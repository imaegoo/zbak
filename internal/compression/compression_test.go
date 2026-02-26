package compression

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// mockFileSystemService is a mock implementation of FileSystemService for testing
type mockFileSystemService struct {
	calculateDirSizeFunc func(path string) (int64, error)
	hasSubdirsFunc       func(path string) (bool, error)
	listFilesFunc        func(path string) ([]string, error)
	createDirFunc        func(path string) error
}

func (m *mockFileSystemService) CalculateDirSize(path string) (int64, error) {
	if m.calculateDirSizeFunc != nil {
		return m.calculateDirSizeFunc(path)
	}
	return 0, nil
}

func (m *mockFileSystemService) HasSubdirs(path string) (bool, error) {
	if m.hasSubdirsFunc != nil {
		return m.hasSubdirsFunc(path)
	}
	return false, nil
}

func (m *mockFileSystemService) ListFiles(path string) ([]string, error) {
	if m.listFilesFunc != nil {
		return m.listFilesFunc(path)
	}
	return []string{}, nil
}

func (m *mockFileSystemService) CreateDir(path string) error {
	if m.createDirFunc != nil {
		return m.createDirFunc(path)
	}
	return nil
}

// mockSevenZipWrapper is a mock implementation of SevenZipWrapper for testing
type mockSevenZipWrapper struct {
	compressFunc func(params CompressParams) error
	compressCalls []CompressParams
	mu           sync.Mutex
}

func (m *mockSevenZipWrapper) Compress(params CompressParams) error {
	m.mu.Lock()
	m.compressCalls = append(m.compressCalls, params)
	m.mu.Unlock()
	if m.compressFunc != nil {
		return m.compressFunc(params)
	}
	return nil
}

func (m *mockSevenZipWrapper) getCompressCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.compressCalls)
}

func TestDetermineStrategy_SmallDirectory(t *testing.T) {
	tests := []struct {
		name       string
		dirSize    int64
		volumeSize int64
		wantErr    bool
	}{
		{
			name:       "directory much smaller than volume size",
			dirSize:    1024,
			volumeSize: 1024 * 1024,
			wantErr:    false,
		},
		{
			name:       "directory exactly one byte smaller than volume size",
			dirSize:    1023,
			volumeSize: 1024,
			wantErr:    false,
		},
		{
			name:       "empty directory",
			dirSize:    0,
			volumeSize: 1024,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := &mockFileSystemService{
				calculateDirSizeFunc: func(path string) (int64, error) {
					return tt.dirSize, nil
				},
			}

			service := NewService(mockFS, &mockSevenZipWrapper{})
			strategy, err := service.DetermineStrategy("/test/path", tt.volumeSize)

			if (err != nil) != tt.wantErr {
				t.Errorf("DetermineStrategy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && strategy != StrategySmallDir {
				t.Errorf("DetermineStrategy() = %v, want %v", strategy, StrategySmallDir)
			}
		})
	}
}

func TestDetermineStrategy_LargeDirectoryNoSubdirs(t *testing.T) {
	tests := []struct {
		name       string
		dirSize    int64
		volumeSize int64
		hasSubdirs bool
		wantErr    bool
	}{
		{
			name:       "directory equal to volume size, no subdirs",
			dirSize:    1024,
			volumeSize: 1024,
			hasSubdirs: false,
			wantErr:    false,
		},
		{
			name:       "directory larger than volume size, no subdirs",
			dirSize:    2048,
			volumeSize: 1024,
			hasSubdirs: false,
			wantErr:    false,
		},
		{
			name:       "directory much larger than volume size, no subdirs",
			dirSize:    1024 * 1024 * 100,
			volumeSize: 1024 * 1024,
			hasSubdirs: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := &mockFileSystemService{
				calculateDirSizeFunc: func(path string) (int64, error) {
					return tt.dirSize, nil
				},
				hasSubdirsFunc: func(path string) (bool, error) {
					return tt.hasSubdirs, nil
				},
			}

			service := NewService(mockFS, &mockSevenZipWrapper{})
			strategy, err := service.DetermineStrategy("/test/path", tt.volumeSize)

			if (err != nil) != tt.wantErr {
				t.Errorf("DetermineStrategy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && strategy != StrategyLargeNoSubdir {
				t.Errorf("DetermineStrategy() = %v, want %v", strategy, StrategyLargeNoSubdir)
			}
		})
	}
}

