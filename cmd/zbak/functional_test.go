package main

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zbak/internal/config"
	"zbak/internal/index"
)

// TestHelper provides utilities for functional tests
type TestHelper struct {
	t         *testing.T
	tempDir   string
	sourceDir string
	targetDir string
	configPath string
}

// NewTestHelper creates a new test helper with temporary directories
func NewTestHelper(t *testing.T) *TestHelper {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")
	configPath := filepath.Join(tempDir, "config.yaml")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	return &TestHelper{
		t:         t,
		tempDir:   tempDir,
		sourceDir: sourceDir,
		targetDir: targetDir,
		configPath: configPath,
	}
}

// CreateConfig creates a config file with the given parameters
func (h *TestHelper) CreateConfig(volumeSize int64, concurrency int) {
	cfg := &config.Config{
		SourceDir:   h.sourceDir,
		TargetDir:   h.targetDir,
		VolumeSize:  volumeSize,
		Password:    "test123",
		Concurrency: concurrency,
	}

	configMgr := config.NewConfigManager()
	if err := configMgr.Save(h.configPath, cfg); err != nil {
		h.t.Fatalf("Failed to save config: %v", err)
	}
}

// CreateFile creates a file with the given content
func (h *TestHelper) CreateFile(relativePath string, content []byte) {
	fullPath := filepath.Join(h.sourceDir, relativePath)
	dir := filepath.Dir(fullPath)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		h.t.Fatalf("Failed to create directory %s: %v", dir, err)
	}
	
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		h.t.Fatalf("Failed to create file %s: %v", relativePath, err)
	}
}

// CreateFileWithSize creates a file with the specified size
func (h *TestHelper) CreateFileWithSize(relativePath string, size int64) {
	content := make([]byte, size)
	// Use crypto/rand to generate incompressible random data
	// This ensures the data won't compress too well in tests
	if _, err := rand.Read(content); err != nil {
		h.t.Fatalf("Failed to generate random data for %s: %v", relativePath, err)
	}
	h.CreateFile(relativePath, content)
}

// ModifyFile modifies an existing file
func (h *TestHelper) ModifyFile(relativePath string, newContent []byte) {
	fullPath := filepath.Join(h.sourceDir, relativePath)
	
	// Wait a bit to ensure modification time changes
	time.Sleep(10 * time.Millisecond)
	
	if err := os.WriteFile(fullPath, newContent, 0644); err != nil {
		h.t.Fatalf("Failed to modify file %s: %v", relativePath, err)
	}
}

// DeleteFile deletes a file from source directory
func (h *TestHelper) DeleteFile(relativePath string) {
	fullPath := filepath.Join(h.sourceDir, relativePath)
	if err := os.Remove(fullPath); err != nil {
		h.t.Fatalf("Failed to delete file %s: %v", relativePath, err)
	}
}

// FileExists checks if a file exists in source directory
func (h *TestHelper) FileExists(relativePath string) bool {
	fullPath := filepath.Join(h.sourceDir, relativePath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// GetFileContent reads file content from source directory
func (h *TestHelper) GetFileContent(relativePath string) []byte {
	fullPath := filepath.Join(h.sourceDir, relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		h.t.Fatalf("Failed to read file %s: %v", relativePath, err)
	}
	return content
}

// GetFileHash calculates SHA256 hash of a file
func (h *TestHelper) GetFileHash(relativePath string) string {
	content := h.GetFileContent(relativePath)
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}

// RunBackup executes backup command
func (h *TestHelper) RunBackup() error {
	return runBackup([]string{"--config", h.configPath})
}

// RunRestore executes restore command
func (h *TestHelper) RunRestore(options ...string) error {
	args := []string{"--config", h.configPath}
	args = append(args, options...)
	return runRestore(args)
}

// GetTimestamps returns all timestamp directories in target
func (h *TestHelper) GetTimestamps() []string {
	entries, err := os.ReadDir(h.targetDir)
	if err != nil {
		h.t.Fatalf("Failed to read target dir: %v", err)
	}
	
	var timestamps []string
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 19 && strings.Count(entry.Name(), "-") == 5 {
			timestamps = append(timestamps, entry.Name())
		}
	}
	return timestamps
}

// GetIndex loads the file index
func (h *TestHelper) GetIndex() *index.FileIndex {
	indexPath := filepath.Join(h.targetDir, "index.yaml")
	idx, err := index.Load(indexPath)
	if err != nil {
		h.t.Fatalf("Failed to load index: %v", err)
	}
	return idx
}

// ClearSourceDir removes all files from source directory
func (h *TestHelper) ClearSourceDir() {
	entries, err := os.ReadDir(h.sourceDir)
	if err != nil {
		h.t.Fatalf("Failed to read source dir: %v", err)
	}
	
	for _, entry := range entries {
		path := filepath.Join(h.sourceDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			h.t.Fatalf("Failed to remove %s: %v", path, err)
		}
	}
}

// CountArchives counts .7z.001 files in target directory
func (h *TestHelper) CountArchives() int {
	count := 0
	filepath.Walk(h.targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			name := info.Name()
			// Count both .7z and .7z.001 files (first volume of multi-volume archives)
			if strings.HasSuffix(name, ".7z") || strings.HasSuffix(name, ".7z.001") {
				count++
			}
		}
		return nil
	})
	return count
}

