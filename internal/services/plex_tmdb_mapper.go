package services

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type PlexTMDBMapper struct {
	db         *sql.DB
	tmdbClient *TMDBClient
}

type PlexTMDBMapping struct {
	ID          int    `json:"id"`
	PlexGUID    string `json:"plexGuid"`
	TMDBID      int    `json:"tmdbId"`
	Title       string `json:"title"`
	Year        *int   `json:"year,omitempty"`
	RatingKey   string `json:"ratingKey,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func NewPlexTMDBMapper(db *sql.DB, tmdbClient *TMDBClient) *PlexTMDBMapper {
	return &PlexTMDBMapper{db: db, tmdbClient: tmdbClient}
}

// ExternalIDInfo represents extracted external ID information from Plex GUID
type ExternalIDInfo struct {
	Type  string // "tmdb", "imdb", "tvdb", "plex"
	Value string // the actual ID value
}

// ExtractExternalIDFromGUID extracts external ID information from various Plex GUID formats
func (m *PlexTMDBMapper) ExtractExternalIDFromGUID(guid string) (*ExternalIDInfo, error) {
	// Plex GUIDs can be in various formats:
	// "plex://movie/5d7768258df361001bdc8b4b" (Plex's own)
	// "com.plexapp.agents.themoviedb://123456?lang=en" (TMDB agent)
	// "tmdb://123456" (TMDB direct)
	// "imdb://tt1234567" (IMDB)
	// "com.plexapp.agents.imdb://tt1234567?lang=en" (IMDB agent)
	// "com.plexapp.agents.thetvdb://123456?lang=en" (TVDB agent)
	
	// TMDB agent format
	tmdbAgentRegex := regexp.MustCompile(`com\.plexapp\.agents\.themoviedb://(\d+)`)
	if matches := tmdbAgentRegex.FindStringSubmatch(guid); len(matches) > 1 {
		return &ExternalIDInfo{Type: "tmdb", Value: matches[1]}, nil
	}
	
	// Direct TMDB format  
	tmdbDirectRegex := regexp.MustCompile(`tmdb://(\d+)`)
	if matches := tmdbDirectRegex.FindStringSubmatch(guid); len(matches) > 1 {
		return &ExternalIDInfo{Type: "tmdb", Value: matches[1]}, nil
	}
	
	// IMDB agent format
	imdbAgentRegex := regexp.MustCompile(`com\.plexapp\.agents\.imdb://(tt\d+)`)  
	if matches := imdbAgentRegex.FindStringSubmatch(guid); len(matches) > 1 {
		return &ExternalIDInfo{Type: "imdb", Value: matches[1]}, nil
	}
	
	// Direct IMDB format
	imdbDirectRegex := regexp.MustCompile(`imdb://(tt\d+)`)
	if matches := imdbDirectRegex.FindStringSubmatch(guid); len(matches) > 1 {
		return &ExternalIDInfo{Type: "imdb", Value: matches[1]}, nil
	}
	
	// TVDB agent format
	tvdbAgentRegex := regexp.MustCompile(`com\.plexapp\.agents\.thetvdb://(\d+)`)
	if matches := tvdbAgentRegex.FindStringSubmatch(guid); len(matches) > 1 {
		return &ExternalIDInfo{Type: "tvdb", Value: matches[1]}, nil
	}
	
	// Direct TVDB format
	tvdbDirectRegex := regexp.MustCompile(`tvdb://(\d+)`)
	if matches := tvdbDirectRegex.FindStringSubmatch(guid); len(matches) > 1 {
		return &ExternalIDInfo{Type: "tvdb", Value: matches[1]}, nil
	}
	
	// Plex's own format (can't directly convert to TMDB)
	plexRegex := regexp.MustCompile(`plex://movie/([a-f0-9]{24})`)
	if matches := plexRegex.FindStringSubmatch(guid); len(matches) > 1 {
		return &ExternalIDInfo{Type: "plex", Value: matches[1]}, nil
	}
	
	return nil, fmt.Errorf("no recognized external ID found in GUID: %s", guid)
}

// ExtractTMDBIDFromGUID extracts TMDB ID from various Plex GUID formats (legacy method)
func (m *PlexTMDBMapper) ExtractTMDBIDFromGUID(guid string) (int, error) {
	extID, err := m.ExtractExternalIDFromGUID(guid)
	if err != nil {
		return 0, err
	}
	
	if extID.Type != "tmdb" {
		return 0, fmt.Errorf("GUID contains %s ID, not TMDB ID: %s", extID.Type, guid)
	}
	
	return strconv.Atoi(extID.Value)
}

