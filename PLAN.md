# Movie Database Web App - Development Plan

## Project Overview
A social movie database web app for tracking movies (watched, watchlist, favorites, owned formats) with a simple social feed feature. Built for ~30 users with public-only privacy model.

## Tech Stack

### Frontend
- **React 18** - Main UI framework
- **React Router v6** - Client-side routing
- **Auth0 React SDK** - Authentication
- **Tailwind CSS** - Styling
- **Lucide React** - Icons
- **Vite** - Build tool and dev server

### Backend
- **Go 1.22+** - Server language (using enhanced ServeMux)
- **SQLite** - Database
- **Auth0** - Authentication provider
- **TMDB API** - Movie metadata
- **Standard library** - HTTP server with pattern matching, database/sql

### Libraries & Dependencies

#### Frontend (package.json)
```json
{
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.8.0",
    "@auth0/auth0-react": "^2.0.0",
    "lucide-react": "^0.263.1"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^4.0.0",
    "vite": "^4.0.0",
    "tailwindcss": "^3.2.0",
    "autoprefixer": "^10.4.13",
    "postcss": "^8.4.21"
  }
}
```

#### Backend (go.mod)
```go
module moviedb

go 1.22

require (
    github.com/auth0/go-jwt-middleware/v2 v2.2.0
    github.com/mattn/go-sqlite3 v1.14.17
)
```

## Project Structure

```
moviedb/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── auth/
│   │   └── middleware.go           # Auth0 JWT validation
│   ├── database/
│   │   ├── migrations.go           # Custom migration system
│   │   ├── connection.go           # Database connection
│   │   └── models.go               # Data models
│   ├── handlers/
│   │   ├── movies.go               # Movie CRUD operations
│   │   ├── users.go                # User management
│   │   ├── feed.go                 # Social feed
│   │   ├── lists.go                # User lists
│   │   └── friends.go              # Friend system
│   ├── services/
│   │   ├── movie_service.go        # Business logic for movies
│   │   ├── tmdb_client.go          # TMDB API client
│   │   └── feed_service.go         # Feed generation logic
│   └── types/
│       └── types.go                # Shared types/structs
├── web/                            # React frontend
│   ├── src/
│   │   ├── components/
│   │   │   ├── auth/
│   │   │   │   ├── LoginButton.jsx
│   │   │   │   ├── LogoutButton.jsx
│   │   │   │   └── ProtectedRoute.jsx
│   │   │   ├── layout/
│   │   │   │   ├── Header.jsx
│   │   │   │   ├── Sidebar.jsx
│   │   │   │   └── Layout.jsx
│   │   │   ├── movies/
│   │   │   │   ├── MovieCard.jsx
│   │   │   │   ├── MovieGrid.jsx
│   │   │   │   ├── MovieDetail.jsx
│   │   │   │   └── MovieSearch.jsx
│   │   │   ├── feed/
│   │   │   │   ├── Feed.jsx
│   │   │   │   ├── FeedPost.jsx
│   │   │   │   └── PostActions.jsx
│   │   │   ├── lists/
│   │   │   │   ├── ListGrid.jsx
│   │   │   │   ├── ListDetail.jsx
│   │   │   │   └── CreateList.jsx
│   │   │   └── users/
│   │   │       ├── UserProfile.jsx
│   │   │       ├── UserGrid.jsx
│   │   │       └── FriendsList.jsx
│   │   ├── hooks/
│   │   │   ├── useApi.js            # API calls with auth
│   │   │   ├── useMovies.js         # Movie state management
│   │   │   └── useFeed.js           # Feed state management
│   │   ├── pages/
│   │   │   ├── Dashboard.jsx
│   │   │   ├── Movies.jsx
│   │   │   ├── Lists.jsx
│   │   │   ├── Feed.jsx
│   │   │   ├── Users.jsx
│   │   │   └── Profile.jsx
│   │   ├── utils/
│   │   │   ├── api.js               # API base configuration
│   │   │   └── constants.js         # App constants
│   │   ├── App.jsx                  # Main app component
│   │   └── main.jsx                 # React entry point
│   ├── public/
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js
│   └── tailwind.config.js
├── db/
│   └── migrations/                  # SQL migration files
├── static/                          # Static assets (if not embedded)
├── .env.example                     # Environment variables template
├── Makefile                         # Build automation
├── go.mod
├── go.sum
└── README.md
```

