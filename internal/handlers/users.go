package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

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
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name, authUser.AvatarURL)
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
	_, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get query parameters for search and pagination
	searchQuery := r.URL.Query().Get("search")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	
	// Set defaults
	page := 1
	limit := 20
	
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	
	offset := (page - 1) * limit

	// Build the SQL query
	var query string
	var args []interface{}
	
	if searchQuery != "" {
		// Search by name or username with list counts and unique movie counts
		query = `
			SELECT u.id, u.auth0_id, u.email, u.name, u.username, u.avatar_url, u.created_at,
			       COUNT(DISTINCT l.id) as list_count,
			       COUNT(DISTINCT lm.movie_id) as movie_count
			FROM users u
			LEFT JOIN lists l ON u.id = l.user_id AND l.is_public = 1
			LEFT JOIN list_movies lm ON l.id = lm.list_id
			WHERE (u.name LIKE ? OR u.username LIKE ?) 
			GROUP BY u.id, u.auth0_id, u.email, u.name, u.username, u.avatar_url, u.created_at
			ORDER BY u.created_at DESC 
			LIMIT ? OFFSET ?
		`
		searchPattern := "%" + searchQuery + "%"
		args = []interface{}{searchPattern, searchPattern, limit, offset}
	} else {
		// TODO: Remove current user from community list later
		// Get all users (including current user for now) with list counts and unique movie counts
		query = `
			SELECT u.id, u.auth0_id, u.email, u.name, u.username, u.avatar_url, u.created_at,
			       COUNT(DISTINCT l.id) as list_count,
			       COUNT(DISTINCT lm.movie_id) as movie_count
			FROM users u
			LEFT JOIN lists l ON u.id = l.user_id AND l.is_public = 1
			LEFT JOIN list_movies lm ON l.id = lm.list_id
			GROUP BY u.id, u.auth0_id, u.email, u.name, u.username, u.avatar_url, u.created_at
			ORDER BY u.created_at DESC 
			LIMIT ? OFFSET ?
		`
		args = []interface{}{limit, offset}
	}

	// Get total count for pagination
	var countQuery string
	var countArgs []interface{}
	
	if searchQuery != "" {
		countQuery = `
			SELECT COUNT(DISTINCT u.id)
			FROM users u
			WHERE (u.name LIKE ? OR u.username LIKE ?)
		`
		searchPattern := "%" + searchQuery + "%"
		countArgs = []interface{}{searchPattern, searchPattern}
	} else {
		countQuery = `SELECT COUNT(*) FROM users`
		countArgs = []interface{}{}
	}
	
	var totalCount int
	err = h.db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		http.Error(w, "Failed to count users", http.StatusInternalServerError)
		return
	}
	
	totalPages := (totalCount + limit - 1) / limit

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var auth0ID, email, name string
		var username *string
		var avatarURL *string
		var createdAt string
		var listCount int
		var movieCount int

		err := rows.Scan(&id, &auth0ID, &email, &name, &username, &avatarURL, &createdAt, &listCount, &movieCount)
		if err != nil {
			continue // Skip problematic rows
		}

		user := map[string]interface{}{
			"id":          id,
			"auth0_id":    auth0ID,
			"name":        name,
			"created_at":  createdAt,
			"list_count":  listCount,
			"movie_count": movieCount,
			// Don't expose email for privacy
		}

		if username != nil {
			user["username"] = *username
		}

		if avatarURL != nil {
			user["avatar_url"] = *avatarURL
		}

		users = append(users, user)
	}

	response := map[string]interface{}{
		"users":       users,
		"count":       len(users),
		"total":       totalCount,
		"total_pages": totalPages,
		"current_page": page,
		"per_page":    limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Get path parameter
	userIDStr := utils.GetPathParam(r, "id")
	
	// Get user by Auth0 ID
	var user types.User
	err := h.db.QueryRow("SELECT id, auth0_id, email, name, username, avatar_url, created_at FROM users WHERE auth0_id = ?", userIDStr).Scan(
		&user.ID, &user.Auth0ID, &user.Email, &user.Name, &user.Username, &user.AvatarURL, &user.Created)
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
		"id":         user.ID,
		"auth0_id":   user.Auth0ID,
		"name":       user.Name,
		"username":   user.Username,
		"created_at": user.Created,
	}

	if user.AvatarURL != nil {
		response["avatar_url"] = *user.AvatarURL
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
	currentUser, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name, authUser.AvatarURL)
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
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name, authUser.AvatarURL)
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
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name, authUser.AvatarURL)
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

