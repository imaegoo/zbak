package index

import (
	"fmt"
	"log"
	"sync"
)

// IndexService 提供线程安全的文件索引操作服务
type IndexService struct {
	mu sync.Mutex
}

// NewIndexService 创建一个新的索引服务实例
func NewIndexService() *IndexService {
	return &IndexService{}
}

// Load 从指定路径加载文件索引
// 如果文件不存在，返回一个新的空索引
// 如果文件格式无效，记录警告并返回新的空索引
func (s *IndexService) Load(path string) (*FileIndex, error) {
	index, err := Load(path)
	if err != nil {
		// 需求2.9: 当文件索引格式无效时，记录警告并创建新的文件索引
		log.Printf("警告: 加载索引文件失败: %v，将创建新的索引", err)
		return NewFileIndex(), nil
	}
	return index, nil
}

// Save 将文件索引保存到指定路径
func (s *IndexService) Save(path string, index *FileIndex) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	return Save(path, index)
}

// AddEntry 向索引中添加一个文件条目
// 此方法是线程安全的
func (s *IndexService) AddEntry(index *FileIndex, entry FileEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if index == nil {
		return fmt.Errorf("索引不能为nil")
	}
	
	if index.Files == nil {
		index.Files = make(map[string]FileEntry)
	}
	
	index.Files[entry.SourcePath] = entry
	return nil
}

// MarkDeleted 在索引中标记文件为已删除
// 此方法是线程安全的
func (s *IndexService) MarkDeleted(index *FileIndex, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if index == nil {
		return fmt.Errorf("索引不能为nil")
	}
	
	if index.Files == nil {
		return fmt.Errorf("文件 %s 在索引中不存在", path)
	}
	
	entry, exists := index.Files[path]
	if !exists {
		return fmt.Errorf("文件 %s 在索引中不存在", path)
	}
	
	entry.Deleted = true
	index.Files[path] = entry
	return nil
}

// GetEntry 从索引中获取指定路径的文件条目
// 返回条目和是否存在的标志
func (s *IndexService) GetEntry(index *FileIndex, path string) (*FileEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if index == nil || index.Files == nil {
		return nil, false
	}
	
	entry, exists := index.Files[path]
	if !exists {
		return nil, false
	}
	
	return &entry, true
}
