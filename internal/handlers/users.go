package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"moviedb/internal/auth"
	"moviedb/internal/database"
	"moviedb/internal/types"
	"moviedb/internal/utils"
)

type UserHandler struct {
	db *sql.DB
}

func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement user update
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *UserHandler) SetupUser(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement user setup
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get all users
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Get path parameter
	userIDStr := utils.GetPathParam(r, "id")
	
	// Get user by Auth0 ID
	var user types.User
	err := h.db.QueryRow("SELECT id, auth0_id, email, name, username, created_at FROM users WHERE auth0_id = ?", userIDStr).Scan(
		&user.ID, &user.Auth0ID, &user.Email, &user.Name, &user.Username, &user.Created)
	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Return public user information (no sensitive data)
	response := map[string]interface{}{
		"id":       user.ID,
		"auth0_id": user.Auth0ID,
		"name":     user.Name,
		"username": user.Username,
		"created_at": user.Created,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *UserHandler) GetUserLists(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get path parameter
	userIDStr := utils.GetPathParam(r, "id")
	
	// Get or create current user in database
	currentUser, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name)
	if err != nil {
		http.Error(w, "Failed to get current user", http.StatusInternalServerError)
		return
	}

	// Determine target user ID
	var targetUserID int
	if userIDStr == "me" || userIDStr == "" {
		targetUserID = currentUser.ID
	} else {
		// For now, treat userID as Auth0 ID - in a real app you might want numeric IDs
		// Get user by Auth0 ID
		var targetUser types.User
		err = h.db.QueryRow("SELECT id, auth0_id, email, name, username, created_at FROM users WHERE auth0_id = ?", userIDStr).Scan(
			&targetUser.ID, &targetUser.Auth0ID, &targetUser.Email, &targetUser.Name, &targetUser.Username, &targetUser.Created)
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Failed to get target user", http.StatusInternalServerError)
			return
		}
		targetUserID = targetUser.ID
	}

	isOwnProfile := targetUserID == currentUser.ID

	// Get lists with privacy filtering
	var query string
	if isOwnProfile {
		// Own profile: show all lists
		query = `
			SELECT l.id, l.name, l.description, l.is_public, l.created_at,
			       COUNT(lm.movie_id) as movie_count
			FROM lists l
			LEFT JOIN list_movies lm ON l.id = lm.list_id
			WHERE l.user_id = ?
			GROUP BY l.id, l.name, l.description, l.is_public, l.created_at
			ORDER BY l.created_at DESC
		`
	} else {
		// Other's profile: only show public lists
		query = `
			SELECT l.id, l.name, l.description, l.is_public, l.created_at,
			       COUNT(lm.movie_id) as movie_count
			FROM lists l
			LEFT JOIN list_movies lm ON l.id = lm.list_id
			WHERE l.user_id = ? AND l.is_public = 1
			GROUP BY l.id, l.name, l.description, l.is_public, l.created_at
			ORDER BY l.created_at DESC
		`
	}

	rows, err := h.db.Query(query, targetUserID)
	if err != nil {
		http.Error(w, "Failed to get user lists", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lists []map[string]interface{}
	for rows.Next() {
		var id int
		var name, description string
		var isPublic bool
		var createdAt string
		var movieCount int

		err := rows.Scan(&id, &name, &description, &isPublic, &createdAt, &movieCount)
		if err != nil {
			continue
		}

		list := map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description,
			"is_public":   isPublic,
			"created_at":  createdAt,
			"movie_count": movieCount,
		}

		lists = append(lists, list)
	}

	response := map[string]interface{}{
		"lists": lists,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *UserHandler) AddFriend(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement add friend
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *UserHandler) RemoveFriend(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement remove friend
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *UserHandler) GetUserPreferences(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Get user preferences
	prefs, err := database.GetUserPreferences(h.db, user.ID)
	if err != nil {
		http.Error(w, "Failed to get preferences", http.StatusInternalServerError)
		return
	}

	// Return preferences in the format expected by frontend
	response := map[string]interface{}{
		"darkMode": prefs.DarkMode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *UserHandler) UpdateUserPreferences(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req types.UpdatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Ensure preferences exist first
	_, err = database.GetUserPreferences(h.db, user.ID)
	if err != nil {
		http.Error(w, "Failed to get preferences", http.StatusInternalServerError)
		return
	}

	// Update preferences
	err = database.UpdateUserPreferences(h.db, user.ID, req.DarkMode)
	if err != nil {
		http.Error(w, "Failed to update preferences", http.StatusInternalServerError)
		return
	}

	// Return success
	response := map[string]interface{}{
		"success":  true,
		"darkMode": req.DarkMode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}