func TestDetermineStrategy_LargeDirectoryWithSubdirs(t *testing.T) {
	tests := []struct {
		name       string
		dirSize    int64
		volumeSize int64
		hasSubdirs bool
		wantErr    bool
	}{
		{
			name:       "directory equal to volume size, has subdirs",
			dirSize:    1024,
			volumeSize: 1024,
			hasSubdirs: true,
			wantErr:    false,
		},
		{
			name:       "directory larger than volume size, has subdirs",
			dirSize:    2048,
			volumeSize: 1024,
			hasSubdirs: true,
			wantErr:    false,
		},
		{
			name:       "directory much larger than volume size, has subdirs",
			dirSize:    1024 * 1024 * 100,
			volumeSize: 1024 * 1024,
			hasSubdirs: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := &mockFileSystemService{
				calculateDirSizeFunc: func(path string) (int64, error) {
					return tt.dirSize, nil
				},
				hasSubdirsFunc: func(path string) (bool, error) {
					return tt.hasSubdirs, nil
				},
			}

			service := NewService(mockFS, &mockSevenZipWrapper{})
			strategy, err := service.DetermineStrategy("/test/path", tt.volumeSize)

			if (err != nil) != tt.wantErr {
				t.Errorf("DetermineStrategy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && strategy != StrategyLargeWithSubdir {
				t.Errorf("DetermineStrategy() = %v, want %v", strategy, StrategyLargeWithSubdir)
			}
		})
	}
}

