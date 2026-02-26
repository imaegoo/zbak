package index

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewFileIndex(t *testing.T) {
	index := NewFileIndex()
	
	if index == nil {
		t.Fatal("NewFileIndex() returned nil")
	}
	
	if index.Files == nil {
		t.Error("NewFileIndex() did not initialize Files map")
	}
	
	if len(index.Files) != 0 {
		t.Errorf("NewFileIndex() should create empty map, got %d entries", len(index.Files))
	}
}

func TestLoad_FileNotExists(t *testing.T) {
	// 测试文件不存在时返回空索引
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.yaml")
	
	index, err := Load(indexPath)
	if err != nil {
		t.Fatalf("Load() should not return error for non-existent file, got: %v", err)
	}
	
	if index == nil {
		t.Fatal("Load() returned nil index")
	}
	
	if len(index.Files) != 0 {
		t.Errorf("Load() should return empty index, got %d entries", len(index.Files))
	}
}

func TestLoad_ValidFile(t *testing.T) {
	// 创建临时目录和测试文件
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.yaml")
	
	// 创建测试索引
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	originalIndex := &FileIndex{
		Files: map[string]FileEntry{
			"file1.txt": {
				SourcePath:   "file1.txt",
				Size:         1024,
				ModTime:      testTime,
				ArchivePath:  "2024-01-15-10-30-00/file1.7z.001",
				TimestampDir: "2024-01-15-10-30-00",
				Deleted:      false,
			},
			"file2.txt": {
				SourcePath:   "file2.txt",
				Size:         2048,
				ModTime:      testTime,
				ArchivePath:  "2024-01-15-10-30-00/file2.7z.001",
				TimestampDir: "2024-01-15-10-30-00",
				Deleted:      true,
			},
		},
	}
	
	// 保存索引
	if err := Save(indexPath, originalIndex); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	
	// 加载索引
	loadedIndex, err := Load(indexPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	
	// 验证加载的索引
	if len(loadedIndex.Files) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(loadedIndex.Files))
	}
	
	// 验证file1.txt
	entry1, exists := loadedIndex.Files["file1.txt"]
	if !exists {
		t.Error("file1.txt not found in loaded index")
	} else {
		if entry1.SourcePath != "file1.txt" {
			t.Errorf("Expected SourcePath 'file1.txt', got '%s'", entry1.SourcePath)
		}
		if entry1.Size != 1024 {
			t.Errorf("Expected Size 1024, got %d", entry1.Size)
		}
		if !entry1.ModTime.Equal(testTime) {
			t.Errorf("Expected ModTime %v, got %v", testTime, entry1.ModTime)
		}
		if entry1.ArchivePath != "2024-01-15-10-30-00/file1.7z.001" {
			t.Errorf("Expected ArchivePath '2024-01-15-10-30-00/file1.7z.001', got '%s'", entry1.ArchivePath)
		}
		if entry1.TimestampDir != "2024-01-15-10-30-00" {
			t.Errorf("Expected TimestampDir '2024-01-15-10-30-00', got '%s'", entry1.TimestampDir)
		}
		if entry1.Deleted {
			t.Error("Expected Deleted to be false")
		}
	}
	
	// 验证file2.txt
	entry2, exists := loadedIndex.Files["file2.txt"]
	if !exists {
		t.Error("file2.txt not found in loaded index")
	} else {
		if !entry2.Deleted {
			t.Error("Expected Deleted to be true for file2.txt")
		}
	}
}

func TestLoad_InvalidFormat(t *testing.T) {
	// 创建临时目录和无效的YAML文件
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.yaml")
	
	// 写入无效的YAML内容
	invalidYAML := []byte("invalid: yaml: content: [")
	if err := os.WriteFile(indexPath, invalidYAML, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// 尝试加载
	_, err := Load(indexPath)
	if err == nil {
		t.Error("Load() should return error for invalid YAML format")
	}
}

func TestSave(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.yaml")
	
	// 创建测试索引
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	index := &FileIndex{
		Files: map[string]FileEntry{
			"test.txt": {
				SourcePath:   "test.txt",
				Size:         512,
				ModTime:      testTime,
				ArchivePath:  "2024-01-15-10-30-00/test.7z.001",
				TimestampDir: "2024-01-15-10-30-00",
				Deleted:      false,
			},
		},
	}
	
	// 保存索引
	if err := Save(indexPath, index); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	
	// 验证文件存在
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("Save() did not create index file")
	}
	
	// 重新加载并验证内容
	loadedIndex, err := Load(indexPath)
	if err != nil {
		t.Fatalf("Failed to reload saved index: %v", err)
	}
	
	if len(loadedIndex.Files) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(loadedIndex.Files))
	}
	
	entry, exists := loadedIndex.Files["test.txt"]
	if !exists {
		t.Error("test.txt not found in reloaded index")
	} else {
		if entry.Size != 512 {
			t.Errorf("Expected Size 512, got %d", entry.Size)
		}
	}
}

func TestSave_EmptyIndex(t *testing.T) {
	// 测试保存空索引
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.yaml")
	
	index := NewFileIndex()
	
	if err := Save(indexPath, index); err != nil {
		t.Fatalf("Save() failed for empty index: %v", err)
	}
	
	// 重新加载
	loadedIndex, err := Load(indexPath)
	if err != nil {
		t.Fatalf("Failed to reload empty index: %v", err)
	}
	
	if len(loadedIndex.Files) != 0 {
		t.Errorf("Expected empty index, got %d entries", len(loadedIndex.Files))
	}
}

func TestFileEntry_AllFields(t *testing.T) {
	// 测试FileEntry的所有字段都能正确序列化和反序列化
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.yaml")
	
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC)
	
	index := &FileIndex{
		Files: map[string]FileEntry{
			"dir/subdir/file.txt": {
				SourcePath:   "dir/subdir/file.txt",
				Size:         9876543210,
				ModTime:      testTime,
				ArchivePath:  "2024-01-15-10-30-45/dir/subdir.7z.001",
				TimestampDir: "2024-01-15-10-30-45",
				Deleted:      true,
			},
		},
	}
	
	// 保存并重新加载
	if err := Save(indexPath, index); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	
	loadedIndex, err := Load(indexPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	
	entry, exists := loadedIndex.Files["dir/subdir/file.txt"]
	if !exists {
		t.Fatal("Entry not found in loaded index")
	}
	
	// 验证所有字段
	if entry.SourcePath != "dir/subdir/file.txt" {
		t.Errorf("SourcePath mismatch: expected 'dir/subdir/file.txt', got '%s'", entry.SourcePath)
	}
	if entry.Size != 9876543210 {
		t.Errorf("Size mismatch: expected 9876543210, got %d", entry.Size)
	}
	if !entry.ModTime.Equal(testTime) {
		t.Errorf("ModTime mismatch: expected %v, got %v", testTime, entry.ModTime)
	}
	if entry.ArchivePath != "2024-01-15-10-30-45/dir/subdir.7z.001" {
		t.Errorf("ArchivePath mismatch: expected '2024-01-15-10-30-45/dir/subdir.7z.001', got '%s'", entry.ArchivePath)
	}
	if entry.TimestampDir != "2024-01-15-10-30-45" {
		t.Errorf("TimestampDir mismatch: expected '2024-01-15-10-30-45', got '%s'", entry.TimestampDir)
	}
	if !entry.Deleted {
		t.Error("Deleted should be true")
	}
}
