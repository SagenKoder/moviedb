package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"moviedb/internal/auth"
	"moviedb/internal/database"
	"moviedb/internal/services"
)

type PlexSyncHandler struct {
	db         *sql.DB
	plexClient *services.PlexClient
	mapper     *services.PlexTMDBMapper
}

func NewPlexSyncHandler(db *sql.DB, tmdbClient *services.TMDBClient) *PlexSyncHandler {
	return &PlexSyncHandler{
		db:         db,
		plexClient: services.NewPlexClient(),
		mapper:     services.NewPlexTMDBMapper(db, tmdbClient),
	}
}

// SyncPlexLibrary syncs a user's Plex library with TMDB mappings
func (h *PlexSyncHandler) SyncPlexLibrary(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's Plex token
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name, authUser.AvatarURL)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	var plexToken string
	err = h.db.QueryRow(`
		SELECT plex_token FROM user_plex_tokens WHERE user_id = ?
	`, user.ID).Scan(&plexToken)

	if err == sql.ErrNoRows {
		http.Error(w, "Plex not connected", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Failed to get Plex token", http.StatusInternalServerError)
		return
	}

	// Get user's Plex servers
	servers, err := h.plexClient.GetServers(plexToken)
	if err != nil {
		http.Error(w, "Failed to get Plex servers", http.StatusInternalServerError)
		return
	}

	var syncResults []map[string]interface{}
	totalSynced := 0
	totalErrors := 0
	var debugInfo []string

	// For each server, get libraries and sync movies
	for _, server := range servers {
		serverName, _ := server["name"].(string)
		
		// Extract server URL from connections array - only use external connections
		var serverURL string
		if connections, ok := server["connections"].([]interface{}); ok {
			// Only use external (non-local) connections
			for _, conn := range connections {
				if connMap, ok := conn.(map[string]interface{}); ok {
					if uri, ok := connMap["uri"].(string); ok {
						if local, ok := connMap["local"].(bool); ok && !local {
							// External connection - use this
							serverURL = uri
							break
						}
					}
				}
			}
		}
		
		fmt.Printf("DEBUG: Processing Plex server: %s\n", serverName)
		fmt.Printf("DEBUG: Selected server URL: '%s'\n", serverURL)
		
		debugInfo = append(debugInfo, fmt.Sprintf("Processing server: %s", serverName))
		debugInfo = append(debugInfo, fmt.Sprintf("  Selected URL: '%s'", serverURL))
		
		if serverURL == "" {
			debugInfo = append(debugInfo, "Skipping server with no accessible URL")
			continue
		}

		// Check if user owns this server
		owned, _ := server["owned"].(bool)
		
		// Get libraries for this server
		libraries, err := h.plexClient.GetLibraries(plexToken, serverURL)
		if err != nil {
			if !owned {
				debugInfo = append(debugInfo, fmt.Sprintf("Cannot access libraries on shared server %s (not owner): %v", serverName, err))
				debugInfo = append(debugInfo, "Trying alternative endpoints for shared users...")
				
				// Try alternative approach for shared users
				movies, err := h.trySharedUserSync(plexToken, serverURL, serverName)
				if err != nil {
					debugInfo = append(debugInfo, fmt.Sprintf("Alternative sync failed: %v", err))
					totalErrors++
					continue
				} else if len(movies) > 0 {
					debugInfo = append(debugInfo, fmt.Sprintf("Found %d movies via alternative method", len(movies)))
					
					// Process movies directly without library structure
					libraryResults := map[string]interface{}{
						"server":   serverName,
						"library":  "Shared Content",
						"movies":   len(movies),
						"synced":   0,
						"errors":   0,
					}
					
					for _, movie := range movies {
						year := &movie.Year
						if movie.Year == 0 {
							year = nil
						}
						
						_, err := h.mapper.GetOrCreateMapping(movie.GUID, movie.Title, year, movie.RatingKey)
						if err != nil {
							libraryResults["errors"] = libraryResults["errors"].(int) + 1
							totalErrors++
						} else {
							libraryResults["synced"] = libraryResults["synced"].(int) + 1
							totalSynced++
						}
					}
					
					syncResults = append(syncResults, libraryResults)
					continue
				}
			} else {
				debugInfo = append(debugInfo, fmt.Sprintf("Error getting libraries from %s: %v", serverName, err))
			}
			totalErrors++
			continue
		}

		debugInfo = append(debugInfo, fmt.Sprintf("Found %d libraries on server %s", len(libraries), serverName))

		// Process movie libraries only
		for _, library := range libraries {
			libType, _ := library["type"].(string)
			if libType != "movie" {
				continue
			}

			libKey, _ := library["key"].(string)
			libTitle, _ := library["title"].(string)
			
			// Get all movies in this library
			movies, err := h.plexClient.GetLibraryContent(plexToken, serverURL, libKey)
			if err != nil {
				totalErrors++
				continue
			}

			// Process each movie
			libraryResults := map[string]interface{}{
				"server":   serverName,
				"library":  libTitle,
				"movies":   len(movies),
				"synced":   0,
				"errors":   0,
			}

			for _, movie := range movies {
				// Try to create mapping
				year := &movie.Year
				if movie.Year == 0 {
					year = nil
				}
				
				_, err := h.mapper.GetOrCreateMapping(movie.GUID, movie.Title, year, movie.RatingKey)
				if err != nil {
					libraryResults["errors"] = libraryResults["errors"].(int) + 1
					totalErrors++
				} else {
					libraryResults["synced"] = libraryResults["synced"].(int) + 1
					totalSynced++
				}
			}

			syncResults = append(syncResults, libraryResults)
		}
	}

	response := map[string]interface{}{
		"success":      true,
		"totalSynced":  totalSynced,
		"totalErrors":  totalErrors,
		"libraries":    syncResults,
		"debugInfo":    debugInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPlexMappings returns all Plex-TMDB mappings with pagination
func (h *PlexSyncHandler) GetPlexMappings(w http.ResponseWriter, r *http.Request) {
	_, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get pagination parameters
	page := 1
	limit := 50
	
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	
	offset := (page - 1) * limit

	// Get mappings
	mappings, totalCount, err := h.mapper.GetAllMappings(limit, offset)
	if err != nil {
		http.Error(w, "Failed to get mappings", http.StatusInternalServerError)
		return
	}

	totalPages := (totalCount + limit - 1) / limit

	response := map[string]interface{}{
		"mappings":     mappings,
		"count":        len(mappings),
		"total":        totalCount,
		"totalPages":   totalPages,
		"currentPage":  page,
		"perPage":      limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SearchPlexMappings searches mappings by title
func (h *PlexSyncHandler) SearchPlexMappings(w http.ResponseWriter, r *http.Request) {
	_, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	title := r.URL.Query().Get("title")
	if title == "" {
		http.Error(w, "Title parameter required", http.StatusBadRequest)
		return
	}

	mappings, err := h.mapper.SearchMappingsByTitle(title)
	if err != nil {
		http.Error(w, "Failed to search mappings", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"mappings": mappings,
		"count":    len(mappings),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// trySharedUserSync attempts to sync movies for shared users using alternative endpoints
func (h *PlexSyncHandler) trySharedUserSync(token, serverURL, serverName string) ([]services.PlexLibraryItem, error) {
	// For shared users, we can't access the full library endpoints
	// This is a placeholder that returns empty results since we've moved to on-demand search
	return []services.PlexLibraryItem{}, fmt.Errorf("shared user sync not supported - use on-demand search instead")
}