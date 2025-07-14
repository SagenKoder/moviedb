package main

import (
	"log"
	"net/http"
	"os"

	"moviedb"
	"moviedb/internal/auth"
	"moviedb/internal/database"
	"moviedb/internal/handlers"
	"moviedb/internal/services"
)


func main() {
	// Get environment variables
	dbPath := getEnv("DATABASE_PATH", "./moviedb.db")
	port := getEnv("PORT", "8080")
	auth0Domain := getEnv("AUTH0_DOMAIN", "")
	auth0Audience := getEnv("AUTH0_AUDIENCE", "")
	tmdbAPIKey := getEnv("TMDB_API_KEY", "")

	if auth0Domain == "" || auth0Audience == "" {
		log.Fatal("AUTH0_DOMAIN and AUTH0_AUDIENCE environment variables are required")
	}

	if tmdbAPIKey == "" {
		log.Fatal("TMDB_API_KEY environment variable is required")
	}

	// Initialize database
	db, err := database.Connect(dbPath)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Migration failed:", err)
	}

	// Initialize auth middleware
	authMiddleware, err := auth.NewMiddleware(auth0Domain, auth0Audience)
	if err != nil {
		log.Fatal("Failed to create auth middleware:", err)
	}

	// Initialize TMDB client and services
	tmdbClient := services.NewTMDBClient(tmdbAPIKey)
	movieSyncService := services.NewMovieSyncService(db, tmdbClient)

	// Start movie sync scheduler
	movieSyncService.StartSyncScheduler()

	// Initialize handlers
	movieHandler := handlers.NewMovieHandler(db, tmdbClient)
	userHandler := handlers.NewUserHandler(db)
	feedHandler := handlers.NewFeedHandler(db)
	listHandler := handlers.NewListHandler(db)
	syncHandler := handlers.NewSyncHandler(movieSyncService)
	plexHandler := handlers.NewPlexHandler(db)

	// Setup router using standard library ServeMux
	mux := http.NewServeMux()

	// Health check (no auth required)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create auth middleware wrapper
	requireAuth := auth.RequireAuth(authMiddleware)

	// User routes
	mux.HandleFunc("GET /api/me", requireAuth(http.HandlerFunc(userHandler.GetCurrentUser)).ServeHTTP)
	mux.HandleFunc("PUT /api/me", requireAuth(http.HandlerFunc(userHandler.UpdateCurrentUser)).ServeHTTP)
	mux.HandleFunc("POST /api/me/setup", requireAuth(http.HandlerFunc(userHandler.SetupUser)).ServeHTTP)
	mux.HandleFunc("GET /api/me/preferences", requireAuth(http.HandlerFunc(userHandler.GetUserPreferences)).ServeHTTP)
	mux.HandleFunc("PUT /api/me/preferences", requireAuth(http.HandlerFunc(userHandler.UpdateUserPreferences)).ServeHTTP)
	mux.HandleFunc("GET /api/users", requireAuth(http.HandlerFunc(userHandler.GetUsers)).ServeHTTP)
	mux.HandleFunc("GET /api/users/{id}", requireAuth(http.HandlerFunc(userHandler.GetUser)).ServeHTTP)
	mux.HandleFunc("GET /api/users/{id}/lists", requireAuth(http.HandlerFunc(userHandler.GetUserLists)).ServeHTTP)
	mux.HandleFunc("GET /api/users/{id}/movies", requireAuth(http.HandlerFunc(userHandler.GetUserMovies)).ServeHTTP)
	mux.HandleFunc("POST /api/users/{id}/friend", requireAuth(http.HandlerFunc(userHandler.AddFriend)).ServeHTTP)
	mux.HandleFunc("DELETE /api/users/{id}/friend", requireAuth(http.HandlerFunc(userHandler.RemoveFriend)).ServeHTTP)

	// Movie routes
	mux.HandleFunc("GET /api/movies", requireAuth(http.HandlerFunc(movieHandler.SearchMovies)).ServeHTTP)
	mux.HandleFunc("GET /api/movies/{id}", requireAuth(http.HandlerFunc(movieHandler.GetMovie)).ServeHTTP)
	mux.HandleFunc("POST /api/movies/{id}/status", requireAuth(http.HandlerFunc(movieHandler.UpdateMovieStatus)).ServeHTTP)
	mux.HandleFunc("POST /api/movies/{id}/rating", requireAuth(http.HandlerFunc(movieHandler.RateMovie)).ServeHTTP)
	mux.HandleFunc("POST /api/movies/{id}/notes", requireAuth(http.HandlerFunc(movieHandler.UpdateNotes)).ServeHTTP)
	mux.HandleFunc("POST /api/movies/{id}/owned", requireAuth(http.HandlerFunc(movieHandler.UpdateOwnedFormats)).ServeHTTP)

	// List routes
	mux.HandleFunc("GET /api/lists", requireAuth(http.HandlerFunc(listHandler.GetLists)).ServeHTTP)
	mux.HandleFunc("POST /api/lists", requireAuth(http.HandlerFunc(listHandler.CreateList)).ServeHTTP)
	mux.HandleFunc("GET /api/lists/{id}", requireAuth(http.HandlerFunc(listHandler.GetList)).ServeHTTP)
	mux.HandleFunc("PUT /api/lists/{id}", requireAuth(http.HandlerFunc(listHandler.UpdateList)).ServeHTTP)
	mux.HandleFunc("DELETE /api/lists/{id}", requireAuth(http.HandlerFunc(listHandler.DeleteList)).ServeHTTP)
	mux.HandleFunc("POST /api/lists/{id}/movies/{movieId}", requireAuth(http.HandlerFunc(listHandler.AddMovieToList)).ServeHTTP)
	mux.HandleFunc("DELETE /api/lists/{id}/movies/{movieId}", requireAuth(http.HandlerFunc(listHandler.RemoveMovieFromList)).ServeHTTP)
	mux.HandleFunc("GET /api/movies/{movieId}/lists", requireAuth(http.HandlerFunc(listHandler.GetMovieInLists)).ServeHTTP)
	mux.HandleFunc("GET /api/me/movies", requireAuth(http.HandlerFunc(listHandler.GetAllUserMovies)).ServeHTTP)

	// Feed routes
	mux.HandleFunc("GET /api/feed/friends", requireAuth(http.HandlerFunc(feedHandler.GetFriendsFeed)).ServeHTTP)
	mux.HandleFunc("GET /api/feed/global", requireAuth(http.HandlerFunc(feedHandler.GetGlobalFeed)).ServeHTTP)
	mux.HandleFunc("POST /api/posts/{id}/like", requireAuth(http.HandlerFunc(feedHandler.LikePost)).ServeHTTP)
	mux.HandleFunc("DELETE /api/posts/{id}/like", requireAuth(http.HandlerFunc(feedHandler.UnlikePost)).ServeHTTP)
	mux.HandleFunc("POST /api/posts/{id}/comments", requireAuth(http.HandlerFunc(feedHandler.AddComment)).ServeHTTP)

	// Sync routes
	mux.HandleFunc("POST /api/sync/movies", requireAuth(http.HandlerFunc(syncHandler.TriggerMovieSync)).ServeHTTP)
	mux.HandleFunc("GET /api/sync/status", requireAuth(http.HandlerFunc(syncHandler.GetSyncStatus)).ServeHTTP)

	// Plex routes
	mux.HandleFunc("POST /api/plex/auth/start", requireAuth(http.HandlerFunc(plexHandler.StartPlexAuth)).ServeHTTP)
	mux.HandleFunc("GET /api/plex/auth/check", requireAuth(http.HandlerFunc(plexHandler.CheckPlexAuth)).ServeHTTP)
	mux.HandleFunc("GET /api/plex/status", requireAuth(http.HandlerFunc(plexHandler.GetPlexStatus)).ServeHTTP)
	mux.HandleFunc("DELETE /api/plex/disconnect", requireAuth(http.HandlerFunc(plexHandler.DisconnectPlex)).ServeHTTP)
	mux.HandleFunc("GET /api/plex/now-playing", requireAuth(http.HandlerFunc(plexHandler.GetNowPlaying)).ServeHTTP)

	// SPA routes - serve index.html for client-side routing
	spaRoutes := []string{"/movies", "/community", "/lists", "/profile", "/search", "/settings"}
	for _, route := range spaRoutes {
		route := route // capture loop variable
		mux.HandleFunc("GET "+route, func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = "/"
			staticDir := getEnv("STATIC_DIR", "./web/dist")
			if _, err := os.Stat(staticDir); err == nil {
				// Development mode
				fs := http.FileServer(http.Dir(staticDir))
				addCacheHeaders(fs).ServeHTTP(w, r)
			} else {
				// Production mode
				distFS, err := moviedb.GetDistFS()
				if err != nil {
					http.Error(w, "Failed to load app", http.StatusInternalServerError)
					return
				}
				addCacheHeaders(http.FileServer(http.FS(distFS))).ServeHTTP(w, r)
			}
		})
	}

	// Static files (React app) - serve embedded files in production or from disk in development
	staticDir := getEnv("STATIC_DIR", "./web/dist")
	if _, err := os.Stat(staticDir); err == nil {
		// Development mode - serve from disk
		log.Println("Serving static files from disk:", staticDir)
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", addCacheHeaders(fs))
	} else {
		// Production mode - serve embedded files
		log.Println("Serving embedded static files")
		distFS, err := moviedb.GetDistFS()
		if err != nil {
			log.Fatal("Failed to create sub filesystem:", err)
		}
		mux.Handle("/", addCacheHeaders(http.FileServer(http.FS(distFS))))
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}


// addCacheHeaders adds appropriate cache headers to prevent browser caching issues
func addCacheHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For HTML files (like index.html), prevent caching to ensure latest version is loaded
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		} else {
			// For assets (JS, CSS), allow caching but add ETag for validation
			w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year for assets
		}
		
		next.ServeHTTP(w, r)
	})
}