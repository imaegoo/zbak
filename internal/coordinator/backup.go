package coordinator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"zbak/internal/compression"
	"zbak/internal/config"
	"zbak/internal/detector"
	"zbak/internal/index"
	"zbak/internal/logger"
)

// BackupReport represents the result of a backup operation
type BackupReport struct {
	StartTime      time.Time
	EndTime        time.Time
	TotalFiles     int
	NewFiles       int
	ModifiedFiles  int
	DeletedFiles   int
	UnchangedFiles int
	SuccessCount   int
	FailureCount   int
	TotalSize      int64
	Errors         []error
}

// BackupCoordinator orchestrates the entire backup process
type BackupCoordinator struct {
	config           *config.Config
	indexService     *index.FileIndex
	detector         *detector.Detector
	compressionSvc   *compression.Service
	timestampMgr     *TimestampManager
	logger           logger.Logger
	indexPath        string
	mu               sync.Mutex // Protects index updates
}

// NewBackupCoordinator creates a new BackupCoordinator instance
func NewBackupCoordinator(
	cfg *config.Config,
	compressionSvc *compression.Service,
	log logger.Logger,
) *BackupCoordinator {
	return &BackupCoordinator{
		config:         cfg,
		detector:       detector.NewDetector(),
		compressionSvc: compressionSvc,
		timestampMgr:   NewTimestampManager(cfg.TargetDir),
		logger:         log,
		indexPath:      filepath.Join(cfg.TargetDir, "index.yaml"),
	}
}

// Execute performs the complete backup operation
// Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 12.5, 12.6, 12.7
func (bc *BackupCoordinator) Execute(ctx context.Context) (*BackupReport, error) {
	report := &BackupReport{
		StartTime: time.Now(),
		Errors:    make([]error, 0),
	}

	bc.logger.Info("开始备份操作")

	// Step 1: Load file index
	bc.logger.Info("加载文件索引: %s", bc.indexPath)
	fileIndex, err := index.Load(bc.indexPath)
	if err != nil {
		return nil, fmt.Errorf("加载文件索引失败: %w", err)
	}
	bc.indexService = fileIndex

	// Step 2: Execute incremental detection
	bc.logger.Info("执行增量检测: %s", bc.config.SourceDir)
	changeSet, err := bc.detector.Detect(bc.config.SourceDir, fileIndex)
	if err != nil {
		return nil, fmt.Errorf("增量检测失败: %w", err)
	}

	// Update report statistics
	report.NewFiles = len(changeSet.NewFiles)
	report.ModifiedFiles = len(changeSet.ModifiedFiles)
	report.DeletedFiles = len(changeSet.DeletedFiles)
	report.UnchangedFiles = len(changeSet.UnchangedFiles)
	report.TotalFiles = report.NewFiles + report.ModifiedFiles + report.DeletedFiles + report.UnchangedFiles

	bc.logger.Info("增量检测完成: 新增=%d, 修改=%d, 删除=%d, 未变化=%d",
		report.NewFiles, report.ModifiedFiles, report.DeletedFiles, report.UnchangedFiles)

	// Check if there are any changes
	if report.NewFiles == 0 && report.ModifiedFiles == 0 && report.DeletedFiles == 0 {
		bc.logger.Info("没有文件变化，跳过备份")
		report.EndTime = time.Now()
		return report, nil
	}

	// Step 3: Create timestamp directory
	timestamp, err := bc.timestampMgr.CreateTimestampDir(time.Now())
	if err != nil {
		return nil, fmt.Errorf("创建时间戳目录失败: %w", err)
	}
	bc.logger.Info("创建时间戳目录: %s", timestamp)

	// Step 4: Build compression tasks
	bc.logger.Info("构建压缩任务")
	tasks, err := bc.buildCompressionTasks(changeSet, timestamp)
	if err != nil {
		return nil, fmt.Errorf("构建压缩任务失败: %w", err)
	}
	bc.logger.Info("构建了 %d 个压缩任务", len(tasks))

	// Step 5: Start worker pool to execute tasks
	bc.logger.Info("启动工作池，并发数: %d", bc.config.Concurrency)
	workerPool := compression.NewWorkerPool(bc.compressionSvc, bc.config.Concurrency)
	workerPool.Start(ctx)

	// Submit all tasks
	for _, task := range tasks {
		workerPool.Submit(task)
	}

	// Wait for all tasks to complete
	errors := workerPool.Wait()
	
	// Collect errors
	report.Errors = errors
	report.FailureCount = len(errors)
	report.SuccessCount = len(tasks) - report.FailureCount

	// Log errors
	for _, err := range errors {
		bc.logger.Error("压缩任务失败: %v", err)
	}

	bc.logger.Info("压缩任务完成: 成功=%d, 失败=%d", report.SuccessCount, report.FailureCount)

	// Step 6: Update file index
	bc.logger.Info("更新文件索引")
	if err := bc.updateFileIndex(changeSet, timestamp, tasks); err != nil {
		return nil, fmt.Errorf("更新文件索引失败: %w", err)
	}

	// Save updated index
	if err := index.Save(bc.indexPath, bc.indexService); err != nil {
		return nil, fmt.Errorf("保存文件索引失败: %w", err)
	}
	bc.logger.Info("文件索引已保存")

	// Step 7: Generate backup report
	report.EndTime = time.Now()
	bc.generateBackupReport(report)

	bc.logger.Info("备份操作完成")
	return report, nil
}

