package sevenzip

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestDetect tests the 7zip tool detection
func TestDetect(t *testing.T) {
	w := NewWrapper()

	cmd, err := w.Detect()

	// Check if 7zip is available on the system
	_, err7z := exec.LookPath("7z")
	_, err7za := exec.LookPath("7za")

	if err7z != nil && err7za != nil {
		// 7zip not available, should return error
		if err == nil {
			t.Error("Expected error when 7zip is not available, got nil")
		}
		if err != Err7zipNotFound {
			t.Errorf("Expected Err7zipNotFound, got %v", err)
		}
		return
	}

	// 7zip is available
	if err != nil {
		t.Errorf("Expected no error when 7zip is available, got %v", err)
	}

	if cmd != "7z" && cmd != "7za" {
		t.Errorf("Expected command to be '7z' or '7za', got '%s'", cmd)
	}
}

// TestCompressWithout7zip tests compress when 7zip is not detected
func TestCompressWithout7zip(t *testing.T) {
	w := &wrapper{command: "nonexistent7z"} // Use non-existent command

	params := CompressParams{
		Sources:    []string{"test.txt"},
		Output:     "test.7z",
		Password:   "password",
		VolumeSize: 0,
	}

	err := w.Compress(params)
	if err == nil {
		t.Error("Expected error when 7zip command is invalid, got nil")
	}
}

// TestExtractWithout7zip tests extract when 7zip is not detected
func TestExtractWithout7zip(t *testing.T) {
	w := &wrapper{command: "nonexistent7z"} // Use non-existent command

	params := ExtractParams{
		Archive:   "test.7z",
		OutputDir: "output",
		Password:  "password",
	}

	err := w.Extract(params)
	if err == nil {
		t.Error("Expected error when 7zip command is invalid, got nil")
	}
}

// TestBuildCommand tests command building logic
func TestBuildCommand(t *testing.T) {
	w := &wrapper{command: "7z"}

	tests := []struct {
		name      string
		operation string
		params    interface{}
		expected  []string
	}{
		{
			name:      "Compress without volume",
			operation: "compress",
			params: CompressParams{
				Sources:          []string{"file1.txt", "file2.txt"},
				Output:           "output.7z",
				Password:         "secret",
				VolumeSize:       0,
				CompressionLevel: 5,
			},
			expected: []string{"a", "-t7z", "-mx=5", "-mhe=on", "-psecret", "output.7z", "file1.txt", "file2.txt"},
		},
		{
			name:      "Compress with volume",
			operation: "compress",
			params: CompressParams{
				Sources:          []string{"dir"},
				Output:           "output.7z",
				Password:         "secret",
				VolumeSize:       1048576,
				CompressionLevel: 5,
			},
			expected: []string{"a", "-t7z", "-mx=5", "-mhe=on", "-psecret", "-v1048576b", "output.7z", "dir"},
		},
		{
			name:      "Extract",
			operation: "extract",
			params: ExtractParams{
				Archive:   "archive.7z",
				OutputDir: "/output/dir",
				Password:  "secret",
			},
			expected: []string{"x", "-psecret", "-o/output/dir", "-y", "archive.7z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := w.buildCommand(tt.operation, tt.params)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d arguments, got %d", len(tt.expected), len(result))
				t.Errorf("Expected: %v", tt.expected)
				t.Errorf("Got: %v", result)
				return
			}

			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("Argument %d: expected '%s', got '%s'", i, tt.expected[i], arg)
				}
			}
		})
	}
}

// TestCompressIntegration tests actual compression if 7zip is available
func TestCompressIntegration(t *testing.T) {
	// Skip if 7zip is not available
	w := NewWrapper()
	_, err := w.Detect()
	if err != nil {
		t.Skip("7zip not available, skipping integration test")
	}

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compress the file
	outputFile := filepath.Join(tempDir, "test.7z")
	params := CompressParams{
		Sources:    []string{testFile},
		Output:     outputFile,
		Password:   "testpassword",
		VolumeSize: 0,
	}

	if err := w.Compress(params); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Verify output file exists
	// When VolumeSize is 0, 7zip creates a single file without .001 suffix
	if _, err := os.Stat(outputFile); err != nil {
		t.Errorf("Output file not created: %v", err)
	}
}

// TestCompressWithVolume tests compression with volume splitting
func TestCompressWithVolume(t *testing.T) {
	// Skip if 7zip is not available
	w := NewWrapper()
	_, err := w.Detect()
	if err != nil {
		t.Skip("7zip not available, skipping integration test")
	}

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create test file with enough content to trigger volume splitting
	testFile := filepath.Join(tempDir, "largefile.txt")
	// Create a 2KB file
	content := make([]byte, 2048)
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compress with 1KB volume size
	outputFile := filepath.Join(tempDir, "test.7z")
	params := CompressParams{
		Sources:    []string{testFile},
		Output:     outputFile,
		Password:   "testpassword",
		VolumeSize: 1024, // 1KB volumes
	}

	if err := w.Compress(params); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Verify volume files exist (.001, .002, etc.)
	// With volume splitting, 7zip creates files with .001, .002 suffixes
	volume1 := outputFile + ".001"
	if _, err := os.Stat(volume1); err != nil {
		t.Errorf("Volume 1 not created: %v", err)
	}

	// Check if multiple volumes were created
	volume2 := outputFile + ".002"
	if _, err := os.Stat(volume2); err == nil {
		t.Logf("Multiple volumes created (at least 2)")
	}
}

// TestExtractIntegration tests actual extraction if 7zip is available
func TestExtractIntegration(t *testing.T) {
	// Skip if 7zip is not available
	w := NewWrapper()
	_, err := w.Detect()
	if err != nil {
		t.Skip("7zip not available, skipping integration test")
	}

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := []byte("test content for extraction")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compress the file first
	archiveFile := filepath.Join(tempDir, "test.7z")
	compressParams := CompressParams{
		Sources:    []string{testFile},
		Output:     archiveFile,
		Password:   "testpassword",
		VolumeSize: 0,
	}

	if err := w.Compress(compressParams); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Remove original file
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("Failed to remove original file: %v", err)
	}

	// Extract the archive
	extractDir := filepath.Join(tempDir, "extracted")
	// When VolumeSize is 0, 7zip creates a single file without .001 suffix
	extractParams := ExtractParams{
		Archive:   archiveFile,
		OutputDir: extractDir,
		Password:  "testpassword",
	}

	if err := w.Extract(extractParams); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify extracted file exists and has correct content
	extractedFile := filepath.Join(extractDir, "test.txt")
	content, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Errorf("Failed to read extracted file: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Extracted content mismatch: expected '%s', got '%s'", testContent, content)
	}
}

// TestParseOutput tests output parsing
func TestParseOutput(t *testing.T) {
	tests := []struct {
		name     string
		stdout   string
		stderr   string
		expected string
	}{
		{
			name:     "Both stdout and stderr",
			stdout:   "stdout content",
			stderr:   "stderr content",
			expected: "Stdout: stdout content\nStderr: stderr content",
		},
		{
			name:     "Only stdout",
			stdout:   "stdout content",
			stderr:   "",
			expected: "Stdout: stdout content",
		},
		{
			name:     "Only stderr",
			stdout:   "",
			stderr:   "stderr content",
			expected: "Stderr: stderr content",
		},
		{
			name:     "Neither",
			stdout:   "",
			stderr:   "",
			expected: "无输出",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOutput(tt.stdout, tt.stderr)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
