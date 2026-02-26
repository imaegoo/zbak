package compression

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"zbak/internal/sevenzip"
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
	Compress(params sevenzip.CompressParams) error
}

// CompressParams represents parameters for 7zip compression
// This is an alias to sevenzip.CompressParams for backward compatibility
type CompressParams = sevenzip.CompressParams

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

	// Log strategy decision for debugging
	fmt.Printf("[DEBUG] DetermineStrategy: path=%s, size=%d bytes (%.2f MB), volumeSize=%d bytes (%.2f GB)\n",
		dirPath, dirSize, float64(dirSize)/(1024*1024), volumeSize, float64(volumeSize)/(1024*1024*1024))

	// Small directory: compress as single file
	if dirSize < volumeSize {
		fmt.Printf("[DEBUG] Strategy: SmallDir (size < volumeSize)\n")
		return StrategySmallDir, nil
	}

	// Large directory: check for subdirectories
	hasSubdirs, err := s.fs.HasSubdirs(dirPath)
	if err != nil {
		return 0, fmt.Errorf("failed to determine compression strategy: %w", err)
	}

	if hasSubdirs {
		fmt.Printf("[DEBUG] Strategy: LargeWithSubdir (size >= volumeSize && hasSubdirs)\n")
		return StrategyLargeWithSubdir, nil
	}

	fmt.Printf("[DEBUG] Strategy: LargeNoSubdir (size >= volumeSize && !hasSubdirs)\n")
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
	// Ensure the output always ends with .7z.001
	outputFile := task.TargetPath
	if !strings.HasSuffix(outputFile, ".7z.001") {
		if strings.HasSuffix(outputFile, ".7z") {
			outputFile = outputFile + ".001"
		} else {
			outputFile = outputFile + ".7z.001"
		}
	}

	fmt.Printf("[DEBUG] compressSmallDir: source=%s, target=%s, output=%s\n", 
		task.SourcePath, task.TargetPath, outputFile)

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

	fmt.Printf("[DEBUG] compressSmallDir: compression completed successfully\n")
	return nil
}

// compressLargeNoSubdir compresses a large directory without subdirectories using volume splitting
// Requirements: 6.1, 6.2, 6.3, 6.4, 6.5
func (s *Service) compressLargeNoSubdir(ctx context.Context, task CompressionTask) error {
	// Generate output filename: dirname.7z.001
	// Ensure the output always ends with .7z.001
	outputFile := task.TargetPath
	if !strings.HasSuffix(outputFile, ".7z.001") {
		if strings.HasSuffix(outputFile, ".7z") {
			outputFile = outputFile + ".001"
		} else {
			outputFile = outputFile + ".7z.001"
		}
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

	// Recursively process each subdirectory FIRST
	// This ensures subdirectory archives are created before files.7z
	// Maintain directory structure by creating subdirectory paths in target
	for _, subdir := range subdirs {
		subdirName := filepath.Base(subdir)
		
		// Preserve directory structure in target
		targetDir := task.TargetPath
		if strings.HasSuffix(task.TargetPath, ".7z.001") || strings.HasSuffix(task.TargetPath, ".7z") {
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

	// Compress non-directory files as "zbaksubfiles.7z.001" AFTER subdirectories
	// These files are placed in the same directory level as subdirectories
	// Using "zbaksubfiles" instead of "files" to avoid conflicts with subdirectories named "files"
	if len(files) > 0 {
		// Ensure the target directory exists for files
		targetDir := task.TargetPath
		if strings.HasSuffix(task.TargetPath, ".7z.001") || strings.HasSuffix(task.TargetPath, ".7z") {
			targetDir = filepath.Dir(task.TargetPath)
		}
		
		filesOutput := filepath.Join(targetDir, "zbaksubfiles.7z.001")
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

	return nil
}

// WorkerPool manages concurrent compression tasks
// Requirements: 9.1, 9.2, 9.3, 9.4
type WorkerPool struct {
	service     *Service
	workerCount int
	taskChan    chan CompressionTask
	errorChan   chan error
	doneChan    chan struct{}
	errors      []error
	wg          sync.WaitGroup
}

// NewWorkerPool creates a new WorkerPool instance
func NewWorkerPool(service *Service, workerCount int) *WorkerPool {
	if workerCount < 1 {
		workerCount = 1
	}
	
	return &WorkerPool{
		service:     service,
		workerCount: workerCount,
		taskChan:    make(chan CompressionTask, workerCount*2),
		errorChan:   make(chan error, workerCount*2),
		doneChan:    make(chan struct{}),
		errors:      make([]error, 0),
	}
}

// Start initializes and starts the worker goroutines
// Requirement 9.1: Create worker goroutines pool
func (wp *WorkerPool) Start(ctx context.Context) {
	// Start error collector goroutine
	go wp.collectErrors()
	
	// Start worker goroutines
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx)
	}
}

// worker is the goroutine that processes compression tasks
// Requirements 9.2, 9.3: Support serial (concurrency=1) and parallel (concurrency>1) execution
func (wp *WorkerPool) worker(ctx context.Context) {
	defer wp.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-wp.taskChan:
			if !ok {
				return
			}
			
			// Execute compression task
			if err := wp.service.CompressDirectory(ctx, task); err != nil {
				wp.errorChan <- fmt.Errorf("compression task failed for %s: %w", task.SourcePath, err)
			}
		}
	}
}

// collectErrors collects errors from worker goroutines
// Requirement 10.1, 10.2: Collect errors from failed tasks
func (wp *WorkerPool) collectErrors() {
	for err := range wp.errorChan {
		wp.errors = append(wp.errors, err)
	}
	close(wp.doneChan)
}

// Submit adds a compression task to the queue
func (wp *WorkerPool) Submit(task CompressionTask) {
	wp.taskChan <- task
}

// Wait waits for all tasks to complete and returns any errors
// Requirement 9.4: Wait for all tasks to complete
func (wp *WorkerPool) Wait() []error {
	// Close task channel to signal workers to stop
	close(wp.taskChan)
	
	// Wait for all workers to finish
	wp.wg.Wait()
	
	// Close error channel after all workers are done
	close(wp.errorChan)
	
	// Wait for error collector to finish
	<-wp.doneChan
	
	return wp.errors
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	close(wp.taskChan)
}
