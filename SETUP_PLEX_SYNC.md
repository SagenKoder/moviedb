# Plex Sync System Setup Guide

## Required Changes for Enhanced Plex Sync

### 1. Database Migration
Run the new migration to create required tables:

```bash
# Apply the new migration
./migrate up
# or however you run migrations in your system
```

**New Tables Created:**
- `plex_servers` - Stores unique Plex servers
- `plex_libraries` - Stores libraries from servers
- `user_plex_access` - Tracks user access to libraries
- `plex_library_items` - Cached library contents
- `sync_jobs` - Background job management
- `tmdb_rate_limits` - Rate limiting tracking

### 2. Server Integration Changes

**In your main server file** (e.g., `cmd/server/main.go`):

```go
// Add these imports
import (
    "moviedb/internal/services"
    "moviedb/internal/handlers"
)

// Initialize Plex integration (add this after database setup)
plexIntegration := services.NewPlexIntegrationManager(db, tmdbClient)

// Start background services
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := plexIntegration.Start(ctx); err != nil {
    log.Fatalf("Failed to start Plex integration: %v", err)
}

// Setup graceful shutdown
go func() {
    <-ctx.Done()
    if err := plexIntegration.Stop(); err != nil {
        log.Printf("Error stopping Plex integration: %v", err)
    }
}()

// Add new API routes
syncHandler := handlers.NewPlexSyncEnhancedHandler(
    plexIntegration.GetSyncService(),
    plexIntegration.GetJobManager(),
)

mux.HandleFunc("POST /api/plex/sync", requireAuth(syncHandler.TriggerFullSync))
mux.HandleFunc("GET /api/plex/sync/status/{jobId}", requireAuth(syncHandler.GetJobStatus))
mux.HandleFunc("GET /api/plex/sync/jobs", requireAuth(syncHandler.GetUserJobs))
mux.HandleFunc("POST /api/plex/sync/cancel/{jobId}", requireAuth(syncHandler.CancelJob))
mux.HandleFunc("GET /api/plex/libraries", requireAuth(syncHandler.GetUserLibraries))
```

### 3. Authentication Integration

**CRITICAL**: Update the `getUserID` function in `internal/handlers/plex_sync_enhanced.go`:

```go
// Replace this placeholder with your actual auth implementation
func getUserID(r *http.Request) int64 {
    // Example for JWT-based auth:
    // token := r.Header.Get("Authorization")
    // claims, err := parseJWT(token)
    // if err != nil { return 0 }
    // return claims.UserID
    
    // Example for session-based auth:
    // session := getSession(r)
    // return session.UserID
    
    // REPLACE THIS WITH YOUR ACTUAL AUTH LOGIC
    return 0
}
```

### 4. Go Module Dependencies

Add to `go.mod` if not already present:

```go
require (
    github.com/LukeHagar/plexgo v0.23.0
    // ... other dependencies
)
```

Run:
```bash
go mod tidy
```

### 5. Environment Variables (Optional)

Add these optional environment variables:

```bash
# Rate limiting configuration
TMDB_RATE_LIMIT_REQUESTS=40
TMDB_RATE_LIMIT_WINDOW=10s

# Job processing
PLEX_SYNC_WORKERS=3
PLEX_SYNC_TIMEOUT=2h

# Cleanup scheduling
PLEX_CLEANUP_INTERVAL=6h
```

### 6. Frontend Build

The frontend changes are already in place. Build the frontend:

```bash
cd web
npm run build
```

### 7. Database Backup (Recommended)

Before running the migration, backup your database:

```bash
# For SQLite
cp your-database.db your-database.backup.db

# For PostgreSQL
pg_dump your_db > backup.sql
```

## What's NOT Required

- **No changes to existing Plex authentication** - The existing PIN flow still works
- **No changes to existing movie data** - All existing data is preserved
- **No changes to user management** - Uses existing user authentication
- **No changes to TMDB client** - Uses existing TMDB integration

## Testing the Setup

1. **Test migration**:
```bash
# Check that new tables exist
sqlite3 your-database.db ".schema" | grep plex_
```

2. **Test API endpoints**:
```bash
# Test sync trigger (with proper auth headers)
curl -X POST http://localhost:8080/api/plex/sync \
  -H "Authorization: Bearer YOUR_TOKEN"
```

3. **Test UI**:
- Connect to Plex via existing flow
- Look for "Sync Plex Data" button in user dropdown
- Click to trigger sync and watch progress

## Troubleshooting

### Common Issues:

1. **Migration fails**: Check database permissions and syntax
2. **Jobs not processing**: Ensure job manager is started in main server
3. **Auth errors**: Verify `getUserID` function is properly implemented
4. **Rate limiting**: Check TMDB API key and rate limits

### Debug Logs:

The system includes extensive debug logging. Look for:
- `DEBUG: [GetServers]` - Server discovery
- `DEBUG: [SearchAllLibraries]` - Library search
- `DEBUG: [searchMovieWithPlexgo]` - Movie search
- Job progress updates in console

## Performance Considerations

- **Initial sync**: May take 10-30 minutes for large libraries
- **Memory usage**: Each worker uses ~50MB during sync
- **Database size**: Expect 1-2MB per 1000 movies
- **TMDB calls**: Rate limited to 40 requests per 10 seconds

## Rollback Plan

If issues occur:

1. **Stop the server**
2. **Restore database backup**
3. **Revert code changes**
4. **Restart with old system**

The new system is additive - it won't break existing functionality.