func TestDetermineStrategy_ErrorHandling(t *testing.T) {
	tests := []struct {
		name              string
		calculateSizeErr  error
		hasSubdirsErr     error
		dirSize           int64
		volumeSize        int64
		expectError       bool
		errorContains     string
	}{
		{
			name:             "error calculating directory size",
			calculateSizeErr: errors.New("permission denied"),
			dirSize:          0,
			volumeSize:       1024,
			expectError:      true,
			errorContains:    "failed to determine compression strategy",
		},
		{
			name:          "error checking subdirectories",
			hasSubdirsErr: errors.New("permission denied"),
			dirSize:       2048,
			volumeSize:    1024,
			expectError:   true,
			errorContains: "failed to determine compression strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := &mockFileSystemService{
				calculateDirSizeFunc: func(path string) (int64, error) {
					if tt.calculateSizeErr != nil {
						return 0, tt.calculateSizeErr
					}
					return tt.dirSize, nil
				},
				hasSubdirsFunc: func(path string) (bool, error) {
					if tt.hasSubdirsErr != nil {
						return false, tt.hasSubdirsErr
					}
					return false, nil
				},
			}

			service := NewService(mockFS, &mockSevenZipWrapper{})
			_, err := service.DetermineStrategy("/test/path", tt.volumeSize)

			if tt.expectError {
				if err == nil {
					t.Errorf("DetermineStrategy() expected error but got nil")
					return
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("DetermineStrategy() error = %v, want error containing %v", err, tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("DetermineStrategy() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestCompressionStrategy_String(t *testing.T) {
	tests := []struct {
		strategy CompressionStrategy
		want     string
	}{
		{StrategySmallDir, "SmallDir"},
		{StrategyLargeNoSubdir, "LargeNoSubdir"},
		{StrategyLargeWithSubdir, "LargeWithSubdir"},
		{CompressionStrategy(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.want {
				t.Errorf("CompressionStrategy.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewService(t *testing.T) {
	mockFS := &mockFileSystemService{}
	mockZip := &mockSevenZipWrapper{}
	service := NewService(mockFS, mockZip)

	if service == nil {
		t.Error("NewService() returned nil")
	}

	if service.fs != mockFS {
		t.Error("NewService() did not set filesystem service correctly")
	}

	if service.sevenZip != mockZip {
		t.Error("NewService() did not set sevenzip wrapper correctly")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCompressDirectory_SmallDir(t *testing.T) {
	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return nil
		},
	}

	mockZip := &mockSevenZipWrapper{}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: "/source/dir",
		TargetPath: "/target/dir",
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategySmallDir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err != nil {
		t.Errorf("CompressDirectory() error = %v", err)
	}

	// Verify compress was called once
	if len(mockZip.compressCalls) != 1 {
		t.Errorf("Expected 1 compress call, got %d", len(mockZip.compressCalls))
	}

	// Verify parameters
	call := mockZip.compressCalls[0]
	if call.Password != "test123" {
		t.Errorf("Expected password 'test123', got '%s'", call.Password)
	}
	if call.VolumeSize != 0 {
		t.Errorf("Expected VolumeSize 0 for small dir, got %d", call.VolumeSize)
	}
	if len(call.Sources) != 1 || call.Sources[0] != "/source/dir" {
		t.Errorf("Expected sources [/source/dir], got %v", call.Sources)
	}
}

func TestCompressDirectory_LargeNoSubdir(t *testing.T) {
	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return nil
		},
	}

	mockZip := &mockSevenZipWrapper{}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: "/source/dir",
		TargetPath: "/target/dir",
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategyLargeNoSubdir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err != nil {
		t.Errorf("CompressDirectory() error = %v", err)
	}

	// Verify compress was called once
	if len(mockZip.compressCalls) != 1 {
		t.Errorf("Expected 1 compress call, got %d", len(mockZip.compressCalls))
	}

	// Verify parameters
	call := mockZip.compressCalls[0]
	if call.Password != "test123" {
		t.Errorf("Expected password 'test123', got '%s'", call.Password)
	}
	if call.VolumeSize != 1024 {
		t.Errorf("Expected VolumeSize 1024, got %d", call.VolumeSize)
	}
}

func TestCompressDirectory_CreateDirError(t *testing.T) {
	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return errors.New("permission denied")
		},
	}

	mockZip := &mockSevenZipWrapper{}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: "/source/dir",
		TargetPath: "/target/dir",
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategySmallDir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err == nil {
		t.Error("Expected error when creating directory fails")
	}

	// Verify compress was not called
	if len(mockZip.compressCalls) != 0 {
		t.Errorf("Expected 0 compress calls, got %d", len(mockZip.compressCalls))
	}
}

func TestCompressDirectory_CompressError(t *testing.T) {
	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return nil
		},
	}

	mockZip := &mockSevenZipWrapper{
		compressFunc: func(params CompressParams) error {
			return errors.New("7zip failed")
		},
	}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: "/source/dir",
		TargetPath: "/target/dir",
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategySmallDir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err == nil {
		t.Error("Expected error when compression fails")
	}
}

func TestCompressDirectory_UnknownStrategy(t *testing.T) {
	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return nil
		},
	}

	mockZip := &mockSevenZipWrapper{}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: "/source/dir",
		TargetPath: "/target/dir",
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   CompressionStrategy(999),
	}

	err := service.CompressDirectory(context.Background(), task)
	if err == nil {
		t.Error("Expected error for unknown strategy")
	}
}

func TestCompressDirectory_OutputFilenameHandling(t *testing.T) {
	tests := []struct {
		name           string
		targetPath     string
		expectedOutput string
	}{
		{
			name:           "target path without .7z.001 suffix",
			targetPath:     "/target/dir",
			expectedOutput: "/target/dir.7z.001",
		},
		{
			name:           "target path with .7z.001 suffix",
			targetPath:     "/target/dir.7z.001",
			expectedOutput: "/target/dir.7z.001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := &mockFileSystemService{
				createDirFunc: func(path string) error {
					return nil
				},
			}

			mockZip := &mockSevenZipWrapper{}

			service := NewService(mockFS, mockZip)

			task := CompressionTask{
				SourcePath: "/source/dir",
				TargetPath: tt.targetPath,
				Password:   "test123",
				VolumeSize: 1024,
				Strategy:   StrategySmallDir,
			}

			err := service.CompressDirectory(context.Background(), task)
			if err != nil {
				t.Errorf("CompressDirectory() error = %v", err)
			}

			if len(mockZip.compressCalls) != 1 {
				t.Fatalf("Expected 1 compress call, got %d", len(mockZip.compressCalls))
			}

			if mockZip.compressCalls[0].Output != tt.expectedOutput {
				t.Errorf("Expected output %s, got %s", tt.expectedOutput, mockZip.compressCalls[0].Output)
			}
		})
	}
}

func TestCompressDirectory_PasswordPropagation(t *testing.T) {
	passwords := []string{"simple", "complex!@#$%", "with spaces", ""}

	for _, password := range passwords {
		t.Run("password_"+password, func(t *testing.T) {
			mockFS := &mockFileSystemService{
				createDirFunc: func(path string) error {
					return nil
				},
			}

			mockZip := &mockSevenZipWrapper{}

			service := NewService(mockFS, mockZip)

			task := CompressionTask{
				SourcePath: "/source/dir",
				TargetPath: "/target/dir",
				Password:   password,
				VolumeSize: 1024,
				Strategy:   StrategySmallDir,
			}

			err := service.CompressDirectory(context.Background(), task)
			if err != nil {
				t.Errorf("CompressDirectory() error = %v", err)
			}

			if len(mockZip.compressCalls) != 1 {
				t.Fatalf("Expected 1 compress call, got %d", len(mockZip.compressCalls))
			}

			if mockZip.compressCalls[0].Password != password {
				t.Errorf("Expected password %s, got %s", password, mockZip.compressCalls[0].Password)
			}
		})
	}
}