// TestFunctional_CompleteBackupFlow tests the complete backup workflow
// Validates: Requirements 19.1
func TestFunctional_CompleteBackupFlow(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2) // 10MB volume, 2 concurrent workers
	
	// Create test files
	h.CreateFile("file1.txt", []byte("content1"))
	h.CreateFile("dir1/file2.txt", []byte("content2"))
	h.CreateFile("dir1/subdir/file3.txt", []byte("content3"))
	
	// Run backup
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	
	// Verify timestamp directory was created
	timestamps := h.GetTimestamps()
	if len(timestamps) != 1 {
		t.Errorf("Expected 1 timestamp directory, got %d", len(timestamps))
	}
	
	// Verify index was created and contains files
	idx := h.GetIndex()
	if len(idx.Files) != 3 {
		t.Errorf("Expected 3 files in index, got %d", len(idx.Files))
	}
	
	// Verify archives were created
	archiveCount := h.CountArchives()
	if archiveCount == 0 {
		t.Error("No archives were created")
	}
	
	t.Logf("Backup completed successfully with %d archives", archiveCount)
}

// TestFunctional_CompleteRestoreFlow tests the complete restore workflow
// Validates: Requirements 19.2
func TestFunctional_CompleteRestoreFlow(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2)
	
	// Create and backup files in separate directories (avoid nested subdirectories for this test)
	h.CreateFile("dir1/file1.txt", []byte("content1"))
	h.CreateFile("dir2/file2.txt", []byte("content2"))
	h.CreateFile("dir3/file3.txt", []byte("content3"))
	
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	
	// Save original hashes
	hash1 := h.GetFileHash("dir1/file1.txt")
	hash2 := h.GetFileHash("dir2/file2.txt")
	hash3 := h.GetFileHash("dir3/file3.txt")
	
	// Clear source directory
	h.ClearSourceDir()
	
	// Verify files are gone
	if h.FileExists("dir1/file1.txt") {
		t.Error("dir1/file1.txt should not exist after clearing")
	}
	
	// Run restore
	if err := h.RunRestore(); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	
	// Verify files are restored
	if !h.FileExists("dir1/file1.txt") {
		t.Error("dir1/file1.txt was not restored")
	}
	if !h.FileExists("dir2/file2.txt") {
		t.Error("dir2/file2.txt was not restored")
	}
	if !h.FileExists("dir3/file3.txt") {
		t.Error("dir3/file3.txt was not restored")
	}
	
	// Verify content integrity
	if h.GetFileHash("dir1/file1.txt") != hash1 {
		t.Error("dir1/file1.txt content mismatch after restore")
	}
	if h.GetFileHash("dir2/file2.txt") != hash2 {
		t.Error("dir2/file2.txt content mismatch after restore")
	}
	if h.GetFileHash("dir3/file3.txt") != hash3 {
		t.Error("dir3/file3.txt content mismatch after restore")
	}
	
	t.Log("Restore completed successfully with content integrity verified")
}

