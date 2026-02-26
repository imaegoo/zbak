package compression

import (
	"fmt"
)

// CompressionStrategy represents the strategy to use for compressing a directory
type CompressionStrategy int

const (
	// StrategySmallDir is used for directories smaller than volume size
	// The entire directory is compressed into a single file
	StrategySmallDir CompressionStrategy = iota

	// StrategyLargeNoSubdir is used for large directories without subdirectories
	// The directory contents are compressed with volume splitting
	StrategyLargeNoSubdir

	// StrategyLargeWithSubdir is used for large directories with subdirectories
	// Each subdirectory is processed recursively with the same logic
	StrategyLargeWithSubdir
)

// String returns a human-readable string representation of the strategy
func (s CompressionStrategy) String() string {
	switch s {
	case StrategySmallDir:
		return "SmallDir"
	case StrategyLargeNoSubdir:
		return "LargeNoSubdir"
	case StrategyLargeWithSubdir:
		return "LargeWithSubdir"
	default:
		return "Unknown"
	}
}

// FileSystemService defines the interface for file system operations
type FileSystemService interface {
	CalculateDirSize(path string) (int64, error)
	HasSubdirs(path string) (bool, error)
}

// Service provides compression operations for the backup tool
type Service struct {
	fs FileSystemService
}

// NewService creates a new CompressionService instance
func NewService(fs FileSystemService) *Service {
	return &Service{
		fs: fs,
	}
}

// DetermineStrategy determines the compression strategy for a directory
// based on its size and structure
//
// Strategy selection logic:
// 1. If directory size < volumeSize: use StrategySmallDir
// 2. If directory size >= volumeSize:
//    - If no subdirectories: use StrategyLargeNoSubdir
//    - If has subdirectories: use StrategyLargeWithSubdir
func (s *Service) DetermineStrategy(dirPath string, volumeSize int64) (CompressionStrategy, error) {
	// Calculate directory size
	dirSize, err := s.fs.CalculateDirSize(dirPath)
	if err != nil {
		return 0, fmt.Errorf("failed to determine compression strategy: %w", err)
	}

	// Small directory: compress as single file
	if dirSize < volumeSize {
		return StrategySmallDir, nil
	}

	// Large directory: check for subdirectories
	hasSubdirs, err := s.fs.HasSubdirs(dirPath)
	if err != nil {
		return 0, fmt.Errorf("failed to determine compression strategy: %w", err)
	}

	if hasSubdirs {
		return StrategyLargeWithSubdir, nil
	}

	return StrategyLargeNoSubdir, nil
}
