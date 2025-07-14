package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2"
	"moviedb/internal/auth"
	"moviedb/internal/database"
	"moviedb/internal/services"
)

// PlexSyncEnhancedHandler handles enhanced Plex sync operations
type PlexSyncEnhancedHandler struct {
	syncService    *services.PlexSyncService
	authMiddleware *jwtmiddleware.JWTMiddleware
}

// NewPlexSyncEnhancedHandler creates a new enhanced Plex sync handler
func NewPlexSyncEnhancedHandler(syncService *services.PlexSyncService, authMiddleware *jwtmiddleware.JWTMiddleware) *PlexSyncEnhancedHandler {
	return &PlexSyncEnhancedHandler{
		syncService:    syncService,
		authMiddleware: authMiddleware,
	}
}

// getUserID extracts the user ID from the request using Auth0 authentication
func (h *PlexSyncEnhancedHandler) getUserID(r *http.Request) int64 {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		return 0
	}

	// Get or create user in database to get the numeric user ID
	user, err := database.GetOrCreateUser(
		h.syncService.DB(),
		authUser.Auth0ID,
		authUser.Email,
		authUser.Name,
		authUser.AvatarURL,
	)
	if err != nil {
		return 0
	}

	return int64(user.ID)
}

// TriggerFullSyncResponse represents the response for triggering a full sync
type TriggerFullSyncResponse struct {
	JobID     int64  `json:"job_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

// JobStatusResponse represents the response for job status queries
type JobStatusResponse struct {
	JobID           int64                  `json:"job_id"`
	Type            string                 `json:"type"`
	Status          string                 `json:"status"`
	Progress        int                    `json:"progress"`
	CurrentStep     string                 `json:"current_step"`
	TotalItems      int                    `json:"total_items"`
	ProcessedItems  int                    `json:"processed_items"`
	SuccessfulItems int                    `json:"successful_items"`
	FailedItems     int                    `json:"failed_items"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// UserJobsResponse represents the response for user job history
type UserJobsResponse struct {
	Jobs []JobStatusResponse `json:"jobs"`
}

// LibraryInfo represents library information
type LibraryInfo struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	ItemCount  int    `json:"item_count"`
	ServerName string `json:"server_name"`
	LastSynced string `json:"last_synced"`
	HasAccess  bool   `json:"has_access"`
}

// UserLibrariesResponse represents the response for user libraries
type UserLibrariesResponse struct {
	Libraries []LibraryInfo `json:"libraries"`
}