// GetOrCreateMapping gets existing mapping or creates new one using TMDB API for external ID lookups
func (m *PlexTMDBMapper) GetOrCreateMapping(plexGUID, title string, year *int, ratingKey string) (*PlexTMDBMapping, error) {
	// First, try to get existing mapping
	existing, err := m.GetMappingByPlexGUID(plexGUID)
	if err == nil {
		return existing, nil
	}
	
	// Extract external ID from GUID
	extID, err := m.ExtractExternalIDFromGUID(plexGUID)
	if err != nil {
		fmt.Printf("DEBUG: Failed to extract external ID from GUID %s: %v\n", plexGUID, err)
		return m.tryFallbackMapping(plexGUID, title, year, ratingKey)
	}
	
	fmt.Printf("DEBUG: Extracted external ID - Type: %s, Value: %s from GUID: %s\n", extID.Type, extID.Value, plexGUID)
	
	var tmdbID int
	
	// Handle different external ID types
	switch extID.Type {
	case "tmdb":
		// Direct TMDB ID - convert to int
		tmdbID, err = strconv.Atoi(extID.Value)
		if err != nil {
			fmt.Printf("DEBUG: Failed to convert TMDB ID %s to int: %v\n", extID.Value, err)
			return m.tryFallbackMapping(plexGUID, title, year, ratingKey)
		}
		
	case "imdb":
		// Use TMDB find API to lookup by IMDb ID
		if m.tmdbClient == nil {
			fmt.Printf("DEBUG: No TMDB client available for external ID lookup, failing for IMDb ID: %s\n", extID.Value)
			return m.tryFallbackMapping(plexGUID, title, year, ratingKey)
		}
		
		findResp, err := m.tmdbClient.FindByExternalID(extID.Value, "imdb_id")
		if err != nil {
			fmt.Printf("DEBUG: TMDB find API failed for IMDb ID %s: %v\n", extID.Value, err)
			return m.tryFallbackMapping(plexGUID, title, year, ratingKey)
		}
		
		if len(findResp.MovieResults) == 0 {
			fmt.Printf("DEBUG: No TMDB movies found for IMDb ID %s\n", extID.Value)
			return m.tryFallbackMapping(plexGUID, title, year, ratingKey)
		}
		
		// Take the first result (should be the best match)
		tmdbID = findResp.MovieResults[0].ID
		fmt.Printf("DEBUG: Found TMDB ID %d via IMDb ID %s\n", tmdbID, extID.Value)
		
	case "tvdb":
		// Use TMDB find API to lookup by TVDB ID
		if m.tmdbClient == nil {
			fmt.Printf("DEBUG: No TMDB client available for external ID lookup, failing for TVDB ID: %s\n", extID.Value)
			return m.tryFallbackMapping(plexGUID, title, year, ratingKey)
		}
		
		findResp, err := m.tmdbClient.FindByExternalID(extID.Value, "tvdb_id")
		if err != nil {
			fmt.Printf("DEBUG: TMDB find API failed for TVDB ID %s: %v\n", extID.Value, err)
			return m.tryFallbackMapping(plexGUID, title, year, ratingKey)
		}
		
		if len(findResp.MovieResults) == 0 {
			fmt.Printf("DEBUG: No TMDB movies found for TVDB ID %s\n", extID.Value)
			return m.tryFallbackMapping(plexGUID, title, year, ratingKey)
		}
		
		// Take the first result (should be the best match)
		tmdbID = findResp.MovieResults[0].ID
		fmt.Printf("DEBUG: Found TMDB ID %d via TVDB ID %s\n", tmdbID, extID.Value)
		
	case "plex":
		// Plex's own format can't be directly converted to TMDB
		fmt.Printf("DEBUG: Cannot convert Plex internal ID %s to TMDB ID, trying fallback\n", extID.Value)
		return m.tryFallbackMapping(plexGUID, title, year, ratingKey)
		
	default:
		fmt.Printf("DEBUG: Unsupported external ID type %s for value %s\n", extID.Type, extID.Value)
		return m.tryFallbackMapping(plexGUID, title, year, ratingKey)
	}
	
	// Check if the TMDB movie exists in our database
	var existsInMovies bool
	err = m.db.QueryRow("SELECT 1 FROM movies WHERE tmdb_id = ?", tmdbID).Scan(&existsInMovies)
	if err == sql.ErrNoRows {
		fmt.Printf("DEBUG: TMDB movie %d not found in local database\n", tmdbID)
		return nil, fmt.Errorf("TMDB movie %d not found in local database", tmdbID)
	}
	if err != nil {
		fmt.Printf("DEBUG: Error checking movie existence for TMDB ID %d: %v\n", tmdbID, err)
		return nil, fmt.Errorf("error checking movie existence: %w", err)
	}
	
	// Create new mapping
	fmt.Printf("DEBUG: Creating mapping - Plex GUID: %s -> TMDB ID: %d\n", plexGUID, tmdbID)
	return m.CreateMapping(plexGUID, tmdbID, title, year, ratingKey)
}

