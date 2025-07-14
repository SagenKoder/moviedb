package services

import (
	"database/sql"
	"fmt"
	"time"
)

type WatchProvidersService struct {
	db           *sql.DB
	tmdbClient   *TMDBClient
	plexClient   *PlexClient   // Keep for backward compatibility
	plexgoClient *PlexgoClient // Use for new permission-aware operations
}

// WatchProvider represents a unified watch provider (TMDB + Plex)
type WatchProvider struct {
	Name         string  `json:"name"`
	LogoPath     string  `json:"logoPath,omitempty"`
	ProviderType string  `json:"providerType"` // "flatrate", "rent", "buy", "free", "plex"
	Price        *string `json:"price,omitempty"`
	Link         string  `json:"link,omitempty"`
	PlexServer   string  `json:"plexServer,omitempty"`  // For Plex providers
	PlexURL      string  `json:"plexUrl,omitempty"`     // Direct Plex URL to launch movie
	LibraryName  string  `json:"libraryName,omitempty"` // Plex library name
}

// WatchProvidersResponse represents the combined response
type WatchProvidersResponse struct {
	TMDBID        int             `json:"tmdbId"`
	Region        string          `json:"region"`
	TMDBLink      string          `json:"tmdbLink,omitempty"`
	Providers     []WatchProvider `json:"providers"`
	PlexAvailable bool            `json:"plexAvailable"`
	CachedAt      time.Time       `json:"cachedAt"`
	ExpiresAt     time.Time       `json:"expiresAt"`
}

func NewWatchProvidersService(db *sql.DB, tmdbClient *TMDBClient, plexClient *PlexClient) *WatchProvidersService {
	return &WatchProvidersService{
		db:           db,
		tmdbClient:   tmdbClient,
		plexClient:   plexClient,        // Keep for backward compatibility during migration
		plexgoClient: NewPlexgoClient(), // Primary client for all operations
	}
}

// GetWatchProviders gets watch provider information with caching
func (s *WatchProvidersService) GetWatchProviders(tmdbID int, region string, userID *int) (*WatchProvidersResponse, error) {
	if region == "" {
		region = "US" // Default to US
	}

	// TEMPORARILY DISABLE CACHE - Try to get from cache first
	// cached, err := s.getCachedWatchProviders(tmdbID, region)
	// if err == nil && cached.ExpiresAt.After(time.Now()) {
	// 	// Add Plex availability if user is provided
	// 	if userID != nil {
	// 		plexAvailable, plexProviders, err := s.getPlexAvailability(tmdbID, *userID)
	// 		if err == nil {
	// 			cached.PlexAvailable = plexAvailable
	// 			// Add Plex providers to the list
	// 			cached.Providers = append(cached.Providers, plexProviders...)
	// 		}
	// 	}
	// 	return cached, nil
	// }

	fmt.Printf("DEBUG: CACHE DISABLED - Forcing fresh lookup for TMDB ID %d\n", tmdbID)

	// Fetch fresh data from TMDB
	tmdbProviders, err := s.tmdbClient.GetMovieWatchProviders(tmdbID)
	if err != nil {
		return nil, fmt.Errorf("failed to get TMDB watch providers: %w", err)
	}

	// Convert TMDB data to our format
	response := &WatchProvidersResponse{
		TMDBID:    tmdbID,
		Region:    region,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(48 * time.Hour), // 48 hour cache
		Providers: []WatchProvider{},
	}

	// Process region-specific providers
	if regionData, exists := tmdbProviders.Results[region]; exists {
		response.TMDBLink = regionData.Link

		// Add flatrate providers (subscriptions like Netflix)
		for _, provider := range regionData.Flatrate {
			response.Providers = append(response.Providers, WatchProvider{
				Name:         provider.ProviderName,
				LogoPath:     s.tmdbClient.GetPosterURL(&provider.LogoPath, "w92"),
				ProviderType: "flatrate",
				Link:         regionData.Link,
			})
		}

		// Add rent providers
		for _, provider := range regionData.Rent {
			response.Providers = append(response.Providers, WatchProvider{
				Name:         provider.ProviderName,
				LogoPath:     s.tmdbClient.GetPosterURL(&provider.LogoPath, "w92"),
				ProviderType: "rent",
				Link:         regionData.Link,
			})
		}

		// Add buy providers
		for _, provider := range regionData.Buy {
			response.Providers = append(response.Providers, WatchProvider{
				Name:         provider.ProviderName,
				LogoPath:     s.tmdbClient.GetPosterURL(&provider.LogoPath, "w92"),
				ProviderType: "buy",
				Link:         regionData.Link,
			})
		}

		// Add free providers
		for _, provider := range regionData.Free {
			response.Providers = append(response.Providers, WatchProvider{
				Name:         provider.ProviderName,
				LogoPath:     s.tmdbClient.GetPosterURL(&provider.LogoPath, "w92"),
				ProviderType: "free",
				Link:         regionData.Link,
			})
		}
	}

	// Add Plex availability if user is provided
	if userID != nil {
		plexAvailable, plexProviders, err := s.getPlexAvailability(tmdbID, *userID)
		if err == nil {
			response.PlexAvailable = plexAvailable
			response.Providers = append(response.Providers, plexProviders...)
		}
	}

	// SKIP CACHING WHILE TESTING - Cache the TMDB data (not including Plex data which is user-specific)
	// err = s.cacheWatchProviders(response)
	// if err != nil {
	// 	fmt.Printf("Failed to cache watch providers: %v\n", err)
	// }
	fmt.Printf("DEBUG: SKIPPING TMDB provider cache write for testing\n")

	return response, nil
}

