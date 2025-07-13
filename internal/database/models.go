package database

import (
	"database/sql"
	"fmt"
	"time"

	"moviedb/internal/types"
)

// GetOrCreateUser finds a user by Auth0 ID or creates a new one
func GetOrCreateUser(db *sql.DB, auth0ID, email, name string) (*types.User, error) {
	// First try to find existing user
	var user types.User
	err := db.QueryRow(`
		SELECT id, auth0_id, email, name, username, created_at 
		FROM users 
		WHERE auth0_id = ?
	`, auth0ID).Scan(&user.ID, &user.Auth0ID, &user.Email, &user.Name, &user.Username, &user.Created)

	if err == nil {
		// User exists, return it
		return &user, nil
	}

	if err != sql.ErrNoRows {
		// Actual error occurred
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// User doesn't exist, create new one
	result, err := db.Exec(`
		INSERT INTO users (auth0_id, email, name, created_at) 
		VALUES (?, ?, ?, ?)
	`, auth0ID, email, name, time.Now())

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	// Return the newly created user
	user = types.User{
		ID:      int(userID),
		Auth0ID: auth0ID,
		Email:   email,
		Name:    name,
		Created: time.Now(),
	}

	return &user, nil
}

// GetUserPreferences gets user preferences, creating default ones if they don't exist
func GetUserPreferences(db *sql.DB, userID int) (*types.UserPreferences, error) {
	var prefs types.UserPreferences
	err := db.QueryRow(`
		SELECT id, user_id, dark_mode, created_at, updated_at 
		FROM user_preferences 
		WHERE user_id = ?
	`, userID).Scan(&prefs.ID, &prefs.UserID, &prefs.DarkMode, &prefs.Created, &prefs.Updated)

	if err == nil {
		// Preferences exist
		return &prefs, nil
	}

	if err != sql.ErrNoRows {
		// Actual error occurred
		return nil, fmt.Errorf("failed to query user preferences: %w", err)
	}

	// Preferences don't exist, create default ones
	result, err := db.Exec(`
		INSERT INTO user_preferences (user_id, dark_mode, created_at, updated_at) 
		VALUES (?, ?, ?, ?)
	`, userID, false, time.Now(), time.Now())

	if err != nil {
		return nil, fmt.Errorf("failed to create user preferences: %w", err)
	}

	prefsID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get preferences ID: %w", err)
	}

	// Return the newly created preferences
	prefs = types.UserPreferences{
		ID:       int(prefsID),
		UserID:   userID,
		DarkMode: false,
		Created:  time.Now(),
		Updated:  time.Now(),
	}

	return &prefs, nil
}

// UpdateUserPreferences updates user preferences
func UpdateUserPreferences(db *sql.DB, userID int, darkMode bool) error {
	_, err := db.Exec(`
		UPDATE user_preferences 
		SET dark_mode = ?, updated_at = ? 
		WHERE user_id = ?
	`, darkMode, time.Now(), userID)

	if err != nil {
		return fmt.Errorf("failed to update user preferences: %w", err)
	}

	return nil
}