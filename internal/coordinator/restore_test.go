package coordinator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zbak/internal/config"
	"zbak/internal/index"
	"zbak/internal/sevenzip"
)

// TestDiscoverTimestamps 测试时间戳目录发现和排序
func TestDiscoverTimestamps(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试时间戳目录
	timestamps := []string{
		"2024-01-15-10-30-00",
		"2024-01-14-09-20-00",
		"2024-01-16-14-45-00",
		"2024-01-13-08-15-00",
	}

	for _, ts := range timestamps {
		tsPath := filepath.Join(tempDir, ts)
		if err := os.MkdirAll(tsPath, 0755); err != nil {
			t.Fatalf("创建时间戳目录失败: %v", err)
		}
	}

	// 创建一些非时间戳目录（应该被忽略）
	invalidDirs := []string{
		"not-a-timestamp",
		"2024-13-01-10-30-00", // 无效月份
		"index.yaml",          // 文件，不是目录
	}

	for _, dir := range invalidDirs {
		if dir == "index.yaml" {
			// 创建文件
			filePath := filepath.Join(tempDir, dir)
			if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
				t.Fatalf("创建测试文件失败: %v", err)
			}
		} else {
			// 创建目录
			dirPath := filepath.Join(tempDir, dir)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				t.Fatalf("创建无效目录失败: %v", err)
			}
		}
	}

	// 创建恢复协调器
	cfg := &config.Config{
		TargetDir: tempDir,
	}
	rc := NewRestoreCoordinator(cfg, index.NewFileIndex(), sevenzip.NewWrapper(), &mockLogger{})

	// 发现时间戳
	discovered, err := rc.discoverTimestamps()
	if err != nil {
		t.Fatalf("发现时间戳失败: %v", err)
	}

	// 验证发现的时间戳数量
	if len(discovered) != len(timestamps) {
		t.Errorf("期望发现 %d 个时间戳，实际发现 %d 个", len(timestamps), len(discovered))
	}

	// 验证时间戳按时间顺序排序（从旧到新）
	expected := []string{
		"2024-01-13-08-15-00",
		"2024-01-14-09-20-00",
		"2024-01-15-10-30-00",
		"2024-01-16-14-45-00",
	}

	for i, ts := range discovered {
		if ts != expected[i] {
			t.Errorf("时间戳顺序错误: 位置 %d 期望 %s，实际 %s", i, expected[i], ts)
		}
	}
}

// TestIsValidTimestamp 测试时间戳格式验证
func TestIsValidTimestamp(t *testing.T) {
	rc := &RestoreCoordinator{}

	tests := []struct {
		name      string
		timestamp string
		valid     bool
	}{
		{"有效时间戳", "2024-01-15-10-30-00", true},
		{"有效时间戳2", "2023-12-31-23-59-59", true},
		{"无效格式", "not-a-timestamp", false},
		{"无效月份", "2024-13-01-10-30-00", false},
		{"无效日期", "2024-01-32-10-30-00", false},
		{"无效小时", "2024-01-15-25-30-00", false},
		{"缺少部分", "2024-01-15", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rc.isValidTimestamp(tt.timestamp)
			if result != tt.valid {
				t.Errorf("isValidTimestamp(%s) = %v, 期望 %v", tt.timestamp, result, tt.valid)
			}
		})
	}
}