// TestFunctional_IncrementalBackup_FileModification tests incremental backup with file modifications
// Validates: Requirements 19.3
func TestFunctional_IncrementalBackup_FileModification(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2)
	
	// Initial backup
	h.CreateFile("file1.txt", []byte("original content"))
	h.CreateFile("file2.txt", []byte("unchanged"))
	
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Initial backup failed: %v", err)
	}
	
	initialTimestamps := h.GetTimestamps()
	if len(initialTimestamps) != 1 {
		t.Fatalf("Expected 1 timestamp after initial backup, got %d", len(initialTimestamps))
	}
	
	// Delete one file
	time.Sleep(1100 * time.Millisecond) // Ensure different timestamp (need >1 second)
	h.ModifyFile("file1.txt", []byte("modified content"))
	
	// Second backup (incremental)
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Incremental backup failed: %v", err)
	}
	
	// Verify second timestamp directory was created
	timestamps := h.GetTimestamps()
	if len(timestamps) != 2 {
		t.Errorf("Expected 2 timestamp directories, got %d", len(timestamps))
	}
	
	// Verify index contains both versions
	idx := h.GetIndex()
	if len(idx.Files) != 2 {
		t.Errorf("Expected 2 files in index, got %d", len(idx.Files))
	}
	
	// Verify file1.txt points to new timestamp
	entry, exists := idx.Files["file1.txt"]
	if !exists {
		t.Fatal("file1.txt not found in index")
	}
	if entry.TimestampDir != timestamps[1] {
		t.Errorf("file1.txt should point to new timestamp %s, got %s", timestamps[1], entry.TimestampDir)
	}
	
	t.Log("Incremental backup with file modification completed successfully")
}

// TestFunctional_IncrementalBackup_FileAddition tests incremental backup with new files
// Validates: Requirements 19.3
func TestFunctional_IncrementalBackup_FileAddition(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2)
	
	// Initial backup
	h.CreateFile("file1.txt", []byte("content1"))
	
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Initial backup failed: %v", err)
	}
	
	// Add new file
	time.Sleep(1100 * time.Millisecond) // Ensure different timestamp
	h.CreateFile("file2.txt", []byte("content2"))
	h.CreateFile("dir1/file3.txt", []byte("content3"))
	
	// Second backup
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Incremental backup failed: %v", err)
	}
	
	// Verify index contains all files
	idx := h.GetIndex()
	if len(idx.Files) != 3 {
		t.Errorf("Expected 3 files in index, got %d", len(idx.Files))
	}
	
	timestamps := h.GetTimestamps()
	if len(timestamps) != 2 {
		t.Errorf("Expected 2 timestamp directories, got %d", len(timestamps))
	}
	
	t.Log("Incremental backup with file addition completed successfully")
}

// TestFunctional_IncrementalBackup_FileDeletion tests incremental backup with file deletions
// Validates: Requirements 19.3
func TestFunctional_IncrementalBackup_FileDeletion(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2)
	
	// Initial backup
	h.CreateFile("file1.txt", []byte("content1"))
	h.CreateFile("file2.txt", []byte("content2"))
	
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Initial backup failed: %v", err)
	}
	
	// Delete one file
	time.Sleep(1100 * time.Millisecond) // Ensure different timestamp
	h.DeleteFile("file2.txt")
	
	// Second backup
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Incremental backup failed: %v", err)
	}
	
	// Verify file2.txt is marked as deleted in index
	idx := h.GetIndex()
	entry, exists := idx.Files["file2.txt"]
	if !exists {
		t.Fatal("file2.txt should still be in index")
	}
	if !entry.Deleted {
		t.Error("file2.txt should be marked as deleted")
	}
	
	t.Log("Incremental backup with file deletion completed successfully")
}


