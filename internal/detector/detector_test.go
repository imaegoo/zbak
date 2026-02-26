package detector

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"zbak/internal/index"
)

func TestDetector_Detect_NewFiles(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	// 创建测试文件
	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建空索引
	fileIndex := index.NewFileIndex()

	// 执行检测
	detector := NewDetector()
	changeSet, err := detector.Detect(sourceDir, fileIndex)
	if err != nil {
		t.Fatalf("检测失败: %v", err)
	}

	// 验证结果
	if len(changeSet.NewFiles) != 1 {
		t.Errorf("期望1个新文件，实际得到%d个", len(changeSet.NewFiles))
	}
	if len(changeSet.NewFiles) > 0 && changeSet.NewFiles[0] != "test.txt" {
		t.Errorf("期望新文件为test.txt，实际为%s", changeSet.NewFiles[0])
	}
	if len(changeSet.ModifiedFiles) != 0 {
		t.Errorf("期望0个修改文件，实际得到%d个", len(changeSet.ModifiedFiles))
	}
	if len(changeSet.DeletedFiles) != 0 {
		t.Errorf("期望0个删除文件，实际得到%d个", len(changeSet.DeletedFiles))
	}
	if len(changeSet.UnchangedFiles) != 0 {
		t.Errorf("期望0个未变化文件，实际得到%d个", len(changeSet.UnchangedFiles))
	}
}

func TestDetector_Detect_ModifiedFiles(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	// 创建测试文件
	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 获取文件信息
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}

	// 创建索引，记录旧的文件信息（不同的大小）
	fileIndex := index.NewFileIndex()
	fileIndex.Files["test.txt"] = index.FileEntry{
		SourcePath: "test.txt",
		Size:       100, // 不同的大小
		ModTime:    info.ModTime(),
		Deleted:    false,
	}

	// 执行检测
	detector := NewDetector()
	changeSet, err := detector.Detect(sourceDir, fileIndex)
	if err != nil {
		t.Fatalf("检测失败: %v", err)
	}

	// 验证结果
	if len(changeSet.ModifiedFiles) != 1 {
		t.Errorf("期望1个修改文件，实际得到%d个", len(changeSet.ModifiedFiles))
	}
	if len(changeSet.ModifiedFiles) > 0 && changeSet.ModifiedFiles[0] != "test.txt" {
		t.Errorf("期望修改文件为test.txt，实际为%s", changeSet.ModifiedFiles[0])
	}
}

func TestDetector_Detect_DeletedFiles(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	// 创建索引，记录一个不存在的文件
	fileIndex := index.NewFileIndex()
	fileIndex.Files["deleted.txt"] = index.FileEntry{
		SourcePath: "deleted.txt",
		Size:       100,
		ModTime:    time.Now(),
		Deleted:    false,
	}

	// 执行检测
	detector := NewDetector()
	changeSet, err := detector.Detect(sourceDir, fileIndex)
	if err != nil {
		t.Fatalf("检测失败: %v", err)
	}

	// 验证结果
	if len(changeSet.DeletedFiles) != 1 {
		t.Errorf("期望1个删除文件，实际得到%d个", len(changeSet.DeletedFiles))
	}
	if len(changeSet.DeletedFiles) > 0 && changeSet.DeletedFiles[0] != "deleted.txt" {
		t.Errorf("期望删除文件为deleted.txt，实际为%s", changeSet.DeletedFiles[0])
	}
}

func TestDetector_Detect_UnchangedFiles(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	// 创建测试文件
	testFile := filepath.Join(sourceDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 获取文件信息
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}

	// 创建索引，记录相同的文件信息
	fileIndex := index.NewFileIndex()
	fileIndex.Files["test.txt"] = index.FileEntry{
		SourcePath: "test.txt",
		Size:       info.Size(),
		ModTime:    info.ModTime(),
		Deleted:    false,
	}

	// 执行检测
	detector := NewDetector()
	changeSet, err := detector.Detect(sourceDir, fileIndex)
	if err != nil {
		t.Fatalf("检测失败: %v", err)
	}

	// 验证结果
	if len(changeSet.UnchangedFiles) != 1 {
		t.Errorf("期望1个未变化文件，实际得到%d个", len(changeSet.UnchangedFiles))
	}
	if len(changeSet.UnchangedFiles) > 0 && changeSet.UnchangedFiles[0] != "test.txt" {
		t.Errorf("期望未变化文件为test.txt，实际为%s", changeSet.UnchangedFiles[0])
	}
	if len(changeSet.NewFiles) != 0 {
		t.Errorf("期望0个新文件，实际得到%d个", len(changeSet.NewFiles))
	}
	if len(changeSet.ModifiedFiles) != 0 {
		t.Errorf("期望0个修改文件，实际得到%d个", len(changeSet.ModifiedFiles))
	}
}

