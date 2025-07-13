package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"moviedb/internal/auth"
	"moviedb/internal/database"
	"moviedb/internal/types"
	"moviedb/internal/utils"
)

type ListHandler struct {
	db *sql.DB
}

func NewListHandler(db *sql.DB) *ListHandler {
	return &ListHandler{db: db}
}

func (h *ListHandler) GetLists(w http.ResponseWriter, r *http.Request) {
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

	// Get user's lists with movie counts
	rows, err := h.db.Query(`
		SELECT l.id, l.name, l.description, l.is_public, l.created_at,
		       COUNT(lm.movie_id) as movie_count
		FROM lists l
		LEFT JOIN list_movies lm ON l.id = lm.list_id
		WHERE l.user_id = ?
		GROUP BY l.id, l.name, l.description, l.is_public, l.created_at
		ORDER BY l.created_at DESC
	`, user.ID)
	if err != nil {
		http.Error(w, "Failed to get lists", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lists []map[string]interface{}
	for rows.Next() {
		var id int
		var name, description string
		var isPublic bool
		var createdAt time.Time
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"lists": lists,
	})
}

func (h *ListHandler) CreateList(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req types.CreateListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		http.Error(w, "List name is required", http.StatusBadRequest)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Create list
	result, err := h.db.Exec(`
		INSERT INTO lists (user_id, name, description, is_public, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, user.ID, req.Name, req.Description, req.IsPublic, time.Now())
	if err != nil {
		http.Error(w, "Failed to create list", http.StatusInternalServerError)
		return
	}

	listID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Failed to get list ID", http.StatusInternalServerError)
		return
	}

	// Return created list
	response := map[string]interface{}{
		"id":          int(listID),
		"name":        req.Name,
		"description": req.Description,
		"is_public":   req.IsPublic,
		"movie_count": 0,
		"created_at":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *ListHandler) GetList(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get path parameter
	listIDStr := utils.GetPathParam(r, "id")
	listID, err := strconv.Atoi(listIDStr)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Get list details with movies
	var listName, listDescription string
	var isPublic bool
	var createdAt time.Time
	var listUserID int

	err = h.db.QueryRow(`
		SELECT user_id, name, description, is_public, created_at
		FROM lists 
		WHERE id = ?
	`, listID).Scan(&listUserID, &listName, &listDescription, &isPublic, &createdAt)
	
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to get list", http.StatusInternalServerError)
		return
	}

	// Check if user has access (owner or public list)
	if listUserID != user.ID && !isPublic {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get movies in this list
	rows, err := h.db.Query(`
		SELECT m.id, m.tmdb_id, m.title, m.year, m.poster_url, m.synopsis, lm.added_at
		FROM list_movies lm
		JOIN movies m ON lm.movie_id = m.id
		WHERE lm.list_id = ?
		ORDER BY lm.added_at DESC
	`, listID)
	if err != nil {
		http.Error(w, "Failed to get list movies", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var movies []map[string]interface{}
	for rows.Next() {
		var movieID, tmdbID int
		var title, synopsis string
		var year *int
		var posterURL *string
		var addedAt time.Time

		err := rows.Scan(&movieID, &tmdbID, &title, &year, &posterURL, &synopsis, &addedAt)
		if err != nil {
			continue
		}

		movie := map[string]interface{}{
			"id":       movieID,
			"tmdb_id":  tmdbID,
			"title":    title,
			"year":     year,
			"synopsis": synopsis,
			"added_at": addedAt,
		}

		if posterURL != nil {
			movie["poster_url"] = *posterURL
		}

		movies = append(movies, movie)
	}

	response := map[string]interface{}{
		"id":          listID,
		"name":        listName,
		"description": listDescription,
		"is_public":   isPublic,
		"created_at":  createdAt,
		"movie_count": len(movies),
		"movies":      movies,
		"is_owner":    listUserID == user.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ListHandler) UpdateList(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get path parameter
	listIDStr := utils.GetPathParam(r, "id")
	listID, err := strconv.Atoi(listIDStr)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req types.CreateListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		http.Error(w, "List name is required", http.StatusBadRequest)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Verify list belongs to user
	var listUserID int
	err = h.db.QueryRow("SELECT user_id FROM lists WHERE id = ?", listID).Scan(&listUserID)
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to verify list ownership", http.StatusInternalServerError)
		return
	}
	if listUserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Update list
	_, err = h.db.Exec(`
		UPDATE lists 
		SET name = ?, description = ?, is_public = ?
		WHERE id = ?
	`, req.Name, req.Description, req.IsPublic, listID)
	if err != nil {
		http.Error(w, "Failed to update list", http.StatusInternalServerError)
		return
	}

	// Get updated list data
	var name, description string
	var isPublic bool
	var createdAt time.Time
	var movieCount int

	err = h.db.QueryRow(`
		SELECT l.name, l.description, l.is_public, l.created_at,
		       COUNT(lm.movie_id) as movie_count
		FROM lists l
		LEFT JOIN list_movies lm ON l.id = lm.list_id
		WHERE l.id = ?
		GROUP BY l.id, l.name, l.description, l.is_public, l.created_at
	`, listID).Scan(&name, &description, &isPublic, &createdAt, &movieCount)
	if err != nil {
		http.Error(w, "Failed to get updated list", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":          listID,
		"name":        name,
		"description": description,
		"is_public":   isPublic,
		"created_at":  createdAt,
		"movie_count": movieCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ListHandler) DeleteList(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get path parameter
	listIDStr := utils.GetPathParam(r, "id")
	listID, err := strconv.Atoi(listIDStr)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Verify list belongs to user
	var listUserID int
	err = h.db.QueryRow("SELECT user_id FROM lists WHERE id = ?", listID).Scan(&listUserID)
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to verify list ownership", http.StatusInternalServerError)
		return
	}
	if listUserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Delete list movies first (foreign key constraint)
	_, err = h.db.Exec("DELETE FROM list_movies WHERE list_id = ?", listID)
	if err != nil {
		http.Error(w, "Failed to delete list movies", http.StatusInternalServerError)
		return
	}

	// Delete list
	_, err = h.db.Exec("DELETE FROM lists WHERE id = ?", listID)
	if err != nil {
		http.Error(w, "Failed to delete list", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "List deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ListHandler) AddMovieToList(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get path parameters
	listIDStr := utils.GetPathParam(r, "id")
	movieIDStr := utils.GetPathParam(r, "movieId")

	listID, err := strconv.Atoi(listIDStr)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	tmdbID, err := strconv.Atoi(movieIDStr)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Verify list belongs to user
	var listUserID int
	err = h.db.QueryRow("SELECT user_id FROM lists WHERE id = ?", listID).Scan(&listUserID)
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to verify list ownership", http.StatusInternalServerError)
		return
	}
	if listUserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Find or create movie in our database using TMDB ID
	var movieID int
	err = h.db.QueryRow("SELECT id FROM movies WHERE tmdb_id = ?", tmdbID).Scan(&movieID)
	if err == sql.ErrNoRows {
		// Movie doesn't exist in our database, we need to fetch it from TMDB first
		http.Error(w, "Movie not found in database. Please view the movie details first to cache it.", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to find movie", http.StatusInternalServerError)
		return
	}

	// Check if movie is already in the list
	var existingID int
	err = h.db.QueryRow("SELECT id FROM list_movies WHERE list_id = ? AND movie_id = ?", listID, movieID).Scan(&existingID)
	if err == nil {
		// Movie is already in the list
		http.Error(w, "Movie is already in this list", http.StatusConflict)
		return
	}
	if err != sql.ErrNoRows {
		http.Error(w, "Failed to check if movie is in list", http.StatusInternalServerError)
		return
	}

	// Add movie to list
	_, err = h.db.Exec(`
		INSERT INTO list_movies (list_id, movie_id, added_at)
		VALUES (?, ?, ?)
	`, listID, movieID, time.Now())
	if err != nil {
		http.Error(w, "Failed to add movie to list", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Movie added to list",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ListHandler) RemoveMovieFromList(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get path parameters
	listIDStr := utils.GetPathParam(r, "id")
	movieIDStr := utils.GetPathParam(r, "movieId")

	listID, err := strconv.Atoi(listIDStr)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	tmdbID, err := strconv.Atoi(movieIDStr)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Verify list belongs to user
	var listUserID int
	err = h.db.QueryRow("SELECT user_id FROM lists WHERE id = ?", listID).Scan(&listUserID)
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to verify list ownership", http.StatusInternalServerError)
		return
	}
	if listUserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Find movie in our database using TMDB ID
	var movieID int
	err = h.db.QueryRow("SELECT id FROM movies WHERE tmdb_id = ?", tmdbID).Scan(&movieID)
	if err == sql.ErrNoRows {
		http.Error(w, "Movie not found in database", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to find movie", http.StatusInternalServerError)
		return
	}

	// Remove movie from list
	_, err = h.db.Exec(`
		DELETE FROM list_movies 
		WHERE list_id = ? AND movie_id = ?
	`, listID, movieID)
	if err != nil {
		http.Error(w, "Failed to remove movie from list", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Movie removed from list",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ListHandler) GetMovieInLists(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get path parameter
	movieIDStr := utils.GetPathParam(r, "movieId")
	tmdbID, err := strconv.Atoi(movieIDStr)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(h.db, authUser.Auth0ID, authUser.Email, authUser.Name)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Find movie in our database using TMDB ID
	var movieID int
	err = h.db.QueryRow("SELECT id FROM movies WHERE tmdb_id = ?", tmdbID).Scan(&movieID)
	if err == sql.ErrNoRows {
		// Movie not in database, return empty list
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"list_ids": []int{},
		})
		return
	}
	if err != nil {
		http.Error(w, "Failed to find movie", http.StatusInternalServerError)
		return
	}

	// Get lists that contain this movie for this user
	rows, err := h.db.Query(`
		SELECT l.id
		FROM lists l
		JOIN list_movies lm ON l.id = lm.list_id
		WHERE l.user_id = ? AND lm.movie_id = ?
	`, user.ID, movieID)
	if err != nil {
		http.Error(w, "Failed to get movie lists", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var listIDs []int
	for rows.Next() {
		var listID int
		if err := rows.Scan(&listID); err != nil {
			continue
		}
		listIDs = append(listIDs, listID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"list_ids": listIDs,
	})
}

func (h *ListHandler) GetAllUserMovies(w http.ResponseWriter, r *http.Request) {
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

	// Get all movies from all user's lists
	rows, err := h.db.Query(`
		SELECT DISTINCT m.id, m.tmdb_id, m.title, m.year, m.poster_url, m.synopsis, lm.added_at,
		       l.id as list_id, l.name as list_name
		FROM list_movies lm
		JOIN movies m ON lm.movie_id = m.id
		JOIN lists l ON lm.list_id = l.id
		WHERE l.user_id = ?
		ORDER BY lm.added_at DESC
	`, user.ID)
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
		var addedAt time.Time

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"movies": movies,
	})
}