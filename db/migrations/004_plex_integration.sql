-- Plex Integration
CREATE TABLE user_plex_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    plex_token TEXT NOT NULL,
    plex_username TEXT,
    plex_email TEXT,
    plex_thumb TEXT, -- User's Plex avatar
    server_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(user_id) -- One Plex account per user
);

-- Track Plex authentication attempts (for pin flow)
CREATE TABLE plex_auth_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    pin_id TEXT NOT NULL,
    pin_code TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    completed BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Indexes
CREATE INDEX idx_user_plex_tokens_user_id ON user_plex_tokens(user_id);
CREATE INDEX idx_plex_auth_attempts_user_id ON plex_auth_attempts(user_id);
CREATE INDEX idx_plex_auth_attempts_pin_id ON plex_auth_attempts(pin_id);