package compression

import (
	"context"
	"errors"
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
}

func (m *mockSevenZipWrapper) Compress(params CompressParams) error {
	m.compressCalls = append(m.compressCalls, params)
	if m.compressFunc != nil {
		return m.compressFunc(params)
	}
	return nil
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