## Database Schema

### Core Tables
```sql
-- Users
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auth0_id TEXT UNIQUE NOT NULL,
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    username TEXT UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Movies (cached from TMDB)
CREATE TABLE movies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tmdb_id INTEGER UNIQUE NOT NULL,
    title TEXT NOT NULL,
    year INTEGER,
    poster_url TEXT,
    synopsis TEXT,
    runtime INTEGER,
    genres TEXT, -- JSON array as string
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User-Movie relationships
CREATE TABLE user_movies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    movie_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'not_watched', -- 'not_watched', 'watched', 'watching'
    rating INTEGER, -- 1-5 stars
    watched_date DATETIME,
    notes TEXT,
    owned_formats TEXT, -- JSON: ["bluray", "digital", "netflix"]
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (movie_id) REFERENCES movies(id),
    UNIQUE(user_id, movie_id)
);

-- Custom Lists
CREATE TABLE lists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_public BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- List-Movie relationships
CREATE TABLE list_movies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    list_id INTEGER NOT NULL,
    movie_id INTEGER NOT NULL,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (list_id) REFERENCES lists(id),
    FOREIGN KEY (movie_id) REFERENCES movies(id),
    UNIQUE(list_id, movie_id)
);

-- Friends
CREATE TABLE friends (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    friend_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (friend_id) REFERENCES users(id),
    UNIQUE(user_id, friend_id)
);

-- Feed Posts
CREATE TABLE feed_posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    type TEXT NOT NULL, -- 'watched', 'rated', 'list_created', 'review'
    movie_id INTEGER,
    list_id INTEGER,
    content TEXT, -- User's review/note
    rating INTEGER,
    metadata TEXT, -- JSON for additional data
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (movie_id) REFERENCES movies(id),
    FOREIGN KEY (list_id) REFERENCES lists(id)
);

-- Post Interactions
CREATE TABLE post_likes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_id) REFERENCES feed_posts(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(post_id, user_id)
);

CREATE TABLE post_comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_id) REFERENCES feed_posts(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Schema Migrations tracking
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Indexes
```sql
CREATE INDEX idx_users_auth0_id ON users(auth0_id);
CREATE INDEX idx_movies_tmdb_id ON movies(tmdb_id);
CREATE INDEX idx_user_movies_user_id ON user_movies(user_id);
CREATE INDEX idx_user_movies_status ON user_movies(status);
CREATE INDEX idx_feed_posts_user_id ON feed_posts(user_id);
CREATE INDEX idx_feed_posts_created_at ON feed_posts(created_at);
CREATE INDEX idx_friends_user_id ON friends(user_id);
```

## API Endpoints

### Authentication
- All endpoints require valid JWT token except health check
- JWT validation via Auth0 middleware

### Movies
```
GET    /api/movies?search=query&page=1     # Search movies (TMDB)
GET    /api/movies/:id                     # Get movie details
POST   /api/movies/:id/status              # Update watch status
POST   /api/movies/:id/rating              # Rate movie
POST   /api/movies/:id/notes               # Add/update notes
POST   /api/movies/:id/owned               # Update owned formats
```

### Lists
```
GET    /api/lists                          # User's lists
POST   /api/lists                          # Create list
GET    /api/lists/:id                      # List details
PUT    /api/lists/:id                      # Update list
DELETE /api/lists/:id                      # Delete list
POST   /api/lists/:id/movies/:movieId      # Add movie to list
DELETE /api/lists/:id/movies/:movieId      # Remove movie from list
```

### Users & Social
```
GET    /api/users                          # Browse all users
GET    /api/users/:id                      # User profile
GET    /api/users/:id/lists                # User's public lists
POST   /api/users/:id/friend               # Add friend
DELETE /api/users/:id/friend               # Remove friend

GET    /api/feed/friends                   # Friends' activity feed
GET    /api/feed/global                    # Global activity feed
POST   /api/posts/:id/like                 # Like post
DELETE /api/posts/:id/like                 # Unlike post
POST   /api/posts/:id/comments             # Comment on post
```

### User Management
```
GET    /api/me                             # Current user info
PUT    /api/me                             # Update profile
POST   /api/me/setup                       # Initial user setup
```

## Code Examples

### Backend - Main Server
```go
// cmd/server/main.go
package main