func TestDetector_Detect_PreviouslyDeletedFileReappears(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	// 创建测试文件
	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建索引，文件被标记为已删除
	fileIndex := index.NewFileIndex()
	fileIndex.Files["test.txt"] = index.FileEntry{
		SourcePath: "test.txt",
		Size:       100,
		ModTime:    time.Now(),
		Deleted:    true, // 标记为已删除
	}

	// 执行检测
	detector := NewDetector()
	changeSet, err := detector.Detect(sourceDir, fileIndex)
	if err != nil {
		t.Fatalf("检测失败: %v", err)
	}

	// 验证结果：之前被删除的文件重新出现，应该被标记为新增
	if len(changeSet.NewFiles) != 1 {
		t.Errorf("期望1个新文件，实际得到%d个", len(changeSet.NewFiles))
	}
	if len(changeSet.NewFiles) > 0 && changeSet.NewFiles[0] != "test.txt" {
		t.Errorf("期望新文件为test.txt，实际为%s", changeSet.NewFiles[0])
	}
}

func TestDetector_Detect_SkipsAlreadyDeletedFiles(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	// 创建索引，文件被标记为已删除
	fileIndex := index.NewFileIndex()
	fileIndex.Files["deleted.txt"] = index.FileEntry{
		SourcePath: "deleted.txt",
		Size:       100,
		ModTime:    time.Now(),
		Deleted:    true, // 已经标记为删除
	}

	// 执行检测
	detector := NewDetector()
	changeSet, err := detector.Detect(sourceDir, fileIndex)
	if err != nil {
		t.Fatalf("检测失败: %v", err)
	}

	// 验证结果：已经标记为删除的文件不应该再次出现在删除列表中
	if len(changeSet.DeletedFiles) != 0 {
		t.Errorf("期望0个删除文件，实际得到%d个", len(changeSet.DeletedFiles))
	}
}

func TestDetector_Detect_ModifiedTimeChange(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	// 创建测试文件
	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 获取文件信息
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}

	// 创建索引，记录不同的修改时间
	fileIndex := index.NewFileIndex()
	fileIndex.Files["test.txt"] = index.FileEntry{
		SourcePath: "test.txt",
		Size:       info.Size(),
		ModTime:    time.Now().Add(-1 * time.Hour), // 不同的修改时间
		Deleted:    false,
	}

	// 执行检测
	detector := NewDetector()
	changeSet, err := detector.Detect(sourceDir, fileIndex)
	if err != nil {
		t.Fatalf("检测失败: %v", err)
	}

	// 验证结果
	if len(changeSet.ModifiedFiles) != 1 {
		t.Errorf("期望1个修改文件，实际得到%d个", len(changeSet.ModifiedFiles))
	}
}

func TestDetector_Detect_NestedDirectories(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	subDir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("创建子目录失败: %v", err)
	}

	// 创建测试文件
	testFile1 := filepath.Join(sourceDir, "test1.txt")
	testFile2 := filepath.Join(subDir, "test2.txt")
	if err := os.WriteFile(testFile1, []byte("content1"), 0644); err != nil {
		t.Fatalf("创建测试文件1失败: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("content2"), 0644); err != nil {
		t.Fatalf("创建测试文件2失败: %v", err)
	}

	// 创建空索引
	fileIndex := index.NewFileIndex()

	// 执行检测
	detector := NewDetector()
	changeSet, err := detector.Detect(sourceDir, fileIndex)
	if err != nil {
		t.Fatalf("检测失败: %v", err)
	}

	// 验证结果：应该检测到两个新文件
	if len(changeSet.NewFiles) != 2 {
		t.Errorf("期望2个新文件，实际得到%d个", len(changeSet.NewFiles))
	}

	// 验证文件路径使用正确的分隔符
	expectedFiles := map[string]bool{
		"test1.txt":                true,
		filepath.Join("subdir", "test2.txt"): true,
	}
	for _, file := range changeSet.NewFiles {
		if !expectedFiles[file] {
			t.Errorf("意外的文件: %s", file)
		}
	}
}
