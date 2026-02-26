package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 表示应用程序配置
type Config struct {
	SourceDir   string `yaml:"source_dir"`
	TargetDir   string `yaml:"target_dir"`
	VolumeSize  int64  `yaml:"volume_size"`
	Password    string `yaml:"password"`
	Concurrency int    `yaml:"concurrency"`
}

// ConfigManager 配置管理器接口
type ConfigManager interface {
	Load(path string) (*Config, error)
	Validate(config *Config) error
}

// configManager 配置管理器实现
type configManager struct{}

// NewConfigManager 创建新的配置管理器
func NewConfigManager() ConfigManager {
	return &configManager{}
}

// Load 从YAML文件加载配置
func (cm *configManager) Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, ErrInvalidConfigFormat
	}

	if err := cm.Validate(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate 验证配置
func (cm *configManager) Validate(config *Config) error {
	if config.SourceDir == "" {
		return ErrMissingRequiredField("source_dir")
	}
	if config.TargetDir == "" {
		return ErrMissingRequiredField("target_dir")
	}
	if config.VolumeSize <= 0 {
		return ErrMissingRequiredField("volume_size")
	}
	if config.Password == "" {
		return ErrMissingRequiredField("password")
	}
	if config.Concurrency < 1 {
		return ErrMissingRequiredField("concurrency")
	}
	return nil
}
