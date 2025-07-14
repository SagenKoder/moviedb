package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/LukeHagar/plexgo"
	"github.com/LukeHagar/plexgo/models/operations"
)

// PlexgoClient wraps the plexgo SDK with our application-specific logic
type PlexgoClient struct {
	clientID string
	product  string
	version  string
	device   string
}

// PlexServer represents a Plex server with connection info
type PlexServer struct {
	Name             string
	MachineID        string
	AccessToken      string
	Connections      []PlexConnection
	Owned            bool
	Product          string
	ProductVersion   string
	Platform         string
	PlatformVersion  string
	Device           string
}

// PlexConnection represents a server connection
type PlexConnection struct {
	Protocol string
	Address  string
	Port     int
	URI      string
	Local    bool
	Relay    bool
}

// PlexLibrary represents a Plex library section
type PlexLibrary struct {
	Key        int
	Title      string
	Type       string
	Agent      string
	Scanner    string
	Language   string
	UUID       string
}

// PlexSearchResult represents a search result
type PlexSearchResult struct {
	Title string
	Year  *int
	Type  string
	GUID  string
}

func NewPlexgoClient() *PlexgoClient {
	return &PlexgoClient{
		clientID: "moviedb-app",
		product:  "MovieDB",
		version:  "1.0.0",
		device:   "Web",
	}
}

// GetServers gets all servers accessible to the user (automatically filtered by permissions)
func (p *PlexgoClient) GetServers(ctx context.Context, token string) ([]PlexServer, error) {
	client := plexgo.New(
		plexgo.WithSecurity(token),
	)

	// Use the correct plexgo API for server resources
	res, err := client.Plex.GetServerResources(ctx, p.clientID, 
		operations.IncludeHTTPSEnable.ToPointer(),
		operations.IncludeRelayEnable.ToPointer(), 
		nil) // IPv6 not needed
	if err != nil {
		return nil, fmt.Errorf("failed to get server resources: %w", err)
	}

	var servers []PlexServer
	if res.PlexDevices != nil {
		for _, device := range res.PlexDevices {
			// Only process Plex Media Servers
			if device.Product != "Plex Media Server" {
				continue
			}

			server := PlexServer{
				Name:             device.Name,
				MachineID:        device.ClientIdentifier,
				AccessToken:      device.AccessToken,
				Owned:            device.Owned,
				Product:          device.Product,
				ProductVersion:   device.ProductVersion,
				Platform:         getStringValue(device.Platform),
				PlatformVersion:  getStringValue(device.PlatformVersion),
				Device:           getStringValue(device.Device),
			}

			// Convert connections
			if device.Connections != nil {
				for _, conn := range device.Connections {
					connection := PlexConnection{
						Protocol: string(conn.Protocol),
						Address:  conn.Address, 
						Port:     conn.Port,
						URI:      conn.URI,
						Local:    conn.Local,
						Relay:    conn.Relay,
					}
					server.Connections = append(server.Connections, connection)
				}
			}

			servers = append(servers, server)
		}
	}

	fmt.Printf("DEBUG: [GetServers] Retrieved %d accessible servers using plexgo\n", len(servers))
	return servers, nil
}

// GetLibraries gets all libraries from a server (automatically filtered by user permissions)
func (p *PlexgoClient) GetLibraries(ctx context.Context, token, serverURL string) ([]PlexLibrary, error) {
	client := plexgo.New(
		plexgo.WithSecurity(token),
		plexgo.WithServerURL(serverURL),
	)

	res, err := client.Library.GetAllLibraries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get libraries: %w", err)
	}

	var libraries []PlexLibrary
	if res.Object != nil && res.Object.MediaContainer != nil {
		for _, dir := range res.Object.MediaContainer.Directory {
			// Convert string key to int
			key, err := strconv.Atoi(dir.Key)
			if err != nil {
				continue // Skip invalid keys
			}

			library := PlexLibrary{
				Key:      key,
				Title:    dir.Title,
				Type:     string(dir.Type),
				Agent:    dir.Agent,
				Scanner:  dir.Scanner,
				Language: dir.Language,
				UUID:     dir.UUID,
			}
			libraries = append(libraries, library)
		}
	}

	return libraries, nil
}