import (
    "log"
    "net/http"
    "os"
    
    "moviedb/internal/auth"
    "moviedb/internal/database"
    "moviedb/internal/handlers"
)

func main() {
    // Get environment variables
    dbPath := getEnv("DATABASE_PATH", "./moviedb.db")
    port := getEnv("PORT", "8080")
    auth0Domain := getEnv("AUTH0_DOMAIN", "")
    auth0Audience := getEnv("AUTH0_AUDIENCE", "")

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

    // Initialize handlers
    movieHandler := handlers.NewMovieHandler(db)
    userHandler := handlers.NewUserHandler(db)
    feedHandler := handlers.NewFeedHandler(db)
    
    // Setup router using Go 1.22+ ServeMux with pattern matching
    mux := http.NewServeMux()
    
    // Health check (no auth required)
    mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    // Create auth middleware wrapper
    requireAuth := auth.RequireAuth(authMiddleware)

    // API routes with HTTP method and path patterns
    mux.HandleFunc("GET /api/movies", requireAuth(http.HandlerFunc(movieHandler.SearchMovies)).ServeHTTP)
    mux.HandleFunc("GET /api/movies/{id}", requireAuth(http.HandlerFunc(movieHandler.GetMovie)).ServeHTTP)
    mux.HandleFunc("GET /api/users", requireAuth(http.HandlerFunc(userHandler.GetUsers)).ServeHTTP)
    // ... more routes
    
    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, mux))
}
```

### Backend - Auth Middleware
```go
// internal/auth/middleware.go
package auth

import (
    "context"
    "fmt"
    "net/http"
    "net/url"
    "time"

    "github.com/auth0/go-jwt-middleware/v2"
    "github.com/auth0/go-jwt-middleware/v2/jwks"
    "github.com/auth0/go-jwt-middleware/v2/validator"
)

type User struct {
    Auth0ID string `json:"auth0_id"`
    Email   string `json:"email"`
    Name    string `json:"name"`
}

// CustomClaims contains custom data we want to extract from the token.
type CustomClaims struct {
    Email string `json:"email"`
    Name  string `json:"name"`
}

// Validate satisfies validator.CustomClaims interface.
func (c CustomClaims) Validate(ctx context.Context) error {
    return nil
}

func NewMiddleware(domain, audience string) (*jwtmiddleware.JWTMiddleware, error) {
    issuerURL, err := url.Parse("https://" + domain + "/")
    if err != nil {
        return nil, fmt.Errorf("failed to parse the issuer url: %w", err)
    }

    provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

    jwtValidator, err := validator.New(
        provider.KeyFunc,
        validator.RS256,
        issuerURL.String(),
        []string{audience},
        validator.WithCustomClaims(
            func() validator.CustomClaims {
                return &CustomClaims{}
            },
        ),
        validator.WithAllowedClockSkew(time.Minute),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create JWT validator: %w", err)
    }

    return jwtmiddleware.New(jwtValidator.ValidateToken), nil
}

func GetUserFromContext(ctx context.Context) (*User, error) {
    claims, ok := ctx.Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
    if !ok {
        return nil, fmt.Errorf("no claims found in context")
    }

    customClaims, ok := claims.CustomClaims.(*CustomClaims)
    if !ok {
        return nil, fmt.Errorf("invalid custom claims format")
    }

    return &User{
        Auth0ID: claims.RegisteredClaims.Subject,
        Email:   customClaims.Email,
        Name:    customClaims.Name,
    }, nil
}
```

### Frontend - API Hook
```jsx
// src/hooks/useApi.js
import { useAuth0 } from '@auth0/auth0-react';

export function useApi() {
    const { getAccessTokenSilently } = useAuth0();
    
    const apiCall = async (url, options = {}) => {
        const token = await getAccessTokenSilently();
        
        return fetch(url, {
            ...options,
            headers: {
                ...options.headers,
                Authorization: `Bearer ${token}`,
                'Content-Type': 'application/json',
            },
        });
    };
    
    return { apiCall };
}
```

### Frontend - Movie Component
```jsx
// src/components/movies/MovieCard.jsx
import { useState } from 'react';
import { Heart, Plus, Check } from 'lucide-react';
import { useApi } from '../../hooks/useApi';

