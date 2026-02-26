package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestConfigManager_Load_Success 测试成功加载有效配置文件
func TestConfigManager_Load_Success(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// 创建有效的配置文件
	validConfig := `source_dir: /path/to/source
target_dir: /path/to/target
volume_size: 4294967296
password: test_password
concurrency: 4
`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	// 加载配置
	cm := NewConfigManager()
	config, err := cm.Load(configPath)

	// 验证结果
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if config.SourceDir != "/path/to/source" {
		t.Errorf("SourceDir = %s; want /path/to/source", config.SourceDir)
	}
	if config.TargetDir != "/path/to/target" {
		t.Errorf("TargetDir = %s; want /path/to/target", config.TargetDir)
	}
	if config.VolumeSize != 4294967296 {
		t.Errorf("VolumeSize = %d; want 4294967296", config.VolumeSize)
	}
	if config.Password != "test_password" {
		t.Errorf("Password = %s; want test_password", config.Password)
	}
	if config.Concurrency != 4 {
		t.Errorf("Concurrency = %d; want 4", config.Concurrency)
	}
	if config.CompressionLevel == nil || *config.CompressionLevel != 1 {
		t.Errorf("CompressionLevel = %v; want 1", config.CompressionLevel)
	}
}

// TestConfigManager_Load_CompressionLevel 测试压缩级别配置
func TestConfigManager_Load_CompressionLevel(t *testing.T) {
	tests := []struct {
		name             string
		compressionLevel string
		expectedLevel    int
		expectError      bool
	}{
		{"默认值(未指定)", "", 1, false},
		{"有效值0", "0", 0, false},
		{"有效值1", "1", 1, false},
		{"有效值9", "9", 9, false},
		{"无效值-1", "-1", 0, true},
		{"无效值10", "10", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			configStr := `source_dir: /path/to/source
target_dir: /path/to/target
volume_size: 4294967296
password: test_password
concurrency: 4
`
			if tt.compressionLevel != "" {
				configStr += "compression_level: " + tt.compressionLevel + "\n"
			}

			if err := os.WriteFile(configPath, []byte(configStr), 0644); err != nil {
				t.Fatalf("创建测试配置文件失败: %v", err)
			}

			cm := NewConfigManager()
			config, err := cm.Load(configPath)

			if tt.expectError {
				if err == nil {
					t.Error("期望错误，但没有返回错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("加载配置失败: %v", err)
			}

			if config.CompressionLevel == nil {
				t.Fatal("CompressionLevel 为 nil")
			}

			if *config.CompressionLevel != tt.expectedLevel {
				t.Errorf("CompressionLevel = %d; want %d", *config.CompressionLevel, tt.expectedLevel)
			}
		})
	}
}

// TestConfigManager_Load_FileNotFound 测试配置文件不存在错误
func TestConfigManager_Load_FileNotFound(t *testing.T) {
	cm := NewConfigManager()
	_, err := cm.Load("/nonexistent/path/config.yaml")

	if err != ErrConfigNotFound {
		t.Errorf("期望错误 ErrConfigNotFound, 得到: %v", err)
	}
}

// TestConfigManager_Load_InvalidFormat 测试配置文件格式无效错误
func TestConfigManager_Load_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// 创建格式无效的配置文件
	invalidConfig := `this is not valid yaml: [[[`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cm := NewConfigManager()
	_, err := cm.Load(configPath)

	if err != ErrInvalidConfigFormat {
		t.Errorf("期望错误 ErrInvalidConfigFormat, 得到: %v", err)
	}
}

// TestConfigManager_Load_MissingSourceDir 测试缺失source_dir字段
func TestConfigManager_Load_MissingSourceDir(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config := `target_dir: /path/to/target
volume_size: 4294967296
password: test_password
concurrency: 4
`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cm := NewConfigManager()
	_, err := cm.Load(configPath)

	if err == nil {
		t.Fatal("期望错误，但没有返回错误")
	}
	if err.Error() != "必需字段缺失: source_dir" {
		t.Errorf("期望错误消息 '必需字段缺失: source_dir', 得到: %v", err)
	}
}

// TestConfigManager_Load_MissingTargetDir 测试缺失target_dir字段
func TestConfigManager_Load_MissingTargetDir(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config := `source_dir: /path/to/source
volume_size: 4294967296
password: test_password
concurrency: 4
`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cm := NewConfigManager()
	_, err := cm.Load(configPath)

	if err == nil {
		t.Fatal("期望错误，但没有返回错误")
	}
	if err.Error() != "必需字段缺失: target_dir" {
		t.Errorf("期望错误消息 '必需字段缺失: target_dir', 得到: %v", err)
	}
}

// TestConfigManager_Load_MissingVolumeSize 测试缺失volume_size字段
func TestConfigManager_Load_MissingVolumeSize(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config := `source_dir: /path/to/source
target_dir: /path/to/target
password: test_password
concurrency: 4
`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cm := NewConfigManager()
	_, err := cm.Load(configPath)

	if err == nil {
		t.Fatal("期望错误，但没有返回错误")
	}
	if err.Error() != "必需字段缺失: volume_size" {
		t.Errorf("期望错误消息 '必需字段缺失: volume_size', 得到: %v", err)
	}
}

