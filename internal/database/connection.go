package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func Connect(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure SQLite for concurrent access
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",    // Enable WAL mode for better concurrent access
		"PRAGMA synchronous = NORMAL",  // Balance between performance and safety
		"PRAGMA cache_size = 1000",     // Increase cache size
		"PRAGMA temp_store = memory",   // Use memory for temporary tables
		"PRAGMA busy_timeout = 5000",   // Wait up to 5 seconds for locks
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)  // Limit concurrent connections
	db.SetMaxIdleConns(5)   // Keep some connections idle
	
	return db, nil
}