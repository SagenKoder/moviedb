package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// PlexSyncService handles comprehensive Plex library synchronization
type PlexSyncService struct {
	db           *sql.DB
	plexgoClient *PlexgoClient
	tmdbClient   *TMDBClient
	rateLimiter  *TMDBRateLimiter
	jobManager   *JobManager
}

// PlexSyncJobProcessor implements JobProcessor for Plex sync operations
type PlexSyncJobProcessor struct {
	syncService *PlexSyncService
}

// NewPlexSyncService creates a new Plex sync service
func NewPlexSyncService(db *sql.DB, plexgoClient *PlexgoClient, tmdbClient *TMDBClient, rateLimiter *TMDBRateLimiter, jobManager *JobManager) *PlexSyncService {
	service := &PlexSyncService{
		db:           db,
		plexgoClient: plexgoClient,
		tmdbClient:   tmdbClient,
		rateLimiter:  rateLimiter,
		jobManager:   jobManager,
	}

	// Register job processor
	processor := &PlexSyncJobProcessor{syncService: service}
	jobManager.RegisterProcessor(processor)

	return service
}

// DB returns the database connection for validation purposes
func (s *PlexSyncService) DB() *sql.DB {
	return s.db
}

// JobManager returns the job manager for external access
func (s *PlexSyncService) JobManager() *JobManager {
	return s.jobManager
}

// GetJobType returns the job type this processor handles
func (p *PlexSyncJobProcessor) GetJobType() JobType {
	return JobTypeFullSync
}

// ProcessJob processes a full sync job
func (p *PlexSyncJobProcessor) ProcessJob(ctx context.Context, job *Job) error {
	fmt.Printf("PlexSyncJobProcessor: Starting to process job %d\n", job.ID)
	
	if job.UserID == nil {
		fmt.Printf("PlexSyncJobProcessor: Job %d missing user ID\n", job.ID)
		return fmt.Errorf("user ID is required for sync job")
	}

	fmt.Printf("PlexSyncJobProcessor: Processing full sync for user %d, job %d\n", *job.UserID, job.ID)
	err := p.syncService.PerformFullSync(ctx, *job.UserID, job.ID)
	
	if err != nil {
		fmt.Printf("PlexSyncJobProcessor: Job %d failed: %v\n", job.ID, err)
	} else {
		fmt.Printf("PlexSyncJobProcessor: Job %d completed successfully\n", job.ID)
	}
	
	return err
}

// TriggerFullSync creates a new full sync job for a user
func (s *PlexSyncService) TriggerFullSync(userID int64) (*Job, error) {
	// Check if there's already a running sync for this user
	var existingJobID int64
	err := s.db.QueryRow(`
		SELECT id FROM sync_jobs 
		WHERE user_id = ? AND type = ? AND status IN (?, ?)
		ORDER BY created_at DESC LIMIT 1
	`, userID, JobTypeFullSync, JobStatusPending, JobStatusRunning).Scan(&existingJobID)

	if err == nil {
		return nil, fmt.Errorf("sync already in progress for user %d (job %d)", userID, existingJobID)
	}

	// Create new sync job
	metadata := map[string]interface{}{
		"sync_type": "full",
		"user_id":   userID,
	}

	job, err := s.jobManager.CreateJob(JobTypeFullSync, &userID, nil, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync job: %w", err)
	}

	return job, nil
}

