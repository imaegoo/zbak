package sevenzip

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Wrapper 7zip包装器接口
type Wrapper interface {
	Detect() (string, error)
	Compress(params CompressParams) error
	Extract(params ExtractParams) error
}

// CompressParams 压缩参数
type CompressParams struct {
	Sources    []string // 源文件或目录列表
	Output     string   // 输出文件路径
	Password         string   // 加密密码
	VolumeSize       int64    // 分卷大小（字节）
	CompressionLevel int      // 压缩级别（0-9）
}

// ExtractParams 解压参数
type ExtractParams struct {
	Archive   string // 压缩文件路径
	OutputDir string // 输出目录
	Password  string // 解密密码
}

// wrapper 7zip包装器实现
type wrapper struct {
	command string // 7zip命令路径（7z或7za）
}

// NewWrapper 创建新的7zip包装器
func NewWrapper() Wrapper {
	return &wrapper{}
}

// Detect 检测系统中可用的7zip命令行工具
// 返回找到的命令名称（7z或7za）和错误
func (w *wrapper) Detect() (string, error) {
	// 首先尝试7z
	if _, err := exec.LookPath("7z"); err == nil {
		w.command = "7z"
		return "7z", nil
	}

	// 然后尝试7za
	if _, err := exec.LookPath("7za"); err == nil {
		w.command = "7za"
		return "7za", nil
	}

	return "", Err7zipNotFound
}

// Compress 执行压缩操作
func (w *wrapper) Compress(params CompressParams) error {
	// 确保已检测到7zip命令
	if w.command == "" {
		if _, err := w.Detect(); err != nil {
			return err
		}
	}

	// 构建命令参数
	args := []string{
		"a",                                            // 添加到压缩文件
		"-t7z",                                         // 使用7z格式
		"-mx=" + strconv.Itoa(params.CompressionLevel), // 压缩级别
		"-mhe=on",                                      // 加密文件头
		"-p" + params.Password,                         // 密码
	}

	// 添加分卷大小参数（如果指定）
	if params.VolumeSize > 0 {
		args = append(args, "-v"+strconv.FormatInt(params.VolumeSize, 10)+"b")
	}

	// 添加输出文件
	args = append(args, params.Output)

	// 添加源文件或目录
	args = append(args, params.Sources...)

	// 执行命令
	cmd := exec.Command(w.command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Err7zipCommandFailed(
			fmt.Sprintf("压缩失败: %v\nStdout: %s\nStderr: %s",
				err, stdout.String(), stderr.String()),
		)
	}

	return nil
}

// Extract 执行解压操作
func (w *wrapper) Extract(params ExtractParams) error {
	// 确保已检测到7zip命令
	if w.command == "" {
		if _, err := w.Detect(); err != nil {
			return err
		}
	}

	// 构建命令参数
	args := []string{
		"x",                    // 解压并保持目录结构
		"-p" + params.Password, // 密码
		"-o" + params.OutputDir, // 输出目录（注意：-o和路径之间没有空格）
		"-y",                   // 对所有提示回答yes
		params.Archive,         // 压缩文件
	}

	// 执行命令
	cmd := exec.Command(w.command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Err7zipCommandFailed(
			fmt.Sprintf("解压失败: %v\nStdout: %s\nStderr: %s",
				err, stdout.String(), stderr.String()),
		)
	}

	return nil
}

// buildCommand 构建7zip命令（用于测试）
func (w *wrapper) buildCommand(operation string, params interface{}) []string {
	var args []string

	switch operation {
	case "compress":
		p := params.(CompressParams)
		args = []string{
			"a",
			"-t7z",
			"-mx=" + strconv.Itoa(p.CompressionLevel),
			"-mhe=on",
			"-p" + p.Password,
		}
		if p.VolumeSize > 0 {
			args = append(args, "-v"+strconv.FormatInt(p.VolumeSize, 10)+"b")
		}
		args = append(args, p.Output)
		args = append(args, p.Sources...)

	case "extract":
		p := params.(ExtractParams)
		args = []string{
			"x",
			"-p" + p.Password,
			"-o" + p.OutputDir,
			"-y",
			p.Archive,
		}
	}

	return args
}

// parseOutput 解析7zip输出（用于错误处理）
func parseOutput(stdout, stderr string) string {
	var parts []string

	if stdout != "" {
		parts = append(parts, "Stdout: "+strings.TrimSpace(stdout))
	}

	if stderr != "" {
		parts = append(parts, "Stderr: "+strings.TrimSpace(stderr))
	}

	if len(parts) == 0 {
		return "无输出"
	}

	return strings.Join(parts, "\n")
}
