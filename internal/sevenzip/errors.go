package sevenzip

import (
	"errors"
	"fmt"
)

var (
	// Err7zipNotFound 7zip工具不存在错误
	Err7zipNotFound = errors.New("7zip工具不存在：未找到7z或7za命令")
)

// Err7zipCommandFailed 7zip命令执行失败错误
func Err7zipCommandFailed(details string) error {
	return fmt.Errorf("7zip命令执行失败: %s", details)
}
