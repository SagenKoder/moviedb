package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PlexClient struct {
	clientID string
	product  string
	version  string
	device   string
}

type PlexPinResponse struct {
	ID       int    `json:"id"`
	Code     string `json:"code"`
	Product  string `json:"product"`
	Trusted  bool   `json:"trusted"`
	ClientID string `json:"clientIdentifier"`
	Location struct {
		Code    string `json:"code"`
		Country string `json:"country"`
	} `json:"location"`
	ExpiresIn int       `json:"expiresIn"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	AuthToken string    `json:"authToken,omitempty"`
}

type PlexUser struct {
	ID           int    `json:"id"`
	UUID         string `json:"uuid"`
	Username     string `json:"username"`
	Title        string `json:"title"`
	FriendlyName string `json:"friendlyName"`
	Email        string `json:"email"`
	Thumb        string `json:"thumb"`
	AuthToken    string `json:"authToken"`
	Country      string `json:"country"`
}

func NewPlexClient() *PlexClient {
	return &PlexClient{
		clientID: "moviedb-app",
		product:  "MovieDB",
		version:  "1.0.0",
		device:   "Web",
	}
}

// RequestPin starts the Plex PIN authentication flow
func (p *PlexClient) RequestPin() (*PlexPinResponse, error) {
	headers := p.getHeaders("")

	resp, err := p.makeRequest("POST", "https://plex.tv/api/v2/pins", headers, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to request PIN: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("PIN request failed with status: %d", resp.StatusCode)
	}

	var pinResp PlexPinResponse
	if err := json.NewDecoder(resp.Body).Decode(&pinResp); err != nil {
		return nil, fmt.Errorf("failed to decode PIN response: %w", err)
	}

	return &pinResp, nil
}

// CheckPin polls Plex to see if the PIN has been authorized
func (p *PlexClient) CheckPin(pinID int) (*PlexPinResponse, error) {
	headers := p.getHeaders("")

	pinURL := fmt.Sprintf("https://plex.tv/api/v2/pins/%d", pinID)
	resp, err := p.makeRequest("GET", pinURL, headers, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check PIN: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PIN check failed with status: %d", resp.StatusCode)
	}

	var pinResp PlexPinResponse
	if err := json.NewDecoder(resp.Body).Decode(&pinResp); err != nil {
		return nil, fmt.Errorf("failed to decode PIN response: %w", err)
	}

	return &pinResp, nil
}

// GetUser gets the authenticated user's information
func (p *PlexClient) GetUser(token string) (*PlexUser, error) {
	headers := p.getHeaders(token)

	resp, err := p.makeRequest("GET", "https://plex.tv/api/v2/user", headers, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user failed with status: %d", resp.StatusCode)
	}

	var user PlexUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &user, nil
}

// GetServers gets the user's available Plex servers
func (p *PlexClient) GetServers(token string) ([]map[string]interface{}, error) {
	headers := p.getHeaders(token)

	resp, err := p.makeRequest("GET", "https://plex.tv/api/v2/resources?includeHttps=1&includeRelay=1", headers, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get servers failed with status: %d", resp.StatusCode)
	}

	var servers []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		return nil, fmt.Errorf("failed to decode servers response: %w", err)
	}

	return servers, nil
}

// PlexLibraryItem represents a movie/show in a Plex library
type PlexLibraryItem struct {
	RatingKey string `json:"ratingKey"`
	GUID      string `json:"guid"`
	Title     string `json:"title"`
	Year      int    `json:"year"`
	Type      string `json:"type"`
	Summary   string `json:"summary"`
	Thumb     string `json:"thumb"`
	AddedAt   int64  `json:"addedAt"`
	UpdatedAt int64  `json:"updatedAt"`
	ViewCount int    `json:"viewCount"`
}

// GetLibraries gets all libraries from a Plex server
func (p *PlexClient) GetLibraries(token, serverURL string) ([]map[string]interface{}, error) {
	headers := p.getHeaders(token)
	
	url := fmt.Sprintf("%s/library/sections", serverURL)
	resp, err := p.makeRequest("GET", url, headers, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get libraries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get libraries failed with status: %d", resp.StatusCode)
	}

	var librariesResp struct {
		MediaContainer struct {
			Directory []map[string]interface{} `json:"Directory"`
		} `json:"MediaContainer"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&librariesResp); err != nil {
		return nil, fmt.Errorf("failed to decode libraries response: %w", err)
	}

	return librariesResp.MediaContainer.Directory, nil
}

