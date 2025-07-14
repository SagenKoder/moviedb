-- Plex to TMDB mapping table
CREATE TABLE plex_tmdb_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plex_guid TEXT NOT NULL UNIQUE,
    tmdb_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    year INTEGER,
    plex_rating_key TEXT, -- For specific server items
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tmdb_id) REFERENCES movies(tmdb_id),
    UNIQUE(plex_guid, tmdb_id)
);

-- Indexes for fast lookups
CREATE INDEX idx_plex_tmdb_mappings_plex_guid ON plex_tmdb_mappings(plex_guid);
CREATE INDEX idx_plex_tmdb_mappings_tmdb_id ON plex_tmdb_mappings(tmdb_id);
CREATE INDEX idx_plex_tmdb_mappings_title ON plex_tmdb_mappings(title);