// TestFilterTimestamps 测试时间戳过滤
func TestFilterTimestamps(t *testing.T) {
	rc := &RestoreCoordinator{}

	allTimestamps := []string{
		"2024-01-13-08-15-00",
		"2024-01-14-09-20-00",
		"2024-01-15-10-30-00",
		"2024-01-16-14-45-00",
	}

	tests := []struct {
		name     string
		options  RestoreOptions
		expected []string
		hasError bool
	}{
		{
			name:     "单个时间戳",
			options:  RestoreOptions{Timestamp: "2024-01-15-10-30-00"},
			expected: []string{"2024-01-15-10-30-00"},
			hasError: false,
		},
		{
			name:     "不存在的时间戳",
			options:  RestoreOptions{Timestamp: "2024-01-17-00-00-00"},
			expected: nil,
			hasError: true,
		},
		{
			name:     "时间戳范围",
			options:  RestoreOptions{FromTime: "2024-01-14-00-00-00", ToTime: "2024-01-15-23-59-59"},
			expected: []string{"2024-01-14-09-20-00", "2024-01-15-10-30-00"},
			hasError: false,
		},
		{
			name:     "只指定起始时间",
			options:  RestoreOptions{FromTime: "2024-01-15-00-00-00"},
			expected: []string{"2024-01-15-10-30-00", "2024-01-16-14-45-00"},
			hasError: false,
		},
		{
			name:     "只指定结束时间",
			options:  RestoreOptions{ToTime: "2024-01-14-23-59-59"},
			expected: []string{"2024-01-13-08-15-00", "2024-01-14-09-20-00"},
			hasError: false,
		},
		{
			name:     "无过滤条件",
			options:  RestoreOptions{},
			expected: allTimestamps,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rc.filterTimestamps(allTimestamps, tt.options)

			if tt.hasError {
				if err == nil {
					t.Errorf("期望返回错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("不期望错误，但返回: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("期望 %d 个时间戳，实际 %d 个", len(tt.expected), len(result))
				return
			}

			for i, ts := range result {
				if ts != tt.expected[i] {
					t.Errorf("位置 %d: 期望 %s，实际 %s", i, tt.expected[i], ts)
				}
			}
		})
	}
}

// TestDiscoverArchives 测试压缩文件发现和分卷关联
func TestDiscoverArchives(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	timestampDir := "2024-01-15-10-30-00"
	timestampPath := filepath.Join(tempDir, timestampDir)

	// 创建测试目录结构
	testDirs := []string{
		"dir1",
		"dir2/subdir1",
		"dir3",
	}

	for _, dir := range testDirs {
		dirPath := filepath.Join(timestampPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("创建测试目录失败: %v", err)
		}
	}

	// 创建测试压缩文件
	testFiles := []struct {
		path string
	}{
		// 单个文件压缩
		{"dir1/archive1.7z.001"},
		// 分卷压缩
		{"dir2/archive2.7z.001"},
		{"dir2/archive2.7z.002"},
		{"dir2/archive2.7z.003"},
		// 子目录中的压缩文件
		{"dir2/subdir1/archive3.7z.001"},
		{"dir2/subdir1/archive3.7z.002"},
		// 另一个单文件压缩
		{"dir3/zbaksubfiles.7z.001"},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(timestampPath, file.path)
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	// 创建恢复协调器
	cfg := &config.Config{
		TargetDir: tempDir,
	}
	rc := NewRestoreCoordinator(cfg, index.NewFileIndex(), sevenzip.NewWrapper(), &mockLogger{})

	// 发现压缩文件
	archives, err := rc.discoverArchives(timestampDir)
	if err != nil {
		t.Fatalf("发现压缩文件失败: %v", err)
	}

	// 验证发现的压缩文件数量
	expectedCount := 4 // archive1, archive2, archive3, zbaksubfiles
	if len(archives) != expectedCount {
		t.Errorf("期望发现 %d 个压缩文件，实际发现 %d 个", expectedCount, len(archives))
	}

	// 验证每个压缩文件的信息
	archiveMap := make(map[string]ArchiveInfo)
	for _, archive := range archives {
		archiveMap[archive.BaseName] = archive
	}

	// 验证 archive1（单文件）
	if archive, exists := archiveMap["archive1"]; exists {
		if len(archive.AllVolumes) != 1 {
			t.Errorf("archive1 期望 1 个分卷，实际 %d 个", len(archive.AllVolumes))
		}
		if archive.TimestampDir != timestampDir {
			t.Errorf("archive1 时间戳目录错误: 期望 %s，实际 %s", timestampDir, archive.TimestampDir)
		}
	} else {
		t.Errorf("未找到 archive1")
	}

	// 验证 archive2（多分卷）
	if archive, exists := archiveMap["archive2"]; exists {
		if len(archive.AllVolumes) != 3 {
			t.Errorf("archive2 期望 3 个分卷，实际 %d 个", len(archive.AllVolumes))
		}
		// 验证分卷排序
		for i := 0; i < len(archive.AllVolumes)-1; i++ {
			if archive.AllVolumes[i] >= archive.AllVolumes[i+1] {
				t.Errorf("archive2 分卷未正确排序")
			}
		}
	} else {
		t.Errorf("未找到 archive2")
	}

	// 验证 archive3（子目录中的多分卷）
	if archive, exists := archiveMap["archive3"]; exists {
		if len(archive.AllVolumes) != 2 {
			t.Errorf("archive3 期望 2 个分卷，实际 %d 个", len(archive.AllVolumes))
		}
	} else {
		t.Errorf("未找到 archive3")
	}

	// 验证 zbaksubfiles
	if archive, exists := archiveMap["zbaksubfiles"]; exists {
		if len(archive.AllVolumes) != 1 {
			t.Errorf("zbaksubfiles 期望 1 个分卷，实际 %d 个", len(archive.AllVolumes))
		}
	} else {
		t.Errorf("未找到 zbaksubfiles")
	}
}

// TestExtractBaseName 测试基础名称提取
func TestExtractBaseName(t *testing.T) {
	rc := &RestoreCoordinator{}

	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"单分卷文件", "archive.7z.001", "archive"},
		{"多分卷文件", "archive.7z.002", "archive"},
		{"带路径的文件", "dir/archive.7z.001", "dir/archive"}, // 函数会正确提取基础名称
		{"复杂名称", "my-backup-2024.7z.001", "my-backup-2024"},
		{"zbaksubfiles压缩", "zbaksubfiles.7z.001", "zbaksubfiles"},
		{"大分卷号", "archive.7z.100", "archive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rc.extractBaseName(tt.filename)
			if result != tt.expected {
				t.Errorf("extractBaseName(%s) = %s, 期望 %s", tt.filename, result, tt.expected)
			}
		})
	}
}

