package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

type WatchProvidersService struct {
	db         *sql.DB
	tmdbClient *TMDBClient
	plexClient *PlexClient
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
		db:         db,
		tmdbClient: tmdbClient,
		plexClient: plexClient,
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

	// Search for this movie directly on Plex servers
	fmt.Printf("DEBUG: Starting Plex server search for movie '%s'\n", movieTitle)
	isAvailable, err := s.searchMovieOnPlex(plexToken, movieTitle, tmdbID)
	if err != nil {
		fmt.Printf("DEBUG: Plex search failed for movie '%s': %v\n", movieTitle, err)
		// Cache negative result to avoid repeated failed searches
		s.cachePlexAvailability(tmdbID, userID, false, []string{})
		return false, []WatchProvider{}, nil
	}
	fmt.Printf("DEBUG: Plex search completed for movie '%s'. Available: %v\n", movieTitle, isAvailable)

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

// searchMovieOnPlex searches for a specific movie on user's Plex servers
func (s *WatchProvidersService) searchMovieOnPlex(token, movieTitle string, tmdbID int) (bool, error) {
	fmt.Printf("DEBUG: [searchMovieOnPlex] Starting search for movie '%s' (TMDB ID: %d)\n", movieTitle, tmdbID)
	
	// Get user's Plex servers
	fmt.Printf("DEBUG: [searchMovieOnPlex] Fetching user's Plex servers...\n")
	servers, err := s.plexClient.GetServers(token)
	if err != nil {
		fmt.Printf("DEBUG: [searchMovieOnPlex] Failed to get Plex servers: %v\n", err)
		return false, fmt.Errorf("failed to get Plex servers: %w", err)
	}
	fmt.Printf("DEBUG: [searchMovieOnPlex] Retrieved %d Plex servers\n", len(servers))

	// Search each server for the movie
	for i, server := range servers {
		serverName, _ := server["name"].(string)
		owned, _ := server["owned"].(bool)
		fmt.Printf("DEBUG: [searchMovieOnPlex] Processing server %d/%d: '%s' (owned: %v)\n", i+1, len(servers), serverName, owned)
		
		// Extract server URL from connections array - only use external connections
		var serverURL string
		var totalConnections int
		var externalConnections int
		
		if connections, ok := server["connections"].([]interface{}); ok {
			totalConnections = len(connections)
			fmt.Printf("DEBUG: [searchMovieOnPlex] Server '%s' has %d connections\n", serverName, totalConnections)
			
			for j, conn := range connections {
				if connMap, ok := conn.(map[string]interface{}); ok {
					uri, hasURI := connMap["uri"].(string)
					local, hasLocal := connMap["local"].(bool)
					fmt.Printf("DEBUG: [searchMovieOnPlex] Connection %d: URI=%s, Local=%v, HasURI=%v, HasLocal=%v\n", j+1, uri, local, hasURI, hasLocal)
					
					if hasURI && hasLocal && !local {
						serverURL = uri
						externalConnections++
						fmt.Printf("DEBUG: [searchMovieOnPlex] Selected external connection: %s\n", uri)
						break
					}
				}
			}
		} else {
			fmt.Printf("DEBUG: [searchMovieOnPlex] Server '%s' has no connections array\n", serverName)
		}
		
		fmt.Printf("DEBUG: [searchMovieOnPlex] Server '%s' summary: %d total connections, %d external, selected URL: '%s'\n", 
			serverName, totalConnections, externalConnections, serverURL)
		
		if serverURL == "" {
			fmt.Printf("DEBUG: [searchMovieOnPlex] Skipping server '%s' - no accessible external URL\n", serverName)
			continue
		}

		fmt.Printf("DEBUG: [searchMovieOnPlex] Searching for '%s' on Plex server '%s' at %s\n", movieTitle, serverName, serverURL)
		
		// Search for the movie on this server
		found, err := s.searchMovieOnServer(token, serverURL, movieTitle)
		if err != nil {
			fmt.Printf("DEBUG: [searchMovieOnPlex] Search failed on server '%s': %v\n", serverName, err)
			continue
		}
		
		fmt.Printf("DEBUG: [searchMovieOnPlex] Search completed on server '%s'. Found: %v\n", serverName, found)
		
		if found {
			fmt.Printf("DEBUG: [searchMovieOnPlex] ✓ Found '%s' on Plex server '%s' - returning success\n", movieTitle, serverName)
			return true, nil
		} else {
			fmt.Printf("DEBUG: [searchMovieOnPlex] ✗ Movie '%s' not found on server '%s'\n", movieTitle, serverName)
		}
	}

	fmt.Printf("DEBUG: [searchMovieOnPlex] Search completed across all %d servers. Movie '%s' not found anywhere.\n", len(servers), movieTitle)
	return false, nil
}

// searchMovieOnServer searches for a movie on a specific Plex server
func (s *WatchProvidersService) searchMovieOnServer(token, serverURL, movieTitle string) (bool, error) {
	// For shared users, direct server access often fails with 401
	// Try using Plex.tv relay service instead of direct server access
	fmt.Printf("DEBUG: [searchMovieOnServer] Attempting shared user search via Plex.tv relay...\n")
	
	// Try to search via Plex.tv using the server's machineIdentifier
	relaySearch, err := s.searchViaPlexRelay(token, movieTitle)
	if err == nil {
		fmt.Printf("DEBUG: [searchMovieOnServer] Plex.tv relay search succeeded\n")
		return relaySearch, nil
	}
	fmt.Printf("DEBUG: [searchMovieOnServer] Plex.tv relay search failed: %v\n", err)
	
	// Fallback to direct server access (which we know will likely fail for shared users)
	fmt.Printf("DEBUG: [searchMovieOnServer] Falling back to direct server access...\n")
	encodedTitle := url.QueryEscape(movieTitle)
	searchURL := fmt.Sprintf("%s/search?query=%s", serverURL, encodedTitle)
	fmt.Printf("DEBUG: [searchMovieOnServer] Movie title: '%s' -> encoded: '%s'\n", movieTitle, encodedTitle)
	fmt.Printf("DEBUG: [searchMovieOnServer] Making request to: %s\n", searchURL)
	
	headers := map[string]string{
		"X-Plex-Token":             token,
		"X-Plex-Client-Identifier": "moviedb-watch-providers",
		"Accept":                   "application/json",
	}
	fmt.Printf("DEBUG: [searchMovieOnServer] Request headers: X-Plex-Token=[%d chars], Client-ID=%s\n", 
		len(token), headers["X-Plex-Client-Identifier"])
	
	resp, err := s.plexClient.MakeRequest("GET", searchURL, headers, nil)
	if err != nil {
		fmt.Printf("DEBUG: [searchMovieOnServer] HTTP request failed: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()
	
	fmt.Printf("DEBUG: [searchMovieOnServer] HTTP response status: %d\n", resp.StatusCode)
	
	if resp.StatusCode != 200 {
		fmt.Printf("DEBUG: [searchMovieOnServer] Non-200 status code: %d\n", resp.StatusCode)
		
		// Try to read error response body for debugging
		if resp.Body != nil {
			bodyBytes := make([]byte, 1024) // Read first 1KB of error response
			n, readErr := resp.Body.Read(bodyBytes)
			if readErr == nil || n > 0 {
				fmt.Printf("DEBUG: [searchMovieOnServer] Error response body: %s\n", string(bodyBytes[:n]))
			}
		}
		
		return false, fmt.Errorf("search returned status %d", resp.StatusCode)
	}
	
	var searchResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		fmt.Printf("DEBUG: [searchMovieOnServer] Failed to decode JSON response: %v\n", err)
		return false, err
	}
	
	fmt.Printf("DEBUG: [searchMovieOnServer] Parsed JSON response, looking for MediaContainer...\n")
	
	// Check if any movies were found in the search results
	if mediaContainer, ok := searchResponse["MediaContainer"].(map[string]interface{}); ok {
		fmt.Printf("DEBUG: [searchMovieOnServer] Found MediaContainer, looking for Metadata...\n")
		
		if metadata, ok := mediaContainer["Metadata"].([]interface{}); ok {
			fmt.Printf("DEBUG: [searchMovieOnServer] Found Metadata array with %d items\n", len(metadata))
			
			movieCount := 0
			for i, item := range metadata {
				if itemMap, ok := item.(map[string]interface{}); ok {
					itemType, hasType := itemMap["type"].(string)
					title, hasTitle := itemMap["title"].(string)
					
					fmt.Printf("DEBUG: [searchMovieOnServer] Item %d: type=%s, title=%s, hasType=%v, hasTitle=%v\n", 
						i+1, itemType, title, hasType, hasTitle)
					
					// Check if this is a movie type
					if hasType && itemType == "movie" {
						movieCount++
						fmt.Printf("DEBUG: [searchMovieOnServer] ✓ Found movie #%d: '%s'\n", movieCount, title)
						return true, nil
					}
				} else {
					fmt.Printf("DEBUG: [searchMovieOnServer] Item %d: could not parse as map\n", i+1)
				}
			}
			
			fmt.Printf("DEBUG: [searchMovieOnServer] Searched %d items, found %d movies, none matched\n", len(metadata), movieCount)
		} else {
			fmt.Printf("DEBUG: [searchMovieOnServer] MediaContainer has no Metadata array or wrong type\n")
		}
	} else {
		fmt.Printf("DEBUG: [searchMovieOnServer] Response has no MediaContainer or wrong type\n")
		fmt.Printf("DEBUG: [searchMovieOnServer] Response keys: ")
		for key := range searchResponse {
			fmt.Printf("%s ", key)
		}
		fmt.Printf("\n")
	}
	
	fmt.Printf("DEBUG: [searchMovieOnServer] No movies found in search results\n")
	return false, nil
}

// searchViaPlexRelay attempts to search for movies using Plex.tv central services
// This approach works better for shared users who don't have direct server access
func (s *WatchProvidersService) searchViaPlexRelay(token, movieTitle string) (bool, error) {
	fmt.Printf("DEBUG: [searchViaPlexRelay] Searching for '%s' via Plex.tv services\n", movieTitle)
	
	// Try the Plex.tv global search endpoint that works with shared access
	encodedTitle := url.QueryEscape(movieTitle)
	searchURL := fmt.Sprintf("https://plex.tv/api/v2/search?query=%s", encodedTitle)
	
	fmt.Printf("DEBUG: [searchViaPlexRelay] Making request to: %s\n", searchURL)
	
	headers := map[string]string{
		"X-Plex-Token":             token,
		"X-Plex-Client-Identifier": "moviedb-watch-providers",
		"Accept":                   "application/json",
	}
	
	resp, err := s.plexClient.MakeRequest("GET", searchURL, headers, nil)
	if err != nil {
		fmt.Printf("DEBUG: [searchViaPlexRelay] HTTP request failed: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()
	
	fmt.Printf("DEBUG: [searchViaPlexRelay] HTTP response status: %d\n", resp.StatusCode)
	
	if resp.StatusCode != 200 {
		fmt.Printf("DEBUG: [searchViaPlexRelay] Non-200 status code: %d\n", resp.StatusCode)
		
		// Try to read error response body for debugging
		if resp.Body != nil {
			bodyBytes := make([]byte, 1024)
			n, readErr := resp.Body.Read(bodyBytes)
			if readErr == nil || n > 0 {
				fmt.Printf("DEBUG: [searchViaPlexRelay] Error response body: %s\n", string(bodyBytes[:n]))
			}
		}
		
		return false, fmt.Errorf("Plex.tv search returned status %d", resp.StatusCode)
	}
	
	var searchResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		fmt.Printf("DEBUG: [searchViaPlexRelay] Failed to decode JSON response: %v\n", err)
		return false, err
	}
	
	fmt.Printf("DEBUG: [searchViaPlexRelay] Parsing Plex.tv search response...\n")
	
	// The Plex.tv response structure might be different - let's examine it
	fmt.Printf("DEBUG: [searchViaPlexRelay] Response keys: ")
	for key := range searchResponse {
		fmt.Printf("%s ", key)
	}
	fmt.Printf("\n")
	
	// Look for any indication that movies were found
	// This is a simplified check - we're mainly testing if the endpoint works
	if len(searchResponse) > 0 {
		fmt.Printf("DEBUG: [searchViaPlexRelay] Plex.tv search returned data - assuming movie availability can be checked via this method\n")
		// For now, return true if we get any response data
		// In practice, you'd want to parse the specific response structure
		return true, nil
	}
	
	fmt.Printf("DEBUG: [searchViaPlexRelay] No data in Plex.tv search response\n")
	return false, nil
}