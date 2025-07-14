package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// JobType represents different types of background jobs
type JobType string

const (
	JobTypeFullSync     JobType = "full_sync"
	JobTypeLibrarySync  JobType = "library_sync"
	JobTypeTMDBMatching JobType = "tmdb_matching"
	JobTypeCleanup      JobType = "cleanup"
)

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// Job represents a background job
type Job struct {
	ID               int64             `json:"id"`
	Type             JobType           `json:"type"`
	UserID           *int64            `json:"user_id,omitempty"`
	LibraryID        *int64            `json:"library_id,omitempty"`
	Status           JobStatus         `json:"status"`
	Progress         int               `json:"progress"`         // 0-100
	CurrentStep      string            `json:"current_step"`
	TotalItems       int               `json:"total_items"`
	ProcessedItems   int               `json:"processed_items"`
	SuccessfulItems  int               `json:"successful_items"`
	FailedItems      int               `json:"failed_items"`
	ErrorMessage     string            `json:"error_message,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	StartedAt        *time.Time        `json:"started_at,omitempty"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
}

// JobProcessor is the interface that job handlers must implement
type JobProcessor interface {
	ProcessJob(ctx context.Context, job *Job) error
	GetJobType() JobType
}

// JobManager manages background job execution
type JobManager struct {
	db         *sql.DB
	processors map[JobType]JobProcessor
	workers    int
	workerPool chan chan *Job
	jobQueue   chan *Job
	quit       chan bool
	wg         sync.WaitGroup
	mutex      sync.RWMutex
	isRunning  bool
}

// NewJobManager creates a new job manager
func NewJobManager(db *sql.DB, workers int) *JobManager {
	manager := &JobManager{
		db:         db,
		processors: make(map[JobType]JobProcessor),
		workers:    workers,
		workerPool: make(chan chan *Job, workers),
		jobQueue:   make(chan *Job, 100), // Buffer up to 100 jobs
		quit:       make(chan bool),
	}
	
	return manager
}

// DB returns the database connection for validation purposes
func (jm *JobManager) DB() *sql.DB {
	return jm.db
}

// RegisterProcessor registers a job processor for a specific job type
func (jm *JobManager) RegisterProcessor(processor JobProcessor) {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()
	jm.processors[processor.GetJobType()] = processor
}

// Start starts the job manager and worker goroutines
func (jm *JobManager) Start() {
	jm.mutex.Lock()
	if jm.isRunning {
		jm.mutex.Unlock()
		return
	}
	jm.isRunning = true
	jm.mutex.Unlock()
	
	// Start workers
	for i := 0; i < jm.workers; i++ {
		worker := NewWorker(i+1, jm.workerPool, jm.quit, jm)
		worker.Start()
		jm.wg.Add(1)
	}
	
	// Start job dispatcher
	go jm.dispatch()
	
	// Resume any jobs that were running when the system shut down
	go jm.resumePendingJobs()
	
	fmt.Printf("Job manager started with %d workers\n", jm.workers)
}

// Stop gracefully stops the job manager
func (jm *JobManager) Stop() {
	jm.mutex.Lock()
	if !jm.isRunning {
		jm.mutex.Unlock()
		return
	}
	jm.isRunning = false
	jm.mutex.Unlock()
	
	fmt.Println("Stopping job manager...")
	
	// Stop accepting new jobs
	close(jm.quit)
	
	// Wait for all workers to finish
	jm.wg.Wait()
	
	fmt.Println("Job manager stopped")
}

// CreateJob creates a new job in the database
func (jm *JobManager) CreateJob(jobType JobType, userID *int64, libraryID *int64, metadata map[string]interface{}) (*Job, error) {
	metadataJSON := "{}"
	if metadata != nil {
		if data, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(data)
		}
	}
	
	var jobID int64
	err := jm.db.QueryRow(`
		INSERT INTO sync_jobs (type, user_id, library_id, status, metadata_json)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id
	`, jobType, userID, libraryID, JobStatusPending, metadataJSON).Scan(&jobID)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}
	
	job, err := jm.GetJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created job: %w", err)
	}
	
	// Queue the job for processing
	select {
	case jm.jobQueue <- job:
		fmt.Printf("Job %d (%s) queued for processing\n", job.ID, job.Type)
	default:
		// Job queue is full, mark job as failed
		jm.updateJobStatus(job.ID, JobStatusFailed, "Job queue is full")
		return nil, fmt.Errorf("job queue is full")
	}
	
	return job, nil
}