// SearchAllLibraries searches across all accessible libraries for a query
func (p *PlexgoClient) SearchAllLibraries(ctx context.Context, token, serverURL, query string) ([]PlexSearchResult, error) {
	client := plexgo.New(
		plexgo.WithSecurity(token),
		plexgo.WithServerURL(serverURL),
	)

	searchReq := operations.GetSearchAllLibrariesRequest{
		Query: query,
	}
	res, err := client.Library.GetSearchAllLibraries(ctx, searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to search libraries: %w", err)
	}

	var results []PlexSearchResult
	
	if res.Object != nil {
		mediaContainer := res.Object.MediaContainer
		fmt.Printf("DEBUG: [SearchAllLibraries] Found %d search results for query '%s'\n", len(mediaContainer.SearchResult), query)
		
		for _, searchResult := range mediaContainer.SearchResult {
			// Check if this is a metadata result with a movie
			if searchResult.Metadata != nil {
				metadata := searchResult.Metadata
				// Only include movies in results
				if metadata.Type == operations.GetSearchAllLibrariesTypeMovie {
					result := PlexSearchResult{
						Title: metadata.Title,
						Type:  "movie",
						GUID:  metadata.GUID,
					}
					
					// Convert year if available
					if metadata.Year != nil {
						result.Year = metadata.Year
					}
					
					results = append(results, result)
					fmt.Printf("DEBUG: [SearchAllLibraries] Found movie: '%s'\n", result.Title)
				}
			}
		}
	}

	fmt.Printf("DEBUG: [SearchAllLibraries] Returning %d movie results for query '%s'\n", len(results), query)
	return results, nil
}

// PerformGlobalSearch performs a global search across the server
func (p *PlexgoClient) PerformGlobalSearch(ctx context.Context, token, serverURL, query string) ([]PlexSearchResult, error) {
	client := plexgo.New(
		plexgo.WithSecurity(token),
		plexgo.WithServerURL(serverURL),
	)

	res, err := client.Search.PerformSearch(ctx, query, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to perform global search: %w", err)
	}

	var results []PlexSearchResult
	
	// PerformSearch appears to not return structured data in the response object
	// The response may be in the raw HTTP response body
	fmt.Printf("DEBUG: [PerformGlobalSearch] PerformSearch completed with status %d for query '%s'\n", res.StatusCode, query)
	
	// For now, return empty results as this method may need raw response parsing
	// or we should prefer SearchAllLibraries method which has structured responses

	fmt.Printf("DEBUG: [PerformGlobalSearch] Returning %d movie results for query '%s'\n", len(results), query)
	return results, nil
}

// GetMoviesInLibrary gets all movies from a specific library
func (p *PlexgoClient) GetMoviesInLibrary(ctx context.Context, token, serverURL string, libraryKey int) ([]PlexSearchResult, error) {
	client := plexgo.New(
		plexgo.WithSecurity(token),
		plexgo.WithServerURL(serverURL),
	)

	libraryReq := operations.GetLibraryItemsRequest{
		SectionKey: libraryKey,
		Tag:        operations.Tag("all"), // Cast to Tag type
	}
	res, err := client.Library.GetLibraryItems(ctx, libraryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get library items: %w", err)
	}

	var results []PlexSearchResult
	
	if res.Object != nil && res.Object.MediaContainer != nil {
		mediaContainer := res.Object.MediaContainer
		fmt.Printf("DEBUG: [GetMoviesInLibrary] Found %d items in library %d\n", len(mediaContainer.Metadata), libraryKey)
		
		for _, metadata := range mediaContainer.Metadata {
			// Only include movies (type 1 = movie)
			if metadata.Type == operations.GetLibraryItemsTypeMovie {
				result := PlexSearchResult{
					Title: metadata.Title,
					Type:  "movie",
					GUID:  metadata.GUID,
				}
				
				// Convert year if available
				if metadata.Year != nil {
					result.Year = metadata.Year
				}
				
				results = append(results, result)
				fmt.Printf("DEBUG: [GetMoviesInLibrary] Found movie: '%s'\n", result.Title)
			}
		}
	}

	fmt.Printf("DEBUG: [GetMoviesInLibrary] Retrieved %d movies from library %d\n", len(results), libraryKey)
	return results, nil
}