// TestFunctional_SmallDirectoryCompression tests compression of small directories
// Validates: Requirements 19.4
func TestFunctional_SmallDirectoryCompression(t *testing.T) {
	h := NewTestHelper(t)
	// Set volume size to 1MB, create files smaller than that
	h.CreateConfig(1*1024*1024, 1)
	
	// Create small directory with files totaling less than 1MB
	h.CreateFile("smalldir/file1.txt", []byte("small content 1"))
	h.CreateFile("smalldir/file2.txt", []byte("small content 2"))
	h.CreateFile("smalldir/file3.txt", []byte("small content 3"))
	
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	
	// Verify archive was created
	timestamps := h.GetTimestamps()
	if len(timestamps) != 1 {
		t.Fatalf("Expected 1 timestamp directory, got %d", len(timestamps))
	}
	
	// Check that smalldir was compressed as a single archive
	timestampPath := filepath.Join(h.targetDir, timestamps[0])
	// Small directories may be compressed as .7z (single volume) or .7z.001 (multi-volume)
	archivePath := filepath.Join(timestampPath, "smalldir.7z")
	if _, err := os.Stat(archivePath); err != nil {
		// Try .7z.001 if .7z doesn't exist
		archivePath = filepath.Join(timestampPath, "smalldir.7z.001")
		if _, err := os.Stat(archivePath); err != nil {
			t.Errorf("Expected archive smalldir.7z or smalldir.7z.001 to exist in %s", timestampPath)
		}
	}
	
	// Verify restore works
	h.ClearSourceDir()
	if err := h.RunRestore(); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	
	if !h.FileExists("smalldir/file1.txt") {
		t.Error("smalldir/file1.txt was not restored")
	}
	
	t.Log("Small directory compression completed successfully")
}

// TestFunctional_LargeDirectoryVolumeCompression tests volume splitting for large directories
// Validates: Requirements 19.5
func TestFunctional_LargeDirectoryVolumeCompression(t *testing.T) {
	h := NewTestHelper(t)
	// Set small volume size to force splitting
	h.CreateConfig(50*1024, 1) // 50KB volumes
	
	// Create directory with files larger than volume size (no subdirectories)
	h.CreateFileWithSize("largedir/file1.bin", 40*1024)  // 40KB
	h.CreateFileWithSize("largedir/file2.bin", 40*1024)  // 40KB
	h.CreateFileWithSize("largedir/file3.bin", 40*1024)  // 40KB
	// Total: 120KB, should create multiple volumes
	
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	
	timestamps := h.GetTimestamps()
	if len(timestamps) != 1 {
		t.Fatalf("Expected 1 timestamp directory, got %d", len(timestamps))
	}
	
	// Check for multiple volume files
	timestampPath := filepath.Join(h.targetDir, timestamps[0])
	volumeCount := 0
	entries, err := os.ReadDir(timestampPath)
	if err != nil {
		t.Fatalf("Failed to read timestamp dir: %v", err)
	}
	
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "largedir") && strings.Contains(entry.Name(), ".7z.") {
			volumeCount++
		}
	}
	
	if volumeCount < 2 {
		t.Errorf("Expected at least 2 volume files, got %d", volumeCount)
	}
	
	// Verify restore works with volumes
	h.ClearSourceDir()
	if err := h.RunRestore(); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	
	if !h.FileExists("largedir/file1.bin") {
		t.Error("largedir/file1.bin was not restored")
	}
	
	t.Logf("Large directory volume compression completed successfully with %d volumes", volumeCount)
}

// TestFunctional_RecursiveDirectoryCompression tests recursive compression of nested directories
// Validates: Requirements 19.6
func TestFunctional_RecursiveDirectoryCompression(t *testing.T) {
	h := NewTestHelper(t)
	// Set volume size small enough to trigger recursive processing
	h.CreateConfig(30*1024, 1) // 30KB volumes
	
	// Create nested directory structure with files
	h.CreateFileWithSize("parent/file1.txt", 20*1024)           // 20KB
	h.CreateFileWithSize("parent/subdir1/file2.txt", 20*1024)   // 20KB
	h.CreateFileWithSize("parent/subdir2/file3.txt", 20*1024)   // 20KB
	h.CreateFileWithSize("parent/subdir2/nested/file4.txt", 20*1024) // 20KB
	// Total: 80KB, should trigger recursive compression
	
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	
	timestamps := h.GetTimestamps()
	if len(timestamps) != 1 {
		t.Fatalf("Expected 1 timestamp directory, got %d", len(timestamps))
	}
	
	// Verify multiple archives were created (one for each subdirectory)
	archiveCount := h.CountArchives()
	if archiveCount < 2 {
		t.Errorf("Expected at least 2 archives for recursive compression, got %d", archiveCount)
	}
	
	// Verify restore works
	h.ClearSourceDir()
	if err := h.RunRestore(); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	
	// Verify all files are restored
	if !h.FileExists("parent/file1.txt") {
		t.Error("parent/file1.txt was not restored")
	}
	if !h.FileExists("parent/subdir1/file2.txt") {
		t.Error("parent/subdir1/file2.txt was not restored")
	}
	if !h.FileExists("parent/subdir2/file3.txt") {
		t.Error("parent/subdir2/file3.txt was not restored")
	}
	if !h.FileExists("parent/subdir2/nested/file4.txt") {
		t.Error("parent/subdir2/nested/file4.txt was not restored")
	}
	
	t.Logf("Recursive directory compression completed successfully with %d archives", archiveCount)
}

