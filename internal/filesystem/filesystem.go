package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Service provides file system operations for the backup tool
type Service struct{}

// NewService creates a new FileSystemService instance
func NewService() *Service {
	return &Service{}
}

// CalculateDirSize calculates the total size of all files in a directory
// It includes all subdirectories and excludes symbolic links
// Returns 0 and logs error if calculation fails
func (s *Service) CalculateDirSize(path string) (int64, error) {
	// Check if path exists
	if _, err := os.Stat(path); err != nil {
		return 0, fmt.Errorf("failed to calculate directory size: %w", err)
	}

	var totalSize int64

	err := filepath.Walk(path, func(filePath string, info fs.FileInfo, err error) error {
		if err != nil {
			// Log error but continue walking
			return nil
		}

		// Skip symbolic links
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// Add file size (directories have size 0)
		if !info.IsDir() {
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate directory size: %w", err)
	}

	return totalSize, nil
}

// HasSubdirs checks if a directory contains any subdirectories
func (s *Service) HasSubdirs(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return true, nil
		}
	}

	return false, nil
}

// ListFiles returns a list of all files in a directory (non-recursive)
func (s *Service) ListFiles(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			// Use filepath.Join for cross-platform path handling
			files = append(files, filepath.Join(path, entry.Name()))
		}
	}

	return files, nil
}

// CreateDir creates a directory and all necessary parent directories
// Uses cross-platform path handling
func (s *Service) CreateDir(path string) error {
	// Clean the path for cross-platform compatibility
	cleanPath := filepath.Clean(path)
	
	err := os.MkdirAll(cleanPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// FileExists checks if a file exists at the given path
func (s *Service) FileExists(path string) bool {
	// Clean the path for cross-platform compatibility
	cleanPath := filepath.Clean(path)
	
	info, err := os.Stat(cleanPath)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
