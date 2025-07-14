package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"moviedb/internal/auth"
	"moviedb/internal/database"
	"moviedb/internal/services"
)

type WatchProvidersHandler struct {
	service *services.WatchProvidersService
	db      *sql.DB
}

func NewWatchProvidersHandler(db *sql.DB, tmdbClient *services.TMDBClient, plexClient *services.PlexClient) *WatchProvidersHandler {
	return &WatchProvidersHandler{
		service: services.NewWatchProvidersService(db, tmdbClient, plexClient),
		db:      db,
	}
}

// GetMovieWatchProviders returns watch provider information for a movie
func (h *WatchProvidersHandler) GetMovieWatchProviders(w http.ResponseWriter, r *http.Request) {
	// Get TMDB ID from URL path
	tmdbIDStr := r.PathValue("id")
	if tmdbIDStr == "" {
		http.Error(w, "Movie ID is required", http.StatusBadRequest)
		return
	}

	tmdbID, err := strconv.Atoi(tmdbIDStr)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	// Get region from query params (default to NO for Norway)
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "NO"
	}

	// Get user ID (authentication is required for this endpoint)
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user ID for Plex availability
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name, authUser.AvatarURL)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	userID := &user.ID

	// Get watch providers
	providers, err := h.service.GetWatchProviders(tmdbID, region, userID)
	if err != nil {
		http.Error(w, "Failed to get watch providers", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

// ClearExpiredCache clears expired cache entries (admin endpoint)
func (h *WatchProvidersHandler) ClearExpiredCache(w http.ResponseWriter, r *http.Request) {
	// This could be protected with admin auth in the future
	_, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = h.service.ClearExpiredCache()
	if err != nil {
		http.Error(w, "Failed to clear cache", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Expired cache entries cleared",
	})
}