func TestCompressDirectory_VolumeSizeHandling(t *testing.T) {
	tests := []struct {
		name               string
		strategy           CompressionStrategy
		volumeSize         int64
		expectedVolumeSize int64
	}{
		{
			name:               "small dir should have zero volume size",
			strategy:           StrategySmallDir,
			volumeSize:         1024,
			expectedVolumeSize: 0,
		},
		{
			name:               "large no subdir should preserve volume size",
			strategy:           StrategyLargeNoSubdir,
			volumeSize:         1024,
			expectedVolumeSize: 1024,
		},
		{
			name:               "large no subdir with large volume size",
			strategy:           StrategyLargeNoSubdir,
			volumeSize:         4294967296,
			expectedVolumeSize: 4294967296,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := &mockFileSystemService{
				createDirFunc: func(path string) error {
					return nil
				},
			}

			mockZip := &mockSevenZipWrapper{}

			service := NewService(mockFS, mockZip)

			task := CompressionTask{
				SourcePath: "/source/dir",
				TargetPath: "/target/dir",
				Password:   "test123",
				VolumeSize: tt.volumeSize,
				Strategy:   tt.strategy,
			}

			err := service.CompressDirectory(context.Background(), task)
			if err != nil {
				t.Errorf("CompressDirectory() error = %v", err)
			}

			if len(mockZip.compressCalls) != 1 {
				t.Fatalf("Expected 1 compress call, got %d", len(mockZip.compressCalls))
			}

			if mockZip.compressCalls[0].VolumeSize != tt.expectedVolumeSize {
				t.Errorf("Expected volume size %d, got %d", tt.expectedVolumeSize, mockZip.compressCalls[0].VolumeSize)
			}
		})
	}
}

func TestCompressDirectory_LargeWithSubdir_OnlyFiles(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create test files
	testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, filename := range testFiles {
		filePath := filepath.Join(sourceDir, filename)
		if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return os.MkdirAll(path, 0755)
		},
		calculateDirSizeFunc: func(path string) (int64, error) {
			return 1024, nil
		},
		hasSubdirsFunc: func(path string) (bool, error) {
			return false, nil
		},
	}

	mockZip := &mockSevenZipWrapper{}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: sourceDir,
		TargetPath: filepath.Join(tempDir, "target", "backup"),
		Password:   "test123",
		VolumeSize: 2048,
		Strategy:   StrategyLargeWithSubdir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err != nil {
		t.Errorf("CompressDirectory() error = %v", err)
	}

	// Verify compress was called once for files
	if len(mockZip.compressCalls) != 1 {
		t.Errorf("Expected 1 compress call, got %d", len(mockZip.compressCalls))
	}

	// Verify the output is zbaksubfiles.7z.001
	call := mockZip.compressCalls[0]
	if !strings.HasSuffix(call.Output, "zbaksubfiles.7z.001") {
		t.Errorf("Expected output to end with zbaksubfiles.7z.001, got %s", call.Output)
	}

	// Verify all files are included
	if len(call.Sources) != len(testFiles) {
		t.Errorf("Expected %d sources, got %d", len(testFiles), len(call.Sources))
	}
}

func TestCompressDirectory_LargeWithSubdir_OnlySubdirs(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create subdirectories
	subdir1 := filepath.Join(sourceDir, "subdir1")
	subdir2 := filepath.Join(sourceDir, "subdir2")
	if err := os.MkdirAll(subdir1, 0755); err != nil {
		t.Fatalf("Failed to create subdir1: %v", err)
	}
	if err := os.MkdirAll(subdir2, 0755); err != nil {
		t.Fatalf("Failed to create subdir2: %v", err)
	}

	// Create files in subdirectories
	if err := os.WriteFile(filepath.Join(subdir1, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file in subdir1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir2, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file in subdir2: %v", err)
	}

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return os.MkdirAll(path, 0755)
		},
		calculateDirSizeFunc: func(path string) (int64, error) {
			// Return small size for subdirectories
			return 512, nil
		},
		hasSubdirsFunc: func(path string) (bool, error) {
			return false, nil
		},
	}

	mockZip := &mockSevenZipWrapper{}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: sourceDir,
		TargetPath: filepath.Join(tempDir, "target", "backup"),
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategyLargeWithSubdir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err != nil {
		t.Errorf("CompressDirectory() error = %v", err)
	}

	// Verify compress was called twice (once for each subdirectory)
	if len(mockZip.compressCalls) != 2 {
		t.Errorf("Expected 2 compress calls, got %d", len(mockZip.compressCalls))
	}
}

