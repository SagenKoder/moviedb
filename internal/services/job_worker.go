package services

import (
	"context"
	"fmt"
	"time"
)

// Worker represents a job worker
type Worker struct {
	id         int
	workerPool chan chan *Job
	jobChannel chan *Job
	quit       chan bool
	manager    *JobManager
}

// NewWorker creates a new worker
func NewWorker(id int, workerPool chan chan *Job, quit chan bool, manager *JobManager) *Worker {
	return &Worker{
		id:         id,
		workerPool: workerPool,
		jobChannel: make(chan *Job),
		quit:       quit,
		manager:    manager,
	}
}

// Start starts the worker
func (w *Worker) Start() {
	go func() {
		defer w.manager.wg.Done()
		
		for {
			// Register worker in the worker pool
			fmt.Printf("Worker %d: Registering for work\n", w.id)
			w.workerPool <- w.jobChannel
			
			select {
			case job := <-w.jobChannel:
				fmt.Printf("Worker %d: Received job %d (%s)\n", w.id, job.ID, job.Type)
				w.processJob(job)
			case <-w.quit:
				fmt.Printf("Worker %d stopping\n", w.id)
				return
			}
		}
	}()
}

// processJob processes a single job
func (w *Worker) processJob(job *Job) {
	fmt.Printf("Worker %d processing job %d (%s)\n", w.id, job.ID, job.Type)
	
	// Mark job as running
	w.manager.updateJobStatus(job.ID, JobStatusRunning, "")
	
	// Update started_at timestamp
	_, err := w.manager.db.Exec(`
		UPDATE sync_jobs SET started_at = datetime('now') WHERE id = ?
	`, job.ID)
	if err != nil {
		fmt.Printf("Failed to update job start time: %v\n", err)
	}
	
	// Find processor for this job type
	w.manager.mutex.RLock()
	processor, exists := w.manager.processors[job.Type]
	w.manager.mutex.RUnlock()
	
	if !exists {
		errMsg := fmt.Sprintf("No processor registered for job type: %s", job.Type)
		fmt.Printf("Worker %d: %s\n", w.id, errMsg)
		w.manager.updateJobStatus(job.ID, JobStatusFailed, errMsg)
		return
	}
	
	// Create context with timeout (jobs shouldn't run longer than 2 hours)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	
	// Process the job
	startTime := time.Now()
	err = processor.ProcessJob(ctx, job)
	duration := time.Since(startTime)
	
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			errMsg := "Job timed out after 2 hours"
			fmt.Printf("Worker %d: Job %d timed out\n", w.id, job.ID)
			w.manager.updateJobStatus(job.ID, JobStatusFailed, errMsg)
		} else {
			errMsg := fmt.Sprintf("Job failed: %v", err)
			fmt.Printf("Worker %d: Job %d failed: %v\n", w.id, job.ID, err)
			w.manager.updateJobStatus(job.ID, JobStatusFailed, errMsg)
		}
	} else {
		// Job completed successfully
		fmt.Printf("Worker %d: Job %d completed successfully in %v\n", w.id, job.ID, duration)
		w.manager.updateJobStatus(job.ID, JobStatusCompleted, "")
		
		// Set progress to 100% if not already set
		w.manager.UpdateJobProgress(job.ID, 100, "Completed", 0, 0, 0)
	}
}