package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"moviedb/internal/services"
	"moviedb/internal/utils"
)

type MovieHandler struct {
	db         *sql.DB
	tmdbClient *services.TMDBClient
}

func NewMovieHandler(db *sql.DB, tmdbClient *services.TMDBClient) *MovieHandler {
	return &MovieHandler{
		db:         db,
		tmdbClient: tmdbClient,
	}
}

func (h *MovieHandler) SearchMovies(w http.ResponseWriter, r *http.Request) {
	query := utils.GetQueryParam(r, "search", "")
	page := utils.GetQueryParamInt(r, "page", 1)

	if query == "" {
		// If no search query, return popular movies from our database
		movies, err := h.getPopularMoviesFromDB(page)
		if err != nil {
			http.Error(w, "Failed to get movies", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": movies,
			"page":    page,
		})
		return
	}

	// Search TMDB for movies
	searchResp, err := h.tmdbClient.SearchMovies(query, page)
	if err != nil {
		http.Error(w, "Failed to search movies", http.StatusInternalServerError)
		return
	}

	// Convert TMDB movies to our format
	movies := make([]map[string]interface{}, len(searchResp.Results))
	for i, tmdbMovie := range searchResp.Results {
		posterURL := h.tmdbClient.GetPosterURL(tmdbMovie.PosterPath, "w500")
		year := services.ExtractYear(tmdbMovie.ReleaseDate)

		movies[i] = map[string]interface{}{
			"id":         tmdbMovie.ID,
			"tmdb_id":    tmdbMovie.ID,
			"title":      tmdbMovie.Title,
			"year":       year,
			"poster_url": posterURL,
			"synopsis":   tmdbMovie.Overview,
			"vote_avg":   tmdbMovie.VoteAverage,
		}
	}

	response := map[string]interface{}{
		"results":       movies,
		"page":          searchResp.Page,
		"total_pages":   searchResp.TotalPages,
		"total_results": searchResp.TotalResults,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *MovieHandler) getPopularMoviesFromDB(page int) ([]map[string]interface{}, error) {
	limit := 20
	offset := (page - 1) * limit

	rows, err := h.db.Query(`
		SELECT id, tmdb_id, title, year, poster_url, synopsis, runtime, genres
		FROM movies 
		ORDER BY id DESC 
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []map[string]interface{}
	for rows.Next() {
		var id, tmdbID int
		var title, synopsis, genres string
		var year, runtime *int
		var posterURL *string

		err := rows.Scan(&id, &tmdbID, &title, &year, &posterURL, &synopsis, &runtime, &genres)
		if err != nil {
			continue
		}

		movie := map[string]interface{}{
			"id":       id,
			"tmdb_id":  tmdbID,
			"title":    title,
			"year":     year,
			"synopsis": synopsis,
			"runtime":  runtime,
			"genres":   genres,
		}

		if posterURL != nil {
			movie["poster_url"] = *posterURL
		}

		movies = append(movies, movie)
	}

	return movies, nil
}

func (h *MovieHandler) GetMovie(w http.ResponseWriter, r *http.Request) {
	movieIDStr := utils.GetPathParam(r, "id")
	if movieIDStr == "" {
		http.Error(w, "Movie ID is required", http.StatusBadRequest)
		return
	}

	movieID, err := strconv.Atoi(movieIDStr)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	// First try to get from our database (by TMDB ID)
	movie, err := h.getMovieFromDB(movieID)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(movie)
		return
	}

	// If not found in DB, get from TMDB
	tmdbMovie, err := h.tmdbClient.GetMovieDetails(movieID)
	if err != nil {
		http.Error(w, "Movie not found", http.StatusNotFound)
		return
	}

	// Convert TMDB movie to our format
	posterURL := h.tmdbClient.GetPosterURL(tmdbMovie.PosterPath, "w500")
	backdropURL := h.tmdbClient.GetBackdropURL(tmdbMovie.BackdropPath, "w1280")
	year := services.ExtractYear(tmdbMovie.ReleaseDate)

	// Convert genres
	genreNames := make([]string, len(tmdbMovie.Genres))
	for i, genre := range tmdbMovie.Genres {
		genreNames[i] = genre.Name
	}

	// Get external IDs (IMDb, etc.)
	externalIDs, err := h.tmdbClient.GetMovieExternalIDs(movieID)
	if err != nil {
		// Continue without external IDs if fetch fails
		externalIDs = nil
	}

	// Save movie to our database for future use
	genresJSON, _ := json.Marshal(genreNames)
	_, err = h.db.Exec(`
		INSERT OR REPLACE INTO movies (tmdb_id, title, year, poster_url, synopsis, runtime, genres, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, tmdbMovie.ID, tmdbMovie.Title, year, posterURL, tmdbMovie.Overview, tmdbMovie.Runtime, string(genresJSON), time.Now())
	if err != nil {
		// Log error but continue - this is not critical
		// TODO: Add proper logging
	}

	movie = map[string]interface{}{
		"id":           tmdbMovie.ID,
		"tmdb_id":      tmdbMovie.ID,
		"title":        tmdbMovie.Title,
		"year":         year,
		"poster_url":   posterURL,
		"backdrop_url": backdropURL,
		"synopsis":     tmdbMovie.Overview,
		"runtime":      tmdbMovie.Runtime,
		"genres":       genreNames,
		"vote_avg":     tmdbMovie.VoteAverage,
		"vote_count":   tmdbMovie.VoteCount,
		"tagline":      tmdbMovie.Tagline,
		"status":       tmdbMovie.Status,
	}

	// Add external IDs if available
	if externalIDs != nil {
		movie["external_ids"] = map[string]interface{}{
			"imdb_id": externalIDs.IMDbID,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movie)
}

func (h *MovieHandler) getMovieFromDB(tmdbID int) (map[string]interface{}, error) {
	var id int
	var title, synopsis, genres string
	var year, runtime *int
	var posterURL *string

	err := h.db.QueryRow(`
		SELECT id, title, year, poster_url, synopsis, runtime, genres
		FROM movies 
		WHERE tmdb_id = ?
	`, tmdbID).Scan(&id, &title, &year, &posterURL, &synopsis, &runtime, &genres)

	if err != nil {
		return nil, err
	}

	movie := map[string]interface{}{
		"id":       id,
		"tmdb_id":  tmdbID,
		"title":    title,
		"year":     year,
		"synopsis": synopsis,
		"runtime":  runtime,
		"genres":   genres,
	}

	if posterURL != nil {
		movie["poster_url"] = *posterURL
	}

	return movie, nil
}

func (h *MovieHandler) UpdateMovieStatus(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement update movie status
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *MovieHandler) RateMovie(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement rate movie
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *MovieHandler) UpdateNotes(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement update movie notes
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *MovieHandler) UpdateOwnedFormats(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement update owned formats
	w.WriteHeader(http.StatusNotImplemented)
}