func (h *UserHandler) GetUserMovies(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get path parameter
	userIDStr := utils.GetPathParam(r, "id")
	
	// Get query parameters for pagination
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	
	// Set defaults
	page := 1
	limit := 20
	
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	
	offset := (page - 1) * limit
	
	// Get current user for authentication
	currentUser, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name, authUser.AvatarURL)
	if err != nil {
		http.Error(w, "Failed to get current user", http.StatusInternalServerError)
		return
	}

	// Determine target user ID
	var targetUserID int
	if userIDStr == "me" || userIDStr == "" {
		targetUserID = currentUser.ID
	} else {
		// Get user by Auth0 ID
		var targetUser types.User
		err = h.db.QueryRow("SELECT id, auth0_id, email, name, username, avatar_url, created_at FROM users WHERE auth0_id = ?", userIDStr).Scan(
			&targetUser.ID, &targetUser.Auth0ID, &targetUser.Email, &targetUser.Name, &targetUser.Username, &targetUser.AvatarURL, &targetUser.Created)
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

	// Get total count for pagination
	var countQuery string
	if isOwnProfile {
		countQuery = `
			SELECT COUNT(DISTINCT m.id)
			FROM list_movies lm
			JOIN movies m ON lm.movie_id = m.id
			JOIN lists l ON lm.list_id = l.id
			WHERE l.user_id = ?
		`
	} else {
		countQuery = `
			SELECT COUNT(DISTINCT m.id)
			FROM list_movies lm
			JOIN movies m ON lm.movie_id = m.id
			JOIN lists l ON lm.list_id = l.id
			WHERE l.user_id = ? AND l.is_public = 1
		`
	}
	
	var totalCount int
	err = h.db.QueryRow(countQuery, targetUserID).Scan(&totalCount)
	if err != nil {
		http.Error(w, "Failed to count user movies", http.StatusInternalServerError)
		return
	}
	
	totalPages := (totalCount + limit - 1) / limit

	// Get movies from user's lists (with privacy filtering and pagination)
	var query string
	if isOwnProfile {
		// Own profile: show movies from all lists
		query = `
			SELECT DISTINCT m.id, m.tmdb_id, m.title, m.year, m.poster_url, m.synopsis, lm.added_at,
			       l.id as list_id, l.name as list_name
			FROM list_movies lm
			JOIN movies m ON lm.movie_id = m.id
			JOIN lists l ON lm.list_id = l.id
			WHERE l.user_id = ?
			ORDER BY lm.added_at DESC
			LIMIT ? OFFSET ?
		`
	} else {
		// Other's profile: only show movies from public lists
		query = `
			SELECT DISTINCT m.id, m.tmdb_id, m.title, m.year, m.poster_url, m.synopsis, lm.added_at,
			       l.id as list_id, l.name as list_name
			FROM list_movies lm
			JOIN movies m ON lm.movie_id = m.id
			JOIN lists l ON lm.list_id = l.id
			WHERE l.user_id = ? AND l.is_public = 1
			ORDER BY lm.added_at DESC
			LIMIT ? OFFSET ?
		`
	}

	rows, err := h.db.Query(query, targetUserID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to get user movies", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var movies []map[string]interface{}
	for rows.Next() {
		var movieID, tmdbID, listID int
		var title, synopsis, listName string
		var year *int
		var posterURL *string
		var addedAt string

		err := rows.Scan(&movieID, &tmdbID, &title, &year, &posterURL, &synopsis, &addedAt, &listID, &listName)
		if err != nil {
			continue
		}

		movie := map[string]interface{}{
			"id":        movieID,
			"tmdb_id":   tmdbID,
			"title":     title,
			"year":      year,
			"synopsis":  synopsis,
			"added_at":  addedAt,
			"list_id":   listID,
			"list_name": listName,
		}

		if posterURL != nil {
			movie["poster_url"] = *posterURL
		}

		movies = append(movies, movie)
	}

	response := map[string]interface{}{
		"movies":       movies,
		"count":        len(movies),
		"total":        totalCount,
		"total_pages":  totalPages,
		"current_page": page,
		"per_page":     limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}