// TestDiscoverArchives_EmptyDirectory 测试空目录
func TestDiscoverArchives_EmptyDirectory(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	timestampDir := "2024-01-15-10-30-00"
	timestampPath := filepath.Join(tempDir, timestampDir)

	// 创建空的时间戳目录
	if err := os.MkdirAll(timestampPath, 0755); err != nil {
		t.Fatalf("创建时间戳目录失败: %v", err)
	}

	// 创建恢复协调器
	cfg := &config.Config{
		TargetDir: tempDir,
	}
	rc := NewRestoreCoordinator(cfg, index.NewFileIndex(), sevenzip.NewWrapper(), &mockLogger{})

	// 发现压缩文件
	archives, err := rc.discoverArchives(timestampDir)
	if err != nil {
		t.Fatalf("发现压缩文件失败: %v", err)
	}

	// 验证返回空列表
	if len(archives) != 0 {
		t.Errorf("期望发现 0 个压缩文件，实际发现 %d 个", len(archives))
	}
}

// TestDiscoverArchives_MissingFirstVolume 测试缺少第一个分卷的情况
func TestDiscoverArchives_MissingFirstVolume(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	timestampDir := "2024-01-15-10-30-00"
	timestampPath := filepath.Join(tempDir, timestampDir)

	// 创建测试目录
	if err := os.MkdirAll(timestampPath, 0755); err != nil {
		t.Fatalf("创建时间戳目录失败: %v", err)
	}

	// 创建测试文件（缺少.7z.001）
	testFiles := []string{
		"archive.7z.002",
		"archive.7z.003",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(timestampPath, file)
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	// 创建恢复协调器
	cfg := &config.Config{
		TargetDir: tempDir,
	}
	rc := NewRestoreCoordinator(cfg, index.NewFileIndex(), sevenzip.NewWrapper(), &mockLogger{})

	// 发现压缩文件
	archives, err := rc.discoverArchives(timestampDir)
	if err != nil {
		t.Fatalf("发现压缩文件失败: %v", err)
	}

	// 验证仍然能发现压缩文件
	if len(archives) != 1 {
		t.Errorf("期望发现 1 个压缩文件，实际发现 %d 个", len(archives))
	}

	// 验证FirstVolume被设置为第一个可用的分卷
	if len(archives) > 0 {
		if archives[0].FirstVolume == "" {
			t.Errorf("FirstVolume 未设置")
		}
		if len(archives[0].AllVolumes) != 2 {
			t.Errorf("期望 2 个分卷，实际 %d 个", len(archives[0].AllVolumes))
		}
	}
}

// TestExecute_FullRestore 测试完整恢复流程
func TestExecute_FullRestore(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	// 创建源目录和目标目录
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("创建目标目录失败: %v", err)
	}

	// 创建时间戳目录和测试文件
	timestamp := "2024-01-15-10-30-00"
	timestampPath := filepath.Join(targetDir, timestamp)
	if err := os.MkdirAll(timestampPath, 0755); err != nil {
		t.Fatalf("创建时间戳目录失败: %v", err)
	}

	// 创建测试压缩文件
	archivePath := filepath.Join(timestampPath, "test.7z.001")
	if err := os.WriteFile(archivePath, []byte("test archive"), 0644); err != nil {
		t.Fatalf("创建测试压缩文件失败: %v", err)
	}

	// 创建文件索引
	idx := index.NewFileIndex()

	// 创建mock 7zip wrapper
	mockSevenZip := &mockSevenZipWrapper{
		extractFunc: func(params sevenzip.ExtractParams) error {
			// 模拟解压：创建一个测试文件
			testFile := filepath.Join(params.OutputDir, "test.txt")
			return os.WriteFile(testFile, []byte("restored content"), 0644)
		},
	}

	// 创建恢复协调器
	cfg := &config.Config{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Password:  "test123",
	}
	rc := NewRestoreCoordinator(cfg, idx, mockSevenZip, &mockLogger{})

	// 执行恢复
	report, err := rc.Execute(RestoreOptions{})
	if err != nil {
		t.Fatalf("恢复失败: %v", err)
	}

	// 验证报告
	if report.RestoredFiles != 1 {
		t.Errorf("期望恢复 1 个文件，实际 %d", report.RestoredFiles)
	}

	if report.FailedFiles != 0 {
		t.Errorf("期望 0 个失败文件，实际 %d", report.FailedFiles)
	}

	// 验证文件已恢复
	restoredFile := filepath.Join(sourceDir, "test.txt")
	if _, err := os.Stat(restoredFile); os.IsNotExist(err) {
		t.Errorf("恢复的文件不存在: %s", restoredFile)
	}
}

