package handlers

import (
	"encoding/json"
	"net/http"

	"moviedb/internal/services"
)

type SyncHandler struct {
	movieSyncService *services.MovieSyncService
}

func NewSyncHandler(movieSyncService *services.MovieSyncService) *SyncHandler {
	return &SyncHandler{
		movieSyncService: movieSyncService,
	}
}

func (h *SyncHandler) TriggerMovieSync(w http.ResponseWriter, r *http.Request) {
	err := h.movieSyncService.ManualSync()
	if err != nil {
		http.Error(w, "Failed to trigger sync", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Movie sync triggered successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *SyncHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.movieSyncService.GetSyncStatus()
	if err != nil {
		http.Error(w, "Failed to get sync status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}