func TestCompressDirectory_LargeWithSubdir_MixedContent(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create files in root
	if err := os.WriteFile(filepath.Join(sourceDir, "root_file.txt"), []byte("root content"), 0644); err != nil {
		t.Fatalf("Failed to create root file: %v", err)
	}

	// Create subdirectory with files
	subdir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "sub_file.txt"), []byte("sub content"), 0644); err != nil {
		t.Fatalf("Failed to create file in subdir: %v", err)
	}

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return os.MkdirAll(path, 0755)
		},
		calculateDirSizeFunc: func(path string) (int64, error) {
			return 512, nil
		},
		hasSubdirsFunc: func(path string) (bool, error) {
			return false, nil
		},
	}

	mockZip := &mockSevenZipWrapper{}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: sourceDir,
		TargetPath: filepath.Join(tempDir, "target", "backup"),
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategyLargeWithSubdir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err != nil {
		t.Errorf("CompressDirectory() error = %v", err)
	}

	// Verify compress was called twice:
	// 1. For subdirectory (processed first)
	// 2. For root files (zbaksubfiles.7z.001)
	if len(mockZip.compressCalls) != 2 {
		t.Errorf("Expected 2 compress calls, got %d", len(mockZip.compressCalls))
	}

	// First call should be for subdirectory (subdirs are processed first)
	if !strings.Contains(mockZip.compressCalls[0].Output, "subdir") {
		t.Errorf("Expected first call to be for subdir, got %s", mockZip.compressCalls[0].Output)
	}

	// Second call should be for zbaksubfiles.7z.001
	if !strings.HasSuffix(mockZip.compressCalls[1].Output, "zbaksubfiles.7z.001") {
		t.Errorf("Expected second call to be for zbaksubfiles.7z.001, got %s", mockZip.compressCalls[1].Output)
	}
}

func TestCompressDirectory_LargeWithSubdir_EmptyDirectory(t *testing.T) {
	// Create temporary empty directory
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return os.MkdirAll(path, 0755)
		},
	}

	mockZip := &mockSevenZipWrapper{}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: sourceDir,
		TargetPath: filepath.Join(tempDir, "target", "backup"),
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategyLargeWithSubdir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err != nil {
		t.Errorf("CompressDirectory() error = %v", err)
	}

	// Verify no compress calls for empty directory
	if len(mockZip.compressCalls) != 0 {
		t.Errorf("Expected 0 compress calls for empty directory, got %d", len(mockZip.compressCalls))
	}
}

func TestCompressDirectory_LargeWithSubdir_ReadDirError(t *testing.T) {
	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return nil
		},
	}

	mockZip := &mockSevenZipWrapper{}

	service := NewService(mockFS, mockZip)

	// Use non-existent directory to trigger ReadDir error
	task := CompressionTask{
		SourcePath: "/non/existent/path",
		TargetPath: "/target/backup",
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategyLargeWithSubdir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err == nil {
		t.Error("Expected error when reading non-existent directory")
	}

	if !strings.Contains(err.Error(), "failed to read directory") {
		t.Errorf("Expected 'failed to read directory' error, got: %v", err)
	}
}

func TestCompressDirectory_LargeWithSubdir_CompressFilesError(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create a test file
	if err := os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return os.MkdirAll(path, 0755)
		},
	}

	mockZip := &mockSevenZipWrapper{
		compressFunc: func(params CompressParams) error {
			return errors.New("compression failed")
		},
	}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: sourceDir,
		TargetPath: filepath.Join(tempDir, "target", "backup"),
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategyLargeWithSubdir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err == nil {
		t.Error("Expected error when compression fails")
	}

	if !strings.Contains(err.Error(), "failed to compress files in directory") {
		t.Errorf("Expected 'failed to compress files in directory' error, got: %v", err)
	}
}

func TestCompressDirectory_LargeWithSubdir_DetermineStrategyError(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create subdirectory
	subdir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return os.MkdirAll(path, 0755)
		},
		calculateDirSizeFunc: func(path string) (int64, error) {
			// Return error for subdirectory
			if strings.Contains(path, "subdir") {
				return 0, errors.New("permission denied")
			}
			return 512, nil
		},
	}

	mockZip := &mockSevenZipWrapper{}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: sourceDir,
		TargetPath: filepath.Join(tempDir, "target", "backup"),
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategyLargeWithSubdir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err == nil {
		t.Error("Expected error when determining strategy fails")
	}

	if !strings.Contains(err.Error(), "failed to determine strategy for subdirectory") {
		t.Errorf("Expected 'failed to determine strategy for subdirectory' error, got: %v", err)
	}
}

