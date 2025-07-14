package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PlexIntegrationManager manages all Plex-related services
type PlexIntegrationManager struct {
	db              *sql.DB
	plexgoClient    *PlexgoClient
	tmdbClient      *TMDBClient
	rateLimiter     *TMDBRateLimiter
	jobManager      *JobManager
	syncService     *PlexSyncService
	cleanupService  *PlexCleanupService
}

// NewPlexIntegrationManager creates a new Plex integration manager
func NewPlexIntegrationManager(db *sql.DB, tmdbClient *TMDBClient) *PlexIntegrationManager {
	// Initialize core services
	plexgoClient := NewPlexgoClient()
	rateLimiter := NewTMDBRateLimiter(db)
	jobManager := NewJobManager(db, 3) // 3 worker threads
	
	// Initialize sync service
	syncService := NewPlexSyncService(db, plexgoClient, tmdbClient, rateLimiter, jobManager)
	
	// Initialize cleanup service
	cleanupService := NewPlexCleanupService(db)
	
	manager := &PlexIntegrationManager{
		db:              db,
		plexgoClient:    plexgoClient,
		tmdbClient:      tmdbClient,
		rateLimiter:     rateLimiter,
		jobManager:      jobManager,
		syncService:     syncService,
		cleanupService:  cleanupService,
	}
	
	return manager
}

// SyncService returns the Plex sync service
func (m *PlexIntegrationManager) SyncService() *PlexSyncService {
	return m.syncService
}

// Start starts all background services
func (m *PlexIntegrationManager) Start(ctx context.Context) error {
	fmt.Println("Starting Plex integration services...")
	
	// Start job manager
	m.jobManager.Start()
	
	// Start periodic cleanup (every 6 hours)
	go m.cleanupService.ScheduleCleanup(ctx, 6*time.Hour)
	
	fmt.Println("Plex integration services started successfully")
	return nil
}

// Stop stops all background services
func (m *PlexIntegrationManager) Stop() error {
	fmt.Println("Stopping Plex integration services...")
	
	// Stop rate limiter
	m.rateLimiter.Stop()
	
	// Stop job manager
	m.jobManager.Stop()
	
	fmt.Println("Plex integration services stopped")
	return nil
}

// GetSyncService returns the sync service
func (m *PlexIntegrationManager) GetSyncService() *PlexSyncService {
	return m.syncService
}

// GetJobManager returns the job manager
func (m *PlexIntegrationManager) GetJobManager() *JobManager {
	return m.jobManager
}

// GetCleanupService returns the cleanup service
func (m *PlexIntegrationManager) GetCleanupService() *PlexCleanupService {
	return m.cleanupService
}

// GetPlexgoClient returns the plexgo client
func (m *PlexIntegrationManager) GetPlexgoClient() *PlexgoClient {
	return m.plexgoClient
}

// GetRateLimiter returns the rate limiter
func (m *PlexIntegrationManager) GetRateLimiter() *TMDBRateLimiter {
	return m.rateLimiter
}

// GetHealthStatus returns the health status of all services
func (m *PlexIntegrationManager) GetHealthStatus() map[string]interface{} {
	status := make(map[string]interface{})
	
	// Rate limiter stats
	status["rate_limiter"] = m.rateLimiter.GetStats()
	
	// Job manager stats
	status["job_manager"] = map[string]interface{}{
		"workers": m.jobManager.workers,
		"running": m.jobManager.isRunning,
	}
	
	// Cleanup stats
	if cleanupStats, err := m.cleanupService.GetCleanupStats(context.Background()); err == nil {
		status["cleanup"] = cleanupStats
	}
	
	return status
}

// TriggerUserSync triggers a sync for a specific user
func (m *PlexIntegrationManager) TriggerUserSync(userID int64) (*Job, error) {
	return m.syncService.TriggerFullSync(userID)
}

// RunManualCleanup runs a manual cleanup operation
func (m *PlexIntegrationManager) RunManualCleanup() error {
	return m.cleanupService.RunFullCleanup(context.Background())
}

// GetUserLibraries returns libraries accessible to a user
func (m *PlexIntegrationManager) GetUserLibraries(userID int64) ([]LibraryInfo, error) {
	query := `
		SELECT pl.id, pl.title, pl.type, pl.item_count, ps.name as server_name, 
			   pl.last_synced_at, upa.is_active
		FROM plex_libraries pl
		JOIN plex_servers ps ON pl.server_id = ps.id
		JOIN user_plex_access upa ON pl.id = upa.library_id
		WHERE upa.user_id = ? AND upa.is_active = 1
		ORDER BY ps.name, pl.title
	`

	rows, err := m.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libraries []LibraryInfo
	for rows.Next() {
		var library LibraryInfo
		var lastSynced sql.NullString

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

		if lastSynced.Valid {
			library.LastSynced = lastSynced.String
		}

		libraries = append(libraries, library)
	}

	return libraries, nil
}