// GetJob retrieves a job by ID
func (jm *JobManager) GetJob(jobID int64) (*Job, error) {
	var job Job
	var userID, libraryID sql.NullInt64
	var currentStep, errorMessage sql.NullString
	var startedAt, completedAt sql.NullString
	var metadataJSON string
	
	err := jm.db.QueryRow(`
		SELECT id, type, user_id, library_id, status, progress, current_step,
			   total_items, processed_items, successful_items, failed_items,
			   error_message, metadata_json, started_at, completed_at, created_at
		FROM sync_jobs WHERE id = ?
	`, jobID).Scan(
		&job.ID, &job.Type, &userID, &libraryID, &job.Status, &job.Progress,
		&currentStep, &job.TotalItems, &job.ProcessedItems, &job.SuccessfulItems,
		&job.FailedItems, &errorMessage, &metadataJSON, &startedAt, &completedAt,
		&job.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Handle nullable fields
	if userID.Valid {
		job.UserID = &userID.Int64
	}
	if libraryID.Valid {
		job.LibraryID = &libraryID.Int64
	}
	if currentStep.Valid {
		job.CurrentStep = currentStep.String
	}
	if errorMessage.Valid {
		job.ErrorMessage = errorMessage.String
	}
	if startedAt.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", startedAt.String); err == nil {
			job.StartedAt = &t
		}
	}
	if completedAt.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", completedAt.String); err == nil {
			job.CompletedAt = &t
		}
	}
	
	// Parse metadata JSON
	if metadataJSON != "" && metadataJSON != "{}" {
		json.Unmarshal([]byte(metadataJSON), &job.Metadata)
	}
	
	return &job, nil
}

// GetUserJobs retrieves all jobs for a specific user
func (jm *JobManager) GetUserJobs(userID int64, limit int) ([]*Job, error) {
	rows, err := jm.db.Query(`
		SELECT id, type, user_id, library_id, status, progress, current_step,
			   total_items, processed_items, successful_items, failed_items,
			   error_message, metadata_json, started_at, completed_at, created_at
		FROM sync_jobs 
		WHERE user_id = ? 
		ORDER BY created_at DESC 
		LIMIT ?
	`, userID, limit)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var jobs []*Job
	for rows.Next() {
		job := &Job{}
		var userID, libraryID sql.NullInt64
		var currentStep, errorMessage sql.NullString
		var startedAt, completedAt sql.NullString
		var metadataJSON string
		
		err := rows.Scan(
			&job.ID, &job.Type, &userID, &libraryID, &job.Status, &job.Progress,
			&currentStep, &job.TotalItems, &job.ProcessedItems, &job.SuccessfulItems,
			&job.FailedItems, &errorMessage, &metadataJSON, &startedAt, &completedAt,
			&job.CreatedAt,
		)
		
		if err != nil {
			return nil, err
		}
		
		// Handle nullable fields (same as GetJob)
		if userID.Valid {
			job.UserID = &userID.Int64
		}
		if libraryID.Valid {
			job.LibraryID = &libraryID.Int64
		}
		if currentStep.Valid {
			job.CurrentStep = currentStep.String
		}
		if errorMessage.Valid {
			job.ErrorMessage = errorMessage.String
		}
		if startedAt.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", startedAt.String); err == nil {
				job.StartedAt = &t
			}
		}
		if completedAt.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", completedAt.String); err == nil {
				job.CompletedAt = &t
			}
		}
		
		// Parse metadata JSON
		if metadataJSON != "" && metadataJSON != "{}" {
			json.Unmarshal([]byte(metadataJSON), &job.Metadata)
		}
		
		jobs = append(jobs, job)
	}
	
	return jobs, nil
}

