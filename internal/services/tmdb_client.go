package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type TMDBClient struct {
	APIKey  string
	BaseURL string
	client  *http.Client
}

// TMDB API Response Types
type TMDBSearchResponse struct {
	Page         int          `json:"page"`
	Results      []TMDBMovie  `json:"results"`
	TotalPages   int          `json:"total_pages"`
	TotalResults int          `json:"total_results"`
}

type TMDBMovie struct {
	ID               int      `json:"id"`
	Title            string   `json:"title"`
	OriginalTitle    string   `json:"original_title"`
	Overview         string   `json:"overview"`
	ReleaseDate      string   `json:"release_date"`
	PosterPath       *string  `json:"poster_path"`
	BackdropPath     *string  `json:"backdrop_path"`
	GenreIDs         []int    `json:"genre_ids"`
	Adult            bool     `json:"adult"`
	OriginalLanguage string   `json:"original_language"`
	Popularity       float64  `json:"popularity"`
	VoteAverage      float64  `json:"vote_average"`
	VoteCount        int      `json:"vote_count"`
	Video            bool     `json:"video"`
}

type TMDBMovieDetails struct {
	TMDBMovie
	Runtime int     `json:"runtime"`
	Genres  []Genre `json:"genres"`
	Budget  int64   `json:"budget"`
	Revenue int64   `json:"revenue"`
	Status  string  `json:"status"`
	Tagline string  `json:"tagline"`
}