// LibraryInfo represents library information for API responses
type LibraryInfo struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	ItemCount   int    `json:"item_count"`
	ServerName  string `json:"server_name"`
	LastSynced  string `json:"last_synced"`
	HasAccess   bool   `json:"has_access"`
}

// GetUserPlexStats returns Plex statistics for a user
func (m *PlexIntegrationManager) GetUserPlexStats(userID int64) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Count accessible libraries
	var libraryCount int
	err := m.db.QueryRow(`
		SELECT COUNT(*) FROM user_plex_access 
		WHERE user_id = ? AND is_active = 1
	`, userID).Scan(&libraryCount)
	if err != nil {
		return nil, err
	}
	stats["accessible_libraries"] = libraryCount
	
	// Count available movies
	var movieCount int
	err = m.db.QueryRow(`
		SELECT COUNT(*) FROM plex_library_items pli
		JOIN plex_libraries pl ON pli.library_id = pl.id
		JOIN user_plex_access upa ON pl.id = upa.library_id
		WHERE upa.user_id = ? AND upa.is_active = 1 
		AND pli.is_active = 1 AND pli.type = 'movie'
	`, userID).Scan(&movieCount)
	if err != nil {
		return nil, err
	}
	stats["available_movies"] = movieCount
	
	// Count matched movies
	var matchedCount int
	err = m.db.QueryRow(`
		SELECT COUNT(*) FROM plex_library_items pli
		JOIN plex_libraries pl ON pli.library_id = pl.id
		JOIN user_plex_access upa ON pl.id = upa.library_id
		WHERE upa.user_id = ? AND upa.is_active = 1 
		AND pli.is_active = 1 AND pli.type = 'movie' AND pli.tmdb_id IS NOT NULL
	`, userID).Scan(&matchedCount)
	if err != nil {
		return nil, err
	}
	stats["matched_movies"] = matchedCount
	
	// Get last sync time
	var lastSync sql.NullString
	err = m.db.QueryRow(`
		SELECT completed_at FROM sync_jobs 
		WHERE user_id = ? AND status = 'completed' 
		ORDER BY completed_at DESC LIMIT 1
	`, userID).Scan(&lastSync)
	if err == nil && lastSync.Valid {
		stats["last_sync"] = lastSync.String
	}
	
	return stats, nil
}

// ProcessTMDBMatching processes a batch of TMDB matching requests
func (m *PlexIntegrationManager) ProcessTMDBMatching(userID int64, batchSize int) error {
	// Get unmatched items for this user
	rows, err := m.db.Query(`
		SELECT pli.id, pli.title, pli.year, pli.plex_guid
		FROM plex_library_items pli
		JOIN plex_libraries pl ON pli.library_id = pl.id
		JOIN user_plex_access upa ON pl.id = upa.library_id
		WHERE upa.user_id = ? AND pli.tmdb_id IS NULL AND pli.is_active = 1
		AND (pli.last_matched_at IS NULL OR pli.matching_attempts < 3)
		ORDER BY pli.created_at DESC
		LIMIT ?
	`, userID, batchSize)
	
	if err != nil {
		return fmt.Errorf("failed to query unmatched items: %w", err)
	}
	defer rows.Close()
	
	var items []struct {
		ID       int64
		Title    string
		Year     *int
		PlexGUID string
	}
	
	for rows.Next() {
		var item struct {
			ID       int64
			Title    string
			Year     *int
			PlexGUID string
		}
		
		if err := rows.Scan(&item.ID, &item.Title, &item.Year, &item.PlexGUID); err != nil {
			continue
		}
		
		items = append(items, item)
	}
	
	// Process each item with rate limiting
	for _, item := range items {
		err := m.rateLimiter.ExecuteWithRateLimit(func() error {
			// This would call the TMDB matching logic
			// For now, just simulate the rate-limited operation
			time.Sleep(50 * time.Millisecond) // Simulate API call
			return nil
		}, 1) // Priority 1 for user-triggered operations
		
		if err != nil {
			fmt.Printf("Failed to match item %s: %v\n", item.Title, err)
			continue
		}
	}
	
	return nil
}