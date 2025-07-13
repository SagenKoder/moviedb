# Sagens Movie Database

A personal movie database web app for tracking movies, creating custom lists, and managing your movie collection. Built with React/TypeScript frontend and Go backend.

## Features

- 🎬 **Movie Search & Discovery** - Search movies powered by TMDB API
- 📋 **Custom Lists** - Create and manage personal movie lists (public/private)
- 👤 **User Profiles** - URL-based profile pages with list filtering
- 📱 **Mobile Responsive** - Optimized for both desktop and mobile
- 🔐 **Authentication** - Secure login via Auth0
- 🎯 **List Management** - Add movies to multiple lists, prevent duplicates
- 🌙 **Dark Mode** - System preference with manual toggle

## Tech Stack

- **Frontend**: React 18, TypeScript, Vite, Tailwind CSS, React Router
- **Backend**: Go 1.22+, SQLite, Standard HTTP library
- **Authentication**: Auth0 (JWT tokens)
- **API**: TMDB (The Movie Database)
- **Database**: SQLite with migration system

## Quick Start

### Prerequisites
- Node.js 18+
- Go 1.22+
- Auth0 account
- TMDB API key

### 1. Install Dependencies
```bash
make install
```

### 2. Environment Configuration

**Backend** (`.env`):
```bash
AUTH0_DOMAIN=your-domain.auth0.com
AUTH0_AUDIENCE=https://your-api-audience
TMDB_API_KEY=your-tmdb-api-key
DATABASE_PATH=./moviedb.db
PORT=8080
```

**Frontend** (`web/.env.local`):
```bash
VITE_AUTH0_DOMAIN=your-domain.auth0.com
VITE_AUTH0_CLIENT_ID=your-client-id
VITE_AUTH0_AUDIENCE=https://your-api-audience
```

### 3. Auth0 Setup
Follow the [Auth0 Setup Guide](docs/AUTH0_SETUP.md) to configure authentication.

### 4. Development
```bash
# Run both frontend and backend
make dev

# Frontend: http://localhost:3000
# Backend: http://localhost:8080
```

### 5. Production Build
```bash
# Build single binary with embedded frontend
make build-prod

# Run production server
./bin/moviedb
```

## Available Commands

| Command | Description |
|---------|-------------|
| `make dev` | Run frontend and backend in development |
| `make build-prod` | Build optimized production binary |
| `make install` | Install all dependencies |
| `make test` | Run all tests |
| `make lint` | Run linters and formatters |
| `make typecheck` | TypeScript type checking |
| `make clean` | Clean build artifacts |
| `make help` | Show all available commands |

## Project Structure

```
moviedb/
├── cmd/server/           # Go server entry point
├── internal/
│   ├── auth/            # Auth0 JWT middleware
│   ├── database/        # SQLite connection & migrations
│   ├── handlers/        # HTTP route handlers
│   ├── services/        # Business logic & TMDB client
│   └── types/           # Shared Go types
├── web/                 # React frontend
│   └── src/
│       ├── components/  # Reusable UI components
│       ├── hooks/       # Custom React hooks
│       ├── pages/       # Page components
│       └── contexts/    # React contexts (dark mode, etc.)
├── db/migrations/       # SQL schema migrations
└── embed.go            # Static file embedding
```

## Key Features

### Movie Management
- Search movies via TMDB API
- View detailed movie information (cast, genres, ratings, etc.)
- Add movies to custom lists
- Duplicate prevention

### Lists & Organization  
- Create unlimited custom lists
- Public/private list visibility
- URL-based list filtering (`/profile/user?list=123`)
- List statistics and movie counts

### User Experience
- Responsive design (mobile-first)
- Dark/light mode toggle
- Keyboard navigation
- Loading states and error handling

### Technical Highlights
- **Single Binary Deployment** - Frontend embedded in Go binary
- **JWT Authentication** - Secure Auth0 integration
- **Mobile Optimized** - Touch-friendly interface
- **URL State Management** - Shareable profile URLs
- **Server-side Privacy** - List visibility enforced at API level

## Deployment

The app builds into a single binary containing both frontend and backend:

1. **Build**: `make build-prod`
2. **Deploy**: Copy `bin/moviedb` to server
3. **Configure**: Set environment variables
4. **Run**: `./bin/moviedb`

No separate frontend hosting needed - everything is self-contained!

## Contributing

1. Fork & clone the repository
2. Create a feature branch
3. Make changes and add tests
4. Run `make lint` and `make typecheck`
5. Submit a pull request

## License

MIT License - see LICENSE file for details.