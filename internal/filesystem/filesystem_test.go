package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCalculateDirSize(t *testing.T) {
	service := NewService()

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		size, err := service.CalculateDirSize(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if size != 0 {
			t.Errorf("expected size 0, got %d", size)
		}
	})

	t.Run("directory with files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test files
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")

		if err := os.WriteFile(file1, []byte("hello"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if err := os.WriteFile(file2, []byte("world!"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		size, err := service.CalculateDirSize(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedSize := int64(5 + 6) // "hello" + "world!"
		if size != expectedSize {
			t.Errorf("expected size %d, got %d", expectedSize, size)
		}
	})

	t.Run("directory with subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create subdirectory with files
		subDir := filepath.Join(tmpDir, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(subDir, "file2.txt")

		if err := os.WriteFile(file1, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if err := os.WriteFile(file2, []byte("data"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		size, err := service.CalculateDirSize(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedSize := int64(4 + 4) // "test" + "data"
		if size != expectedSize {
			t.Errorf("expected size %d, got %d", expectedSize, size)
		}
	})

	t.Run("excludes symbolic links", func(t *testing.T) {
		// Skip on Windows as symlinks require admin privileges
		if runtime.GOOS == "windows" {
			t.Skip("skipping symlink test on Windows")
		}

		tmpDir := t.TempDir()

		// Create a real file
		realFile := filepath.Join(tmpDir, "real.txt")
		if err := os.WriteFile(realFile, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Create a symbolic link
		linkFile := filepath.Join(tmpDir, "link.txt")
		if err := os.Symlink(realFile, linkFile); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		size, err := service.CalculateDirSize(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only count the real file once
		expectedSize := int64(7) // "content"
		if size != expectedSize {
			t.Errorf("expected size %d, got %d", expectedSize, size)
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		size, err := service.CalculateDirSize("/non/existent/path")
		if err == nil {
			t.Error("expected error for non-existent directory")
		}

		if size != 0 {
			t.Errorf("expected size 0 on error, got %d", size)
		}
	})
}

func TestHasSubdirs(t *testing.T) {
	service := NewService()

	t.Run("directory with subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		subDir := filepath.Join(tmpDir, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		hasSubdirs, err := service.HasSubdirs(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !hasSubdirs {
			t.Error("expected HasSubdirs to return true")
		}
	})

	t.Run("directory without subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create only files
		file := filepath.Join(tmpDir, "file.txt")
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		hasSubdirs, err := service.HasSubdirs(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if hasSubdirs {
			t.Error("expected HasSubdirs to return false")
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		hasSubdirs, err := service.HasSubdirs(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if hasSubdirs {
			t.Error("expected HasSubdirs to return false for empty directory")
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := service.HasSubdirs("/non/existent/path")
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})
}

func TestListFiles(t *testing.T) {
	service := NewService()

	t.Run("directory with files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test files
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")

		if err := os.WriteFile(file1, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if err := os.WriteFile(file2, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		files, err := service.ListFiles(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d", len(files))
		}
	})

	t.Run("excludes subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a file and a subdirectory
		file := filepath.Join(tmpDir, "file.txt")
		subDir := filepath.Join(tmpDir, "subdir")

		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		files, err := service.ListFiles(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(files) != 1 {
			t.Errorf("expected 1 file, got %d", len(files))
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		files, err := service.ListFiles(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := service.ListFiles("/non/existent/path")
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})
}

func TestCreateDir(t *testing.T) {
	service := NewService()

	t.Run("create single directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		newDir := filepath.Join(tmpDir, "newdir")

		err := service.CreateDir(newDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(newDir)
		if err != nil {
			t.Fatalf("directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("expected path to be a directory")
		}
	})

	t.Run("create nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		nestedDir := filepath.Join(tmpDir, "level1", "level2", "level3")

		err := service.CreateDir(nestedDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(nestedDir)
		if err != nil {
			t.Fatalf("nested directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("expected path to be a directory")
		}
	})

	t.Run("create existing directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := service.CreateDir(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error when creating existing directory: %v", err)
		}
	})

	t.Run("cross-platform path handling", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Use forward slashes which should be converted to platform-specific
		newDir := filepath.Join(tmpDir, "path/to/dir")

		err := service.CreateDir(newDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(newDir)
		if err != nil {
			t.Fatalf("directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("expected path to be a directory")
		}
	})
}

func TestFileExists(t *testing.T) {
	service := NewService()

	t.Run("existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "file.txt")

		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		if !service.FileExists(file) {
			t.Error("expected FileExists to return true for existing file")
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		if service.FileExists("/non/existent/file.txt") {
			t.Error("expected FileExists to return false for non-existent file")
		}
	})

	t.Run("directory path", func(t *testing.T) {
		tmpDir := t.TempDir()

		if service.FileExists(tmpDir) {
			t.Error("expected FileExists to return false for directory")
		}
	})

	t.Run("cross-platform path handling", func(t *testing.T) {
		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "file.txt")

		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Test with cleaned path
		if !service.FileExists(file) {
			t.Error("expected FileExists to return true with cross-platform path")
		}
	})
}
