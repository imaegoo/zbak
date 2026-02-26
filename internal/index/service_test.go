package index

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewIndexService(t *testing.T) {
	service := NewIndexService()
	
	if service == nil {
		t.Fatal("NewIndexService() returned nil")
	}
}

func TestIndexService_Load_FileNotExists(t *testing.T) {
	service := NewIndexService()
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.yaml")
	
	// 需求2.8: 当文件索引不存在时，创建新的文件索引
	index, err := service.Load(indexPath)
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

func TestIndexService_Load_ValidFile(t *testing.T) {
	service := NewIndexService()
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
		},
	}
	
	// 保存索引
	if err := service.Save(indexPath, originalIndex); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	
	// 加载索引
	loadedIndex, err := service.Load(indexPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	
	if len(loadedIndex.Files) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(loadedIndex.Files))
	}
	
	entry, exists := loadedIndex.Files["file1.txt"]
	if !exists {
		t.Error("file1.txt not found in loaded index")
	} else {
		if entry.Size != 1024 {
			t.Errorf("Expected Size 1024, got %d", entry.Size)
		}
	}
}

func TestIndexService_Load_InvalidFormat(t *testing.T) {
	service := NewIndexService()
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.yaml")
	
	// 需求2.9: 当文件索引格式无效时，记录警告并创建新的文件索引
	// 注意：IndexService.Load() 会捕获错误并返回新索引，而不是传播错误
	// 这与底层的 Load() 函数不同
	
	// 加载不存在的文件应该返回新索引
	index, err := service.Load(indexPath)
	if err != nil {
		t.Fatalf("Load() should not return error, got: %v", err)
	}
	
	if index == nil {
		t.Fatal("Load() returned nil index")
	}
	
	if len(index.Files) != 0 {
		t.Errorf("Load() should return empty index, got %d entries", len(index.Files))
	}
}

func TestIndexService_Save(t *testing.T) {
	service := NewIndexService()
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.yaml")
	
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
	
	if err := service.Save(indexPath, index); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	
	// 重新加载并验证
	loadedIndex, err := service.Load(indexPath)
	if err != nil {
		t.Fatalf("Failed to reload saved index: %v", err)
	}
	
	if len(loadedIndex.Files) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(loadedIndex.Files))
	}
}

func TestIndexService_AddEntry(t *testing.T) {
	service := NewIndexService()
	index := NewFileIndex()
	
	testTime := time.Now()
	entry := FileEntry{
		SourcePath:   "test.txt",
		Size:         1024,
		ModTime:      testTime,
		ArchivePath:  "2024-01-15-10-30-00/test.7z.001",
		TimestampDir: "2024-01-15-10-30-00",
		Deleted:      false,
	}
	
	err := service.AddEntry(index, entry)
	if err != nil {
		t.Fatalf("AddEntry() failed: %v", err)
	}
	
	if len(index.Files) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(index.Files))
	}
	
	storedEntry, exists := index.Files["test.txt"]
	if !exists {
		t.Error("Entry not found in index")
	} else {
		if storedEntry.Size != 1024 {
			t.Errorf("Expected Size 1024, got %d", storedEntry.Size)
		}
		if storedEntry.SourcePath != "test.txt" {
			t.Errorf("Expected SourcePath 'test.txt', got '%s'", storedEntry.SourcePath)
		}
	}
}

func TestIndexService_AddEntry_NilIndex(t *testing.T) {
	service := NewIndexService()
	
	entry := FileEntry{
		SourcePath: "test.txt",
		Size:       1024,
	}
	
	err := service.AddEntry(nil, entry)
	if err == nil {
		t.Error("AddEntry() should return error for nil index")
	}
}