// TriggerFullSync triggers a full Plex sync for the authenticated user
func (h *PlexSyncEnhancedHandler) TriggerFullSync(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == 0 {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	job, err := h.syncService.TriggerFullSync(userID)
	if err != nil {
		fmt.Printf("Failed to trigger full sync for user %d: %v\n", userID, err)
		http.Error(w, fmt.Sprintf("Failed to trigger sync: %v", err), http.StatusInternalServerError)
		return
	}

	response := TriggerFullSyncResponse{
		JobID:     job.ID,
		Status:    string(job.Status),
		Message:   "Sync job created successfully",
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetJobStatus returns the status of a specific job
func (h *PlexSyncEnhancedHandler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == 0 {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Extract job ID from URL path
	jobIDStr := r.PathValue("jobId")

	// Validate input
	if err := validateInput(jobIDStr, 20, "job ID"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid job ID format", http.StatusBadRequest)
		return
	}

	// Validate user has access to this job
	if err := h.validateUserJobAccess(userID, jobID); err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	job, err := h.syncService.JobManager().GetJob(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	response := JobStatusResponse{
		JobID:           job.ID,
		Type:            string(job.Type),
		Status:          string(job.Status),
		Progress:        job.Progress,
		CurrentStep:     job.CurrentStep,
		TotalItems:      job.TotalItems,
		ProcessedItems:  job.ProcessedItems,
		SuccessfulItems: job.SuccessfulItems,
		FailedItems:     job.FailedItems,
		ErrorMessage:    job.ErrorMessage,
		StartedAt:       job.StartedAt,
		CompletedAt:     job.CompletedAt,
		CreatedAt:       job.CreatedAt,
		Metadata:        job.Metadata,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUserJobs returns the job history for the authenticated user
func (h *PlexSyncEnhancedHandler) GetUserJobs(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == 0 {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get limit from query parameter (default 10)
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		// Validate limit parameter
		if err := validateInput(limitStr, 3, "limit"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		} else {
			http.Error(w, "Invalid limit parameter (must be 1-100)", http.StatusBadRequest)
			return
		}
	}

	jobs, err := h.syncService.JobManager().GetUserJobs(userID, limit)
	if err != nil {
		fmt.Printf("Failed to get user jobs for user %d: %v\n", userID, err)
		http.Error(w, "Failed to get jobs", http.StatusInternalServerError)
		return
	}

	var jobResponses []JobStatusResponse
	for _, job := range jobs {
		jobResponse := JobStatusResponse{
			JobID:           job.ID,
			Type:            string(job.Type),
			Status:          string(job.Status),
			Progress:        job.Progress,
			CurrentStep:     job.CurrentStep,
			TotalItems:      job.TotalItems,
			ProcessedItems:  job.ProcessedItems,
			SuccessfulItems: job.SuccessfulItems,
			FailedItems:     job.FailedItems,
			ErrorMessage:    job.ErrorMessage,
			StartedAt:       job.StartedAt,
			CompletedAt:     job.CompletedAt,
			CreatedAt:       job.CreatedAt,
			Metadata:        job.Metadata,
		}
		jobResponses = append(jobResponses, jobResponse)
	}

	response := UserJobsResponse{
		Jobs: jobResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUserLibraries returns the libraries accessible to the authenticated user
func (h *PlexSyncEnhancedHandler) GetUserLibraries(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == 0 {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	libraries, err := h.getUserLibraries(userID)
	if err != nil {
		fmt.Printf("Failed to get user libraries for user %d: %v\n", userID, err)
		http.Error(w, "Failed to get libraries", http.StatusInternalServerError)
		return
	}

	response := UserLibrariesResponse{
		Libraries: libraries,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CancelJob cancels a running job
func (h *PlexSyncEnhancedHandler) CancelJob(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == 0 {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Extract job ID from URL path
	jobIDStr := r.PathValue("jobId")

	// Validate input
	if err := validateInput(jobIDStr, 20, "job ID"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid job ID format", http.StatusBadRequest)
		return
	}

	// Validate user has access to this job
	if err := h.validateUserJobAccess(userID, jobID); err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Cancel the job
	err = h.syncService.JobManager().CancelJob(jobID)
	if err != nil {
		fmt.Printf("Failed to cancel job %d: %v\n", jobID, err)
		http.Error(w, "Failed to cancel job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// getUserLibraries retrieves libraries accessible to a user
func (h *PlexSyncEnhancedHandler) getUserLibraries(userID int64) ([]LibraryInfo, error) {
	query := `
		SELECT pl.id, pl.title, pl.type, pl.item_count, ps.name as server_name, 
			   pl.last_synced_at, upa.is_active
		FROM plex_libraries pl
		JOIN plex_servers ps ON pl.server_id = ps.id
		JOIN user_plex_access upa ON pl.id = upa.library_id
		WHERE upa.user_id = ?
		ORDER BY ps.name, pl.title
	`

	rows, err := h.syncService.DB().Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libraries []LibraryInfo
	for rows.Next() {
		var library LibraryInfo
		var lastSynced *string

		err := rows.Scan(
			&library.ID,
			&library.Title,
			&library.Type,
			&library.ItemCount,
			&library.ServerName,
			&lastSynced,
			&library.HasAccess,
		)
		if err != nil {
			continue
		}

		if lastSynced != nil {
			library.LastSynced = *lastSynced
		}

		libraries = append(libraries, library)
	}

	return libraries, nil
}

// validateUserJobAccess validates that the user owns the specified job
func (h *PlexSyncEnhancedHandler) validateUserJobAccess(userID int64, jobID int64) error {
	var jobUserID sql.NullInt64
	err := h.syncService.JobManager().DB().QueryRow(`
		SELECT user_id FROM sync_jobs WHERE id = ?
	`, jobID).Scan(&jobUserID)

	if err != nil {
		return fmt.Errorf("job not found")
	}

	if !jobUserID.Valid || jobUserID.Int64 != userID {
		return fmt.Errorf("access denied: job belongs to different user")
	}

	return nil
}

// validateInput performs basic input validation
func validateInput(input string, maxLength int, fieldName string) error {
	if len(input) > maxLength {
		return fmt.Errorf("%s exceeds maximum length of %d characters", fieldName, maxLength)
	}

	// Basic SQL injection prevention (additional to parameterized queries)
	if strings.Contains(strings.ToLower(input), "drop ") ||
		strings.Contains(strings.ToLower(input), "delete ") ||
		strings.Contains(strings.ToLower(input), "truncate ") ||
		strings.Contains(strings.ToLower(input), "alter ") {
		return fmt.Errorf("invalid characters in %s", fieldName)
	}

	return nil
}
