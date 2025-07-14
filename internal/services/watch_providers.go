package services

import (
	"context"
	"database/sql"
	"encoding/json"
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
	PlexServer   string  `json:"plexServer,omitempty"` // For Plex providers
}

// WatchProvidersResponse represents the combined response
type WatchProvidersResponse struct {
	TMDBID         int             `json:"tmdbId"`
	Region         string          `json:"region"`
	TMDBLink       string          `json:"tmdbLink,omitempty"`
	Providers      []WatchProvider `json:"providers"`
	PlexAvailable  bool            `json:"plexAvailable"`
	CachedAt       time.Time       `json:"cachedAt"`
	ExpiresAt      time.Time       `json:"expiresAt"`
}

func NewWatchProvidersService(db *sql.DB, tmdbClient *TMDBClient, plexClient *PlexClient) *WatchProvidersService {
	return &WatchProvidersService{
		db:           db,
		tmdbClient:   tmdbClient,
		plexClient:   plexClient,     // Keep for backward compatibility during migration
		plexgoClient: NewPlexgoClient(), // Primary client for all operations
	}
}

// GetWatchProviders gets watch provider information with caching
func (s *WatchProvidersService) GetWatchProviders(tmdbID int, region string, userID *int) (*WatchProvidersResponse, error) {
	if region == "" {
		region = "US" // Default to US
	}

	// Try to get from cache first
	cached, err := s.getCachedWatchProviders(tmdbID, region)
	if err == nil && cached.ExpiresAt.After(time.Now()) {
		// Add Plex availability if user is provided
		if userID != nil {
			plexAvailable, plexProviders, err := s.getPlexAvailability(tmdbID, *userID)
			if err == nil {
				cached.PlexAvailable = plexAvailable
				// Add Plex providers to the list
				cached.Providers = append(cached.Providers, plexProviders...)
			}
		}
		return cached, nil
	}

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

	// Cache the TMDB data (not including Plex data which is user-specific)
	err = s.cacheWatchProviders(response)
	if err != nil {
		fmt.Printf("Failed to cache watch providers: %v\n", err)
	}

	return response, nil
}

// getCachedWatchProviders retrieves cached watch provider data
func (s *WatchProvidersService) getCachedWatchProviders(tmdbID int, region string) (*WatchProvidersResponse, error) {
	query := `
		SELECT providers_data, cached_at, expires_at 
		FROM watch_providers_cache 
		WHERE tmdb_id = ? AND region_code = ? AND expires_at > datetime('now')
	`
	
	var providersJSON string
	var cachedAt, expiresAt time.Time
	
	err := s.db.QueryRow(query, tmdbID, region).Scan(&providersJSON, &cachedAt, &expiresAt)
	if err != nil {
		return nil, err
	}
	
	var response WatchProvidersResponse
	err = json.Unmarshal([]byte(providersJSON), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached providers: %w", err)
	}
	
	response.CachedAt = cachedAt
	response.ExpiresAt = expiresAt
	
	return &response, nil
}

// cacheWatchProviders stores watch provider data in cache
func (s *WatchProvidersService) cacheWatchProviders(response *WatchProvidersResponse) error {
	// Create a copy without Plex data for caching
	cacheResponse := *response
	var tmdbOnlyProviders []WatchProvider
	for _, provider := range response.Providers {
		if provider.ProviderType != "plex" {
			tmdbOnlyProviders = append(tmdbOnlyProviders, provider)
		}
	}
	cacheResponse.Providers = tmdbOnlyProviders
	cacheResponse.PlexAvailable = false // Don't cache user-specific Plex data
	
	providersJSON, err := json.Marshal(cacheResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal providers for caching: %w", err)
	}
	
	query := `
		INSERT INTO watch_providers_cache (tmdb_id, region_code, providers_data, expires_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(tmdb_id, region_code) DO UPDATE SET
			providers_data = excluded.providers_data,
			cached_at = CURRENT_TIMESTAMP,
			expires_at = excluded.expires_at
	`
	
	_, err = s.db.Exec(query, response.TMDBID, response.Region, string(providersJSON), response.ExpiresAt)
	return err
}