func TestCompressDirectory_LargeWithSubdir_RecursiveCompressionError(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create subdirectory
	subdir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Create a file in subdirectory to ensure it's not empty
	if err := os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file in subdir: %v", err)
	}

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return os.MkdirAll(path, 0755)
		},
		calculateDirSizeFunc: func(path string) (int64, error) {
			return 512, nil
		},
		hasSubdirsFunc: func(path string) (bool, error) {
			return false, nil
		},
	}

	mockZip := &mockSevenZipWrapper{
		compressFunc: func(params CompressParams) error {
			// Fail on subdirectory compression
			return errors.New("compression failed")
		},
	}

	service := NewService(mockFS, mockZip)

	task := CompressionTask{
		SourcePath: sourceDir,
		TargetPath: filepath.Join(tempDir, "target", "backup"),
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategyLargeWithSubdir,
	}

	err := service.CompressDirectory(context.Background(), task)
	if err == nil {
		t.Error("Expected error when recursive compression fails")
		return
	}

	if !strings.Contains(err.Error(), "failed to compress subdirectory") {
		t.Errorf("Expected 'failed to compress subdirectory' error, got: %v", err)
	}
}

func TestCompressDirectory_LargeWithSubdir_TargetPathHandling(t *testing.T) {
	tests := []struct {
		name       string
		targetPath string
	}{
		{
			name:       "target path without .7z.001 suffix",
			targetPath: "/target/backup",
		},
		{
			name:       "target path with .7z.001 suffix",
			targetPath: "/target/backup.7z.001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tempDir := t.TempDir()
			sourceDir := filepath.Join(tempDir, "source")
			if err := os.MkdirAll(sourceDir, 0755); err != nil {
				t.Fatalf("Failed to create source directory: %v", err)
			}

			// Create a test file
			if err := os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("content"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			mockFS := &mockFileSystemService{
				createDirFunc: func(path string) error {
					return os.MkdirAll(path, 0755)
				},
			}

			mockZip := &mockSevenZipWrapper{}

			service := NewService(mockFS, mockZip)

			// Use temp directory for target path
			targetPath := tt.targetPath
			if filepath.IsAbs(tt.targetPath) {
				targetPath = filepath.Join(tempDir, filepath.Base(tt.targetPath))
			}

			task := CompressionTask{
				SourcePath: sourceDir,
				TargetPath: targetPath,
				Password:   "test123",
				VolumeSize: 1024,
				Strategy:   StrategyLargeWithSubdir,
			}

			err := service.CompressDirectory(context.Background(), task)
			if err != nil {
				t.Errorf("CompressDirectory() error = %v", err)
			}

			// Verify compress was called
			if len(mockZip.compressCalls) != 1 {
				t.Errorf("Expected 1 compress call, got %d", len(mockZip.compressCalls))
			}

			// Verify output ends with zbaksubfiles.7z.001
			if !strings.HasSuffix(mockZip.compressCalls[0].Output, "zbaksubfiles.7z.001") {
				t.Errorf("Expected output to end with zbaksubfiles.7z.001, got %s", mockZip.compressCalls[0].Output)
			}
		})
	}
}

// WorkerPool Tests

func TestNewWorkerPool(t *testing.T) {
	mockFS := &mockFileSystemService{}
	mockZip := &mockSevenZipWrapper{}
	service := NewService(mockFS, mockZip)

	tests := []struct {
		name            string
		workerCount     int
		expectedWorkers int
	}{
		{
			name:            "normal worker count",
			workerCount:     4,
			expectedWorkers: 4,
		},
		{
			name:            "single worker (serial mode)",
			workerCount:     1,
			expectedWorkers: 1,
		},
		{
			name:            "zero workers should default to 1",
			workerCount:     0,
			expectedWorkers: 1,
		},
		{
			name:            "negative workers should default to 1",
			workerCount:     -5,
			expectedWorkers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewWorkerPool(service, tt.workerCount)

			if pool == nil {
				t.Error("NewWorkerPool() returned nil")
			}

			if pool.workerCount != tt.expectedWorkers {
				t.Errorf("Expected worker count %d, got %d", tt.expectedWorkers, pool.workerCount)
			}

			if pool.service != service {
				t.Error("NewWorkerPool() did not set service correctly")
			}

			if pool.taskChan == nil {
				t.Error("NewWorkerPool() did not initialize taskChan")
			}

			if pool.errorChan == nil {
				t.Error("NewWorkerPool() did not initialize errorChan")
			}

			if pool.doneChan == nil {
				t.Error("NewWorkerPool() did not initialize doneChan")
			}
		})
	}
}

