package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PlexIntegrationManager manages all Plex-related services
type PlexIntegrationManager struct {
	db             *sql.DB
	plexgoClient   *PlexgoClient
	tmdbClient     *TMDBClient
	rateLimiter    *TMDBRateLimiter
	jobManager     *JobManager
	syncService    *PlexSyncService
	cleanupService *PlexCleanupService
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
		db:             db,
		plexgoClient:   plexgoClient,
		tmdbClient:     tmdbClient,
		rateLimiter:    rateLimiter,
		jobManager:     jobManager,
		syncService:    syncService,
		cleanupService: cleanupService,
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

// LibraryInfo represents library information for API responses
type LibraryInfo struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	ItemCount  int    `json:"item_count"`
	ServerName string `json:"server_name"`
	LastSynced string `json:"last_synced"`
	HasAccess  bool   `json:"has_access"`
}