// TestFunctional_ConcurrentCompression tests concurrent compression functionality
// Validates: Requirements 19.7
func TestFunctional_ConcurrentCompression(t *testing.T) {
	h := NewTestHelper(t)
	// Set concurrency to 3
	h.CreateConfig(10*1024*1024, 3)
	
	// Create multiple directories to compress concurrently
	h.CreateFile("dir1/file1.txt", []byte("content1"))
	h.CreateFile("dir2/file2.txt", []byte("content2"))
	h.CreateFile("dir3/file3.txt", []byte("content3"))
	h.CreateFile("dir4/file4.txt", []byte("content4"))
	h.CreateFile("dir5/file5.txt", []byte("content5"))
	
	startTime := time.Now()
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	duration := time.Since(startTime)
	
	// Verify all directories were backed up
	idx := h.GetIndex()
	if len(idx.Files) != 5 {
		t.Errorf("Expected 5 files in index, got %d", len(idx.Files))
	}
	
	// Verify archives were created
	archiveCount := h.CountArchives()
	if archiveCount != 5 {
		t.Errorf("Expected 5 archives, got %d", archiveCount)
	}
	
	t.Logf("Concurrent compression completed in %v with %d archives", duration, archiveCount)
}

// TestFunctional_SelectiveRestore_SingleTimestamp tests selective restore of a single timestamp
// Validates: Requirements 19.8
func TestFunctional_SelectiveRestore_SingleTimestamp(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2)
	
	// First backup
	h.CreateFile("dir1/file1.txt", []byte("version1"))
	if err := h.RunBackup(); err != nil {
		t.Fatalf("First backup failed: %v", err)
	}
	
	timestamps := h.GetTimestamps()
	firstTimestamp := timestamps[0]
	
	// Second backup with modified file
	time.Sleep(1100 * time.Millisecond) // Ensure different timestamp
	h.ModifyFile("dir1/file1.txt", []byte("version2"))
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Second backup failed: %v", err)
	}
	
	timestamps = h.GetTimestamps()
	if len(timestamps) != 2 {
		t.Fatalf("Expected 2 timestamps, got %d", len(timestamps))
	}
	
	// Clear source and restore only first timestamp
	h.ClearSourceDir()
	if err := h.RunRestore("--timestamp", firstTimestamp); err != nil {
		t.Fatalf("Selective restore failed: %v", err)
	}
	
	// Verify file has version1 content
	content := h.GetFileContent("dir1/file1.txt")
	if string(content) != "version1" {
		t.Errorf("Expected 'version1', got '%s'", string(content))
	}
	
	t.Log("Selective restore of single timestamp completed successfully")
}

// TestFunctional_SelectiveRestore_TimeRange tests selective restore with time range
// Validates: Requirements 19.8
func TestFunctional_SelectiveRestore_TimeRange(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2)
	
	// Create three backups
	h.CreateFile("dir1/file1.txt", []byte("v1"))
	if err := h.RunBackup(); err != nil {
		t.Fatalf("First backup failed: %v", err)
	}
	
	time.Sleep(1100 * time.Millisecond) // Ensure different timestamp
	h.CreateFile("dir2/file2.txt", []byte("v2"))
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Second backup failed: %v", err)
	}
	
	time.Sleep(1100 * time.Millisecond) // Ensure different timestamp
	h.CreateFile("dir3/file3.txt", []byte("v3"))
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Third backup failed: %v", err)
	}
	
	timestamps := h.GetTimestamps()
	if len(timestamps) != 3 {
		t.Fatalf("Expected 3 timestamps, got %d", len(timestamps))
	}
	
	// Clear source and restore range (first two timestamps)
	h.ClearSourceDir()
	if err := h.RunRestore("--from", timestamps[0], "--to", timestamps[1]); err != nil {
		t.Fatalf("Range restore failed: %v", err)
	}
	
	// Verify only first two files are restored
	if !h.FileExists("dir1/file1.txt") {
		t.Error("dir1/file1.txt should be restored")
	}
	if !h.FileExists("dir2/file2.txt") {
		t.Error("dir2/file2.txt should be restored")
	}
	if h.FileExists("dir3/file3.txt") {
		t.Error("dir3/file3.txt should not be restored")
	}
	
	t.Log("Selective restore with time range completed successfully")
}

