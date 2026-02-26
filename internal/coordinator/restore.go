package coordinator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"zbak/internal/config"
	"zbak/internal/index"
	"zbak/internal/logger"
	"zbak/internal/sevenzip"
)

// RestoreCoordinator 恢复协调器
type RestoreCoordinator struct {
	config           *config.Config
	indexService     *index.FileIndex
	sevenZipWrapper  sevenzip.Wrapper
	timestampManager *TimestampManager
	logger           logger.Logger
}

// RestoreOptions 恢复选项
type RestoreOptions struct {
	Timestamp string // 单个时间戳
	FromTime  string // 时间戳范围起始
	ToTime    string // 时间戳范围结束
}

// ArchiveInfo 压缩文件信息
type ArchiveInfo struct {
	BaseName      string   // 基础名称（不含.7z.001后缀）
	FirstVolume   string   // 第一个分卷的完整路径
	AllVolumes    []string // 所有分卷的完整路径
	TimestampDir  string   // 所属时间戳目录
	RelativePath  string   // 在时间戳目录中的相对路径
}

// RestoreReport 恢复报告
type RestoreReport struct {
	StartTime     time.Time
	EndTime       time.Time
	RestoredFiles int
	DeletedFiles  int
	FailedFiles   int
	TotalSize     int64
	Errors        []error
}

// NewRestoreCoordinator 创建新的恢复协调器
func NewRestoreCoordinator(
	cfg *config.Config,
	idx *index.FileIndex,
	sevenZip sevenzip.Wrapper,
	log logger.Logger,
) *RestoreCoordinator {
	return &RestoreCoordinator{
		config:           cfg,
		indexService:     idx,
		sevenZipWrapper:  sevenZip,
		timestampManager: NewTimestampManager(cfg.TargetDir),
		logger:           log,
	}
}

// discoverTimestamps 发现并排序时间戳目录
// 返回按时间顺序排序的时间戳列表（从旧到新）
func (rc *RestoreCoordinator) discoverTimestamps() ([]string, error) {
	entries, err := os.ReadDir(rc.config.TargetDir)
	if err != nil {
		return nil, fmt.Errorf("读取目标目录失败: %w", err)
	}

	var timestamps []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 检查目录名是否符合时间戳格式 YYYY-MM-DD-HH-MM-SS
		name := entry.Name()
		if rc.isValidTimestamp(name) {
			timestamps = append(timestamps, name)
		}
	}

	// 按时间顺序排序（从旧到新）
	sort.Strings(timestamps)

	return timestamps, nil
}

// isValidTimestamp 检查字符串是否为有效的时间戳格式
func (rc *RestoreCoordinator) isValidTimestamp(s string) bool {
	_, err := time.Parse("2006-01-02-15-04-05", s)
	return err == nil
}

// filterTimestamps 根据恢复选项过滤时间戳列表
func (rc *RestoreCoordinator) filterTimestamps(timestamps []string, options RestoreOptions) ([]string, error) {
	// 如果指定了单个时间戳
	if options.Timestamp != "" {
		// 验证时间戳是否存在
		found := false
		for _, ts := range timestamps {
			if ts == options.Timestamp {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: %s", ErrTimestampNotFound, options.Timestamp)
		}
		return []string{options.Timestamp}, nil
	}

	// 如果指定了时间戳范围
	if options.FromTime != "" || options.ToTime != "" {
		var filtered []string
		for _, ts := range timestamps {
			// 检查是否在范围内
			if options.FromTime != "" && ts < options.FromTime {
				continue
			}
			if options.ToTime != "" && ts > options.ToTime {
				continue
			}
			filtered = append(filtered, ts)
		}
		return filtered, nil
	}

	// 如果没有指定任何选项，返回所有时间戳
	return timestamps, nil
}

// discoverArchives 发现时间戳目录中的所有压缩文件
// 识别.7z.001文件作为压缩包的起始文件，并关联所有分卷
func (rc *RestoreCoordinator) discoverArchives(timestampDir string) ([]ArchiveInfo, error) {
	timestampPath := rc.timestampManager.GetTimestampPath(timestampDir)
	
	// 存储已发现的压缩文件信息
	archiveMap := make(map[string]*ArchiveInfo)

	// 遍历时间戳目录
	err := filepath.Walk(timestampPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 检查是否为7z文件
		if !strings.HasSuffix(info.Name(), ".7z.001") && !strings.Contains(info.Name(), ".7z.") {
			return nil
		}

		// 获取相对于时间戳目录的路径
		relPath, err := filepath.Rel(timestampPath, path)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %w", err)
		}

		// 提取基础名称（去除.7z.XXX后缀）
		baseName := rc.extractBaseName(info.Name())
		baseDir := filepath.Dir(relPath)
		fullBaseName := filepath.Join(baseDir, baseName)

		// 如果是.7z.001文件，创建新的ArchiveInfo
		if strings.HasSuffix(info.Name(), ".7z.001") {
			archiveInfo := &ArchiveInfo{
				BaseName:     baseName,
				FirstVolume:  path,
				AllVolumes:   []string{path},
				TimestampDir: timestampDir,
				RelativePath: relPath,
			}
			archiveMap[fullBaseName] = archiveInfo
		} else {
			// 如果是其他分卷，添加到对应的ArchiveInfo
			if archiveInfo, exists := archiveMap[fullBaseName]; exists {
				archiveInfo.AllVolumes = append(archiveInfo.AllVolumes, path)
			} else {
				// 如果还没有找到.7z.001，先创建一个临时的ArchiveInfo
				archiveInfo := &ArchiveInfo{
					BaseName:     baseName,
					FirstVolume:  "", // 稍后会被.7z.001更新
					AllVolumes:   []string{path},
					TimestampDir: timestampDir,
					RelativePath: relPath,
				}
				archiveMap[fullBaseName] = archiveInfo
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历时间戳目录失败: %w", err)
	}

	// 转换为切片并排序
	var archives []ArchiveInfo
	for _, archiveInfo := range archiveMap {
		// 确保FirstVolume已设置
		if archiveInfo.FirstVolume == "" {
			// 如果没有找到.7z.001，使用第一个分卷
			if len(archiveInfo.AllVolumes) > 0 {
				sort.Strings(archiveInfo.AllVolumes)
				archiveInfo.FirstVolume = archiveInfo.AllVolumes[0]
			}
		} else {
			// 对分卷进行排序
			sort.Strings(archiveInfo.AllVolumes)
		}
		archives = append(archives, *archiveInfo)
	}

	// 按相对路径排序，确保处理顺序一致
	sort.Slice(archives, func(i, j int) bool {
		return archives[i].RelativePath < archives[j].RelativePath
	})

	return archives, nil
}

// extractBaseName 从文件名中提取基础名称（去除.7z.XXX后缀）
func (rc *RestoreCoordinator) extractBaseName(filename string) string {
	// 查找.7z的位置
	idx := strings.Index(filename, ".7z.")
	if idx == -1 {
		// 如果没有找到.7z.，尝试查找.7z.001
		if strings.HasSuffix(filename, ".7z.001") {
			return strings.TrimSuffix(filename, ".7z.001")
		}
		return filename
	}
	return filename[:idx]
}
