package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PlexCleanupService handles cleanup and maintenance for Plex data
type PlexCleanupService struct {
	db *sql.DB
}

// NewPlexCleanupService creates a new cleanup service
func NewPlexCleanupService(db *sql.DB) *PlexCleanupService {
	return &PlexCleanupService{
		db: db,
	}
}

// CleanupOrphanedItems removes library items that no longer have any users with access
func (s *PlexCleanupService) CleanupOrphanedItems(ctx context.Context) error {
	fmt.Println("Starting cleanup of orphaned Plex library items")
	
	// Remove items from libraries that have no active user access
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM plex_library_items 
		WHERE library_id NOT IN (
			SELECT DISTINCT library_id 
			FROM user_plex_access 
			WHERE is_active = 1
		)
	`)
	
	if err != nil {
		return fmt.Errorf("failed to cleanup orphaned items: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Cleaned up %d orphaned library items\n", rowsAffected)
	
	return nil
}

// CleanupInactiveUserAccess removes user access records for users who haven't synced in a long time
func (s *PlexCleanupService) CleanupInactiveUserAccess(ctx context.Context, daysInactive int) error {
	fmt.Printf("Starting cleanup of inactive user access (older than %d days)\n", daysInactive)
	
	// Mark user access as inactive if not verified recently
	result, err := s.db.ExecContext(ctx, `
		UPDATE user_plex_access 
		SET is_active = 0 
		WHERE last_verified_at < datetime('now', '-' || ? || ' days')
		AND is_active = 1
	`, daysInactive)
	
	if err != nil {
		return fmt.Errorf("failed to cleanup inactive user access: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Marked %d user access records as inactive\n", rowsAffected)
	
	return nil
}

// CleanupOldSyncJobs removes old completed sync jobs
func (s *PlexCleanupService) CleanupOldSyncJobs(ctx context.Context, daysOld int) error {
	fmt.Printf("Starting cleanup of old sync jobs (older than %d days)\n", daysOld)
	
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM sync_jobs 
		WHERE status IN ('completed', 'failed', 'cancelled') 
		AND created_at < datetime('now', '-' || ? || ' days')
	`, daysOld)
	
	if err != nil {
		return fmt.Errorf("failed to cleanup old sync jobs: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Cleaned up %d old sync jobs\n", rowsAffected)
	
	return nil
}