// TestExecute_SelectiveRestore_SingleTimestamp 测试选择性恢复（单个时间戳）
func TestExecute_SelectiveRestore_SingleTimestamp(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("创建目标目录失败: %v", err)
	}

	// 创建多个时间戳目录
	timestamps := []string{
		"2024-01-13-08-15-00",
		"2024-01-14-09-20-00",
		"2024-01-15-10-30-00",
	}

	for _, ts := range timestamps {
		tsPath := filepath.Join(targetDir, ts)
		if err := os.MkdirAll(tsPath, 0755); err != nil {
			t.Fatalf("创建时间戳目录失败: %v", err)
		}

		// 创建测试压缩文件
		archivePath := filepath.Join(tsPath, "test.7z.001")
		if err := os.WriteFile(archivePath, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试压缩文件失败: %v", err)
		}
	}

	// 创建文件索引
	idx := index.NewFileIndex()

	// 记录解压的时间戳
	var extractedTimestamps []string
	mockSevenZip := &mockSevenZipWrapper{
		extractFunc: func(params sevenzip.ExtractParams) error {
			// 记录哪个时间戳被解压
			for _, ts := range timestamps {
				if strings.Contains(params.Archive, ts) {
					extractedTimestamps = append(extractedTimestamps, ts)
					break
				}
			}
			return nil
		},
	}

	// 创建恢复协调器
	cfg := &config.Config{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Password:  "test123",
	}
	rc := NewRestoreCoordinator(cfg, idx, mockSevenZip, &mockLogger{})

	// 执行选择性恢复（只恢复中间的时间戳）
	report, err := rc.Execute(RestoreOptions{
		Timestamp: "2024-01-14-09-20-00",
	})
	if err != nil {
		t.Fatalf("恢复失败: %v", err)
	}

	// 验证只恢复了一个时间戳
	if len(extractedTimestamps) != 1 {
		t.Errorf("期望恢复 1 个时间戳，实际 %d", len(extractedTimestamps))
	}

	if len(extractedTimestamps) > 0 && extractedTimestamps[0] != "2024-01-14-09-20-00" {
		t.Errorf("期望恢复时间戳 2024-01-14-09-20-00，实际 %s", extractedTimestamps[0])
	}

	if report.RestoredFiles != 1 {
		t.Errorf("期望恢复 1 个文件，实际 %d", report.RestoredFiles)
	}
}

