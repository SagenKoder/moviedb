# Movie Database Web App

A social movie database web app for tracking movies (watched, watchlist, favorites, owned formats) with a simple social feed feature. Built with React/TypeScript frontend and Go backend.

## Tech Stack

- **Frontend**: React 18, TypeScript, Vite, Tailwind CSS, Auth0 React SDK
- **Backend**: Go 1.21+, SQLite, Auth0, TMDB API
- **Database**: SQLite with custom migration system

## Quick Start

1. **Install dependencies**:
   ```bash
   make install
   ```

2. **Set up environment variables**:
   ```bash
   # Copy and configure backend environment
   cp .env.example .env
   
   # Copy and configure frontend environment
   cp web/.env.example web/.env.local
   ```

3. **Configure Auth0**:
   - Follow the detailed guide: [Auth0 Setup Guide](docs/AUTH0_SETUP.md)
   - Create both API and Single Page Application in Auth0
   - Configure callback URLs, CORS, and environment variables

4. **Get TMDB API Key**:
   - Sign up at [TMDB](https://www.themoviedb.org/settings/api)
   - Add your API key to the environment files

5. **Run in development**:
   ```bash
   make dev
   ```
   This starts both frontend (http://localhost:3000) and backend (http://localhost:8080)

## Available Commands

- `make dev` - Run both frontend and backend in development
- `make build` - Build both frontend and backend for production
- `make install` - Install all dependencies
- `make test` - Run all tests
- `make lint` - Run all linters
- `make typecheck` - Run TypeScript type checking
- `make clean` - Clean build artifacts and database
- `make help` - Show all available commands

## Project Structure

```
moviedb/
├── cmd/server/           # Go application entry point
├── internal/             # Go internal packages
│   ├── auth/            # Auth0 middleware
│   ├── database/        # Database connection and migrations
│   ├── handlers/        # HTTP handlers
│   ├── services/        # Business logic
│   └── types/           # Shared types
├── web/                 # React/TypeScript frontend
│   └── src/
│       ├── components/  # React components
│       ├── hooks/       # Custom React hooks
│       ├── pages/       # Page components
│       └── utils/       # Utility functions
├── db/migrations/       # SQL migration files
└── static/             # Static assets
```

## Development Notes

- The project uses TypeScript throughout the frontend
- Database migrations run automatically on server start
- Auth0 is configured for authentication
- TMDB API integration is planned for movie data
- The frontend proxies API calls to the backend during development

## Next Steps (Phase 1 Implementation)

1. Implement user management and Auth0 integration
2. Add TMDB API integration for movie search
3. Implement movie tracking functionality
4. Create basic React components and pages
5. Add proper error handling and validation