export function MovieCard({ movie, userMovie }) {
    const { apiCall } = useApi();
    const [status, setStatus] = useState(userMovie?.status || 'not_watched');
    const [isFavorite, setIsFavorite] = useState(userMovie?.is_favorite || false);
    
    const updateStatus = async (newStatus) => {
        await apiCall(`/api/movies/${movie.id}/status`, {
            method: 'POST',
            body: JSON.stringify({ status: newStatus })
        });
        setStatus(newStatus);
    };
    
    const toggleFavorite = async () => {
        await apiCall(`/api/movies/${movie.id}/favorite`, {
            method: 'POST',
            body: JSON.stringify({ is_favorite: !isFavorite })
        });
        setIsFavorite(!isFavorite);
    };
    
    return (
        <div className="relative group">
            <img 
                src={movie.poster_url} 
                alt={movie.title}
                className="w-full rounded-lg shadow-md"
            />
            
            {/* Hover overlay */}
            <div className="absolute inset-0 bg-black bg-opacity-60 opacity-0 group-hover:opacity-100 transition-opacity rounded-lg flex flex-col justify-end p-4">
                <h3 className="text-white font-semibold">{movie.title}</h3>
                <p className="text-gray-300 text-sm">{movie.year}</p>
                
                {/* Action buttons */}
                <div className="flex gap-2 mt-2">
                    {status === 'not_watched' ? (
                        <button 
                            onClick={() => updateStatus('watched')}
                            className="flex items-center gap-1 bg-green-600 text-white px-2 py-1 rounded text-sm"
                        >
                            <Check size={16} /> Watched
                        </button>
                    ) : (
                        <span className="flex items-center gap-1 bg-green-500 text-white px-2 py-1 rounded text-sm">
                            <Check size={16} /> Watched
                        </span>
                    )}
                    
                    <button 
                        onClick={toggleFavorite}
                        className={`p-1 rounded ${isFavorite ? 'bg-red-500 text-white' : 'bg-gray-600 text-gray-300'}`}
                    >
                        <Heart size={16} />
                    </button>
                </div>
            </div>
        </div>
    );
}
```

## Environment Setup

### Environment Variables
```bash
# .env.local (React)
VITE_AUTH0_DOMAIN=your-domain.auth0.com
VITE_AUTH0_CLIENT_ID=your-client-id
VITE_AUTH0_AUDIENCE=your-api-audience
VITE_TMDB_API_KEY=your-tmdb-key

# Backend
AUTH0_DOMAIN=your-domain.auth0.com
AUTH0_AUDIENCE=your-api-audience
TMDB_API_KEY=your-tmdb-key
DATABASE_PATH=./moviedb.db
PORT=8080
```

### Build Scripts
```makefile
# Makefile
.PHONY: dev build clean

dev-frontend:
	cd web && npm run dev

dev-backend:
	go run cmd/server/main.go

build-frontend:
	cd web && npm run build

build-backend:
	go build -o bin/moviedb cmd/server/main.go

build: build-frontend build-backend

clean:
	rm -rf web/dist bin/

dev: 
	# Run both frontend and backend in development
	make -j2 dev-frontend dev-backend
```

## Development Phases

### Phase 1: Core Infrastructure
1. Set up project structure
2. Implement Auth0 integration
3. Create database schema and migrations
4. Basic React app with routing
5. API endpoints for movies and users

### Phase 2: Movie Management
1. TMDB API integration
2. Movie search and details
3. Watch status tracking
4. Rating system
5. Custom lists functionality

### Phase 3: Social Features
1. Friend system
2. Activity feed
3. Post interactions (like, comment)
4. User profiles
5. Global discovery features

### Phase 4: Polish & Deploy
1. UI/UX improvements
2. Performance optimization
3. Error handling
4. Testing
5. Deployment setup

## Deployment Strategy

### Single Binary Deployment
- Embed React build into Go binary
- SQLite database file
- Single executable for easy deployment
- Environment variables for configuration

### Docker Option
```dockerfile
FROM node:18 AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

FROM golang:1.21 AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN go build -o moviedb cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=backend /app/moviedb ./
EXPOSE 8080
CMD ["./moviedb"]
```

This plan provides a complete roadmap for building the movie database web app with all the discussed features and technical requirements.

