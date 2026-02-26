package coordinator

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TimestampManager handles timestamp directory creation and management
type TimestampManager struct {
	targetDir string
}

// NewTimestampManager creates a new TimestampManager
func NewTimestampManager(targetDir string) *TimestampManager {
	return &TimestampManager{
		targetDir: targetDir,
	}
}

// CreateTimestampDir creates a new timestamp directory with format YYYY-MM-DD-HH-MM-SS
// Returns the timestamp directory name and error if directory already exists
func (tm *TimestampManager) CreateTimestampDir(t time.Time) (string, error) {
	// Format: YYYY-MM-DD-HH-MM-SS
	timestamp := t.Format("2006-01-02-15-04-05")
	timestampPath := filepath.Join(tm.targetDir, timestamp)

	// Check if directory already exists (conflict detection)
	if _, err := os.Stat(timestampPath); err == nil {
		return "", fmt.Errorf("%w: %s", ErrTimestampDirExists, timestamp)
	}

	// Create the timestamp directory
	if err := os.MkdirAll(timestampPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create timestamp directory: %w", err)
	}

	return timestamp, nil
}

// GetTimestampPath returns the full path to a timestamp directory
func (tm *TimestampManager) GetTimestampPath(timestamp string) string {
	return filepath.Join(tm.targetDir, timestamp)
}

// CreateRelativePathInTimestamp creates the relative path structure within the timestamp directory
// sourcePath: the file/directory path relative to the source directory
// timestamp: the timestamp directory name
// Returns the full path where the backup should be placed
func (tm *TimestampManager) CreateRelativePathInTimestamp(timestamp, sourcePath string) (string, error) {
	// Get the directory part of the source path
	sourceDir := filepath.Dir(sourcePath)
	
	// Build the full path in the timestamp directory
	targetPath := filepath.Join(tm.targetDir, timestamp, sourceDir)
	
	// Create the directory structure if it doesn't exist
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create relative path structure: %w", err)
	}
	
	return filepath.Join(tm.targetDir, timestamp, sourcePath), nil
}

// TimestampExists checks if a timestamp directory exists
func (tm *TimestampManager) TimestampExists(timestamp string) bool {
	timestampPath := filepath.Join(tm.targetDir, timestamp)
	_, err := os.Stat(timestampPath)
	return err == nil
}