// TestExecute_SelectiveRestore_TimeRange 测试选择性恢复（时间范围）
func TestExecute_SelectiveRestore_TimeRange(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("创建目标目录失败: %v", err)
	}

	// 创建多个时间戳目录
	timestamps := []string{
		"2024-01-13-08-15-00",
		"2024-01-14-09-20-00",
		"2024-01-15-10-30-00",
		"2024-01-16-14-45-00",
	}

	for _, ts := range timestamps {
		tsPath := filepath.Join(targetDir, ts)
		if err := os.MkdirAll(tsPath, 0755); err != nil {
			t.Fatalf("创建时间戳目录失败: %v", err)
		}

		// 创建测试压缩文件
		archivePath := filepath.Join(tsPath, "test.7z.001")
		if err := os.WriteFile(archivePath, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试压缩文件失败: %v", err)
		}
	}

	// 创建文件索引
	idx := index.NewFileIndex()

	// 记录解压的时间戳
	var extractedTimestamps []string
	mockSevenZip := &mockSevenZipWrapper{
		extractFunc: func(params sevenzip.ExtractParams) error {
			for _, ts := range timestamps {
				if strings.Contains(params.Archive, ts) {
					extractedTimestamps = append(extractedTimestamps, ts)
					break
				}
			}
			return nil
		},
	}

	// 创建恢复协调器
	cfg := &config.Config{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Password:  "test123",
	}
	rc := NewRestoreCoordinator(cfg, idx, mockSevenZip, &mockLogger{})

	// 执行范围恢复
	report, err := rc.Execute(RestoreOptions{
		FromTime: "2024-01-14-00-00-00",
		ToTime:   "2024-01-15-23-59-59",
	})
	if err != nil {
		t.Fatalf("恢复失败: %v", err)
	}

	// 验证恢复了正确的时间戳
	expectedTimestamps := []string{
		"2024-01-14-09-20-00",
		"2024-01-15-10-30-00",
	}

	if len(extractedTimestamps) != len(expectedTimestamps) {
		t.Errorf("期望恢复 %d 个时间戳，实际 %d", len(expectedTimestamps), len(extractedTimestamps))
	}

	for i, expected := range expectedTimestamps {
		if i >= len(extractedTimestamps) || extractedTimestamps[i] != expected {
			t.Errorf("时间戳 %d: 期望 %s，实际 %s", i, expected, extractedTimestamps[i])
		}
	}

	if report.RestoredFiles != 2 {
		t.Errorf("期望恢复 2 个文件，实际 %d", report.RestoredFiles)
	}
}

