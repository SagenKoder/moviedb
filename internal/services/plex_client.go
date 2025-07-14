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
