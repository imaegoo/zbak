package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"zbak/internal/compression"
	"zbak/internal/config"
	"zbak/internal/coordinator"
	"zbak/internal/filesystem"
	"zbak/internal/index"
	"zbak/internal/logger"
	"zbak/internal/sevenzip"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	// 如果没有参数，显示帮助信息
	if len(args) == 0 {
		printHelp()
		return nil
	}

	// 解析子命令
	subcommand := args[0]

	switch subcommand {
	case "backup":
		return runBackup(args[1:])
	case "restore":
		return runRestore(args[1:])
	case "--help", "-h", "help":
		printHelp()
		return nil
	case "--version", "-v", "version":
		printVersion()
		return nil
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n\n", subcommand)
		printHelp()
		return fmt.Errorf("未知子命令: %s", subcommand)
	}
}

func runBackup(args []string) error {
	// 创建backup子命令的flag set
	flagSet := flag.NewFlagSet("backup", flag.ExitOnError)
	configPath := flagSet.String("config", "config.yaml", "配置文件路径")
	
	// 解析参数
	if err := flagSet.Parse(args); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	// 加载配置
	configMgr := config.NewConfigManager()
	cfg, err := configMgr.Load(*configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建日志记录器
	log, err := logger.NewLogger(cfg.TargetDir)
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %w", err)
	}
	defer log.Close()

	// 检测7zip工具
	sevenZip := sevenzip.NewWrapper()
	if _, err := sevenZip.Detect(); err != nil {
		return fmt.Errorf("检测7zip工具失败: %w", err)
	}

	// 创建文件系统服务
	fsService := filesystem.NewService()

	// 创建压缩服务
	compressionSvc := compression.NewService(fsService, sevenZip)

	// 创建备份协调器
	backupCoord := coordinator.NewBackupCoordinator(cfg, compressionSvc, log)

	// 执行备份
	ctx := context.Background()
	report, err := backupCoord.Execute(ctx)
	if err != nil {
		return fmt.Errorf("备份失败: %w", err)
	}

	// 如果有失败的任务，返回非零退出码
	if report.FailureCount > 0 {
		return fmt.Errorf("备份完成，但有 %d 个任务失败", report.FailureCount)
	}

	return nil
}

func runRestore(args []string) error {
	// 创建restore子命令的flag set
	flagSet := flag.NewFlagSet("restore", flag.ExitOnError)
	configPath := flagSet.String("config", "config.yaml", "配置文件路径")
	timestamp := flagSet.String("timestamp", "", "恢复指定时间戳的备份")
	fromTime := flagSet.String("from", "", "恢复时间戳范围的起始时间")
	toTime := flagSet.String("to", "", "恢复时间戳范围的结束时间")
	
	// 解析参数
	if err := flagSet.Parse(args); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	// 验证参数
	if *timestamp != "" && (*fromTime != "" || *toTime != "") {
		return fmt.Errorf("不能同时指定 --timestamp 和 --from/--to 参数")
	}

	// 加载配置
	configMgr := config.NewConfigManager()
	cfg, err := configMgr.Load(*configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建日志记录器
	log, err := logger.NewLogger(cfg.TargetDir)
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %w", err)
	}
	defer log.Close()

	// 检测7zip工具
	sevenZip := sevenzip.NewWrapper()
	if _, err := sevenZip.Detect(); err != nil {
		return fmt.Errorf("检测7zip工具失败: %w", err)
	}

	// 加载文件索引
	indexPath := filepath.Join(cfg.TargetDir, "index.yaml")
	fileIndex, err := index.Load(indexPath)
	if err != nil {
		return fmt.Errorf("加载文件索引失败: %w", err)
	}

	// 创建恢复协调器
	restoreCoord := coordinator.NewRestoreCoordinator(cfg, fileIndex, sevenZip, log)

	// 构建恢复选项
	options := coordinator.RestoreOptions{
		Timestamp: *timestamp,
		FromTime:  *fromTime,
		ToTime:    *toTime,
	}

	// 执行恢复
	report, err := restoreCoord.Execute(options)
	if err != nil {
		return fmt.Errorf("恢复失败: %w", err)
	}

	// 如果有失败的文件，返回非零退出码
	if report.FailedFiles > 0 {
		return fmt.Errorf("恢复完成，但有 %d 个文件失败", report.FailedFiles)
	}

	return nil
}

func printHelp() {
	fmt.Println("zbak - NAS备份工具")
	fmt.Printf("版本: %s\n\n", version)
	fmt.Println("用法:")
	fmt.Println("  zbak <子命令> [选项]")
	fmt.Println()
	fmt.Println("子命令:")
	fmt.Println("  backup              执行备份操作")
	fmt.Println("  restore             执行恢复操作")
	fmt.Println("  help, --help, -h    显示帮助信息")
	fmt.Println("  version, --version, -v  显示版本信息")
	fmt.Println()
	fmt.Println("备份选项:")
	fmt.Println("  --config <路径>     配置文件路径 (默认: config.yaml)")
	fmt.Println()
	fmt.Println("恢复选项:")
	fmt.Println("  --config <路径>     配置文件路径 (默认: config.yaml)")
	fmt.Println("  --timestamp <时间戳> 恢复指定时间戳的备份")
	fmt.Println("  --from <时间戳>     恢复时间戳范围的起始时间")
	fmt.Println("  --to <时间戳>       恢复时间戳范围的结束时间")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  zbak backup --config /path/to/config.yaml")
	fmt.Println("  zbak restore --config /path/to/config.yaml")
	fmt.Println("  zbak restore --config /path/to/config.yaml --timestamp 2024-01-15-10-30-00")
	fmt.Println("  zbak restore --config /path/to/config.yaml --from 2024-01-15-10-30-00 --to 2024-01-16-14-20-00")
}

func printVersion() {
	fmt.Printf("zbak 版本 %s\n", version)
}
