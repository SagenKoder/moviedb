package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"moviedb/internal/auth"
	"moviedb/internal/database"
	"moviedb/internal/services"
)

type PlexHandler struct {
	db         *sql.DB
	plexClient *services.PlexClient
}

type PlexPinRequest struct {
	PinID     int    `json:"pinId"`
	PinCode   string `json:"pinCode"`
	ExpiresAt string `json:"expiresAt"`
}

type PlexStatusResponse struct {
	Connected    bool   `json:"connected"`
	Username     string `json:"username,omitempty"`
	FriendlyName string `json:"friendlyName,omitempty"`
	Email        string `json:"email,omitempty"`
	Thumb        string `json:"thumb,omitempty"`
	ServerCount  int    `json:"serverCount,omitempty"`
	ConnectedAt  string `json:"connectedAt,omitempty"`
}

func NewPlexHandler(db *sql.DB) *PlexHandler {
	return &PlexHandler{
		db:         db,
		plexClient: services.NewPlexClient(),
	}
}

// StartPlexAuth initiates the Plex PIN-based authentication flow
func (h *PlexHandler) StartPlexAuth(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name, authUser.AvatarURL)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Check if user already has Plex connected
	var existingToken string
	err = h.db.QueryRow(`
		SELECT plex_token FROM user_plex_tokens WHERE user_id = ?
	`, user.ID).Scan(&existingToken)

	if err == nil {
		http.Error(w, "Plex account already connected", http.StatusConflict)
		return
	}

	// Request PIN from Plex
	pinResp, err := h.plexClient.RequestPin()
	if err != nil {
		http.Error(w, "Failed to request Plex PIN", http.StatusInternalServerError)
		return
	}

	// Store PIN attempt in database
	_, err = h.db.Exec(`
		INSERT INTO plex_auth_attempts (user_id, pin_id, pin_code, expires_at)
		VALUES (?, ?, ?, ?)
	`, user.ID, pinResp.ID, pinResp.Code, pinResp.ExpiresAt)

	if err != nil {
		http.Error(w, "Failed to store PIN attempt", http.StatusInternalServerError)
		return
	}

	response := PlexPinRequest{
		PinID:     pinResp.ID,
		PinCode:   pinResp.Code,
		ExpiresAt: pinResp.ExpiresAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckPlexAuth checks if the PIN has been authorized
func (h *PlexHandler) CheckPlexAuth(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name, authUser.AvatarURL)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	pinIDStr := r.URL.Query().Get("pinId")

	if pinIDStr == "" {
		http.Error(w, "Pin ID is required", http.StatusBadRequest)
		return
	}

	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(w, "Invalid pin ID", http.StatusBadRequest)
		return
	}

	// Check if this PIN attempt belongs to the user
	var storedPinID int
	var expiresAt time.Time
	err = h.db.QueryRow(`
		SELECT pin_id, expires_at FROM plex_auth_attempts 
		WHERE user_id = ? AND pin_id = ? AND completed = 0
	`, user.ID, pinID).Scan(&storedPinID, &expiresAt)

	if err != nil {
		http.Error(w, "PIN attempt not found", http.StatusNotFound)
		return
	}

	// Check if PIN has expired
	if time.Now().After(expiresAt) {
		http.Error(w, "PIN has expired", http.StatusGone)
		return
	}

	// Check PIN status with Plex
	pinResp, err := h.plexClient.CheckPin(pinID)
	if err != nil {
		http.Error(w, "Failed to check PIN status", http.StatusInternalServerError)
		return
	}

	// If no token yet, PIN hasn't been authorized
	if pinResp.AuthToken == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authorized": false,
			"expiresAt":  expiresAt.Format(time.RFC3339),
		})
		return
	}

	// PIN has been authorized, get user info
	plexUser, err := h.plexClient.GetUser(pinResp.AuthToken)
	if err != nil {
		http.Error(w, "Failed to get Plex user info", http.StatusInternalServerError)
		return
	}

	// Get server count
	servers, err := h.plexClient.GetServers(pinResp.AuthToken)
	if err != nil {
		// Don't fail if we can't get servers, just set count to 0
		servers = []map[string]interface{}{}
	}

	// Store the Plex token and user info
	_, err = h.db.Exec(`
		INSERT INTO user_plex_tokens (user_id, plex_token, plex_username, plex_friendly_name, plex_email, plex_thumb, server_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			plex_token = excluded.plex_token,
			plex_username = excluded.plex_username,
			plex_friendly_name = excluded.plex_friendly_name,
			plex_email = excluded.plex_email,
			plex_thumb = excluded.plex_thumb,
			server_count = excluded.server_count,
			updated_at = CURRENT_TIMESTAMP
	`, user.ID, pinResp.AuthToken, plexUser.Username, plexUser.FriendlyName, plexUser.Email, plexUser.Thumb, len(servers))

	if err != nil {
		http.Error(w, "Failed to store Plex token", http.StatusInternalServerError)
		return
	}

	// Mark PIN attempt as completed
	_, err = h.db.Exec(`
		UPDATE plex_auth_attempts SET completed = 1 WHERE pin_id = ?
	`, pinID)

	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to mark PIN attempt as completed: %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authorized": true,
		"user": map[string]interface{}{
			"username":    plexUser.Username,
			"email":       plexUser.Email,
			"thumb":       plexUser.Thumb,
			"serverCount": len(servers),
		},
	})
}

// GetPlexStatus returns the current Plex connection status
func (h *PlexHandler) GetPlexStatus(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name, authUser.AvatarURL)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	var token, username, email, thumb string
	var friendlyName *string // Use pointer to handle NULL
	var serverCount int
	var createdAt time.Time

	err = h.db.QueryRow(`
		SELECT plex_token, plex_username, plex_friendly_name, plex_email, plex_thumb, server_count, created_at
		FROM user_plex_tokens WHERE user_id = ?
	`, user.ID).Scan(&token, &username, &friendlyName, &email, &thumb, &serverCount, &createdAt)

	if err == sql.ErrNoRows {
		// Not connected
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PlexStatusResponse{Connected: false})
		return
	}

	if err != nil {
		http.Error(w, "Failed to get Plex status", http.StatusInternalServerError)
		return
	}

	// Connected - handle NULL friendlyName
	friendlyNameStr := ""
	if friendlyName != nil {
		friendlyNameStr = *friendlyName
	}
	
	response := PlexStatusResponse{
		Connected:    true,
		Username:     username,
		FriendlyName: friendlyNameStr,
		Email:        email,
		Thumb:        thumb,
		ServerCount:  serverCount,
		ConnectedAt:  createdAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DisconnectPlex removes the Plex integration
func (h *PlexHandler) DisconnectPlex(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name, authUser.AvatarURL)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	_, err = h.db.Exec(`DELETE FROM user_plex_tokens WHERE user_id = ?`, user.ID)
	if err != nil {
		http.Error(w, "Failed to disconnect Plex", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}