// GetLibraryContent gets all movies from a specific library
func (p *PlexClient) GetLibraryContent(token, serverURL, libraryKey string) ([]PlexLibraryItem, error) {
	headers := p.getHeaders(token)
	
	url := fmt.Sprintf("%s/library/sections/%s/all", serverURL, libraryKey)
	resp, err := p.makeRequest("GET", url, headers, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get library content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get library content failed with status: %d", resp.StatusCode)
	}

	var contentResp struct {
		MediaContainer struct {
			Metadata []PlexLibraryItem `json:"Metadata"`
		} `json:"MediaContainer"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&contentResp); err != nil {
		return nil, fmt.Errorf("failed to decode library content response: %w", err)
	}

	return contentResp.MediaContainer.Metadata, nil
}

// PlexNowPlayingItem represents currently playing content
type PlexNowPlayingItem struct {
	RatingKey string `json:"ratingKey"`
	GUID      string `json:"guid"`
	Title     string `json:"title"`
	Year      int    `json:"year"`
	Type      string `json:"type"`
	Summary   string `json:"summary"`
	Thumb     string `json:"thumb"`
	Duration  int    `json:"duration"`
	ViewOffset int   `json:"viewOffset"`
	Player    struct {
		Title   string `json:"title"`
		Product string `json:"product"`
		State   string `json:"state"` // "playing", "paused", "stopped"
	} `json:"Player"`
	Session struct {
		ID       string `json:"id"`
		Bandwidth int   `json:"bandwidth"`
		Location string `json:"location"`
	} `json:"Session"`
}

// GetNowPlaying gets what the user is currently watching via Plex.tv global API
func (p *PlexClient) GetNowPlaying(token string) ([]PlexNowPlayingItem, error) {
	headers := p.getHeaders(token)
	
	// Try Plex.tv global sessions API first
	fmt.Printf("DEBUG: Trying Plex.tv global sessions API\n")
	resp, err := p.makeRequest("GET", "https://plex.tv/api/v2/user/sessions", headers, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get global sessions: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: Global sessions API returned status: %d\n", resp.StatusCode)
	
	if resp.StatusCode == http.StatusOK {
		var sessionsResp struct {
			MediaContainer struct {
				Metadata []PlexNowPlayingItem `json:"Metadata"`
			} `json:"MediaContainer"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&sessionsResp); err != nil {
			fmt.Printf("DEBUG: Failed to decode global sessions response: %v\n", err)
		} else {
			fmt.Printf("DEBUG: Global sessions API returned %d items\n", len(sessionsResp.MediaContainer.Metadata))
			if len(sessionsResp.MediaContainer.Metadata) > 0 {
				return sessionsResp.MediaContainer.Metadata, nil
			}
		}
	}
	
	// If global API doesn't work, fall back to checking individual servers
	fmt.Printf("DEBUG: Global API failed or returned no results, trying individual servers\n")
	return p.getNowPlayingFromServers(token)
}

// getNowPlayingFromServers gets now playing from individual servers (fallback method)
func (p *PlexClient) getNowPlayingFromServers(token string) ([]PlexNowPlayingItem, error) {
	// Get user's servers first
	servers, err := p.GetServers(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}
	
	fmt.Printf("DEBUG: Found %d Plex servers\n", len(servers))
	
	var allNowPlaying []PlexNowPlayingItem
	
	// Check each server for now playing content
	for i, server := range servers {
		serverName, _ := server["name"].(string)
		
		// Try to get a valid connection URL from the connections array
		var serverURL string
		var remoteURL string
		var localURL string
		
		if connections, ok := server["connections"].([]interface{}); ok && len(connections) > 0 {
			// First pass: categorize connections
			for _, conn := range connections {
				if connMap, ok := conn.(map[string]interface{}); ok {
					if uri, ok := connMap["uri"].(string); ok && uri != "" {
						isLocal, _ := connMap["local"].(bool)
						if isLocal {
							localURL = uri
						} else {
							remoteURL = uri
						}
					}
				}
			}
			
			// Prefer remote connection over local
			if remoteURL != "" {
				serverURL = remoteURL
			} else if localURL != "" {
				serverURL = localURL
			}
		}
		
		if serverURL == "" {
			fmt.Printf("DEBUG: Server %d (%s) has no valid connections\n", i, serverName)
			continue
		}
		
		fmt.Printf("DEBUG: Checking server %d: %s at %s\n", i, serverName, serverURL)
		
		// Use server-specific access token if available
		serverToken := token
		if accessToken, ok := server["accessToken"].(string); ok && accessToken != "" {
			fmt.Printf("DEBUG: Using server access token: %s\n", accessToken)
			serverToken = accessToken
		} else {
			fmt.Printf("DEBUG: No server access token, using user token\n")
		}
		
		// Get now playing from this server
		nowPlaying, err := p.getNowPlayingFromServer(serverToken, serverURL)
		if err != nil {
			fmt.Printf("DEBUG: Error getting now playing from server %s: %v\n", serverName, err)
			// Don't fail completely if one server fails
			continue
		}
		
		fmt.Printf("DEBUG: Server %s returned %d now playing items\n", serverName, len(nowPlaying))
		allNowPlaying = append(allNowPlaying, nowPlaying...)
	}
	
	fmt.Printf("DEBUG: Total now playing items across all servers: %d\n", len(allNowPlaying))
	return allNowPlaying, nil
}

// getNowPlayingFromServer gets now playing content from a specific server
func (p *PlexClient) getNowPlayingFromServer(token, serverURL string) ([]PlexNowPlayingItem, error) {
	headers := p.getHeaders(token)
	
	url := fmt.Sprintf("%s/status/sessions", serverURL)
	resp, err := p.makeRequest("GET", url, headers, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get now playing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get now playing failed with status: %d", resp.StatusCode)
	}

	var sessionsResp struct {
		MediaContainer struct {
			Metadata []PlexNowPlayingItem `json:"Metadata"`
		} `json:"MediaContainer"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sessionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode now playing response: %w", err)
	}

	return sessionsResp.MediaContainer.Metadata, nil
}

func (p *PlexClient) getHeaders(token string) map[string]string {
	headers := map[string]string{
		"Accept":                   "application/json",
		"Content-Type":             "application/x-www-form-urlencoded",
		"X-Plex-Product":           p.product,
		"X-Plex-Version":           p.version,
		"X-Plex-Client-Name":       p.product,
		"X-Plex-Client-Version":    p.version,
		"X-Plex-Device":            p.device,
		"X-Plex-Device-Name":       p.device,
		"X-Plex-Client-Identifier": p.clientID,
	}

	if token != "" {
		headers["X-Plex-Token"] = token
	}

	return headers
}

func (p *PlexClient) makeRequest(method, url string, headers map[string]string, body *bytes.Buffer) (*http.Response, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, body)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}