// TestConfigManager_Load_InvalidVolumeSize 测试无效的volume_size值（负数或零）
func TestConfigManager_Load_InvalidVolumeSize(t *testing.T) {
	tests := []struct {
		name        string
		volumeSize  string
		expectError bool
	}{
		{"零值", "0", true},
		{"负数", "-1", true},
		{"有效值", "1024", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			config := `source_dir: /path/to/source
target_dir: /path/to/target
volume_size: ` + tt.volumeSize + `
password: test_password
concurrency: 4
`
			if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
				t.Fatalf("创建测试配置文件失败: %v", err)
			}

			cm := NewConfigManager()
			_, err := cm.Load(configPath)

			if tt.expectError && err == nil {
				t.Error("期望错误，但没有返回错误")
			}
			if !tt.expectError && err != nil {
				t.Errorf("不期望错误，但得到: %v", err)
			}
		})
	}
}

// TestConfigManager_Load_MissingPassword 测试缺失password字段
func TestConfigManager_Load_MissingPassword(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config := `source_dir: /path/to/source
target_dir: /path/to/target
volume_size: 4294967296
concurrency: 4
`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cm := NewConfigManager()
	_, err := cm.Load(configPath)

	if err == nil {
		t.Fatal("期望错误，但没有返回错误")
	}
	if err.Error() != "必需字段缺失: password" {
		t.Errorf("期望错误消息 '必需字段缺失: password', 得到: %v", err)
	}
}

// TestConfigManager_Load_MissingConcurrency 测试缺失concurrency字段
func TestConfigManager_Load_MissingConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config := `source_dir: /path/to/source
target_dir: /path/to/target
volume_size: 4294967296
password: test_password
`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cm := NewConfigManager()
	_, err := cm.Load(configPath)

	if err == nil {
		t.Fatal("期望错误，但没有返回错误")
	}
	if err.Error() != "必需字段缺失: concurrency" {
		t.Errorf("期望错误消息 '必需字段缺失: concurrency', 得到: %v", err)
	}
}

// TestConfigManager_Load_InvalidConcurrency 测试无效的concurrency值（小于1）
func TestConfigManager_Load_InvalidConcurrency(t *testing.T) {
	tests := []struct {
		name        string
		concurrency string
		expectError bool
	}{
		{"零值", "0", true},
		{"负数", "-1", true},
		{"有效值1", "1", false},
		{"有效值4", "4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			config := `source_dir: /path/to/source
target_dir: /path/to/target
volume_size: 4294967296
password: test_password
concurrency: ` + tt.concurrency + `
`
			if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
				t.Fatalf("创建测试配置文件失败: %v", err)
			}

			cm := NewConfigManager()
			_, err := cm.Load(configPath)

			if tt.expectError && err == nil {
				t.Error("期望错误，但没有返回错误")
			}
			if !tt.expectError && err != nil {
				t.Errorf("不期望错误，但得到: %v", err)
			}
		})
	}
}

// TestConfigManager_Validate_AllFieldsValid 测试所有字段都有效的配置
func TestConfigManager_Validate_AllFieldsValid(t *testing.T) {
	cm := NewConfigManager()
	config := &Config{
		SourceDir:   "/path/to/source",
		TargetDir:   "/path/to/target",
		VolumeSize:  4294967296,
		Password:    "test_password",
		Concurrency: 4,
	}

	err := cm.Validate(config)
	if err != nil {
		t.Errorf("验证有效配置失败: %v", err)
	}
}

// TestConfigManager_Validate_EmptyFields 测试空字段验证
func TestConfigManager_Validate_EmptyFields(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectedErr string
	}{
		{
			name: "空source_dir",
			config: &Config{
				SourceDir:   "",
				TargetDir:   "/path/to/target",
				VolumeSize:  4294967296,
				Password:    "test_password",
				Concurrency: 4,
			},
			expectedErr: "必需字段缺失: source_dir",
		},
		{
			name: "空target_dir",
			config: &Config{
				SourceDir:   "/path/to/source",
				TargetDir:   "",
				VolumeSize:  4294967296,
				Password:    "test_password",
				Concurrency: 4,
			},
			expectedErr: "必需字段缺失: target_dir",
		},
		{
			name: "空password",
			config: &Config{
				SourceDir:   "/path/to/source",
				TargetDir:   "/path/to/target",
				VolumeSize:  4294967296,
				Password:    "",
				Concurrency: 4,
			},
			expectedErr: "必需字段缺失: password",
		},
	}

	cm := NewConfigManager()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.Validate(tt.config)
			if err == nil {
				t.Fatal("期望错误，但没有返回错误")
			}
			if err.Error() != tt.expectedErr {
				t.Errorf("期望错误消息 '%s', 得到: %v", tt.expectedErr, err)
			}
		})
	}
}

// TestConfigManager_Validate_MinimumConcurrency 测试并发数最小值为1
func TestConfigManager_Validate_MinimumConcurrency(t *testing.T) {
	cm := NewConfigManager()
	config := &Config{
		SourceDir:   "/path/to/source",
		TargetDir:   "/path/to/target",
		VolumeSize:  4294967296,
		Password:    "test_password",
		Concurrency: 1,
	}

	err := cm.Validate(config)
	if err != nil {
		t.Errorf("并发数为1应该有效，但得到错误: %v", err)
	}
}