// tryFallbackMapping attempts to find TMDB ID using title/year fuzzy matching
func (m *PlexTMDBMapper) tryFallbackMapping(plexGUID, title string, year *int, ratingKey string) (*PlexTMDBMapping, error) {
	if m.tmdbClient == nil {
		return nil, fmt.Errorf("no TMDB client available for fallback search and no direct ID mapping found")
	}
	
	// Search TMDB by title
	fmt.Printf("DEBUG: Attempting fallback search for title: %s, year: %v\n", title, year)
	searchResp, err := m.tmdbClient.SearchMovies(title, 1)
	if err != nil {
		fmt.Printf("DEBUG: TMDB search failed for title %s: %v\n", title, err)
		return nil, fmt.Errorf("failed to search TMDB for title %s: %w", title, err)
	}
	
	if len(searchResp.Results) == 0 {
		fmt.Printf("DEBUG: No TMDB search results for title: %s\n", title)
		return nil, fmt.Errorf("no TMDB results found for title: %s", title)
	}
	
	// Try to find best match by year if provided
	var bestMatch *TMDBMovie
	if year != nil {
		for _, movie := range searchResp.Results {
			movieYear := ExtractYear(movie.ReleaseDate)
			if movieYear != nil && *movieYear == *year {
				bestMatch = &movie
				fmt.Printf("DEBUG: Found exact year match - TMDB ID: %d, Title: %s, Year: %d\n", movie.ID, movie.Title, *movieYear)
				break
			}
		}
	}
	
	// If no year match, take the first (most popular) result
	if bestMatch == nil {
		bestMatch = &searchResp.Results[0]
		movieYear := ExtractYear(bestMatch.ReleaseDate)
		fmt.Printf("DEBUG: Using first search result - TMDB ID: %d, Title: %s, Year: %v\n", bestMatch.ID, bestMatch.Title, movieYear)
	}
	
	// Check if the TMDB movie exists in our database
	var existsInMovies bool
	err = m.db.QueryRow("SELECT 1 FROM movies WHERE tmdb_id = ?", bestMatch.ID).Scan(&existsInMovies)
	if err == sql.ErrNoRows {
		fmt.Printf("DEBUG: TMDB movie %d from fallback search not found in local database\n", bestMatch.ID)
		return nil, fmt.Errorf("TMDB movie %d not found in local database", bestMatch.ID)
	}
	if err != nil {
		fmt.Printf("DEBUG: Error checking movie existence for TMDB ID %d from fallback: %v\n", bestMatch.ID, err)
		return nil, fmt.Errorf("error checking movie existence: %w", err)
	}
	
	// Create new mapping
	fmt.Printf("DEBUG: Creating fallback mapping - Plex GUID: %s -> TMDB ID: %d (via search)\n", plexGUID, bestMatch.ID)
	return m.CreateMapping(plexGUID, bestMatch.ID, title, year, ratingKey)
}

// CreateMapping creates a new Plex-TMDB mapping
func (m *PlexTMDBMapper) CreateMapping(plexGUID string, tmdbID int, title string, year *int, ratingKey string) (*PlexTMDBMapping, error) {
	query := `
		INSERT INTO plex_tmdb_mappings (plex_guid, tmdb_id, title, year, plex_rating_key)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(plex_guid, tmdb_id) DO UPDATE SET
			title = excluded.title,
			year = excluded.year,
			plex_rating_key = excluded.plex_rating_key,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, plex_guid, tmdb_id, title, year, plex_rating_key, created_at, updated_at
	`
	
	var mapping PlexTMDBMapping
	err := m.db.QueryRow(query, plexGUID, tmdbID, title, year, ratingKey).Scan(
		&mapping.ID, &mapping.PlexGUID, &mapping.TMDBID, &mapping.Title, 
		&mapping.Year, &mapping.RatingKey, &mapping.CreatedAt, &mapping.UpdatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create mapping: %w", err)
	}
	
	return &mapping, nil
}

