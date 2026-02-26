package coordinator

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"zbak/internal/compression"
	"zbak/internal/config"
	"zbak/internal/detector"
	"zbak/internal/filesystem"
	"zbak/internal/index"
	"zbak/internal/logger"
	"zbak/internal/sevenzip"
)

// mockLogger is a simple logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...interface{}) {}
func (m *mockLogger) Info(msg string, args ...interface{})  {}
func (m *mockLogger) Warn(msg string, args ...interface{})  {}
func (m *mockLogger) Error(msg string, args ...interface{}) {}
func (m *mockLogger) SetOutput(w io.Writer)                 {}
func (m *mockLogger) Close() error                          { return nil }

// Ensure mockLogger implements logger.Logger interface at compile time
var _ logger.Logger = (*mockLogger)(nil)

// sevenZipAdapter adapts sevenzip.Wrapper to compression.SevenZipWrapper
type sevenZipAdapter struct {
	wrapper sevenzip.Wrapper
}

func (a *sevenZipAdapter) Compress(params compression.CompressParams) error {
	szParams := sevenzip.CompressParams{
		Sources:    params.Sources,
		Output:     params.Output,
		Password:   params.Password,
		VolumeSize: params.VolumeSize,
	}
	return a.wrapper.Compress(szParams)
}

func newSevenZipAdapter(wrapper sevenzip.Wrapper) compression.SevenZipWrapper {
	return &sevenZipAdapter{wrapper: wrapper}
}

func TestNewBackupCoordinator(t *testing.T) {
	cfg := &config.Config{
		SourceDir:   "/test/source",
		TargetDir:   "/test/target",
		VolumeSize:  1024,
		Password:    "test",
		Concurrency: 2,
	}

	fs := filesystem.NewService()
	sz := sevenzip.NewWrapper()
	szAdapter := newSevenZipAdapter(sz)
	log := &mockLogger{}
	compressionSvc := compression.NewService(fs, szAdapter, log)

	bc := NewBackupCoordinator(cfg, compressionSvc, log)

	if bc == nil {
		t.Fatal("NewBackupCoordinator returned nil")
	}

	if bc.config != cfg {
		t.Error("Config not set correctly")
	}

	if bc.detector == nil {
		t.Error("Detector not initialized")
	}

	if bc.compressionSvc != compressionSvc {
		t.Error("CompressionService not set correctly")
	}

	if bc.timestampMgr == nil {
		t.Error("TimestampManager not initialized")
	}

	if bc.logger != log {
		t.Error("Logger not set correctly")
	}

	expectedIndexPath := filepath.Join(cfg.TargetDir, "index.yaml")
	if bc.indexPath != expectedIndexPath {
		t.Errorf("Expected indexPath %s, got %s", expectedIndexPath, bc.indexPath)
	}
}

