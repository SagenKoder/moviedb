-- Enhanced Plex Library Sync System
-- This migration creates the tables needed for comprehensive Plex library syncing

-- Plex Servers table - stores unique Plex servers
CREATE TABLE plex_servers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    machine_id TEXT UNIQUE NOT NULL, -- Plex server machine identifier
    name TEXT NOT NULL,
    owner_user_id INTEGER, -- User who owns this server (if known)
    base_url TEXT, -- Server URL for direct access
    version TEXT,
    platform TEXT,
    last_synced_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Plex Libraries table - stores unique libraries from servers
CREATE TABLE plex_libraries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER NOT NULL,
    section_key INTEGER NOT NULL, -- Plex library section key
    title TEXT NOT NULL,
    type TEXT NOT NULL, -- 'movie', 'show', etc.
    agent TEXT, -- Library agent (e.g., com.plexapp.agents.imdb)
    scanner TEXT, -- Library scanner
    language TEXT,
    uuid TEXT,
    item_count INTEGER DEFAULT 0, -- Cached count of items
    last_synced_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (server_id) REFERENCES plex_servers(id) ON DELETE CASCADE,
    UNIQUE(server_id, section_key)
);

-- User access to Plex libraries - tracks which users can access which libraries
CREATE TABLE user_plex_access (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    library_id INTEGER NOT NULL,
    access_level TEXT DEFAULT 'read', -- 'read', 'admin' (for future use)
    discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_verified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT 1, -- False when access is revoked
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (library_id) REFERENCES plex_libraries(id) ON DELETE CASCADE,
    UNIQUE(user_id, library_id)
);

-- Plex Library Items - caches all items in each library
CREATE TABLE plex_library_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL,
    plex_rating_key TEXT NOT NULL, -- Plex item rating key
    plex_guid TEXT NOT NULL, -- Plex GUID for TMDB matching
    title TEXT NOT NULL,
    year INTEGER,
    tmdb_id INTEGER, -- Matched TMDB ID (nullable until matched)
    type TEXT NOT NULL, -- 'movie', 'episode', 'season', 'show'
    metadata_json TEXT, -- Full Plex metadata as JSON
    added_at DATETIME, -- When added to Plex
    updated_at_plex DATETIME, -- Last updated in Plex
    last_matched_at DATETIME, -- Last time TMDB matching was attempted
    matching_attempts INTEGER DEFAULT 0, -- Number of TMDB matching attempts
    is_active BOOLEAN DEFAULT 1, -- False when item is removed from Plex
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (library_id) REFERENCES plex_libraries(id) ON DELETE CASCADE,
    FOREIGN KEY (tmdb_id) REFERENCES movies(tmdb_id),
    UNIQUE(library_id, plex_rating_key)
);

-- Background sync jobs - manages sync operations
CREATE TABLE sync_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL, -- 'full_sync', 'library_sync', 'tmdb_matching'
    user_id INTEGER, -- User who triggered the sync (nullable for system jobs)
    library_id INTEGER, -- Specific library being synced (nullable for full sync)
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed', 'cancelled'
    progress INTEGER DEFAULT 0, -- Progress percentage (0-100)
    current_step TEXT, -- Current operation description
    total_items INTEGER DEFAULT 0, -- Total items to process
    processed_items INTEGER DEFAULT 0, -- Items processed so far
    successful_items INTEGER DEFAULT 0, -- Successfully processed items
    failed_items INTEGER DEFAULT 0, -- Failed items
    error_message TEXT, -- Error details if failed
    metadata_json TEXT, -- Additional job metadata as JSON
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (library_id) REFERENCES plex_libraries(id) ON DELETE CASCADE
);

-- TMDB API rate limiting tracking
CREATE TABLE tmdb_rate_limits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    time_window_start DATETIME NOT NULL, -- Start of current time window
    requests_count INTEGER DEFAULT 0, -- Requests made in current window
    last_request_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Initialize rate limiting with first entry
INSERT INTO tmdb_rate_limits (time_window_start, requests_count) 
VALUES (datetime('now'), 0);

-- Indexes for performance
CREATE INDEX idx_plex_servers_machine_id ON plex_servers(machine_id);
CREATE INDEX idx_plex_libraries_server_id ON plex_libraries(server_id);
CREATE INDEX idx_plex_libraries_type ON plex_libraries(type);
CREATE INDEX idx_user_plex_access_user_id ON user_plex_access(user_id);
CREATE INDEX idx_user_plex_access_library_id ON user_plex_access(library_id);
CREATE INDEX idx_user_plex_access_active ON user_plex_access(is_active);
CREATE INDEX idx_plex_library_items_library_id ON plex_library_items(library_id);
CREATE INDEX idx_plex_library_items_plex_guid ON plex_library_items(plex_guid);
CREATE INDEX idx_plex_library_items_tmdb_id ON plex_library_items(tmdb_id);
CREATE INDEX idx_plex_library_items_type ON plex_library_items(type);
CREATE INDEX idx_plex_library_items_active ON plex_library_items(is_active);
CREATE INDEX idx_sync_jobs_status ON sync_jobs(status);
CREATE INDEX idx_sync_jobs_user_id ON sync_jobs(user_id);
CREATE INDEX idx_sync_jobs_created_at ON sync_jobs(created_at);