// getPlexAvailability checks if movie is available on user's Plex servers using database query
func (s *WatchProvidersService) getPlexAvailability(tmdbID int, userID int) (bool, []WatchProvider, error) {
	fmt.Printf("DEBUG: Starting Plex availability check for TMDB ID %d, User ID %d\n", tmdbID, userID)

	// TEMPORARILY DISABLE CACHE - Check cache first
	// cachedAvailable, cachedProviders, err := s.getCachedPlexAvailability(tmdbID, userID)
	// if err == nil {
	// 	fmt.Printf("DEBUG: Found cached Plex availability: %v (expires check passed)\n", cachedAvailable)
	// 	return cachedAvailable, cachedProviders, nil
	// }
	fmt.Printf("DEBUG: CACHE DISABLED - Skipping cache lookup for testing\n")

	// Get detailed Plex availability with server information for clickable links
	fmt.Printf("DEBUG: Getting detailed Plex availability using database query\n")
	plexProviders, err := s.getPlexProvidersFromDatabase(tmdbID, userID)
	if err != nil {
		fmt.Printf("DEBUG: Database query failed: %v\n", err)
		return false, []WatchProvider{}, nil
	}
	fmt.Printf("DEBUG: Database query completed. Found %d Plex providers\n", len(plexProviders))

	isAvailable := len(plexProviders) > 0

	// SKIP CACHING WHILE TESTING - Cache the result
	fmt.Printf("DEBUG: SKIPPING cache write for testing: available=%v\n", isAvailable)
	// s.cachePlexAvailability(tmdbID, userID, isAvailable, []string{})

	fmt.Printf("DEBUG: Completed Plex availability check. Final result: %v\n", isAvailable)
	return isAvailable, plexProviders, nil
}

// ClearExpiredCache removes expired cache entries
func (s *WatchProvidersService) ClearExpiredCache() error {
	// Clear expired TMDB watch providers cache
	_, err := s.db.Exec("DELETE FROM watch_providers_cache WHERE expires_at <= datetime('now')")
	if err != nil {
		return fmt.Errorf("failed to clear expired watch providers cache: %w", err)
	}

	// Clear expired Plex availability cache
	_, err = s.db.Exec("DELETE FROM plex_availability_cache WHERE expires_at <= datetime('now')")
	if err != nil {
		return fmt.Errorf("failed to clear expired Plex availability cache: %w", err)
	}

	return nil
}

// getPlexProvidersFromDatabase gets detailed Plex provider information with clickable URLs
func (s *WatchProvidersService) getPlexProvidersFromDatabase(tmdbID int, userID int) ([]WatchProvider, error) {
	query := `
		SELECT DISTINCT 
			ps.name as server_name,
			ps.machine_id,
			pl.title as library_name,
			pl.section_key,
			pli.plex_rating_key,
			pli.plex_guid,
			pli.title as movie_title
		FROM plex_library_items pli
		JOIN plex_libraries pl ON pli.library_id = pl.id
		JOIN plex_servers ps ON pl.server_id = ps.id
		JOIN user_plex_access upa ON pl.id = upa.library_id
		WHERE upa.user_id = ? AND pli.tmdb_id = ? AND pli.is_active = 1 AND upa.is_active = 1
	`

	rows, err := s.db.Query(query, userID, tmdbID)
	if err != nil {
		return nil, fmt.Errorf("failed to query Plex providers: %w", err)
	}
	defer rows.Close()

	var providers []WatchProvider
	for rows.Next() {
		var serverName, machineID, libraryName, movieTitle, ratingKey, plexGUID string
		var sectionKey int

		err := rows.Scan(&serverName, &machineID, &libraryName, &sectionKey, &ratingKey, &plexGUID, &movieTitle)
		if err != nil {
			continue
		}

		// Use the rating key directly from the database - now that we've updated the sync
		// to store the actual numeric rating key from the Plex API
		actualRatingKey := ratingKey

		// Create Plex web link URL that works in any browser
		// Format: https://app.plex.tv/desktop/#!/server/{machineID}/details?key=%2Flibrary%2Fmetadata%2F{ratingKey}
		plexURL := fmt.Sprintf("https://app.plex.tv/desktop/#!/server/%s/details?key=%%2Flibrary%%2Fmetadata%%2F%s", machineID, actualRatingKey)

		provider := WatchProvider{
			Name:         fmt.Sprintf("Plex (%s)", serverName),
			ProviderType: "plex",
			PlexServer:   serverName,
			PlexURL:      plexURL,
			LibraryName:  libraryName,
			Link:         plexURL, // Also set as generic link for UI consistency
		}

		providers = append(providers, provider)
	}

	return providers, nil
}