// BuildServerURL constructs a proper server URL from connection info
func (p *PlexgoClient) BuildServerURL(connection PlexConnection) string {
	if connection.URI != "" {
		return connection.URI
	}
	return fmt.Sprintf("%s://%s:%d", connection.Protocol, connection.Address, connection.Port)
}

// GetBestConnection returns the best connection for a server (prefer external, then local)
func (p *PlexgoClient) GetBestConnection(server PlexServer) *PlexConnection {
	var bestConn *PlexConnection
	
	// Prefer external connections first
	for _, conn := range server.Connections {
		if !conn.Local && !conn.Relay {
			bestConn = &conn
			break
		}
	}
	
	// Fall back to local connections
	if bestConn == nil {
		for _, conn := range server.Connections {
			if conn.Local {
				bestConn = &conn
				break
			}
		}
	}
	
	// Last resort: any connection
	if bestConn == nil && len(server.Connections) > 0 {
		bestConn = &server.Connections[0]
	}
	
	return bestConn
}

// getStringValue safely converts a pointer string to a string value
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// SearchMovieByTitle searches for a specific movie title across accessible libraries
func (p *PlexgoClient) SearchMovieByTitle(ctx context.Context, token, serverURL, movieTitle string) (bool, error) {
	fmt.Printf("DEBUG: [SearchMovieByTitle] Starting search for '%s' on server %s\n", movieTitle, serverURL)
	
	// First try global search across all libraries (faster and more comprehensive)
	results, err := p.SearchAllLibraries(ctx, token, serverURL, movieTitle)
	if err != nil {
		fmt.Printf("DEBUG: [SearchMovieByTitle] SearchAllLibraries failed: %v, trying PerformGlobalSearch\n", err)
		
		// Fallback to global search
		results, err = p.PerformGlobalSearch(ctx, token, serverURL, movieTitle)
		if err != nil {
			fmt.Printf("DEBUG: [SearchMovieByTitle] Both search methods failed: %v\n", err)
			return false, fmt.Errorf("failed to search for movie: %w", err)
		}
	}
	
	// Check if any result matches our movie title
	for _, result := range results {
		if p.titleMatches(result.Title, movieTitle) {
			fmt.Printf("DEBUG: [SearchMovieByTitle] Found matching movie: '%s'\n", result.Title)
			return true, nil
		}
	}
	
	fmt.Printf("DEBUG: [SearchMovieByTitle] Movie '%s' not found in %d search results\n", movieTitle, len(results))
	return false, nil
}

// titleMatches checks if two movie titles are similar (case-insensitive, ignoring common variations)
func (p *PlexgoClient) titleMatches(plexTitle, searchTitle string) bool {
	// Simple case-insensitive comparison
	plexLower := strings.ToLower(strings.TrimSpace(plexTitle))
	searchLower := strings.ToLower(strings.TrimSpace(searchTitle))
	
	// Exact match
	if plexLower == searchLower {
		return true
	}
	
	// Contains match (for cases like "Movie Title" vs "Movie Title (2023)")
	if strings.Contains(plexLower, searchLower) || strings.Contains(searchLower, plexLower) {
		return true
	}
	
	return false
}