// TestExecute_HandleDeletedFiles 测试已删除文件处理
func TestExecute_HandleDeletedFiles(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("创建目标目录失败: %v", err)
	}

	// 创建时间戳目录
	timestamp := "2024-01-15-10-30-00"
	timestampPath := filepath.Join(targetDir, timestamp)
	if err := os.MkdirAll(timestampPath, 0755); err != nil {
		t.Fatalf("创建时间戳目录失败: %v", err)
	}

	// 创建测试压缩文件
	archivePath := filepath.Join(timestampPath, "test.7z.001")
	if err := os.WriteFile(archivePath, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试压缩文件失败: %v", err)
	}

	// 在源目录创建一个文件（将被删除）
	deletedFile := filepath.Join(sourceDir, "deleted.txt")
	if err := os.WriteFile(deletedFile, []byte("to be deleted"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建文件索引，标记文件为已删除
	idx := index.NewFileIndex()
	idx.Files["deleted.txt"] = index.FileEntry{
		SourcePath:   "deleted.txt",
		Size:         13,
		ModTime:      time.Now(),
		ArchivePath:  filepath.Join(timestamp, "test.7z.001"),
		TimestampDir: timestamp,
		Deleted:      true,
	}

	mockSevenZip := &mockSevenZipWrapper{
		extractFunc: func(params sevenzip.ExtractParams) error {
			return nil
		},
	}

	// 创建恢复协调器
	cfg := &config.Config{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Password:  "test123",
	}
	rc := NewRestoreCoordinator(cfg, idx, mockSevenZip, &mockLogger{})

	// 执行恢复
	report, err := rc.Execute(RestoreOptions{})
	if err != nil {
		t.Fatalf("恢复失败: %v", err)
	}

	// 验证文件已被删除
	if _, err := os.Stat(deletedFile); !os.IsNotExist(err) {
		t.Errorf("期望文件被删除，但文件仍然存在: %s", deletedFile)
	}

	if report.DeletedFiles != 1 {
		t.Errorf("期望删除 1 个文件，实际 %d", report.DeletedFiles)
	}
}

// TestExecute_ErrorHandling 测试错误处理
func TestExecute_ErrorHandling(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("创建目标目录失败: %v", err)
	}

	// 创建时间戳目录
	timestamp := "2024-01-15-10-30-00"
	timestampPath := filepath.Join(targetDir, timestamp)
	if err := os.MkdirAll(timestampPath, 0755); err != nil {
		t.Fatalf("创建时间戳目录失败: %v", err)
	}

	// 创建多个测试压缩文件
	for i := 1; i <= 3; i++ {
		archivePath := filepath.Join(timestampPath, fmt.Sprintf("test%d.7z.001", i))
		if err := os.WriteFile(archivePath, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试压缩文件失败: %v", err)
		}
	}

	// 创建文件索引
	idx := index.NewFileIndex()

	// 创建mock 7zip wrapper，第二个文件解压失败
	extractCount := 0
	mockSevenZip := &mockSevenZipWrapper{
		extractFunc: func(params sevenzip.ExtractParams) error {
			extractCount++
			if extractCount == 2 {
				return fmt.Errorf("模拟解压失败")
			}
			return nil
		},
	}

	// 创建恢复协调器
	cfg := &config.Config{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Password:  "test123",
	}
	rc := NewRestoreCoordinator(cfg, idx, mockSevenZip, &mockLogger{})

	// 执行恢复
	report, err := rc.Execute(RestoreOptions{})
	if err != nil {
		t.Fatalf("恢复失败: %v", err)
	}

	// 验证报告
	if report.RestoredFiles != 2 {
		t.Errorf("期望恢复 2 个文件，实际 %d", report.RestoredFiles)
	}

	if report.FailedFiles != 1 {
		t.Errorf("期望 1 个失败文件，实际 %d", report.FailedFiles)
	}

	if len(report.Errors) != 1 {
		t.Errorf("期望 1 个错误，实际 %d", len(report.Errors))
	}
}

// TestExecute_EmptyTimestamps 测试空时间戳列表
func TestExecute_EmptyTimestamps(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("创建目标目录失败: %v", err)
	}

	// 不创建任何时间戳目录

	// 创建文件索引
	idx := index.NewFileIndex()

	mockSevenZip := &mockSevenZipWrapper{
		extractFunc: func(params sevenzip.ExtractParams) error {
			return nil
		},
	}

	// 创建恢复协调器
	cfg := &config.Config{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Password:  "test123",
	}
	rc := NewRestoreCoordinator(cfg, idx, mockSevenZip, &mockLogger{})

	// 执行恢复
	report, err := rc.Execute(RestoreOptions{})
	if err != nil {
		t.Fatalf("恢复失败: %v", err)
	}

	// 验证报告
	if report.RestoredFiles != 0 {
		t.Errorf("期望恢复 0 个文件，实际 %d", report.RestoredFiles)
	}

	if report.FailedFiles != 0 {
		t.Errorf("期望 0 个失败文件，实际 %d", report.FailedFiles)
	}
}

// mockSevenZipWrapper 是用于测试的mock 7zip wrapper
type mockSevenZipWrapper struct {
	detectFunc  func() (string, error)
	compressFunc func(params sevenzip.CompressParams) error
	extractFunc func(params sevenzip.ExtractParams) error
}

func (m *mockSevenZipWrapper) Detect() (string, error) {
	if m.detectFunc != nil {
		return m.detectFunc()
	}
	return "7z", nil
}

func (m *mockSevenZipWrapper) Compress(params sevenzip.CompressParams) error {
	if m.compressFunc != nil {
		return m.compressFunc(params)
	}
	return nil
}

func (m *mockSevenZipWrapper) Extract(params sevenzip.ExtractParams) error {
	if m.extractFunc != nil {
		return m.extractFunc(params)
	}
	return nil
}