// PerformFullSync performs a complete sync for a user
func (s *PlexSyncService) PerformFullSync(ctx context.Context, userID int64, jobID int64) error {
	fmt.Printf("Starting full Plex sync for user %d\n", userID)

	// Get user's Plex token
	var plexToken string
	err := s.db.QueryRow(`SELECT plex_token FROM user_plex_tokens WHERE user_id = ?`, userID).Scan(&plexToken)
	if err != nil {
		return fmt.Errorf("failed to get Plex token: %w", err)
	}

	// Phase 1: Server and Library Discovery
	s.jobManager.UpdateJobProgress(jobID, 10, "Discovering Plex servers and libraries", 0, 0, 0)

	serverLibraries, err := s.discoverUserLibraries(ctx, plexToken, userID)
	if err != nil {
		return fmt.Errorf("failed to discover libraries: %w", err)
	}

	fmt.Printf("DEBUG: [PerformFullSync] Found %d libraries from discovery\n", len(serverLibraries))
	for i, lib := range serverLibraries {
		fmt.Printf("DEBUG: [PerformFullSync] Library %d: %s (Type: %s)\n", i, lib.Title, lib.Type)
	}

	if len(serverLibraries) == 0 {
		s.jobManager.UpdateJobProgress(jobID, 100, "No accessible libraries found", 0, 0, 0)
		return nil
	}

	// Phase 2: Sync Library Contents
	s.jobManager.UpdateJobProgress(jobID, 20, "Syncing library contents", 0, 0, 0)

	totalItems := 0
	processedItems := 0
	successfulItems := 0
	failedItems := 0

	for _, library := range serverLibraries {
		fmt.Printf("DEBUG: [PerformFullSync] Found library: %s (Type: %s)\n", library.Title, library.Type)
		
		// Only sync movie libraries for now
		if library.Type != "movie" {
			fmt.Printf("DEBUG: [PerformFullSync] Skipping non-movie library: %s\n", library.Title)
			continue
		}

		fmt.Printf("Syncing library: %s (%s)\n", library.Title, library.Type)

		// Sync this library using its server-specific access token
		items, err := s.syncLibraryItems(ctx, library.AccessToken, library, jobID)
		if err != nil {
			fmt.Printf("Failed to sync library %s: %v\n", library.Title, err)
			failedItems++
			continue
		}

		totalItems += len(items)
		processedItems += len(items)
		successfulItems += len(items)

		// Update progress
		progress := 20 + (processedItems * 60 / max(totalItems, 1))
		s.jobManager.UpdateJobProgress(jobID, progress, fmt.Sprintf("Synced library: %s", library.Title), processedItems, successfulItems, failedItems)
	}

	fmt.Printf("DEBUG: [PerformFullSync] Library sync completed, starting TMDB matching phase\n")
	
	// Phase 3: TMDB Matching
	s.jobManager.UpdateJobProgress(jobID, 80, "Matching items with TMDB", processedItems, successfulItems, failedItems)

	fmt.Printf("DEBUG: [PerformFullSync] About to call performTMDBMatching for user %d\n", userID)
	matchedItems, err := s.performTMDBMatching(ctx, userID, jobID)
	if err != nil {
		fmt.Printf("TMDB matching failed: %v\n", err)
		// Don't fail the entire sync for TMDB matching issues
	}
	fmt.Printf("DEBUG: [PerformFullSync] TMDB matching returned %d matched items\n", matchedItems)

	// Phase 4: Cleanup
	s.jobManager.UpdateJobProgress(jobID, 95, "Cleaning up removed items", processedItems, successfulItems, failedItems)

	err = s.cleanupRemovedItems(ctx, userID)
	if err != nil {
		fmt.Printf("Cleanup failed: %v\n", err)
		// Don't fail the entire sync for cleanup issues
	}

	// Final progress update
	s.jobManager.UpdateJobProgress(jobID, 100, "Sync completed", processedItems, successfulItems, failedItems)

	fmt.Printf("Full sync completed for user %d: %d items processed, %d successful, %d failed, %d TMDB matched\n",
		userID, processedItems, successfulItems, failedItems, matchedItems)

	return nil
}

// discoverUserLibraries discovers all servers and libraries accessible to a user
func (s *PlexSyncService) discoverUserLibraries(ctx context.Context, plexToken string, userID int64) ([]PlexLibrary, error) {
	// Get user's accessible servers
	servers, err := s.plexgoClient.GetServers(ctx, plexToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}

	var allLibraries []PlexLibrary

	for _, server := range servers {
		// Store or update server in database
		serverID, err := s.storeServer(server)
		if err != nil {
			fmt.Printf("Failed to store server %s: %v\n", server.Name, err)
			continue
		}

		// Get best connection for this server
		bestConnection := s.plexgoClient.GetBestConnection(server)
		if bestConnection == nil {
			fmt.Printf("No accessible connection for server %s\n", server.Name)
			continue
		}

		serverURL := s.plexgoClient.BuildServerURL(*bestConnection)

		// Get libraries for this server using the server-specific access token
		libraries, err := s.plexgoClient.GetLibraries(ctx, server.AccessToken, serverURL)
		if err != nil {
			fmt.Printf("Failed to get libraries for server %s: %v\n", server.Name, err)
			continue
		}

		// Store libraries and user access
		for _, library := range libraries {
			library.ServerID = serverID
			library.ServerURL = serverURL
			library.AccessToken = server.AccessToken  // Store server-specific token

			// Store library in database
			libraryID, err := s.storeLibrary(library)
			if err != nil {
				fmt.Printf("Failed to store library %s: %v\n", library.Title, err)
				continue
			}

			// Record user access to this library
			err = s.recordUserAccess(userID, libraryID)
			if err != nil {
				fmt.Printf("Failed to record user access to library %s: %v\n", library.Title, err)
			}

			library.ID = libraryID
			allLibraries = append(allLibraries, library)
		}
	}

	return allLibraries, nil
}