// buildCompressionTasks builds compression tasks for changed files
// Groups files by their parent directory and determines compression strategy
func (bc *BackupCoordinator) buildCompressionTasks(changeSet *detector.ChangeSet, timestamp string) ([]compression.CompressionTask, error) {
	tasks := make([]compression.CompressionTask, 0)

	// Combine new and modified files
	changedFiles := append(changeSet.NewFiles, changeSet.ModifiedFiles...)
	if len(changedFiles) == 0 {
		return tasks, nil
	}

	// Group files by their parent directory
	dirMap := make(map[string][]string)
	for _, relPath := range changedFiles {
		dir := filepath.Dir(relPath)
		if dir == "." {
			dir = ""
		}
		dirMap[dir] = append(dirMap[dir], relPath)
	}

	// Build tasks for each directory
	for dir := range dirMap {
		// Determine the source directory path
		var sourcePath string
		if dir == "" {
			// Files in root directory
			sourcePath = bc.config.SourceDir
		} else {
			sourcePath = filepath.Join(bc.config.SourceDir, dir)
		}

		// Check if source path is a directory
		info, err := os.Stat(sourcePath)
		if err != nil {
			bc.logger.Warn("跳过无法访问的路径: %s, 错误: %v", sourcePath, err)
			continue
		}

		if !info.IsDir() {
			// If it's a file, we need to handle it differently
			// This shouldn't happen in normal cases, but handle it gracefully
			bc.logger.Warn("路径不是目录: %s", sourcePath)
			continue
		}

		// Determine compression strategy
		strategy, err := bc.compressionSvc.DetermineStrategy(sourcePath, bc.config.VolumeSize)
		if err != nil {
			bc.logger.Warn("无法确定压缩策略: %s, 错误: %v", sourcePath, err)
			continue
		}

		// Create target path in timestamp directory
		var targetPath string
		if dir == "" {
			targetPath = bc.timestampMgr.GetTimestampPath(timestamp)
		} else {
			targetPath = filepath.Join(bc.timestampMgr.GetTimestampPath(timestamp), dir)
		}

		// Create compression task
		task := compression.CompressionTask{
			SourcePath: sourcePath,
			TargetPath: targetPath,
			Password:   bc.config.Password,
			VolumeSize: bc.config.VolumeSize,
			Strategy:   strategy,
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// updateFileIndex updates the file index with backup results
// Requirements: 9.5 - Thread-safe index updates
func (bc *BackupCoordinator) updateFileIndex(changeSet *detector.ChangeSet, timestamp string, tasks []compression.CompressionTask) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Mark deleted files
	for _, relPath := range changeSet.DeletedFiles {
		if entry, exists := bc.indexService.Files[relPath]; exists {
			entry.Deleted = true
			bc.indexService.Files[relPath] = entry
		}
	}

	// Update entries for new and modified files
	changedFiles := append(changeSet.NewFiles, changeSet.ModifiedFiles...)
	for _, relPath := range changedFiles {
		// Get file info
		fullPath := filepath.Join(bc.config.SourceDir, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			bc.logger.Warn("无法获取文件信息: %s, 错误: %v", relPath, err)
			continue
		}

		// Determine archive path
		// The archive path is the directory where the file was compressed
		dir := filepath.Dir(relPath)
		var archivePath string
		if dir == "." || dir == "" {
			archivePath = filepath.Join(timestamp, filepath.Base(bc.config.SourceDir)+".7z.001")
		} else {
			// Find the appropriate archive based on directory structure
			archivePath = filepath.Join(timestamp, dir+".7z.001")
		}

		// Create or update file entry
		entry := index.FileEntry{
			SourcePath:   relPath,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			ArchivePath:  archivePath,
			TimestampDir: timestamp,
			Deleted:      false,
		}

		bc.indexService.Files[relPath] = entry
	}

	return nil
}

// generateBackupReport logs the backup report
// Requirements: 12.5, 12.6, 12.7
func (bc *BackupCoordinator) generateBackupReport(report *BackupReport) {
	duration := report.EndTime.Sub(report.StartTime)
	
	bc.logger.Info("========== 备份报告 ==========")
	bc.logger.Info("开始时间: %s", report.StartTime.Format("2006-01-02 15:04:05"))
	bc.logger.Info("结束时间: %s", report.EndTime.Format("2006-01-02 15:04:05"))
	bc.logger.Info("耗时: %s", duration)
	bc.logger.Info("总文件数: %d", report.TotalFiles)
	bc.logger.Info("新增文件: %d", report.NewFiles)
	bc.logger.Info("修改文件: %d", report.ModifiedFiles)
	bc.logger.Info("删除文件: %d", report.DeletedFiles)
	bc.logger.Info("未变化文件: %d", report.UnchangedFiles)
	bc.logger.Info("成功任务: %d", report.SuccessCount)
	bc.logger.Info("失败任务: %d", report.FailureCount)
	bc.logger.Info("==============================")
}
