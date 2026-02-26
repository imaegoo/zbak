package detector

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"zbak/internal/index"
)

// ChangeSet 表示文件变化集合
type ChangeSet struct {
	NewFiles       []string // 新增文件列表
	ModifiedFiles  []string // 修改文件列表
	DeletedFiles   []string // 删除文件列表
	UnchangedFiles []string // 未变化文件列表
}

// Detector 提供增量检测功能
type Detector struct{}

// NewDetector 创建一个新的增量检测器
func NewDetector() *Detector {
	return &Detector{}
}

// Detect 检测源目录和文件索引之间的变化
// 返回变化集，包含新增、修改、删除和未变化的文件
func (d *Detector) Detect(sourceDir string, fileIndex *index.FileIndex) (*ChangeSet, error) {
	changeSet := &ChangeSet{
		NewFiles:       []string{},
		ModifiedFiles:  []string{},
		DeletedFiles:   []string{},
		UnchangedFiles: []string{},
	}

	// 遍历源目录中的所有文件
	sourceFiles := make(map[string]bool)
	err := filepath.Walk(sourceDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录和符号链接
		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %w", err)
		}

		// Normalize path to use forward slashes for consistency
		relPath = filepath.ToSlash(relPath)

		// 标记文件存在
		sourceFiles[relPath] = true

		// 检查文件是否在索引中
		entry, exists := fileIndex.Files[relPath]
		if !exists {
			// 文件在索引中不存在，标记为新增
			changeSet.NewFiles = append(changeSet.NewFiles, relPath)
			return nil
		}

		// 检查文件是否被标记为已删除
		if entry.Deleted {
			// 文件之前被删除，现在又出现了，标记为新增
			changeSet.NewFiles = append(changeSet.NewFiles, relPath)
			return nil
		}

		// 比对文件大小和修改时间
		if info.Size() != entry.Size || !info.ModTime().Equal(entry.ModTime) {
			// 文件大小或修改时间不同，标记为修改
			changeSet.ModifiedFiles = append(changeSet.ModifiedFiles, relPath)
			return nil
		}

		// 文件未发生变化
		changeSet.UnchangedFiles = append(changeSet.UnchangedFiles, relPath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历源目录失败: %w", err)
	}

	// 检查索引中的文件是否在源目录中存在
	for relPath, entry := range fileIndex.Files {
		// 跳过已经标记为删除的文件
		if entry.Deleted {
			continue
		}

		// If文件不在源目录中，标记为删除
		if !sourceFiles[relPath] {
			changeSet.DeletedFiles = append(changeSet.DeletedFiles, relPath)
		}
	}

	return changeSet, nil
}