type TMDBExternalIDs struct {
	IMDbID      *string `json:"imdb_id"`
	FacebookID  *string `json:"facebook_id"`
	InstagramID *string `json:"instagram_id"`
	TwitterID   *string `json:"twitter_id"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func NewTMDBClient(apiKey string) *TMDBClient {
	return &TMDBClient{
		APIKey:  apiKey,
		BaseURL: "https://api.themoviedb.org/3",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IsValidAPIKey checks if the API key looks valid (basic validation)
func (c *TMDBClient) IsValidAPIKey() bool {
	// Basic validation - TMDB API keys are typically 32 characters long
	if len(c.APIKey) < 20 || c.APIKey == "test" || c.APIKey == "fake-api-key" {
		return false
	}
	return true
}

func (c *TMDBClient) makeRequest(endpoint string, params map[string]string) (*http.Response, error) {
	u, err := url.Parse(c.BaseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	query := u.Query()
	
	// Add request parameters
	for key, value := range params {
		query.Set(key, value)
	}
	
	u.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use Bearer token authentication (recommended for TMDB API v3)
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Read the response body to get detailed error information
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d, response: %s, URL: %s", resp.StatusCode, string(body), req.URL.String())
	}

	return resp, nil
}

// SearchMovies searches for movies by query string
func (c *TMDBClient) SearchMovies(query string, year int) (*TMDBSearchResponse, error) {
	params := map[string]string{
		"query": query,
	}

	// Add year parameter if provided
	if year > 0 {
		params["year"] = strconv.Itoa(year)
	}

	resp, err := c.makeRequest("/search/movie", params)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	var searchResp TMDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return &searchResp, nil
}

// GetMovieDetails gets detailed information about a specific movie
func (c *TMDBClient) GetMovieDetails(tmdbID int) (*TMDBMovieDetails, error) {
	endpoint := fmt.Sprintf("/movie/%d", tmdbID)
	
	resp, err := c.makeRequest(endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("movie details request failed: %w", err)
	}
	defer resp.Body.Close()

	var movie TMDBMovieDetails
	if err := json.NewDecoder(resp.Body).Decode(&movie); err != nil {
		return nil, fmt.Errorf("failed to decode movie details: %w", err)
	}

	return &movie, nil
}

// GetPopularMovies gets a list of popular movies
func (c *TMDBClient) GetPopularMovies(page int) (*TMDBSearchResponse, error) {
	if page <= 0 {
		page = 1
	}

	params := map[string]string{
		"page": strconv.Itoa(page),
	}

	resp, err := c.makeRequest("/movie/popular", params)
	if err != nil {
		return nil, fmt.Errorf("popular movies request failed: %w", err)
	}
	defer resp.Body.Close()

	var searchResp TMDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode popular movies response: %w", err)
	}

	return &searchResp, nil
}

// GetTrendingMovies gets a list of trending movies
func (c *TMDBClient) GetTrendingMovies(timeWindow string) (*TMDBSearchResponse, error) {
	if timeWindow != "day" && timeWindow != "week" {
		timeWindow = "week"
	}

	endpoint := fmt.Sprintf("/trending/movie/%s", timeWindow)
	
	resp, err := c.makeRequest(endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("trending movies request failed: %w", err)
	}
	defer resp.Body.Close()

	var searchResp TMDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode trending movies response: %w", err)
	}

	return &searchResp, nil
}

// GetMovieExternalIDs gets external IDs (IMDb, etc.) for a movie
func (c *TMDBClient) GetMovieExternalIDs(tmdbID int) (*TMDBExternalIDs, error) {
	endpoint := fmt.Sprintf("/movie/%d/external_ids", tmdbID)
	
	resp, err := c.makeRequest(endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("external IDs request failed: %w", err)
	}
	defer resp.Body.Close()

	var externalIDs TMDBExternalIDs
	if err := json.NewDecoder(resp.Body).Decode(&externalIDs); err != nil {
		return nil, fmt.Errorf("failed to decode external IDs: %w", err)
	}

	return &externalIDs, nil
}

// TMDBFindResponse represents the response from TMDB find API
type TMDBFindResponse struct {
	MovieResults []TMDBMovie `json:"movie_results"`
	PersonResults []interface{} `json:"person_results"`
	TVResults []interface{} `json:"tv_results"`
}

// FindByExternalID finds TMDB movie by external ID (IMDb, TVDB, etc.)
func (c *TMDBClient) FindByExternalID(externalID string, source string) (*TMDBFindResponse, error) {
	// Validate source parameter
	validSources := map[string]bool{
		"imdb_id": true,
		"freebase_mid": true,
		"freebase_id": true,
		"tvdb_id": true,
		"tvrage_id": true,
		"facebook_id": true,
		"twitter_id": true,
		"instagram_id": true,
	}
	
	if !validSources[source] {
		return nil, fmt.Errorf("invalid external source: %s", source)
	}
	
	endpoint := fmt.Sprintf("/find/%s", externalID)
	params := map[string]string{
		"external_source": source,
	}
	
	resp, err := c.makeRequest(endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("find request failed: %w", err)
	}
	defer resp.Body.Close()

	var findResp TMDBFindResponse
	if err := json.NewDecoder(resp.Body).Decode(&findResp); err != nil {
		return nil, fmt.Errorf("failed to decode find response: %w", err)
	}

	return &findResp, nil
}

// TMDBWatchProvider represents a streaming/rental provider
type TMDBWatchProvider struct {
	DisplayPriority int     `json:"display_priority"`
	LogoPath        string  `json:"logo_path"`
	ProviderID      int     `json:"provider_id"`
	ProviderName    string  `json:"provider_name"`
}

// TMDBWatchProvidersRegion represents watch providers for a specific region
type TMDBWatchProvidersRegion struct {
	Link      string              `json:"link,omitempty"`
	Flatrate  []TMDBWatchProvider `json:"flatrate,omitempty"`  // Subscription services like Netflix
	Rent      []TMDBWatchProvider `json:"rent,omitempty"`      // Rental services like Amazon Video
	Buy       []TMDBWatchProvider `json:"buy,omitempty"`       // Purchase services like iTunes
	Free      []TMDBWatchProvider `json:"free,omitempty"`      // Free services like YouTube
}

// TMDBWatchProvidersResponse represents the response from TMDB watch providers API
type TMDBWatchProvidersResponse struct {
	ID      int                                     `json:"id"`
	Results map[string]TMDBWatchProvidersRegion    `json:"results"` // Region code -> providers
}

// GetMovieWatchProviders gets watch provider information for a movie
func (c *TMDBClient) GetMovieWatchProviders(tmdbID int) (*TMDBWatchProvidersResponse, error) {
	endpoint := fmt.Sprintf("/movie/%d/watch/providers", tmdbID)
	
	resp, err := c.makeRequest(endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("watch providers request failed: %w", err)
	}
	defer resp.Body.Close()

	var watchProviders TMDBWatchProvidersResponse
	if err := json.NewDecoder(resp.Body).Decode(&watchProviders); err != nil {
		return nil, fmt.Errorf("failed to decode watch providers: %w", err)
	}

	return &watchProviders, nil
}

// GetPosterURL generates the full URL for a movie poster
func (c *TMDBClient) GetPosterURL(posterPath *string, size string) string {
	if posterPath == nil || *posterPath == "" {
		return ""
	}

	if size == "" {
		size = "w500" // Default poster size
	}

	return fmt.Sprintf("https://image.tmdb.org/t/p/%s%s", size, *posterPath)
}

// GetBackdropURL generates the full URL for a movie backdrop
func (c *TMDBClient) GetBackdropURL(backdropPath *string, size string) string {
	if backdropPath == nil || *backdropPath == "" {
		return ""
	}

	if size == "" {
		size = "w1280" // Default backdrop size
	}

	return fmt.Sprintf("https://image.tmdb.org/t/p/%s%s", size, *backdropPath)
}

// Helper function to extract year from release date
func ExtractYear(releaseDate string) *int {
	if releaseDate == "" {
		return nil
	}

	parts := strings.Split(releaseDate, "-")
	if len(parts) == 0 {
		return nil
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}

	return &year
}