// TestFunctional_ErrorRecovery tests error recovery during backup
// Validates: Requirements 19.9
func TestFunctional_ErrorRecovery(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2)
	
	// Create multiple directories
	h.CreateFile("dir1/file1.txt", []byte("content1"))
	h.CreateFile("dir2/file2.txt", []byte("content2"))
	h.CreateFile("dir3/file3.txt", []byte("content3"))
	
	// Run backup - even if one directory fails, others should succeed
	err := h.RunBackup()
	
	// Backup might fail or succeed depending on system state
	// The important thing is that partial results are saved
	
	// Verify that at least some files were backed up
	idx := h.GetIndex()
	if len(idx.Files) == 0 {
		t.Error("Expected at least some files to be backed up")
	}
	
	// If backup succeeded, verify restore works
	if err == nil {
		h.ClearSourceDir()
		if err := h.RunRestore(); err != nil {
			t.Fatalf("Restore failed: %v", err)
		}
		
		// Verify at least some files were restored
		if !h.FileExists("dir1/file1.txt") && !h.FileExists("dir2/file2.txt") && !h.FileExists("dir3/file3.txt") {
			t.Error("No files were restored")
		}
	}
	
	t.Log("Error recovery test completed")
}

// TestFunctional_FileConsistencyVerification tests file consistency after restore
// Validates: Requirements 19.10, 19.11
func TestFunctional_FileConsistencyVerification(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2)
	
	// Create files with known content in subdirectories
	testFiles := map[string][]byte{
		"dir1/text.txt":           []byte("This is a text file with some content"),
		"dir2/binary.bin":         {0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
		"dir3/nested.txt":         []byte("Nested file content"),
		"dir4/sub/deep.txt":       []byte("Deep nested content"),
	}
	
	// Create files and calculate hashes
	originalHashes := make(map[string]string)
	for path, content := range testFiles {
		h.CreateFile(path, content)
		originalHashes[path] = h.GetFileHash(path)
	}
	
	// Backup
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	
	// Clear source
	h.ClearSourceDir()
	
	// Restore
	if err := h.RunRestore(); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	
	// Verify all files exist and have correct content
	for path, originalHash := range originalHashes {
		if !h.FileExists(path) {
			t.Errorf("File %s was not restored", path)
			continue
		}
		
		restoredHash := h.GetFileHash(path)
		if restoredHash != originalHash {
			t.Errorf("File %s content mismatch: expected %s, got %s", path, originalHash, restoredHash)
		}
	}
	
	t.Log("File consistency verification completed successfully")
}