// storeServer stores or updates a Plex server in the database
func (s *PlexSyncService) storeServer(server PlexServer) (int64, error) {
	var serverID int64

	// Try to get existing server
	err := s.db.QueryRow(`
		SELECT id FROM plex_servers WHERE machine_id = ?
	`, server.MachineID).Scan(&serverID)

	if err == sql.ErrNoRows {
		// Create new server
		err = s.db.QueryRow(`
			INSERT INTO plex_servers (machine_id, name, platform, version, last_synced_at, updated_at)
			VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
			RETURNING id
		`, server.MachineID, server.Name, server.Platform, server.ProductVersion).Scan(&serverID)

		if err != nil {
			return 0, fmt.Errorf("failed to create server: %w", err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("failed to query server: %w", err)
	} else {
		// Update existing server
		_, err = s.db.Exec(`
			UPDATE plex_servers 
			SET name = ?, platform = ?, version = ?, last_synced_at = datetime('now'), updated_at = datetime('now')
			WHERE id = ?
		`, server.Name, server.Platform, server.ProductVersion, serverID)

		if err != nil {
			return 0, fmt.Errorf("failed to update server: %w", err)
		}
	}

	return serverID, nil
}

// storeLibrary stores or updates a Plex library in the database
func (s *PlexSyncService) storeLibrary(library PlexLibrary) (int64, error) {
	var libraryID int64

	// Try to get existing library
	err := s.db.QueryRow(`
		SELECT id FROM plex_libraries WHERE server_id = ? AND section_key = ?
	`, library.ServerID, library.Key).Scan(&libraryID)

	if err == sql.ErrNoRows {
		// Create new library
		err = s.db.QueryRow(`
			INSERT INTO plex_libraries (server_id, section_key, title, type, agent, scanner, language, uuid, last_synced_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
			RETURNING id
		`, library.ServerID, library.Key, library.Title, library.Type, library.Agent, library.Scanner, library.Language, library.UUID).Scan(&libraryID)

		if err != nil {
			return 0, fmt.Errorf("failed to create library: %w", err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("failed to query library: %w", err)
	} else {
		// Update existing library
		_, err = s.db.Exec(`
			UPDATE plex_libraries 
			SET title = ?, type = ?, agent = ?, scanner = ?, language = ?, uuid = ?, last_synced_at = datetime('now'), updated_at = datetime('now')
			WHERE id = ?
		`, library.Title, library.Type, library.Agent, library.Scanner, library.Language, library.UUID, libraryID)

		if err != nil {
			return 0, fmt.Errorf("failed to update library: %w", err)
		}
	}

	return libraryID, nil
}

// recordUserAccess records or updates user access to a library
func (s *PlexSyncService) recordUserAccess(userID, libraryID int64) error {
	_, err := s.db.Exec(`
		INSERT INTO user_plex_access (user_id, library_id, access_level, last_verified_at)
		VALUES (?, ?, 'read', datetime('now'))
		ON CONFLICT(user_id, library_id) DO UPDATE SET
			last_verified_at = datetime('now'),
			is_active = 1
	`, userID, libraryID)

	return err
}

// syncLibraryItems syncs all items in a library
func (s *PlexSyncService) syncLibraryItems(ctx context.Context, plexToken string, library PlexLibrary, jobID int64) ([]PlexSearchResult, error) {
	items, err := s.plexgoClient.GetMoviesInLibrary(ctx, plexToken, library.ServerURL, library.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to get library items: %w", err)
	}

	for _, item := range items {
		// Store item in database
		err = s.storeLibraryItem(library.ID, item)
		if err != nil {
			fmt.Printf("Failed to store item %s: %v\n", item.Title, err)
			continue
		}
	}

	// Update library item count
	_, err = s.db.Exec(`
		UPDATE plex_libraries SET item_count = ? WHERE id = ?
	`, len(items), library.ID)

	if err != nil {
		fmt.Printf("Failed to update library item count: %v\n", err)
	}

	return items, nil
}

// storeLibraryItem stores or updates a library item
func (s *PlexSyncService) storeLibraryItem(libraryID int64, item PlexSearchResult) error {
	// Convert item to JSON for metadata storage
	metadata, _ := json.Marshal(item)

	// Use the actual rating key from the Plex API response
	ratingKey := item.RatingKey

	_, err := s.db.Exec(`
		INSERT INTO plex_library_items (library_id, plex_rating_key, plex_guid, title, year, type, metadata_json, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(library_id, plex_rating_key) DO UPDATE SET
			title = excluded.title,
			year = excluded.year,
			type = excluded.type,
			metadata_json = excluded.metadata_json,
			updated_at = datetime('now'),
			is_active = 1
	`, libraryID, ratingKey, item.GUID, item.Title, item.Year, item.Type, string(metadata))

	return err
}

// performTMDBMatching matches Plex items with TMDB using rate limiting
func (s *PlexSyncService) performTMDBMatching(ctx context.Context, userID int64, jobID int64) (int, error) {
	fmt.Printf("DEBUG: [performTMDBMatching] Starting TMDB matching for user %d\n", userID)
	
	// Debug: Check total items in database
	var totalItems int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM plex_library_items WHERE is_active = 1`).Scan(&totalItems)
	if err != nil {
		fmt.Printf("DEBUG: [performTMDBMatching] Error counting total items: %v\n", err)
	} else {
		fmt.Printf("DEBUG: [performTMDBMatching] Total active items in database: %d\n", totalItems)
	}
	
	// Debug: Check user access entries
	var userAccessCount int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM user_plex_access WHERE user_id = ? AND is_active = 1`, userID).Scan(&userAccessCount)
	if err != nil {
		fmt.Printf("DEBUG: [performTMDBMatching] Error counting user access: %v\n", err)
	} else {
		fmt.Printf("DEBUG: [performTMDBMatching] User %d has access to %d libraries\n", userID, userAccessCount)
	}
	
	// Get unmatched items
	rows, err := s.db.Query(`
		SELECT pli.id, pli.title, pli.year, pli.plex_guid
		FROM plex_library_items pli
		JOIN plex_libraries pl ON pli.library_id = pl.id
		JOIN user_plex_access upa ON pl.id = upa.library_id
		WHERE upa.user_id = ? AND pli.tmdb_id IS NULL AND pli.is_active = 1
		AND (pli.last_matched_at IS NULL OR pli.matching_attempts < 3)
		ORDER BY pli.created_at DESC
	`, userID)

	if err != nil {
		return 0, fmt.Errorf("failed to query unmatched items: %w", err)
	}
	defer rows.Close()

	var unmatchedItems []struct {
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

		err := rows.Scan(&item.ID, &item.Title, &item.Year, &item.PlexGUID)
		if err != nil {
			continue
		}

		unmatchedItems = append(unmatchedItems, item)
	}

	fmt.Printf("DEBUG: [performTMDBMatching] Found %d unmatched items for user %d\n", len(unmatchedItems), userID)
	
	matchedCount := 0

	for i, item := range unmatchedItems {
		// Update progress
		progress := 80 + (i * 15 / max(len(unmatchedItems), 1))
		s.jobManager.UpdateJobProgress(jobID, progress, fmt.Sprintf("Matching with TMDB: %s", item.Title), 0, 0, 0)

		// Try to match with TMDB using rate limiting
		err := s.rateLimiter.ExecuteWithRateLimit(func() error {
			return s.matchItemWithTMDB(item.ID, item.Title, item.Year, item.PlexGUID)
		}, 0) // Priority 0 for background sync

		if err != nil {
			fmt.Printf("Failed to match %s with TMDB: %v\n", item.Title, err)
			// Update attempt count
			s.db.Exec(`
				UPDATE plex_library_items 
				SET matching_attempts = matching_attempts + 1, last_matched_at = datetime('now')
				WHERE id = ?
			`, item.ID)
		} else {
			matchedCount++
		}
	}

	return matchedCount, nil
}

// matchItemWithTMDB attempts to match a Plex item with TMDB
func (s *PlexSyncService) matchItemWithTMDB(itemID int64, title string, year *int, plexGUID string) error {
	// Try to extract TMDB ID from Plex GUID first
	if tmdbID := extractTMDBFromGUID(plexGUID); tmdbID > 0 {
		// Verify the movie exists in TMDB
		movie, err := s.tmdbClient.GetMovieDetails(tmdbID)
		if err == nil {
			// Update the item with TMDB ID
			_, err = s.db.Exec(`
				UPDATE plex_library_items 
				SET tmdb_id = ?, last_matched_at = datetime('now')
				WHERE id = ?
			`, tmdbID, itemID)

			if err == nil {
				// Also add to movies table if not exists
				s.storeMovieFromTMDB(movie)
				return nil
			}
		}
	}

	// Fallback to search by title and year
	yearInt := 0
	if year != nil {
		yearInt = *year
	}

	searchResp, err := s.tmdbClient.SearchMovies(title, yearInt)
	if err != nil {
		return fmt.Errorf("TMDB search failed: %w", err)
	}

	if len(searchResp.Results) == 0 {
		return fmt.Errorf("no TMDB matches found for %s (%d)", title, yearInt)
	}

	// Use the first match (most relevant)
	bestMatch := searchResp.Results[0]

	// Store movie in movies table first (to satisfy foreign key constraint)
	err = s.storeMovieFromTMDB(bestMatch)
	if err != nil {
		return fmt.Errorf("failed to store movie from TMDB: %w", err)
	}

	// Update the item with TMDB ID
	_, err = s.db.Exec(`
		UPDATE plex_library_items 
		SET tmdb_id = ?, last_matched_at = datetime('now')
		WHERE id = ?
	`, bestMatch.ID, itemID)

	if err != nil {
		return fmt.Errorf("failed to update item with TMDB ID: %w", err)
	}

	return nil
}

// storeMovieFromTMDB stores a movie from TMDB API response
func (s *PlexSyncService) storeMovieFromTMDB(movie interface{}) error {
	// Handle both TMDBMovie and TMDBMovieDetails types
	var tmdbID int
	var title string
	var posterURL string
	var synopsis string
	var runtime *int
	var year *int
	var genresJSON string = "[]"

	switch m := movie.(type) {
	case TMDBMovie:
		tmdbID = m.ID
		title = m.Title
		synopsis = m.Overview
		if m.PosterPath != nil && *m.PosterPath != "" {
			posterURL = "https://image.tmdb.org/t/p/w500" + *m.PosterPath
		}
		if m.ReleaseDate != "" && len(m.ReleaseDate) >= 4 {
			if parsedYear, err := strconv.Atoi(m.ReleaseDate[:4]); err == nil {
				year = &parsedYear
			}
		}
		
	case *TMDBMovieDetails:
		tmdbID = m.ID
		title = m.Title
		synopsis = m.Overview
		if m.PosterPath != nil && *m.PosterPath != "" {
			posterURL = "https://image.tmdb.org/t/p/w500" + *m.PosterPath
		}
		if m.ReleaseDate != "" && len(m.ReleaseDate) >= 4 {
			if parsedYear, err := strconv.Atoi(m.ReleaseDate[:4]); err == nil {
				year = &parsedYear
			}
		}
		if m.Runtime > 0 {
			runtime = &m.Runtime
		}
		// Handle genres for details
		if len(m.Genres) > 0 {
			genreNames := make([]string, 0, len(m.Genres))
			for _, genre := range m.Genres {
				genreNames = append(genreNames, genre.Name)
			}
			if genresBytes, err := json.Marshal(genreNames); err == nil {
				genresJSON = string(genresBytes)
			}
		}
		
	default:
		return fmt.Errorf("unsupported movie data type: %T", movie)
	}

	// Insert or update movie in database
	_, err := s.db.Exec(`
		INSERT INTO movies (tmdb_id, title, year, poster_url, synopsis, runtime, genres, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(tmdb_id) DO UPDATE SET
			title = excluded.title,
			year = excluded.year,
			poster_url = excluded.poster_url,
			synopsis = excluded.synopsis,
			runtime = excluded.runtime,
			genres = excluded.genres
	`, tmdbID, title, year, posterURL, synopsis, runtime, genresJSON)

	if err != nil {
		return fmt.Errorf("failed to store movie in database: %w", err)
	}

	return nil
}

// cleanupRemovedItems removes items that are no longer in Plex
func (s *PlexSyncService) cleanupRemovedItems(ctx context.Context, userID int64) error {
	// Mark items as inactive if they weren't updated in the last sync
	// This is a simplified approach - in a real implementation, you'd want to
	// actually check if the item still exists in Plex

	_, err := s.db.Exec(`
		UPDATE plex_library_items 
		SET is_active = 0
		WHERE library_id IN (
			SELECT pl.id FROM plex_libraries pl
			JOIN user_plex_access upa ON pl.id = upa.library_id
			WHERE upa.user_id = ? AND upa.is_active = 1
		) AND updated_at < datetime('now', '-1 hour')
	`, userID)

	return err
}

// Helper functions
func extractTMDBFromGUID(plexGUID string) int {
	// Extract TMDB ID from Plex GUID formats like:
	// com.plexapp.agents.themoviedb://12345
	// plex://movie/5d776b59ad5437001f79c6f8

	if strings.Contains(plexGUID, "themoviedb://") {
		parts := strings.Split(plexGUID, "://")
		if len(parts) == 2 {
			if id, err := strconv.Atoi(parts[1]); err == nil {
				return id
			}
		}
	}

	return 0
}

func getYear(year *int) int {
	if year == nil {
		return 0
	}
	return *year
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
