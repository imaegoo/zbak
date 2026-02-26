package index

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// FileIndex 表示文件索引，记录所有已备份文件的信息
type FileIndex struct {
	Files map[string]FileEntry `yaml:"files"`
}

// FileEntry 表示单个文件的索引条目
type FileEntry struct {
	SourcePath   string    `yaml:"source_path"`
	Size         int64     `yaml:"size"`
	ModTime      time.Time `yaml:"mod_time"`
	ArchivePath  string    `yaml:"archive_path"`
	TimestampDir string    `yaml:"timestamp_dir"`
	Deleted      bool      `yaml:"deleted"`
}

// NewFileIndex 创建一个新的空文件索引
func NewFileIndex() *FileIndex {
	return &FileIndex{
		Files: make(map[string]FileEntry),
	}
}

// Load 从YAML文件加载文件索引
// 如果文件不存在，返回一个新的空索引
// 如果文件格式无效，返回错误
func Load(path string) (*FileIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，返回新的空索引
			return NewFileIndex(), nil
		}
		return nil, fmt.Errorf("读取索引文件失败: %w", err)
	}

	var index FileIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("解析索引文件失败: %w", err)
	}

	// 确保Files map已初始化
	if index.Files == nil {
		index.Files = make(map[string]FileEntry)
	}

	return &index, nil
}

// Save 将文件索引保存到YAML文件
func Save(path string, index *FileIndex) error {
	data, err := yaml.Marshal(index)
	if err != nil {
		return fmt.Errorf("序列化索引失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入索引文件失败: %w", err)
	}

	return nil
}