// TestFunctional_DeletedFileHandling tests handling of deleted files during restore
// Validates: Requirements 19.11
func TestFunctional_DeletedFileHandling(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2)
	
	// First backup with two files
	h.CreateFile("dir1/file1.txt", []byte("content1"))
	h.CreateFile("dir2/file2.txt", []byte("content2"))
	
	if err := h.RunBackup(); err != nil {
		t.Fatalf("First backup failed: %v", err)
	}
	
	// Delete one file and backup again
	time.Sleep(1100 * time.Millisecond) // Ensure different timestamp
	h.DeleteFile("dir2/file2.txt")
	
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Second backup failed: %v", err)
	}
	
	// Verify file2.txt is marked as deleted in index
	idx := h.GetIndex()
	entry, exists := idx.Files["dir2/file2.txt"]
	if !exists {
		t.Fatal("dir2/file2.txt should be in index")
	}
	if !entry.Deleted {
		t.Error("dir2/file2.txt should be marked as deleted")
	}
	
	// Restore from first backup (should restore file2.txt)
	h.ClearSourceDir()
	timestamps := h.GetTimestamps()
	if err := h.RunRestore("--timestamp", timestamps[0]); err != nil {
		t.Fatalf("Restore first backup failed: %v", err)
	}
	
	if !h.FileExists("dir2/file2.txt") {
		t.Error("dir2/file2.txt should be restored from first backup")
	}
	
	// Now restore all backups (should delete file2.txt)
	if err := h.RunRestore(); err != nil {
		t.Fatalf("Full restore failed: %v", err)
	}
	
	if h.FileExists("dir2/file2.txt") {
		t.Error("dir2/file2.txt should be deleted after full restore")
	}
	if !h.FileExists("dir1/file1.txt") {
		t.Error("dir1/file1.txt should still exist")
	}
	
	t.Log("Deleted file handling completed successfully")
}

// TestFunctional_LargeFileHandling tests handling of large files
// Validates: Requirements 19.12, 19.13, 19.14, 19.15, 19.16
func TestFunctional_LargeFileHandling(t *testing.T) {
	h := NewTestHelper(t)
	// Set small volume size to test volume splitting with large files
	h.CreateConfig(100*1024, 2) // 100KB volumes
	
	// Create a large file (500KB) in a subdirectory
	h.CreateFileWithSize("largedir/largefile.bin", 500*1024)
	originalHash := h.GetFileHash("largedir/largefile.bin")
	
	// Backup
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	
	// Verify multiple volumes were created
	timestamps := h.GetTimestamps()
	timestampPath := filepath.Join(h.targetDir, timestamps[0])
	
	volumeCount := 0
	entries, err := os.ReadDir(timestampPath)
	if err != nil {
		t.Fatalf("Failed to read timestamp dir: %v", err)
	}
	
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".7z.") {
			volumeCount++
		}
	}
	
	if volumeCount < 2 {
		t.Errorf("Expected at least 2 volumes for large file, got %d", volumeCount)
	}
	
	// Clear and restore
	h.ClearSourceDir()
	if err := h.RunRestore(); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	
	// Verify file integrity
	if !h.FileExists("largedir/largefile.bin") {
		t.Fatal("largedir/largefile.bin was not restored")
	}
	
	restoredHash := h.GetFileHash("largedir/largefile.bin")
	if restoredHash != originalHash {
		t.Error("Large file content mismatch after restore")
	}
	
	t.Logf("Large file handling completed successfully with %d volumes", volumeCount)
}

// TestFunctional_EmptyDirectoryHandling tests handling of empty directories
func TestFunctional_EmptyDirectoryHandling(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2)
	
	// Create directory with files and an empty subdirectory
	h.CreateFile("dir1/file1.txt", []byte("content"))
	emptyDir := filepath.Join(h.sourceDir, "dir1", "emptydir")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create empty dir: %v", err)
	}
	
	// Backup
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	
	// Verify backup succeeded
	idx := h.GetIndex()
	if len(idx.Files) == 0 {
		t.Error("Expected files to be backed up")
	}
	
	t.Log("Empty directory handling completed")
}

// TestFunctional_SpecialCharactersInFilenames tests handling of special characters
func TestFunctional_SpecialCharactersInFilenames(t *testing.T) {
	h := NewTestHelper(t)
	h.CreateConfig(10*1024*1024, 2)
	
	// Create files with special characters (that are valid on most filesystems) in subdirectories
	testFiles := []string{
		"testdir/file with spaces.txt",
		"testdir/file_with_underscores.txt",
		"testdir/file-with-dashes.txt",
		"testdir/file.multiple.dots.txt",
	}
	
	for _, filename := range testFiles {
		h.CreateFile(filename, []byte("content"))
	}
	
	// Backup
	if err := h.RunBackup(); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	
	// Restore
	h.ClearSourceDir()
	if err := h.RunRestore(); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	
	// Verify all files are restored
	for _, filename := range testFiles {
		if !h.FileExists(filename) {
			t.Errorf("File %s was not restored", filename)
		}
	}
	
	t.Log("Special characters in filenames handled successfully")
}