// getPlexAvailability checks if movie is available on user's Plex servers by searching directly
func (s *WatchProvidersService) getPlexAvailability(tmdbID int, userID int) (bool, []WatchProvider, error) {
	fmt.Printf("DEBUG: Starting Plex availability check for TMDB ID %d, User ID %d\n", tmdbID, userID)
	
	// Check cache first
	cachedAvailable, cachedProviders, err := s.getCachedPlexAvailability(tmdbID, userID)
	if err == nil {
		fmt.Printf("DEBUG: Found cached Plex availability: %v (expires check passed)\n", cachedAvailable)
		return cachedAvailable, cachedProviders, nil
	}
	fmt.Printf("DEBUG: No valid cache found for Plex availability: %v\n", err)

	// Get user's Plex token
	var plexToken string
	err = s.db.QueryRow(`
		SELECT plex_token FROM user_plex_tokens WHERE user_id = ?
	`, userID).Scan(&plexToken)
	
	if err == sql.ErrNoRows {
		fmt.Printf("DEBUG: User %d not connected to Plex - caching negative result\n", userID)
		// User not connected to Plex - cache negative result
		s.cachePlexAvailability(tmdbID, userID, false, []string{})
		return false, []WatchProvider{}, nil
	}
	if err != nil {
		fmt.Printf("DEBUG: Failed to get Plex token for user %d: %v\n", userID, err)
		return false, []WatchProvider{}, fmt.Errorf("failed to get Plex token: %w", err)
	}
	// Show first 8 characters of token for debugging (safely)
	tokenPreview := plexToken
	if len(plexToken) > 8 {
		tokenPreview = plexToken[:8] + "..."
	}
	fmt.Printf("DEBUG: Retrieved Plex token for user %d (length: %d chars, preview: %s)\n", 
		userID, len(plexToken), tokenPreview)

	// Get movie title from TMDB ID to search Plex
	var movieTitle string
	err = s.db.QueryRow(`SELECT title FROM movies WHERE tmdb_id = ?`, tmdbID).Scan(&movieTitle)
	if err != nil {
		fmt.Printf("DEBUG: Failed to get movie title for TMDB ID %d: %v\n", tmdbID, err)
		return false, []WatchProvider{}, fmt.Errorf("movie not found in database: %w", err)
	}
	fmt.Printf("DEBUG: Retrieved movie title for TMDB ID %d: '%s'\n", tmdbID, movieTitle)

	// Search for this movie using plexgo (automatically respects user permissions)
	fmt.Printf("DEBUG: Starting plexgo-based Plex search for movie '%s'\n", movieTitle)
	isAvailable, err := s.searchMovieWithPlexgo(plexToken, movieTitle, tmdbID)
	if err != nil {
		fmt.Printf("DEBUG: Plexgo search failed for movie '%s': %v\n", movieTitle, err)
		// Cache negative result to avoid repeated failed searches
		s.cachePlexAvailability(tmdbID, userID, false, []string{})
		return false, []WatchProvider{}, nil
	}
	fmt.Printf("DEBUG: Plexgo search completed for movie '%s'. Available: %v\n", movieTitle, isAvailable)

	var plexProviders []WatchProvider
	if isAvailable {
		plexProviders = append(plexProviders, WatchProvider{
			Name:         "Plex",
			ProviderType: "plex",
			PlexServer:   "Your Plex Server",
		})
		fmt.Printf("DEBUG: Created Plex provider entry for movie '%s'\n", movieTitle)
	}
	
	// Cache the result
	var servers []string
	if isAvailable {
		servers = []string{"found"}
	}
	fmt.Printf("DEBUG: Caching Plex availability result: available=%v, servers=%v\n", isAvailable, servers)
	s.cachePlexAvailability(tmdbID, userID, isAvailable, servers)
	
	fmt.Printf("DEBUG: Completed Plex availability check for movie '%s'. Final result: %v\n", movieTitle, isAvailable)
	return isAvailable, plexProviders, nil
}

