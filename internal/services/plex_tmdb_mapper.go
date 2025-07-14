package services

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type PlexTMDBMapper struct {
	db *sql.DB
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

func NewPlexTMDBMapper(db *sql.DB) *PlexTMDBMapper {
	return &PlexTMDBMapper{db: db}
}

// ExtractTMDBIDFromGUID extracts TMDB ID from various Plex GUID formats
func (m *PlexTMDBMapper) ExtractTMDBIDFromGUID(guid string) (int, error) {
	// Plex GUIDs can be in various formats:
	// "plex://movie/5d7768258df361001bdc8b4b" (Plex's own)
	// "com.plexapp.agents.themoviedb://123456?lang=en" (TMDB agent)
	// "tmdb://123456" (TMDB direct)
	// "imdb://tt1234567" (IMDB - we'd need to convert)
	
	// TMDB agent format
	tmdbAgentRegex := regexp.MustCompile(`com\.plexapp\.agents\.themoviedb://(\d+)`)
	if matches := tmdbAgentRegex.FindStringSubmatch(guid); len(matches) > 1 {
		return strconv.Atoi(matches[1])
	}
	
	// Direct TMDB format
	tmdbDirectRegex := regexp.MustCompile(`tmdb://(\d+)`)
	if matches := tmdbDirectRegex.FindStringSubmatch(guid); len(matches) > 1 {
		return strconv.Atoi(matches[1])
	}
	
	return 0, fmt.Errorf("no TMDB ID found in GUID: %s", guid)
}

// GetOrCreateMapping gets existing mapping or creates new one
func (m *PlexTMDBMapper) GetOrCreateMapping(plexGUID, title string, year *int, ratingKey string) (*PlexTMDBMapping, error) {
	// First, try to get existing mapping
	existing, err := m.GetMappingByPlexGUID(plexGUID)
	if err == nil {
		return existing, nil
	}
	
	// Try to extract TMDB ID from GUID
	tmdbID, err := m.ExtractTMDBIDFromGUID(plexGUID)
	if err != nil {
		// If we can't extract TMDB ID, we'd need to search TMDB by title/year
		// For now, return error - we can implement fuzzy matching later
		return nil, fmt.Errorf("cannot extract TMDB ID from GUID and fuzzy matching not implemented: %w", err)
	}
	
	// Check if the TMDB movie exists in our database
	var existsInMovies bool
	err = m.db.QueryRow("SELECT 1 FROM movies WHERE tmdb_id = ?", tmdbID).Scan(&existsInMovies)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("TMDB movie %d not found in local database", tmdbID)
	}
	if err != nil {
		return nil, fmt.Errorf("error checking movie existence: %w", err)
	}
	
	// Create new mapping
	return m.CreateMapping(plexGUID, tmdbID, title, year, ratingKey)
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