// GetMappingByPlexGUID gets mapping by Plex GUID
func (m *PlexTMDBMapper) GetMappingByPlexGUID(plexGUID string) (*PlexTMDBMapping, error) {
	query := `
		SELECT id, plex_guid, tmdb_id, title, year, plex_rating_key, created_at, updated_at
		FROM plex_tmdb_mappings 
		WHERE plex_guid = ?
	`
	
	var mapping PlexTMDBMapping
	err := m.db.QueryRow(query, plexGUID).Scan(
		&mapping.ID, &mapping.PlexGUID, &mapping.TMDBID, &mapping.Title,
		&mapping.Year, &mapping.RatingKey, &mapping.CreatedAt, &mapping.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &mapping, nil
}

// GetMappingByTMDBID gets mapping by TMDB ID
func (m *PlexTMDBMapper) GetMappingByTMDBID(tmdbID int) ([]*PlexTMDBMapping, error) {
	query := `
		SELECT id, plex_guid, tmdb_id, title, year, plex_rating_key, created_at, updated_at
		FROM plex_tmdb_mappings 
		WHERE tmdb_id = ?
	`
	
	rows, err := m.db.Query(query, tmdbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var mappings []*PlexTMDBMapping
	for rows.Next() {
		var mapping PlexTMDBMapping
		err := rows.Scan(
			&mapping.ID, &mapping.PlexGUID, &mapping.TMDBID, &mapping.Title,
			&mapping.Year, &mapping.RatingKey, &mapping.CreatedAt, &mapping.UpdatedAt,
		)
		if err != nil {
			continue
		}
		mappings = append(mappings, &mapping)
	}
	
	return mappings, nil
}

// SearchMappingsByTitle searches mappings by title (fuzzy)
func (m *PlexTMDBMapper) SearchMappingsByTitle(title string) ([]*PlexTMDBMapping, error) {
	query := `
		SELECT id, plex_guid, tmdb_id, title, year, plex_rating_key, created_at, updated_at
		FROM plex_tmdb_mappings 
		WHERE title LIKE ? 
		ORDER BY title
		LIMIT 50
	`
	
	searchPattern := "%" + strings.ToLower(title) + "%"
	rows, err := m.db.Query(query, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var mappings []*PlexTMDBMapping
	for rows.Next() {
		var mapping PlexTMDBMapping
		err := rows.Scan(
			&mapping.ID, &mapping.PlexGUID, &mapping.TMDBID, &mapping.Title,
			&mapping.Year, &mapping.RatingKey, &mapping.CreatedAt, &mapping.UpdatedAt,
		)
		if err != nil {
			continue
		}
		mappings = append(mappings, &mapping)
	}
	
	return mappings, nil
}

// GetAllMappings gets all mappings with pagination
func (m *PlexTMDBMapper) GetAllMappings(limit, offset int) ([]*PlexTMDBMapping, int, error) {
	// Get total count
	var totalCount int
	err := m.db.QueryRow("SELECT COUNT(*) FROM plex_tmdb_mappings").Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}
	
	// Get mappings
	query := `
		SELECT id, plex_guid, tmdb_id, title, year, plex_rating_key, created_at, updated_at
		FROM plex_tmdb_mappings 
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := m.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	var mappings []*PlexTMDBMapping
	for rows.Next() {
		var mapping PlexTMDBMapping
		err := rows.Scan(
			&mapping.ID, &mapping.PlexGUID, &mapping.TMDBID, &mapping.Title,
			&mapping.Year, &mapping.RatingKey, &mapping.CreatedAt, &mapping.UpdatedAt,
		)
		if err != nil {
			continue
		}
		mappings = append(mappings, &mapping)
	}
	
	return mappings, totalCount, nil
}

// DeleteMapping deletes a mapping
func (m *PlexTMDBMapper) DeleteMapping(id int) error {
	_, err := m.db.Exec("DELETE FROM plex_tmdb_mappings WHERE id = ?", id)
	return err
}