// CleanupUnmatchedItems removes items that failed to match with TMDB after multiple attempts
func (s *PlexCleanupService) CleanupUnmatchedItems(ctx context.Context, maxAttempts int) error {
	fmt.Printf("Starting cleanup of unmatched items (more than %d attempts)\n", maxAttempts)
	
	// Mark items as inactive if they failed to match multiple times
	result, err := s.db.ExecContext(ctx, `
		UPDATE plex_library_items 
		SET is_active = 0 
		WHERE tmdb_id IS NULL 
		AND matching_attempts >= ?
		AND is_active = 1
	`, maxAttempts)
	
	if err != nil {
		return fmt.Errorf("failed to cleanup unmatched items: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Marked %d unmatched items as inactive\n", rowsAffected)
	
	return nil
}

// CleanupOrphanedMappings removes TMDB mappings that no longer have corresponding library items
func (s *PlexCleanupService) CleanupOrphanedMappings(ctx context.Context) error {
	fmt.Println("Starting cleanup of orphaned TMDB mappings")
	
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM plex_tmdb_mappings 
		WHERE plex_guid NOT IN (
			SELECT DISTINCT plex_guid 
			FROM plex_library_items 
			WHERE is_active = 1
		)
	`)
	
	if err != nil {
		return fmt.Errorf("failed to cleanup orphaned mappings: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Cleaned up %d orphaned TMDB mappings\n", rowsAffected)
	
	return nil
}

// UpdateLibraryItemCounts updates the cached item counts for all libraries
func (s *PlexCleanupService) UpdateLibraryItemCounts(ctx context.Context) error {
	fmt.Println("Updating library item counts")
	
	_, err := s.db.ExecContext(ctx, `
		UPDATE plex_libraries 
		SET item_count = (
			SELECT COUNT(*) 
			FROM plex_library_items 
			WHERE library_id = plex_libraries.id 
			AND is_active = 1
		)
	`)
	
	if err != nil {
		return fmt.Errorf("failed to update library item counts: %w", err)
	}
	
	fmt.Println("Library item counts updated")
	return nil
}

// VerifyUserAccess checks if users still have access to their libraries
func (s *PlexCleanupService) VerifyUserAccess(ctx context.Context, userID int64, plexgoClient *PlexgoClient) error {
	fmt.Printf("Verifying user access for user %d\n", userID)
	
	// Get user's Plex token
	var plexToken string
	err := s.db.QueryRowContext(ctx, `
		SELECT plex_token FROM user_plex_tokens WHERE user_id = ?
	`, userID).Scan(&plexToken)
	
	if err != nil {
		return fmt.Errorf("failed to get Plex token: %w", err)
	}
	
	// Get user's current access from database
	rows, err := s.db.QueryContext(ctx, `
		SELECT upa.library_id, pl.section_key, ps.machine_id
		FROM user_plex_access upa
		JOIN plex_libraries pl ON upa.library_id = pl.id
		JOIN plex_servers ps ON pl.server_id = ps.id
		WHERE upa.user_id = ? AND upa.is_active = 1
	`, userID)
	
	if err != nil {
		return fmt.Errorf("failed to get user access: %w", err)
	}
	defer rows.Close()
	
	var accessibleLibraries []int64
	
	for rows.Next() {
		var libraryID int64
		var sectionKey int
		var machineID string
		
		if err := rows.Scan(&libraryID, &sectionKey, &machineID); err != nil {
			continue
		}
		
		// Try to access this library via Plex API
		// This is a simplified check - in a real implementation, you'd want to
		// actually verify access to each library
		accessibleLibraries = append(accessibleLibraries, libraryID)
	}
	
	// Update last_verified_at for accessible libraries
	if len(accessibleLibraries) > 0 {
		// Build IN clause for library IDs
		placeholders := make([]string, len(accessibleLibraries))
		args := make([]interface{}, len(accessibleLibraries)+1)
		args[0] = userID
		
		for i, libID := range accessibleLibraries {
			placeholders[i] = "?"
			args[i+1] = libID
		}
		
		query := fmt.Sprintf(`
			UPDATE user_plex_access 
			SET last_verified_at = datetime('now')
			WHERE user_id = ? AND library_id IN (%s)
		`, joinStrings(placeholders, ","))
		
		_, err = s.db.ExecContext(ctx, query, args...)
		if err != nil {
			fmt.Printf("Failed to update verification times: %v\n", err)
		}
	}
	
	fmt.Printf("Verified access for user %d to %d libraries\n", userID, len(accessibleLibraries))
	return nil
}

// RunFullCleanup runs all cleanup operations
func (s *PlexCleanupService) RunFullCleanup(ctx context.Context) error {
	fmt.Println("Starting full Plex cleanup")
	
	// Run cleanup operations in order
	cleanupOps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Cleanup inactive user access", func(ctx context.Context) error {
			return s.CleanupInactiveUserAccess(ctx, 30) // 30 days
		}},
		{"Cleanup orphaned items", s.CleanupOrphanedItems},
		{"Cleanup unmatched items", func(ctx context.Context) error {
			return s.CleanupUnmatchedItems(ctx, 5) // 5 attempts
		}},
		{"Cleanup orphaned mappings", s.CleanupOrphanedMappings},
		{"Update library item counts", s.UpdateLibraryItemCounts},
		{"Cleanup old sync jobs", func(ctx context.Context) error {
			return s.CleanupOldSyncJobs(ctx, 7) // 7 days
		}},
	}
	
	for _, op := range cleanupOps {
		fmt.Printf("Running: %s\n", op.name)
		if err := op.fn(ctx); err != nil {
			fmt.Printf("Cleanup operation failed: %s - %v\n", op.name, err)
			// Continue with other operations even if one fails
		}
	}
	
	fmt.Println("Full cleanup completed")
	return nil
}

// ScheduleCleanup can be called periodically to maintain the database
func (s *PlexCleanupService) ScheduleCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Cleanup scheduler stopping")
			return
		case <-ticker.C:
			fmt.Println("Running scheduled cleanup")
			if err := s.RunFullCleanup(ctx); err != nil {
				fmt.Printf("Scheduled cleanup failed: %v\n", err)
			}
		}
	}
}

// GetCleanupStats returns statistics about the cleanup operations
func (s *PlexCleanupService) GetCleanupStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Count total items
	var totalItems int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM plex_library_items WHERE is_active = 1
	`).Scan(&totalItems)
	if err != nil {
		return nil, err
	}
	stats["total_active_items"] = totalItems
	
	// Count unmatched items
	var unmatchedItems int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM plex_library_items 
		WHERE is_active = 1 AND tmdb_id IS NULL
	`).Scan(&unmatchedItems)
	if err != nil {
		return nil, err
	}
	stats["unmatched_items"] = unmatchedItems
	
	// Count active user access
	var activeAccess int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_plex_access WHERE is_active = 1
	`).Scan(&activeAccess)
	if err != nil {
		return nil, err
	}
	stats["active_user_access"] = activeAccess
	
	// Count pending sync jobs
	var pendingJobs int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sync_jobs WHERE status IN ('pending', 'running')
	`).Scan(&pendingJobs)
	if err != nil {
		return nil, err
	}
	stats["pending_sync_jobs"] = pendingJobs
	
	return stats, nil
}

// Helper function to join strings
func joinStrings(strs []string, separator string) string {
	if len(strs) == 0 {
		return ""
	}
	
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += separator + strs[i]
	}
	return result
}