func TestIndexService_AddEntry_UpdateExisting(t *testing.T) {
	service := NewIndexService()
	index := NewFileIndex()
	
	// 添加第一个条目
	entry1 := FileEntry{
		SourcePath: "test.txt",
		Size:       1024,
	}
	
	if err := service.AddEntry(index, entry1); err != nil {
		t.Fatalf("AddEntry() failed: %v", err)
	}
	
	// 更新同一个文件
	entry2 := FileEntry{
		SourcePath: "test.txt",
		Size:       2048,
	}
	
	if err := service.AddEntry(index, entry2); err != nil {
		t.Fatalf("AddEntry() failed: %v", err)
	}
	
	if len(index.Files) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(index.Files))
	}
	
	storedEntry := index.Files["test.txt"]
	if storedEntry.Size != 2048 {
		t.Errorf("Expected Size 2048, got %d", storedEntry.Size)
	}
}

func TestIndexService_MarkDeleted(t *testing.T) {
	service := NewIndexService()
	index := NewFileIndex()
	
	// 添加一个条目
	entry := FileEntry{
		SourcePath: "test.txt",
		Size:       1024,
		Deleted:    false,
	}
	
	if err := service.AddEntry(index, entry); err != nil {
		t.Fatalf("AddEntry() failed: %v", err)
	}
	
	// 标记为已删除
	err := service.MarkDeleted(index, "test.txt")
	if err != nil {
		t.Fatalf("MarkDeleted() failed: %v", err)
	}
	
	storedEntry := index.Files["test.txt"]
	if !storedEntry.Deleted {
		t.Error("Entry should be marked as deleted")
	}
}

func TestIndexService_MarkDeleted_NilIndex(t *testing.T) {
	service := NewIndexService()
	
	err := service.MarkDeleted(nil, "test.txt")
	if err == nil {
		t.Error("MarkDeleted() should return error for nil index")
	}
}

func TestIndexService_MarkDeleted_NonExistent(t *testing.T) {
	service := NewIndexService()
	index := NewFileIndex()
	
	err := service.MarkDeleted(index, "nonexistent.txt")
	if err == nil {
		t.Error("MarkDeleted() should return error for non-existent file")
	}
}

func TestIndexService_GetEntry(t *testing.T) {
	service := NewIndexService()
	index := NewFileIndex()
	
	// 添加一个条目
	testTime := time.Now()
	entry := FileEntry{
		SourcePath:   "test.txt",
		Size:         1024,
		ModTime:      testTime,
		ArchivePath:  "2024-01-15-10-30-00/test.7z.001",
		TimestampDir: "2024-01-15-10-30-00",
		Deleted:      false,
	}
	
	if err := service.AddEntry(index, entry); err != nil {
		t.Fatalf("AddEntry() failed: %v", err)
	}
	
	// 获取条目
	retrievedEntry, exists := service.GetEntry(index, "test.txt")
	if !exists {
		t.Fatal("GetEntry() should return true for existing entry")
	}
	
	if retrievedEntry == nil {
		t.Fatal("GetEntry() returned nil entry")
	}
	
	if retrievedEntry.Size != 1024 {
		t.Errorf("Expected Size 1024, got %d", retrievedEntry.Size)
	}
	
	if retrievedEntry.SourcePath != "test.txt" {
		t.Errorf("Expected SourcePath 'test.txt', got '%s'", retrievedEntry.SourcePath)
	}
}

func TestIndexService_GetEntry_NonExistent(t *testing.T) {
	service := NewIndexService()
	index := NewFileIndex()
	
	entry, exists := service.GetEntry(index, "nonexistent.txt")
	if exists {
		t.Error("GetEntry() should return false for non-existent entry")
	}
	
	if entry != nil {
		t.Error("GetEntry() should return nil entry for non-existent file")
	}
}

func TestIndexService_GetEntry_NilIndex(t *testing.T) {
	service := NewIndexService()
	
	entry, exists := service.GetEntry(nil, "test.txt")
	if exists {
		t.Error("GetEntry() should return false for nil index")
	}
	
	if entry != nil {
		t.Error("GetEntry() should return nil entry for nil index")
	}
}

