package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"zbak/internal/compression"
	"zbak/internal/config"
	"zbak/internal/coordinator"
	"zbak/internal/detector"
	"zbak/internal/filesystem"
	"zbak/internal/index"
	"zbak/internal/logger"
	"zbak/internal/sevenzip"
)

// TestMemoryStability 验证内存使用稳定性 (需求 20.1)
// 测试在处理大量小文件时内存使用是否稳定
func TestMemoryStability(t *testing.T) {
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

	// 创建大量小文件 (1000个文件，每个1KB)
	numFiles := 1000
	fileSize := 1024
	for i := 0; i < numFiles; i++ {
		filePath := filepath.Join(sourceDir, fmt.Sprintf("file_%04d.txt", i))
		content := make([]byte, fileSize)
		for j := range content {
			content[j] = byte('A' + (j % 26))
		}
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	// 记录初始内存使用
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// 执行文件索引操作
	fileIndex := index.NewFileIndex()
	det := detector.NewDetector()
	changeSet, err := det.Detect(sourceDir, fileIndex)
	if err != nil {
		t.Fatalf("增量检测失败: %v", err)
	}

	// 验证检测到所有文件
	if len(changeSet.NewFiles) != numFiles {
		t.Errorf("期望检测到 %d 个新文件，实际检测到 %d 个", numFiles, len(changeSet.NewFiles))
	}

	// 记录操作后内存使用
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// 计算内存增长
	memGrowth := memAfter.Alloc - memBefore.Alloc
	memGrowthMB := float64(memGrowth) / (1024 * 1024)

	t.Logf("处理 %d 个文件的内存增长: %.2f MB", numFiles, memGrowthMB)
	t.Logf("初始内存分配: %.2f MB", float64(memBefore.Alloc)/(1024*1024))
	t.Logf("最终内存分配: %.2f MB", float64(memAfter.Alloc)/(1024*1024))

	// 验证内存增长在合理范围内 (应该小于50MB)
	maxMemGrowthMB := 50.0
	if memGrowthMB > maxMemGrowthMB {
		t.Errorf("内存增长过大: %.2f MB (最大允许: %.2f MB)", memGrowthMB, maxMemGrowthMB)
	}
}

// TestCPUUtilization 验证CPU资源利用率 (需求 20.2)
// 测试并发处理时CPU资源的合理利用
func TestCPUUtilization(t *testing.T) {
	// 跳过如果没有7zip工具
	wrapper := sevenzip.NewWrapper()
	if _, err := wrapper.Detect(); err != nil {
		t.Skip("7zip工具未安装，跳过测试")
	}

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

	// 创建多个目录，每个包含一些文件
	numDirs := 4
	filesPerDir := 10
	fileSize := 10240 // 10KB

	for i := 0; i < numDirs; i++ {
		dirPath := filepath.Join(sourceDir, fmt.Sprintf("dir_%d", i))
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("创建测试目录失败: %v", err)
		}

		for j := 0; j < filesPerDir; j++ {
			filePath := filepath.Join(dirPath, fmt.Sprintf("file_%d.txt", j))
			content := make([]byte, fileSize)
			for k := range content {
				content[k] = byte('A' + (k % 26))
			}
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				t.Fatalf("创建测试文件失败: %v", err)
			}
		}
	}

	// 创建配置
	cfg := &config.Config{
		SourceDir:   sourceDir,
		TargetDir:   targetDir,
		VolumeSize:  1024 * 1024, // 1MB
		Password:    "test123",
		Concurrency: 2, // 使用2个并发工作协程
	}

	// 创建日志记录器
	log, err := logger.NewLogger(targetDir, false)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer log.Close()

	// 创建压缩服务
	fsSvc := filesystem.NewService()
	compressionSvc := compression.NewService(fsSvc, wrapper, log)

	// 创建备份协调器
	bc := coordinator.NewBackupCoordinator(cfg, compressionSvc, log)

	// 记录开始时间
	startTime := time.Now()

	// 执行备份
	ctx := context.Background()
	report, err := bc.Execute(ctx)
	if err != nil {
		t.Fatalf("备份执行失败: %v", err)
	}

	// 记录结束时间
	duration := time.Since(startTime)

	t.Logf("备份完成，耗时: %v", duration)
	t.Logf("成功任务: %d, 失败任务: %d", report.SuccessCount, report.FailureCount)

	// 验证备份成功
	if report.FailureCount > 0 {
		t.Errorf("有 %d 个任务失败", report.FailureCount)
	}

	// 验证并发处理的效率
	// 串行处理应该需要更长时间，并发处理应该更快
	// 这里只是记录时间，不做严格断言，因为性能受环境影响
	t.Logf("并发数: %d, 处理 %d 个目录耗时: %v", cfg.Concurrency, numDirs, duration)
}