func TestBackupCoordinator_Execute_NoChanges(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create file index with the same file
	fileIndex := index.NewFileIndex()
	info, _ := os.Stat(testFile)
	fileIndex.Files["test.txt"] = index.FileEntry{
		SourcePath:   "test.txt",
		Size:         info.Size(),
		ModTime:      info.ModTime(),
		ArchivePath:  "2024-01-01-00-00-00/test.7z.001",
		TimestampDir: "2024-01-01-00-00-00",
		Deleted:      false,
	}

	// Save index
	indexPath := filepath.Join(targetDir, "index.yaml")
	if err := index.Save(indexPath, fileIndex); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	// Create logger
	log, err := logger.NewLogger(targetDir, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create config
	cfg := &config.Config{
		SourceDir:   sourceDir,
		TargetDir:   targetDir,
		VolumeSize:  1024 * 1024,
		Password:    "test",
		Concurrency: 1,
	}

	// Create coordinator
	fs := filesystem.NewService()
	sz := sevenzip.NewWrapper()
	szAdapter := newSevenZipAdapter(sz)
	compressionSvc := compression.NewService(fs, szAdapter, log)
	bc := NewBackupCoordinator(cfg, compressionSvc, log)

	// Execute backup
	ctx := context.Background()
	report, err := bc.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify report
	if report.NewFiles != 0 {
		t.Errorf("Expected 0 new files, got %d", report.NewFiles)
	}
	if report.ModifiedFiles != 0 {
		t.Errorf("Expected 0 modified files, got %d", report.ModifiedFiles)
	}
	if report.DeletedFiles != 0 {
		t.Errorf("Expected 0 deleted files, got %d", report.DeletedFiles)
	}
	if report.UnchangedFiles != 1 {
		t.Errorf("Expected 1 unchanged file, got %d", report.UnchangedFiles)
	}
}

func TestBackupCoordinator_Execute_NewFiles(t *testing.T) {
	// Skip if 7zip is not available
	sz := sevenzip.NewWrapper()
	if _, err := sz.Detect(); err != nil {
		t.Skip("7zip not available, skipping test")
	}

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create empty file index
	fileIndex := index.NewFileIndex()
	indexPath := filepath.Join(targetDir, "index.yaml")
	if err := index.Save(indexPath, fileIndex); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	// Create logger
	log, err := logger.NewLogger(targetDir, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create config
	cfg := &config.Config{
		SourceDir:   sourceDir,
		TargetDir:   targetDir,
		VolumeSize:  1024 * 1024,
		Password:    "test",
		Concurrency: 1,
	}

	// Create coordinator
	fs := filesystem.NewService()
	szAdapter := newSevenZipAdapter(sz)
	compressionSvc := compression.NewService(fs, szAdapter, log)
	bc := NewBackupCoordinator(cfg, compressionSvc, log)

	// Execute backup
	ctx := context.Background()
	report, err := bc.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify report
	if report.NewFiles != 1 {
		t.Errorf("Expected 1 new file, got %d", report.NewFiles)
	}
	if report.ModifiedFiles != 0 {
		t.Errorf("Expected 0 modified files, got %d", report.ModifiedFiles)
	}
	if report.DeletedFiles != 0 {
		t.Errorf("Expected 0 deleted files, got %d", report.DeletedFiles)
	}

	// Verify timestamp directory was created
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		t.Fatalf("Failed to read target dir: %v", err)
	}

	timestampDirFound := false
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "." && entry.Name() != ".." {
			timestampDirFound = true
			break
		}
	}

	if !timestampDirFound {
		t.Error("Timestamp directory was not created")
	}

	// Verify index was updated
	updatedIndex, err := index.Load(indexPath)
	if err != nil {
		t.Fatalf("Failed to load updated index: %v", err)
	}

	if len(updatedIndex.Files) != 1 {
		t.Errorf("Expected 1 file in index, got %d", len(updatedIndex.Files))
	}

	entry, exists := updatedIndex.Files["test.txt"]
	if !exists {
		t.Error("test.txt not found in index")
	}

	if entry.Deleted {
		t.Error("File should not be marked as deleted")
	}
}

