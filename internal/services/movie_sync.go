package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type MovieSyncService struct {
	db         *sql.DB
	tmdbClient *TMDBClient
	ticker     *time.Ticker
	stopChan   chan bool
}

type SyncStatus struct {
	LastSync    time.Time `json:"last_sync"`
	MoviesCount int       `json:"movies_count"`
	IsRunning   bool      `json:"is_running"`
}

func NewMovieSyncService(db *sql.DB, tmdbClient *TMDBClient) *MovieSyncService {
	return &MovieSyncService{
		db:         db,
		tmdbClient: tmdbClient,
		stopChan:   make(chan bool),
	}
}

// StartSyncScheduler starts the automatic daily sync scheduler
func (s *MovieSyncService) StartSyncScheduler() {
	log.Println("Starting movie sync scheduler...")

	// Check if we need to sync immediately (empty table)
	movieCount, err := s.getMovieCount()
	if err != nil {
		log.Printf("Error checking movie count: %v", err)
	} else if movieCount == 0 {
		log.Println("Movies table is empty, starting initial sync...")
		go s.performSync()
	} else {
		log.Printf("Movies table has %d movies, checking last sync...", movieCount)
		if s.shouldSync() {
			log.Println("Starting sync (last sync was more than 24 hours ago)...")
			go s.performSync()
		}
	}

	// Set up daily ticker (24 hours)
	s.ticker = time.NewTicker(24 * time.Hour)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				log.Println("Daily sync triggered...")
				s.performSync()
			case <-s.stopChan:
				log.Println("Movie sync scheduler stopped")
				return
			}
		}
	}()
}

// StopSyncScheduler stops the automatic sync scheduler
func (s *MovieSyncService) StopSyncScheduler() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.stopChan <- true
}

// ManualSync triggers a manual sync (can be called from API)
func (s *MovieSyncService) ManualSync() error {
	log.Println("Manual sync triggered...")
	return s.performSync()
}

// GetSyncStatus returns the current sync status
func (s *MovieSyncService) GetSyncStatus() (*SyncStatus, error) {
	movieCount, err := s.getMovieCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get movie count: %w", err)
	}

	lastSync, err := s.getLastSyncTime()
	if err != nil {
		return nil, fmt.Errorf("failed to get last sync time: %w", err)
	}

	return &SyncStatus{
		LastSync:    lastSync,
		MoviesCount: movieCount,
		IsRunning:   false, // TODO: Track actual sync status
	}, nil
}

func (s *MovieSyncService) performSync() error {
	log.Println("Starting movie sync with TMDB...")
	start := time.Now()

	// Sync popular movies (first 5 pages = ~100 movies)
	if err := s.syncPopularMovies(5); err != nil {
		log.Printf("Error syncing popular movies: %v", err)
		return err
	}

	// Sync trending movies for this week
	if err := s.syncTrendingMovies(); err != nil {
		log.Printf("Error syncing trending movies: %v", err)
		return err
	}

	// Update last sync time
	if err := s.updateLastSyncTime(); err != nil {
		log.Printf("Error updating last sync time: %v", err)
	}

	duration := time.Since(start)
	movieCount, _ := s.getMovieCount()
	log.Printf("Movie sync completed in %v. Total movies: %d", duration, movieCount)

	return nil
}

