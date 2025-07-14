-- Watch providers cache table for TMDB data
CREATE TABLE IF NOT EXISTS watch_providers_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tmdb_id INTEGER NOT NULL,
    region_code TEXT NOT NULL DEFAULT 'US',
    providers_data TEXT NOT NULL, -- JSON data of watch providers
    cached_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    UNIQUE(tmdb_id, region_code)
);

-- Index for efficient lookups
CREATE INDEX IF NOT EXISTS idx_watch_providers_tmdb_region ON watch_providers_cache(tmdb_id, region_code);
CREATE INDEX IF NOT EXISTS idx_watch_providers_expires ON watch_providers_cache(expires_at);

-- Plex availability cache table
CREATE TABLE IF NOT EXISTS plex_availability_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tmdb_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    is_available BOOLEAN NOT NULL DEFAULT FALSE,
    plex_servers TEXT, -- JSON array of servers where movie is available
    cached_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    UNIQUE(tmdb_id, user_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Index for efficient lookups
CREATE INDEX IF NOT EXISTS idx_plex_availability_user_tmdb ON plex_availability_cache(user_id, tmdb_id);
CREATE INDEX IF NOT EXISTS idx_plex_availability_expires ON plex_availability_cache(expires_at);