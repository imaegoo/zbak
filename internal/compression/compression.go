package compression

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	ListFiles(path string) ([]string, error)
	CreateDir(path string) error
}

// SevenZipWrapper defines the interface for 7zip operations
type SevenZipWrapper interface {
	Compress(params CompressParams) error
}

// CompressParams represents parameters for 7zip compression
type CompressParams struct {
	Sources    []string
	Output     string
	Password   string
	VolumeSize int64
}

// CompressionTask represents a compression task
type CompressionTask struct {
	SourcePath string
	TargetPath string
	Password   string
	VolumeSize int64
	Strategy   CompressionStrategy
}

// Service provides compression operations for the backup tool
type Service struct {
	fs      FileSystemService
	sevenZip SevenZipWrapper
}

// NewService creates a new CompressionService instance
func NewService(fs FileSystemService, sevenZip SevenZipWrapper) *Service {
	return &Service{
		fs:      fs,
		sevenZip: sevenZip,
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

// CompressDirectory compresses a directory according to the specified task
func (s *Service) CompressDirectory(ctx context.Context, task CompressionTask) error {
	// Ensure target directory exists
	targetDir := filepath.Dir(task.TargetPath)
	if err := s.fs.CreateDir(targetDir); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	switch task.Strategy {
	case StrategySmallDir:
		return s.compressSmallDir(ctx, task)
	case StrategyLargeNoSubdir:
		return s.compressLargeNoSubdir(ctx, task)
	case StrategyLargeWithSubdir:
		return s.compressLargeWithSubdir(ctx, task)
	default:
		return fmt.Errorf("unknown compression strategy: %v", task.Strategy)
	}
}

// compressSmallDir compresses a small directory into a single file
// Requirements: 5.1, 5.2, 5.3, 5.4, 5.5
func (s *Service) compressSmallDir(ctx context.Context, task CompressionTask) error {
	// Generate output filename: dirname.7z.001
	outputFile := task.TargetPath
	if !strings.HasSuffix(outputFile, ".7z.001") {
		outputFile = outputFile + ".7z.001"
	}

	// Compress the entire directory
	params := CompressParams{
		Sources:    []string{task.SourcePath},
		Output:     outputFile,
		Password:   task.Password,
		VolumeSize: 0, // No volume splitting for small directories
	}

	if err := s.sevenZip.Compress(params); err != nil {
		return fmt.Errorf("failed to compress small directory: %w", err)
	}

	return nil
}

// compressLargeNoSubdir compresses a large directory without subdirectories using volume splitting
// Requirements: 6.1, 6.2, 6.3, 6.4, 6.5
func (s *Service) compressLargeNoSubdir(ctx context.Context, task CompressionTask) error {
	// Generate output filename: dirname.7z.001
	outputFile := task.TargetPath
	if !strings.HasSuffix(outputFile, ".7z.001") {
		outputFile = outputFile + ".7z.001"
	}

	// Compress with volume splitting
	params := CompressParams{
		Sources:    []string{task.SourcePath},
		Output:     outputFile,
		Password:   task.Password,
		VolumeSize: task.VolumeSize,
	}

	if err := s.sevenZip.Compress(params); err != nil {
		return fmt.Errorf("failed to compress large directory without subdirs: %w", err)
	}

	return nil
}

// compressLargeWithSubdir recursively compresses a large directory with subdirectories
// Requirements: 7.1, 7.2, 7.3, 7.4
func (s *Service) compressLargeWithSubdir(ctx context.Context, task CompressionTask) error {
	// Read directory entries
	entries, err := os.ReadDir(task.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	var subdirs []string

	// Separate files and subdirectories
	for _, entry := range entries {
		fullPath := filepath.Join(task.SourcePath, entry.Name())
		
		if entry.IsDir() {
			subdirs = append(subdirs, fullPath)
		} else {
			files = append(files, fullPath)
		}
	}

	// Compress non-directory files as "files.7z.001"
	// These files are placed in the same directory level as subdirectories
	if len(files) > 0 {
		// Ensure the target directory exists for files
		targetDir := task.TargetPath
		if !strings.HasSuffix(task.TargetPath, ".7z.001") {
			targetDir = filepath.Dir(task.TargetPath)
		}
		
		filesOutput := filepath.Join(targetDir, "files.7z.001")
		params := CompressParams{
			Sources:    files,
			Output:     filesOutput,
			Password:   task.Password,
			VolumeSize: task.VolumeSize,
		}

		if err := s.sevenZip.Compress(params); err != nil {
			return fmt.Errorf("failed to compress files in directory: %w", err)
		}
	}

	// Recursively process each subdirectory
	// Maintain directory structure by creating subdirectory paths in target
	for _, subdir := range subdirs {
		subdirName := filepath.Base(subdir)
		
		// Preserve directory structure in target
		targetDir := task.TargetPath
		if strings.HasSuffix(task.TargetPath, ".7z.001") {
			targetDir = filepath.Dir(task.TargetPath)
		}
		subdirTarget := filepath.Join(targetDir, subdirName)

		// Determine strategy for subdirectory
		strategy, err := s.DetermineStrategy(subdir, task.VolumeSize)
		if err != nil {
			return fmt.Errorf("failed to determine strategy for subdirectory %s: %w", subdirName, err)
		}

		// Create subtask
		subtask := CompressionTask{
			SourcePath: subdir,
			TargetPath: subdirTarget,
			Password:   task.Password,
			VolumeSize: task.VolumeSize,
			Strategy:   strategy,
		}

		// Recursively compress subdirectory
		if err := s.CompressDirectory(ctx, subtask); err != nil {
			return fmt.Errorf("failed to compress subdirectory %s: %w", subdirName, err)
		}
	}

	return nil
}