func (s *MovieSyncService) syncPopularMovies(maxPages int) error {
	for page := 1; page <= maxPages; page++ {
		log.Printf("Syncing popular movies page %d/%d...", page, maxPages)

		resp, err := s.tmdbClient.GetPopularMovies(page)
		if err != nil {
			return fmt.Errorf("failed to get popular movies page %d: %w", page, err)
		}

		for _, tmdbMovie := range resp.Results {
			if err := s.syncMovie(tmdbMovie); err != nil {
				log.Printf("Error syncing movie %s (ID: %d): %v", tmdbMovie.Title, tmdbMovie.ID, err)
				continue
			}
		}

		// Small delay to be nice to TMDB API
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func (s *MovieSyncService) syncTrendingMovies() error {
	log.Println("Syncing trending movies...")

	resp, err := s.tmdbClient.GetTrendingMovies("week")
	if err != nil {
		return fmt.Errorf("failed to get trending movies: %w", err)
	}

	for _, tmdbMovie := range resp.Results {
		if err := s.syncMovie(tmdbMovie); err != nil {
			log.Printf("Error syncing trending movie %s (ID: %d): %v", tmdbMovie.Title, tmdbMovie.ID, err)
			continue
		}
	}

	return nil
}

func (s *MovieSyncService) syncMovie(tmdbMovie TMDBMovie) error {
	// Check if movie already exists
	exists, err := s.movieExists(tmdbMovie.ID)
	if err != nil {
		return fmt.Errorf("failed to check if movie exists: %w", err)
	}

	if exists {
		// Movie exists, update it
		return s.updateMovie(tmdbMovie)
	} else {
		// New movie, insert it
		return s.insertMovie(tmdbMovie)
	}
}

func (s *MovieSyncService) movieExists(tmdbID int) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM movies WHERE tmdb_id = ?", tmdbID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *MovieSyncService) insertMovie(tmdbMovie TMDBMovie) error {
	// Get detailed movie info for runtime and genres
	details, err := s.tmdbClient.GetMovieDetails(tmdbMovie.ID)
	if err != nil {
		log.Printf("Warning: Could not get details for movie %d, using basic info", tmdbMovie.ID)
		details = &TMDBMovieDetails{TMDBMovie: tmdbMovie}
	}

	// Convert genres to JSON
	genresJSON, err := s.convertGenresToJSON(details.Genres)
	if err != nil {
		log.Printf("Warning: Could not convert genres for movie %d: %v", tmdbMovie.ID, err)
		genresJSON = "[]"
	}

	// Get poster URL
	posterURL := s.tmdbClient.GetPosterURL(tmdbMovie.PosterPath, "w500")
	var posterURLPtr *string
	if posterURL != "" {
		posterURLPtr = &posterURL
	}

	// Extract year from release date
	year := ExtractYear(tmdbMovie.ReleaseDate)

	// Insert movie
	_, err = s.db.Exec(`
		INSERT INTO movies (tmdb_id, title, year, poster_url, synopsis, runtime, genres, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, tmdbMovie.ID, tmdbMovie.Title, year, posterURLPtr, tmdbMovie.Overview, 
		details.Runtime, genresJSON, time.Now())

	if err != nil {
		return fmt.Errorf("failed to insert movie: %w", err)
	}

	return nil
}

func (s *MovieSyncService) updateMovie(tmdbMovie TMDBMovie) error {
	// Get detailed movie info
	details, err := s.tmdbClient.GetMovieDetails(tmdbMovie.ID)
	if err != nil {
		log.Printf("Warning: Could not get details for movie %d during update", tmdbMovie.ID)
		return nil // Skip update if we can't get details
	}

	// Convert genres to JSON
	genresJSON, err := s.convertGenresToJSON(details.Genres)
	if err != nil {
		log.Printf("Warning: Could not convert genres for movie %d: %v", tmdbMovie.ID, err)
		genresJSON = "[]"
	}

	// Get poster URL
	posterURL := s.tmdbClient.GetPosterURL(tmdbMovie.PosterPath, "w500")
	var posterURLPtr *string
	if posterURL != "" {
		posterURLPtr = &posterURL
	}

	// Extract year from release date
	year := ExtractYear(tmdbMovie.ReleaseDate)

	// Update movie
	_, err = s.db.Exec(`
		UPDATE movies 
		SET title = ?, year = ?, poster_url = ?, synopsis = ?, runtime = ?, genres = ?
		WHERE tmdb_id = ?
	`, tmdbMovie.Title, year, posterURLPtr, tmdbMovie.Overview, 
		details.Runtime, genresJSON, tmdbMovie.ID)

	if err != nil {
		return fmt.Errorf("failed to update movie: %w", err)
	}

	return nil
}

func (s *MovieSyncService) convertGenresToJSON(genres []Genre) (string, error) {
	if len(genres) == 0 {
		return "[]", nil
	}

	genreNames := make([]string, len(genres))
	for i, genre := range genres {
		genreNames[i] = genre.Name
	}

	jsonBytes, err := json.Marshal(genreNames)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (s *MovieSyncService) getMovieCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM movies").Scan(&count)
	return count, err
}

func (s *MovieSyncService) shouldSync() bool {
	lastSync, err := s.getLastSyncTime()
	if err != nil {
		return true // If we can't determine last sync, sync anyway
	}

	// Sync if last sync was more than 24 hours ago
	return time.Since(lastSync) > 24*time.Hour
}

func (s *MovieSyncService) getLastSyncTime() (time.Time, error) {
	// We'll store the last sync time in a simple key-value table
	// First, create the table if it doesn't exist
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to create app_settings table: %w", err)
	}

	var syncTimeStr string
	err = s.db.QueryRow("SELECT value FROM app_settings WHERE key = 'last_movie_sync'").Scan(&syncTimeStr)
	if err == sql.ErrNoRows {
		// Never synced before
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}

	syncTime, err := time.Parse(time.RFC3339, syncTimeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse sync time: %w", err)
	}

	return syncTime, nil
}

func (s *MovieSyncService) updateLastSyncTime() error {
	// Create the table if it doesn't exist
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create app_settings table: %w", err)
	}

	now := time.Now()
	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO app_settings (key, value, updated_at)
		VALUES ('last_movie_sync', ?, ?)
	`, now.Format(time.RFC3339), now)

	return err
}