// UpdateJobProgress updates job progress information
func (jm *JobManager) UpdateJobProgress(jobID int64, progress int, currentStep string, processedItems, successfulItems, failedItems int) error {
	_, err := jm.db.Exec(`
		UPDATE sync_jobs 
		SET progress = ?, current_step = ?, processed_items = ?, 
			successful_items = ?, failed_items = ?
		WHERE id = ?
	`, progress, currentStep, processedItems, successfulItems, failedItems, jobID)
	
	return err
}

// updateJobStatus updates job status and error message
func (jm *JobManager) updateJobStatus(jobID int64, status JobStatus, errorMessage string) error {
	now := time.Now()
	var completedAt *time.Time
	
	if status == JobStatusCompleted || status == JobStatusFailed || status == JobStatusCancelled {
		completedAt = &now
	}
	
	_, err := jm.db.Exec(`
		UPDATE sync_jobs 
		SET status = ?, error_message = ?, completed_at = ?
		WHERE id = ?
	`, status, errorMessage, completedAt, jobID)
	
	return err
}

// dispatch continuously dispatches jobs to available workers
func (jm *JobManager) dispatch() {
	fmt.Println("Job dispatcher started")
	for {
		select {
		case job := <-jm.jobQueue:
			fmt.Printf("Dispatcher: Received job %d (%s) from queue\n", job.ID, job.Type)
			// Wait for an available worker
			go func(job *Job) {
				fmt.Printf("Dispatcher: Waiting for available worker for job %d\n", job.ID)
				worker := <-jm.workerPool
				fmt.Printf("Dispatcher: Dispatching job %d to worker\n", job.ID)
				worker <- job
			}(job)
		case <-jm.quit:
			fmt.Println("Job dispatcher stopping")
			return
		}
	}
}

// resumePendingJobs finds jobs that were running when system shut down and requeues them
func (jm *JobManager) resumePendingJobs() {
	fmt.Println("Checking for pending jobs to resume...")
	rows, err := jm.db.Query(`
		SELECT id FROM sync_jobs 
		WHERE status IN (?, ?) 
		ORDER BY created_at ASC
	`, JobStatusPending, JobStatusRunning)
	
	if err != nil {
		fmt.Printf("Failed to query pending jobs: %v\n", err)
		return
	}
	defer rows.Close()
	
	var resumedCount int
	for rows.Next() {
		var jobID int64
		if err := rows.Scan(&jobID); err != nil {
			continue
		}
		
		fmt.Printf("Found pending job %d, resetting status\n", jobID)
		
		// Reset status to pending
		if err := jm.updateJobStatus(jobID, JobStatusPending, ""); err != nil {
			fmt.Printf("Failed to reset job %d status: %v\n", jobID, err)
			continue
		}
		
		// Load and requeue the job
		if job, err := jm.GetJob(jobID); err == nil {
			fmt.Printf("Requeuing job %d (%s)\n", jobID, job.Type)
			select {
			case jm.jobQueue <- job:
				resumedCount++
				fmt.Printf("Successfully requeued job %d\n", jobID)
			default:
				// Queue full, leave as pending
				fmt.Printf("Job queue full, leaving job %d as pending\n", jobID)
				break
			}
		} else {
			fmt.Printf("Failed to load job %d: %v\n", jobID, err)
		}
	}
	
	if resumedCount > 0 {
		fmt.Printf("Resumed %d pending jobs\n", resumedCount)
	} else {
		fmt.Println("No pending jobs to resume")
	}
}

// CancelJob cancels a running or pending job
func (jm *JobManager) CancelJob(jobID int64) error {
	return jm.updateJobStatus(jobID, JobStatusCancelled, "Job cancelled by user")
}

// CleanupOldJobs removes old completed jobs (older than specified days)
func (jm *JobManager) CleanupOldJobs(daysOld int) error {
	result, err := jm.db.Exec(`
		DELETE FROM sync_jobs 
		WHERE status IN (?, ?, ?) 
		AND created_at < datetime('now', '-' || ? || ' days')
	`, JobStatusCompleted, JobStatusFailed, JobStatusCancelled, daysOld)
	
	if err != nil {
		return err
	}
	
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Cleaned up %d old jobs\n", rowsAffected)
	return nil
}