// getCachedPlexAvailability retrieves cached Plex availability
func (s *WatchProvidersService) getCachedPlexAvailability(tmdbID int, userID int) (bool, []WatchProvider, error) {
	query := `
		SELECT is_available, plex_servers
		FROM plex_availability_cache 
		WHERE tmdb_id = ? AND user_id = ? AND expires_at > datetime('now')
	`
	
	var isAvailable bool
	var plexServersJSON string
	
	err := s.db.QueryRow(query, tmdbID, userID).Scan(&isAvailable, &plexServersJSON)
	if err != nil {
		return false, []WatchProvider{}, err
	}
	
	var plexProviders []WatchProvider
	if isAvailable {
		plexProviders = append(plexProviders, WatchProvider{
			Name:         "Plex",
			ProviderType: "plex",
			PlexServer:   "Your Plex Server",
		})
	}
	
	return isAvailable, plexProviders, nil
}

// cachePlexAvailability stores Plex availability in cache
func (s *WatchProvidersService) cachePlexAvailability(tmdbID int, userID int, isAvailable bool, servers []string) error {
	serversJSON, _ := json.Marshal(servers)
	expiresAt := time.Now().Add(48 * time.Hour)
	
	query := `
		INSERT INTO plex_availability_cache (tmdb_id, user_id, is_available, plex_servers, expires_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(tmdb_id, user_id) DO UPDATE SET
			is_available = excluded.is_available,
			plex_servers = excluded.plex_servers,
			cached_at = CURRENT_TIMESTAMP,
			expires_at = excluded.expires_at
	`
	
	_, err := s.db.Exec(query, tmdbID, userID, isAvailable, string(serversJSON), expiresAt)
	return err
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




// searchMovieWithPlexgo searches for a movie using plexgo SDK (with automatic permission filtering)
func (s *WatchProvidersService) searchMovieWithPlexgo(token, movieTitle string, tmdbID int) (bool, error) {
	fmt.Printf("DEBUG: [searchMovieWithPlexgo] Starting permission-aware search for movie '%s' (TMDB ID: %d)\n", movieTitle, tmdbID)
	ctx := context.Background()
	
	// Get user's accessible servers using plexgo (automatically filtered by permissions)
	servers, err := s.plexgoClient.GetServers(ctx, token)
	if err != nil {
		fmt.Printf("DEBUG: [searchMovieWithPlexgo] Failed to get accessible servers: %v\n", err)
		return false, fmt.Errorf("failed to get accessible servers: %w", err)
	}
	
	fmt.Printf("DEBUG: [searchMovieWithPlexgo] Found %d accessible servers for user\n", len(servers))
	
	// If no servers accessible, movie can't be available
	if len(servers) == 0 {
		fmt.Printf("DEBUG: [searchMovieWithPlexgo] User has no accessible Plex servers\n")
		return false, nil
	}
	
	// Search each accessible server
	for i, server := range servers {
		fmt.Printf("DEBUG: [searchMovieWithPlexgo] Searching server %d/%d: '%s' (owned: %v)\n", 
			i+1, len(servers), server.Name, server.Owned)
		
		// Get the best connection for this server
		connection := s.plexgoClient.GetBestConnection(server)
		if connection == nil {
			fmt.Printf("DEBUG: [searchMovieWithPlexgo] Server '%s' has no usable connections\n", server.Name)
			continue
		}
		
		serverURL := s.plexgoClient.BuildServerURL(*connection)
		fmt.Printf("DEBUG: [searchMovieWithPlexgo] Using server URL: %s\n", serverURL)
		
		// Use the server's access token for authentication
		searchToken := server.AccessToken
		if searchToken == "" {
			searchToken = token // Fallback to user token
		}
		
		// Search for the movie on this server using plexgo
		found, err := s.plexgoClient.SearchMovieByTitle(ctx, searchToken, serverURL, movieTitle)
		if err != nil {
			fmt.Printf("DEBUG: [searchMovieWithPlexgo] Search failed on server '%s': %v\n", server.Name, err)
			continue
		}
		
		fmt.Printf("DEBUG: [searchMovieWithPlexgo] Search completed on server '%s'. Found: %v\n", server.Name, found)
		
		if found {
			fmt.Printf("DEBUG: [searchMovieWithPlexgo] ✓ Found '%s' on accessible Plex server '%s'\n", movieTitle, server.Name)
			return true, nil
		}
	}
	
	fmt.Printf("DEBUG: [searchMovieWithPlexgo] Movie '%s' not found on any of %d accessible servers\n", movieTitle, len(servers))
	return false, nil
}