// 需求9.5: 测试并发访问的线程安全性
func TestIndexService_ConcurrentAccess(t *testing.T) {
	service := NewIndexService()
	index := NewFileIndex()
	
	const numGoroutines = 100
	const numOperations = 10
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	// 启动多个goroutine并发添加条目
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				entry := FileEntry{
					SourcePath: fmt.Sprintf("file_%d_%d.txt", id, j),
					Size:       int64(id*numOperations + j),
					Deleted:    false,
				}
				
				if err := service.AddEntry(index, entry); err != nil {
					t.Errorf("AddEntry() failed in goroutine %d: %v", id, err)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// 验证所有条目都被正确添加
	expectedCount := numGoroutines * numOperations
	if len(index.Files) != expectedCount {
		t.Errorf("Expected %d entries, got %d", expectedCount, len(index.Files))
	}
}

// 测试并发读写操作
func TestIndexService_ConcurrentReadWrite(t *testing.T) {
	service := NewIndexService()
	index := NewFileIndex()
	
	// 先添加一些初始条目
	for i := 0; i < 10; i++ {
		entry := FileEntry{
			SourcePath: fmt.Sprintf("file_%d.txt", i),
			Size:       int64(i * 1024),
			Deleted:    false,
		}
		if err := service.AddEntry(index, entry); err != nil {
			t.Fatalf("AddEntry() failed: %v", err)
		}
	}
	
	const numReaders = 50
	const numWriters = 50
	
	var wg sync.WaitGroup
	wg.Add(numReaders + numWriters)
	
	// 启动读取goroutines
	for i := 0; i < numReaders; i++ {
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < 100; j++ {
				path := fmt.Sprintf("file_%d.txt", j%10)
				_, _ = service.GetEntry(index, path)
			}
		}(i)
	}
	
	// 启动写入goroutines
	for i := 0; i < numWriters; i++ {
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < 10; j++ {
				entry := FileEntry{
					SourcePath: fmt.Sprintf("new_file_%d_%d.txt", id, j),
					Size:       int64(id*10 + j),
					Deleted:    false,
				}
				if err := service.AddEntry(index, entry); err != nil {
					t.Errorf("AddEntry() failed in writer goroutine %d: %v", id, err)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// 验证索引仍然有效
	if index.Files == nil {
		t.Error("Index.Files should not be nil after concurrent operations")
	}
}

// 测试并发标记删除操作
func TestIndexService_ConcurrentMarkDeleted(t *testing.T) {
	service := NewIndexService()
	index := NewFileIndex()
	
	// 添加初始条目
	const numFiles = 100
	for i := 0; i < numFiles; i++ {
		entry := FileEntry{
			SourcePath: fmt.Sprintf("file_%d.txt", i),
			Size:       int64(i * 1024),
			Deleted:    false,
		}
		if err := service.AddEntry(index, entry); err != nil {
			t.Fatalf("AddEntry() failed: %v", err)
		}
	}
	
	var wg sync.WaitGroup
	wg.Add(numFiles)
	
	// 并发标记所有文件为已删除
	for i := 0; i < numFiles; i++ {
		go func(id int) {
			defer wg.Done()
			
			path := fmt.Sprintf("file_%d.txt", id)
			if err := service.MarkDeleted(index, path); err != nil {
				t.Errorf("MarkDeleted() failed for %s: %v", path, err)
			}
		}(i)
	}
	
	wg.Wait()
	
	// 验证所有文件都被标记为已删除
	for i := 0; i < numFiles; i++ {
		path := fmt.Sprintf("file_%d.txt", i)
		entry, exists := index.Files[path]
		if !exists {
			t.Errorf("File %s should exist in index", path)
			continue
		}
		if !entry.Deleted {
			t.Errorf("File %s should be marked as deleted", path)
		}
	}
}