func TestWorkerPool_SerialExecution(t *testing.T) {
	// Test serial execution (concurrency = 1)
	// Requirement 9.2: Support serial execution when concurrency=1

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return nil
		},
	}

	mockZip := &mockSevenZipWrapper{}
	service := NewService(mockFS, mockZip)

	pool := NewWorkerPool(service, 1)
	ctx := context.Background()
	pool.Start(ctx)

	// Submit multiple tasks
	tasks := []CompressionTask{
		{
			SourcePath: "/source/dir1",
			TargetPath: "/target/dir1",
			Password:   "test123",
			VolumeSize: 1024,
			Strategy:   StrategySmallDir,
		},
		{
			SourcePath: "/source/dir2",
			TargetPath: "/target/dir2",
			Password:   "test123",
			VolumeSize: 1024,
			Strategy:   StrategySmallDir,
		},
		{
			SourcePath: "/source/dir3",
			TargetPath: "/target/dir3",
			Password:   "test123",
			VolumeSize: 1024,
			Strategy:   StrategySmallDir,
		},
	}

	for _, task := range tasks {
		pool.Submit(task)
	}

	errors := pool.Wait()

	// Verify no errors
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %d errors", len(errors))
	}

	// Verify all tasks were executed
	if len(mockZip.compressCalls) != len(tasks) {
		t.Errorf("Expected %d compress calls, got %d", len(tasks), len(mockZip.compressCalls))
	}
}

func TestWorkerPool_ParallelExecution(t *testing.T) {
	// Test parallel execution (concurrency > 1)
	// Requirement 9.3: Support parallel execution when concurrency>1

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return nil
		},
	}

	mockZip := &mockSevenZipWrapper{}
	service := NewService(mockFS, mockZip)

	pool := NewWorkerPool(service, 4)
	ctx := context.Background()
	pool.Start(ctx)

	// Submit multiple tasks
	taskCount := 10
	for i := 0; i < taskCount; i++ {
		task := CompressionTask{
			SourcePath: filepath.Join("/source", "dir"+string(rune(i))),
			TargetPath: filepath.Join("/target", "dir"+string(rune(i))),
			Password:   "test123",
			VolumeSize: 1024,
			Strategy:   StrategySmallDir,
		}
		pool.Submit(task)
	}

	errors := pool.Wait()

	// Verify no errors
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %d errors", len(errors))
	}

	// Verify all tasks were executed
	callCount := mockZip.getCompressCallCount()
	if callCount != taskCount {
		t.Errorf("Expected %d compress calls, got %d", taskCount, callCount)
	}
}

func TestWorkerPool_ErrorCollection(t *testing.T) {
	// Test error collection from failed tasks
	// Requirements 10.1, 10.2: Collect errors from failed tasks

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return nil
		},
	}

	// Mock that fails on specific paths
	mockZip := &mockSevenZipWrapper{
		compressFunc: func(params CompressParams) error {
			if strings.Contains(params.Sources[0], "fail") {
				return errors.New("compression failed")
			}
			return nil
		},
	}

	service := NewService(mockFS, mockZip)
	pool := NewWorkerPool(service, 2)
	ctx := context.Background()
	pool.Start(ctx)

	// Submit tasks, some will fail
	tasks := []CompressionTask{
		{
			SourcePath: "/source/success1",
			TargetPath: "/target/success1",
			Password:   "test123",
			VolumeSize: 1024,
			Strategy:   StrategySmallDir,
		},
		{
			SourcePath: "/source/fail1",
			TargetPath: "/target/fail1",
			Password:   "test123",
			VolumeSize: 1024,
			Strategy:   StrategySmallDir,
		},
		{
			SourcePath: "/source/success2",
			TargetPath: "/target/success2",
			Password:   "test123",
			VolumeSize: 1024,
			Strategy:   StrategySmallDir,
		},
		{
			SourcePath: "/source/fail2",
			TargetPath: "/target/fail2",
			Password:   "test123",
			VolumeSize: 1024,
			Strategy:   StrategySmallDir,
		},
	}

	for _, task := range tasks {
		pool.Submit(task)
	}

	errors := pool.Wait()

	// Verify we collected 2 errors
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	// Verify error messages contain the failed paths
	errorMessages := make([]string, len(errors))
	for i, err := range errors {
		errorMessages[i] = err.Error()
	}

	failCount := 0
	for _, msg := range errorMessages {
		if strings.Contains(msg, "fail") {
			failCount++
		}
	}

	if failCount != 2 {
		t.Errorf("Expected 2 errors containing 'fail', got %d", failCount)
	}

	// Verify all tasks were attempted (4 compress calls)
	if len(mockZip.compressCalls) != len(tasks) {
		t.Errorf("Expected %d compress calls, got %d", len(tasks), len(mockZip.compressCalls))
	}
}

