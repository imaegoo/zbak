package config

import (
	"errors"
	"fmt"
)

var (
	// ErrConfigNotFound 配置文件不存在错误
	ErrConfigNotFound = errors.New("配置文件不存在")

	// ErrInvalidConfigFormat 配置格式无效错误
	ErrInvalidConfigFormat = errors.New("配置文件格式无效")
)

// ErrMissingRequiredField 必需字段缺失错误
func ErrMissingRequiredField(field string) error {
	return fmt.Errorf("必需字段缺失: %s", field)
}