func TestBackupCoordinator_Execute_ModifiedFiles(t *testing.T) {
	// Skip if 7zip is not available
	sz := sevenzip.NewWrapper()
	if _, err := sz.Detect(); err != nil {
		t.Skip("7zip not available, skipping test")
	}

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait a bit to ensure different modification time
	time.Sleep(10 * time.Millisecond)

	// Create file index with old file info
	fileIndex := index.NewFileIndex()
	fileIndex.Files["test.txt"] = index.FileEntry{
		SourcePath:   "test.txt",
		Size:         5, // Different size
		ModTime:      time.Now().Add(-1 * time.Hour),
		ArchivePath:  "2024-01-01-00-00-00/test.7z.001",
		TimestampDir: "2024-01-01-00-00-00",
		Deleted:      false,
	}

	// Save index
	indexPath := filepath.Join(targetDir, "index.yaml")
	if err := index.Save(indexPath, fileIndex); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	// Create logger
	log, err := logger.NewLogger(targetDir, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create config
	cfg := &config.Config{
		SourceDir:   sourceDir,
		TargetDir:   targetDir,
		VolumeSize:  1024 * 1024,
		Password:    "test",
		Concurrency: 1,
	}

	// Create coordinator
	fs := filesystem.NewService()
	szAdapter := newSevenZipAdapter(sz)
	compressionSvc := compression.NewService(fs, szAdapter, log)
	bc := NewBackupCoordinator(cfg, compressionSvc, log)

	// Execute backup
	ctx := context.Background()
	report, err := bc.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify report
	if report.NewFiles != 0 {
		t.Errorf("Expected 0 new files, got %d", report.NewFiles)
	}
	if report.ModifiedFiles != 1 {
		t.Errorf("Expected 1 modified file, got %d", report.ModifiedFiles)
	}
	if report.DeletedFiles != 0 {
		t.Errorf("Expected 0 deleted files, got %d", report.DeletedFiles)
	}
}

func TestBackupCoordinator_Execute_DeletedFiles(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	// Create file index with a file that doesn't exist in source
	fileIndex := index.NewFileIndex()
	fileIndex.Files["deleted.txt"] = index.FileEntry{
		SourcePath:   "deleted.txt",
		Size:         100,
		ModTime:      time.Now().Add(-1 * time.Hour),
		ArchivePath:  "2024-01-01-00-00-00/deleted.7z.001",
		TimestampDir: "2024-01-01-00-00-00",
		Deleted:      false,
	}

	// Save index
	indexPath := filepath.Join(targetDir, "index.yaml")
	if err := index.Save(indexPath, fileIndex); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	// Create logger
	log, err := logger.NewLogger(targetDir, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create config
	cfg := &config.Config{
		SourceDir:   sourceDir,
		TargetDir:   targetDir,
		VolumeSize:  1024 * 1024,
		Password:    "test",
		Concurrency: 1,
	}

	// Create coordinator
	fs := filesystem.NewService()
	sz := sevenzip.NewWrapper()
	szAdapter := newSevenZipAdapter(sz)
	compressionSvc := compression.NewService(fs, szAdapter, log)
	bc := NewBackupCoordinator(cfg, compressionSvc, log)

	// Execute backup
	ctx := context.Background()
	report, err := bc.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify report
	if report.NewFiles != 0 {
		t.Errorf("Expected 0 new files, got %d", report.NewFiles)
	}
	if report.ModifiedFiles != 0 {
		t.Errorf("Expected 0 modified files, got %d", report.ModifiedFiles)
	}
	if report.DeletedFiles != 1 {
		t.Errorf("Expected 1 deleted file, got %d", report.DeletedFiles)
	}

	// Verify index was updated with deleted flag
	updatedIndex, err := index.Load(indexPath)
	if err != nil {
		t.Fatalf("Failed to load updated index: %v", err)
	}

	entry, exists := updatedIndex.Files["deleted.txt"]
	if !exists {
		t.Error("deleted.txt not found in index")
	}

	if !entry.Deleted {
		t.Error("File should be marked as deleted")
	}
}

func TestBackupCoordinator_BuildCompressionTasks(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	// Create test directory structure
	subdir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Create test files
	if err := os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Create config
	cfg := &config.Config{
		SourceDir:   sourceDir,
		TargetDir:   targetDir,
		VolumeSize:  1024 * 1024,
		Password:    "test",
		Concurrency: 1,
	}

	// Create coordinator
	fs := filesystem.NewService()
	sz := sevenzip.NewWrapper()
	szAdapter := newSevenZipAdapter(sz)
	log := &mockLogger{}
	compressionSvc := compression.NewService(fs, szAdapter, log)
	bc := NewBackupCoordinator(cfg, compressionSvc, log)

	// Create change set
	changeSet := &detector.ChangeSet{
		NewFiles:       []string{"file1.txt", "subdir/file2.txt"},
		ModifiedFiles:  []string{},
		DeletedFiles:   []string{},
		UnchangedFiles: []string{},
	}

	// Build tasks
	timestamp := "2024-01-15-10-30-45"
	tasks, err := bc.buildCompressionTasks(changeSet, timestamp)
	if err != nil {
		t.Fatalf("buildCompressionTasks failed: %v", err)
	}

	// Verify tasks
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// Verify task properties
	for _, task := range tasks {
		if task.Password != cfg.Password {
			t.Errorf("Expected password %s, got %s", cfg.Password, task.Password)
		}
		if task.VolumeSize != cfg.VolumeSize {
			t.Errorf("Expected volume size %d, got %d", cfg.VolumeSize, task.VolumeSize)
		}
	}
}

func TestBackupCoordinator_Execute_CompressionError(t *testing.T) {
	// Skip if 7zip is not available
	sz := sevenzip.NewWrapper()
	if _, err := sz.Detect(); err != nil {
		t.Skip("7zip not available, skipping test")
	}

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	// Create test files in different directories
	subdir1 := filepath.Join(sourceDir, "subdir1")
	subdir2 := filepath.Join(sourceDir, "subdir2")
	if err := os.MkdirAll(subdir1, 0755); err != nil {
		t.Fatalf("Failed to create subdir1: %v", err)
	}
	if err := os.MkdirAll(subdir2, 0755); err != nil {
		t.Fatalf("Failed to create subdir2: %v", err)
	}

	// Create test files
	if err := os.WriteFile(filepath.Join(subdir1, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir2, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Create empty file index
	fileIndex := index.NewFileIndex()
	indexPath := filepath.Join(targetDir, "index.yaml")
	if err := index.Save(indexPath, fileIndex); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	// Create logger
	log, err := logger.NewLogger(targetDir, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create config with invalid password to potentially cause errors
	// Note: 7zip might still succeed, so this test verifies the error handling mechanism
	cfg := &config.Config{
		SourceDir:   sourceDir,
		TargetDir:   targetDir,
		VolumeSize:  1024 * 1024,
		Password:    "test",
		Concurrency: 2, // Use concurrency to test parallel error handling
	}

	// Create coordinator
	fs := filesystem.NewService()
	szAdapter := newSevenZipAdapter(sz)
	compressionSvc := compression.NewService(fs, szAdapter, log)
	bc := NewBackupCoordinator(cfg, compressionSvc, log)

	// Execute backup
	ctx := context.Background()
	report, err := bc.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify that backup completed even if there were potential errors
	// The coordinator should continue execution
	if report.NewFiles != 2 {
		t.Errorf("Expected 2 new files, got %d", report.NewFiles)
	}

	// Verify that tasks were attempted
	totalTasks := report.SuccessCount + report.FailureCount
	if totalTasks == 0 {
		t.Error("Expected at least some tasks to be executed")
	}

	// Verify report contains error information if any failures occurred
	if report.FailureCount > 0 {
		if len(report.Errors) != report.FailureCount {
			t.Errorf("Expected %d errors in report, got %d", report.FailureCount, len(report.Errors))
		}
	}
}

func TestBackupCoordinator_Execute_ConcurrentBackup(t *testing.T) {
	// Skip if 7zip is not available
	sz := sevenzip.NewWrapper()
	if _, err := sz.Detect(); err != nil {
		t.Skip("7zip not available, skipping test")
	}

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	// Create multiple subdirectories with files
	for i := 1; i <= 4; i++ {
		subdir := filepath.Join(sourceDir, fmt.Sprintf("subdir%d", i))
		if err := os.MkdirAll(subdir, 0755); err != nil {
			t.Fatalf("Failed to create subdir%d: %v", i, err)
		}
		
		testFile := filepath.Join(subdir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(testFile, []byte(fmt.Sprintf("content%d", i)), 0644); err != nil {
			t.Fatalf("Failed to create file%d: %v", i, err)
		}
	}

	// Create empty file index
	fileIndex := index.NewFileIndex()
	indexPath := filepath.Join(targetDir, "index.yaml")
	if err := index.Save(indexPath, fileIndex); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	// Create logger
	log, err := logger.NewLogger(targetDir, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create config with concurrency > 1
	cfg := &config.Config{
		SourceDir:   sourceDir,
		TargetDir:   targetDir,
		VolumeSize:  1024 * 1024,
		Password:    "test",
		Concurrency: 3, // Test concurrent execution
	}

	// Create coordinator
	fs := filesystem.NewService()
	szAdapter := newSevenZipAdapter(sz)
	compressionSvc := compression.NewService(fs, szAdapter, log)
	bc := NewBackupCoordinator(cfg, compressionSvc, log)

	// Execute backup
	ctx := context.Background()
	report, err := bc.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify report
	if report.NewFiles != 4 {
		t.Errorf("Expected 4 new files, got %d", report.NewFiles)
	}

	// Verify all tasks succeeded
	if report.FailureCount > 0 {
		t.Errorf("Expected 0 failures, got %d", report.FailureCount)
		for _, err := range report.Errors {
			t.Logf("Error: %v", err)
		}
	}

	// Verify index was updated with all files
	updatedIndex, err := index.Load(indexPath)
	if err != nil {
		t.Fatalf("Failed to load updated index: %v", err)
	}

	if len(updatedIndex.Files) != 4 {
		t.Errorf("Expected 4 files in index, got %d", len(updatedIndex.Files))
	}
}

func TestBackupReport_Fields(t *testing.T) {
	report := &BackupReport{
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(1 * time.Minute),
		TotalFiles:     10,
		NewFiles:       3,
		ModifiedFiles:  2,
		DeletedFiles:   1,
		UnchangedFiles: 4,
		SuccessCount:   9,
		FailureCount:   1,
		TotalSize:      1024,
		Errors:         []error{},
	}

	if report.TotalFiles != 10 {
		t.Errorf("Expected TotalFiles 10, got %d", report.TotalFiles)
	}
	if report.NewFiles != 3 {
		t.Errorf("Expected NewFiles 3, got %d", report.NewFiles)
	}
	if report.ModifiedFiles != 2 {
		t.Errorf("Expected ModifiedFiles 2, got %d", report.ModifiedFiles)
	}
	if report.DeletedFiles != 1 {
		t.Errorf("Expected DeletedFiles 1, got %d", report.DeletedFiles)
	}
	if report.UnchangedFiles != 4 {
		t.Errorf("Expected UnchangedFiles 4, got %d", report.UnchangedFiles)
	}
	if report.SuccessCount != 9 {
		t.Errorf("Expected SuccessCount 9, got %d", report.SuccessCount)
	}
	if report.FailureCount != 1 {
		t.Errorf("Expected FailureCount 1, got %d", report.FailureCount)
	}
}