func TestWorkerPool_WaitForCompletion(t *testing.T) {
	// Test that Wait() blocks until all tasks complete
	// Requirement 9.4: Wait for all tasks to complete

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return nil
		},
	}

	mockZip := &mockSevenZipWrapper{}
	service := NewService(mockFS, mockZip)

	pool := NewWorkerPool(service, 2)
	ctx := context.Background()
	pool.Start(ctx)

	// Submit tasks
	taskCount := 5
	for i := 0; i < taskCount; i++ {
		task := CompressionTask{
			SourcePath: filepath.Join("/source", "dir"+string(rune(i))),
			TargetPath: filepath.Join("/target", "dir"+string(rune(i))),
			Password:   "test123",
			VolumeSize: 1024,
			Strategy:   StrategySmallDir,
		}
		pool.Submit(task)
	}

	// Wait should block until all tasks complete
	errors := pool.Wait()

	// After Wait() returns, all tasks should be complete
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(errors))
	}

	if len(mockZip.compressCalls) != taskCount {
		t.Errorf("Expected %d compress calls after Wait(), got %d", taskCount, len(mockZip.compressCalls))
	}
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	// Test that workers respect context cancellation

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return nil
		},
	}

	mockZip := &mockSevenZipWrapper{}
	service := NewService(mockFS, mockZip)

	pool := NewWorkerPool(service, 2)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Submit a task
	task := CompressionTask{
		SourcePath: "/source/dir1",
		TargetPath: "/target/dir1",
		Password:   "test123",
		VolumeSize: 1024,
		Strategy:   StrategySmallDir,
	}
	pool.Submit(task)

	// Cancel context immediately
	cancel()

	// Wait for completion
	errors := pool.Wait()

	// We should have either 0 or 1 compress calls depending on timing
	// The important thing is that it doesn't hang
	if len(mockZip.compressCalls) > 1 {
		t.Errorf("Expected at most 1 compress call, got %d", len(mockZip.compressCalls))
	}

	// Errors may or may not be present depending on timing
	_ = errors
}

func TestWorkerPool_EmptyQueue(t *testing.T) {
	// Test that Wait() works correctly with no tasks submitted

	mockFS := &mockFileSystemService{}
	mockZip := &mockSevenZipWrapper{}
	service := NewService(mockFS, mockZip)

	pool := NewWorkerPool(service, 2)
	ctx := context.Background()
	pool.Start(ctx)

	// Don't submit any tasks, just wait
	errors := pool.Wait()

	// Should have no errors
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(errors))
	}

	// Should have no compress calls
	if len(mockZip.compressCalls) != 0 {
		t.Errorf("Expected no compress calls, got %d", len(mockZip.compressCalls))
	}
}

func TestWorkerPool_MultipleErrors(t *testing.T) {
	// Test that all errors are collected when multiple tasks fail

	mockFS := &mockFileSystemService{
		createDirFunc: func(path string) error {
			return nil
		},
	}

	// Mock that always fails
	mockZip := &mockSevenZipWrapper{
		compressFunc: func(params CompressParams) error {
			return errors.New("compression failed")
		},
	}

	service := NewService(mockFS, mockZip)
	pool := NewWorkerPool(service, 3)
	ctx := context.Background()
	pool.Start(ctx)

	// Submit multiple tasks that will all fail
	taskCount := 10
	for i := 0; i < taskCount; i++ {
		task := CompressionTask{
			SourcePath: filepath.Join("/source", "dir"+string(rune(i))),
			TargetPath: filepath.Join("/target", "dir"+string(rune(i))),
			Password:   "test123",
			VolumeSize: 1024,
			Strategy:   StrategySmallDir,
		}
		pool.Submit(task)
	}

	errors := pool.Wait()

	// All tasks should have failed
	if len(errors) != taskCount {
		t.Errorf("Expected %d errors, got %d", taskCount, len(errors))
	}

	// All tasks should have been attempted
	callCount := mockZip.getCompressCallCount()
	if callCount != taskCount {
		t.Errorf("Expected %d compress calls, got %d", taskCount, callCount)
	}
}