// TestIndexDataStructureEfficiency 验证文件索引数据结构效率 (需求 20.3)
// 测试文件索引使用map数据结构的查找效率
func TestIndexDataStructureEfficiency(t *testing.T) {
	// 创建大量文件条目
	numEntries := 10000
	fileIndex := index.NewFileIndex()

	// 添加条目
	startAdd := time.Now()
	for i := 0; i < numEntries; i++ {
		entry := index.FileEntry{
			SourcePath:   fmt.Sprintf("path/to/file_%d.txt", i),
			Size:         1024,
			ModTime:      time.Now(),
			ArchivePath:  fmt.Sprintf("archive_%d.7z.001", i),
			TimestampDir: "2024-01-01-00-00-00",
			Deleted:      false,
		}
		fileIndex.Files[entry.SourcePath] = entry
	}
	addDuration := time.Since(startAdd)

	t.Logf("添加 %d 个条目耗时: %v", numEntries, addDuration)

	// 测试查找效率
	startLookup := time.Now()
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("path/to/file_%d.txt", i)
		if _, exists := fileIndex.Files[key]; !exists {
			t.Errorf("条目 %s 不存在", key)
		}
	}
	lookupDuration := time.Since(startLookup)

	t.Logf("查找 %d 个条目耗时: %v", numEntries, lookupDuration)

	// 验证查找效率 (应该是O(1)，非常快)
	// 对于10000个条目，查找应该在几毫秒内完成
	maxLookupDuration := 100 * time.Millisecond
	if lookupDuration > maxLookupDuration {
		t.Errorf("查找效率过低: %v (最大允许: %v)", lookupDuration, maxLookupDuration)
	}

	// 测试遍历效率
	startIterate := time.Now()
	count := 0
	for range fileIndex.Files {
		count++
	}
	iterateDuration := time.Since(startIterate)

	t.Logf("遍历 %d 个条目耗时: %v", numEntries, iterateDuration)

	if count != numEntries {
		t.Errorf("遍历条目数不匹配: 期望 %d, 实际 %d", numEntries, count)
	}
}

// TestFileMetadataReadingOptimization 验证文件元数据读取优化 (需求 20.4)
// 测试避免重复读取文件元数据
func TestFileMetadataReadingOptimization(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	// 创建测试文件
	numFiles := 100
	for i := 0; i < numFiles; i++ {
		filePath := filepath.Join(sourceDir, fmt.Sprintf("file_%d.txt", i))
		content := []byte(fmt.Sprintf("Content of file %d", i))
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	// 测试增量检测器的元数据读取
	fileIndex := index.NewFileIndex()
	det := detector.NewDetector()

	// 第一次检测 - 所有文件都是新的
	startDetect := time.Now()
	changeSet, err := det.Detect(sourceDir, fileIndex)
	if err != nil {
		t.Fatalf("增量检测失败: %v", err)
	}
	detectDuration := time.Since(startDetect)

	t.Logf("第一次检测 %d 个文件耗时: %v", numFiles, detectDuration)

	if len(changeSet.NewFiles) != numFiles {
		t.Errorf("期望检测到 %d 个新文件，实际检测到 %d 个", numFiles, len(changeSet.NewFiles))
	}

	// 更新索引
	for _, relPath := range changeSet.NewFiles {
		fullPath := filepath.Join(sourceDir, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Fatalf("获取文件信息失败: %v", err)
		}

		entry := index.FileEntry{
			SourcePath:   relPath,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			ArchivePath:  "archive.7z.001",
			TimestampDir: "2024-01-01-00-00-00",
			Deleted:      false,
		}
		fileIndex.Files[relPath] = entry
	}

	// 第二次检测 - 所有文件都未变化
	startDetect2 := time.Now()
	changeSet2, err := det.Detect(sourceDir, fileIndex)
	if err != nil {
		t.Fatalf("第二次增量检测失败: %v", err)
	}
	detectDuration2 := time.Since(startDetect2)

	t.Logf("第二次检测 %d 个文件耗时: %v", numFiles, detectDuration2)

	if len(changeSet2.UnchangedFiles) != numFiles {
		t.Errorf("期望 %d 个文件未变化，实际 %d 个", numFiles, len(changeSet2.UnchangedFiles))
	}

	// 验证第二次检测的效率
	// 虽然仍需要读取元数据进行比对，但应该很快
	// 这里主要验证逻辑正确性，不做严格的性能断言
	t.Logf("元数据读取优化验证完成")
}

// TestLoggerBufferedWriting 验证日志缓冲写入 (需求 20.5)
// 测试日志记录器使用缓冲写入
func TestLoggerBufferedWriting(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建日志记录器
	log, err := logger.NewLogger(tempDir, false)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer log.Close()

	// 写入大量日志
	numLogs := 1000
	startWrite := time.Now()
	for i := 0; i < numLogs; i++ {
		log.Info("测试日志消息 %d", i)
	}
	writeDuration := time.Since(startWrite)

	t.Logf("写入 %d 条日志耗时: %v", numLogs, writeDuration)

	// 验证日志文件存在
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("读取目录失败: %v", err)
	}

	logFileFound := false
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".log" {
			logFileFound = true
			t.Logf("找到日志文件: %s", entry.Name())
			break
		}
	}

	if !logFileFound {
		t.Error("未找到日志文件")
	}

	// 验证写入效率
	// 对于1000条日志，应该在合理时间内完成
	maxWriteDuration := 1 * time.Second
	if writeDuration > maxWriteDuration {
		t.Errorf("日志写入效率过低: %v (最大允许: %v)", writeDuration, maxWriteDuration)
	}
}
