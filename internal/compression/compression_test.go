package compression

import (
	"errors"
	"testing"
)

// mockFileSystemService is a mock implementation of FileSystemService for testing
type mockFileSystemService struct {
	calculateDirSizeFunc func(path string) (int64, error)
	hasSubdirsFunc       func(path string) (bool, error)
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

			service := NewService(mockFS)
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

			service := NewService(mockFS)
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

			service := NewService(mockFS)
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

			service := NewService(mockFS)
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
	service := NewService(mockFS)

	if service == nil {
		t.Error("NewService() returned nil")
	}

	if service.fs != mockFS {
		t.Error("NewService() did not set filesystem service correctly")
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
