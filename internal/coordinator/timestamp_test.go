package coordinator

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewTimestampManager(t *testing.T) {
	targetDir := "/test/target"
	tm := NewTimestampManager(targetDir)
	
	if tm == nil {
		t.Fatal("NewTimestampManager returned nil")
	}
	
	if tm.targetDir != targetDir {
		t.Errorf("Expected targetDir %s, got %s", targetDir, tm.targetDir)
	}
}

func TestCreateTimestampDir(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timestamp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	tm := NewTimestampManager(tempDir)
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	
	// Test creating timestamp directory
	timestamp, err := tm.CreateTimestampDir(testTime)
	if err != nil {
		t.Fatalf("CreateTimestampDir failed: %v", err)
	}
	
	expectedTimestamp := "2024-01-15-10-30-45"
	if timestamp != expectedTimestamp {
		t.Errorf("Expected timestamp %s, got %s", expectedTimestamp, timestamp)
	}
	
	// Verify directory was created
	timestampPath := filepath.Join(tempDir, timestamp)
	info, err := os.Stat(timestampPath)
	if err != nil {
		t.Fatalf("Timestamp directory was not created: %v", err)
	}
	
	if !info.IsDir() {
		t.Error("Timestamp path is not a directory")
	}
}

func TestCreateTimestampDir_Conflict(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timestamp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	tm := NewTimestampManager(tempDir)
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	
	// Create timestamp directory first time
	_, err = tm.CreateTimestampDir(testTime)
	if err != nil {
		t.Fatalf("First CreateTimestampDir failed: %v", err)
	}
	
	// Try to create the same timestamp directory again (should fail)
	_, err = tm.CreateTimestampDir(testTime)
	if err == nil {
		t.Error("Expected error when creating duplicate timestamp directory, got nil")
	}
	
	if !errors.Is(err, ErrTimestampDirExists) {
		t.Errorf("Expected ErrTimestampDirExists, got %v", err)
	}
}

func TestGetTimestampPath(t *testing.T) {
	targetDir := "/test/target"
	tm := NewTimestampManager(targetDir)
	timestamp := "2024-01-15-10-30-45"
	
	path := tm.GetTimestampPath(timestamp)
	expected := filepath.Join(targetDir, timestamp)
	
	if path != expected {
		t.Errorf("Expected path %s, got %s", expected, path)
	}
}

func TestCreateRelativePathInTimestamp(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timestamp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	tm := NewTimestampManager(tempDir)
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	
	// Create timestamp directory
	timestamp, err := tm.CreateTimestampDir(testTime)
	if err != nil {
		t.Fatalf("CreateTimestampDir failed: %v", err)
	}
	
	// Test creating relative path structure
	sourcePath := "subdir1/subdir2/file.txt"
	targetPath, err := tm.CreateRelativePathInTimestamp(timestamp, sourcePath)
	if err != nil {
		t.Fatalf("CreateRelativePathInTimestamp failed: %v", err)
	}
	
	// Verify the returned path is correct
	expectedPath := filepath.Join(tempDir, timestamp, sourcePath)
	if targetPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, targetPath)
	}
	
	// Verify the directory structure was created
	expectedDir := filepath.Join(tempDir, timestamp, "subdir1", "subdir2")
	info, err := os.Stat(expectedDir)
	if err != nil {
		t.Fatalf("Directory structure was not created: %v", err)
	}
	
	if !info.IsDir() {
		t.Error("Path is not a directory")
	}
}

func TestCreateRelativePathInTimestamp_RootFile(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timestamp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	tm := NewTimestampManager(tempDir)
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	
	// Create timestamp directory
	timestamp, err := tm.CreateTimestampDir(testTime)
	if err != nil {
		t.Fatalf("CreateTimestampDir failed: %v", err)
	}
	
	// Test creating relative path for a root-level file
	sourcePath := "file.txt"
	targetPath, err := tm.CreateRelativePathInTimestamp(timestamp, sourcePath)
	if err != nil {
		t.Fatalf("CreateRelativePathInTimestamp failed: %v", err)
	}
	
	// Verify the returned path is correct
	expectedPath := filepath.Join(tempDir, timestamp, sourcePath)
	if targetPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, targetPath)
	}
}

func TestCreateRelativePathInTimestamp_MultipleFiles(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timestamp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	tm := NewTimestampManager(tempDir)
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	
	// Create timestamp directory
	timestamp, err := tm.CreateTimestampDir(testTime)
	if err != nil {
		t.Fatalf("CreateTimestampDir failed: %v", err)
	}
	
	// Test creating multiple relative paths
	testPaths := []string{
		"dir1/file1.txt",
		"dir1/file2.txt",
		"dir2/subdir/file3.txt",
		"dir3/file4.txt",
	}
	
	for _, sourcePath := range testPaths {
		targetPath, err := tm.CreateRelativePathInTimestamp(timestamp, sourcePath)
		if err != nil {
			t.Fatalf("CreateRelativePathInTimestamp failed for %s: %v", sourcePath, err)
		}
		
		expectedPath := filepath.Join(tempDir, timestamp, sourcePath)
		if targetPath != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, targetPath)
		}
		
		// Verify directory exists
		dir := filepath.Dir(targetPath)
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Directory was not created for %s: %v", sourcePath, err)
		}
		
		if !info.IsDir() {
			t.Errorf("Path is not a directory for %s", sourcePath)
		}
	}
}

func TestTimestampExists(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timestamp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	tm := NewTimestampManager(tempDir)
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	
	// Test non-existent timestamp
	timestamp := "2024-01-15-10-30-45"
	if tm.TimestampExists(timestamp) {
		t.Error("TimestampExists returned true for non-existent timestamp")
	}
	
	// Create timestamp directory
	createdTimestamp, err := tm.CreateTimestampDir(testTime)
	if err != nil {
		t.Fatalf("CreateTimestampDir failed: %v", err)
	}
	
	// Test existing timestamp
	if !tm.TimestampExists(createdTimestamp) {
		t.Error("TimestampExists returned false for existing timestamp")
	}
}

func TestTimestampFormat(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "timestamp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	tm := NewTimestampManager(tempDir)
	
	// Test various times to ensure format is correct
	testCases := []struct {
		time     time.Time
		expected string
	}{
		{
			time:     time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
			expected: "2024-01-15-10-30-45",
		},
		{
			time:     time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
			expected: "2023-12-31-23-59-59",
		},
		{
			time:     time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), // Leap year
			expected: "2024-02-29-00-00-00",
		},
	}
	
	for _, tc := range testCases {
		timestamp, err := tm.CreateTimestampDir(tc.time)
		if err != nil {
			t.Fatalf("CreateTimestampDir failed for %v: %v", tc.time, err)
		}
		
		if timestamp != tc.expected {
			t.Errorf("Expected timestamp %s for time %v, got %s", tc.expected, tc.time, timestamp)
		}
		
		// Clean up for next iteration
		os.RemoveAll(filepath.Join(tempDir, timestamp))
	}
}
