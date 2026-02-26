# 开发文档

本文档提供zbak项目的技术细节、架构设计和开发指南。

## 目录

- [项目结构](#项目结构)
- [系统架构](#系统架构)
- [核心组件](#核心组件)
- [开发环境设置](#开发环境设置)
- [构建和测试](#构建和测试)
- [代码贡献指南](#代码贡献指南)

## 项目结构

```
zbak/
├── cmd/
│   └── zbak/                    # 主程序入口
│       ├── main.go              # CLI入口和路由
│       ├── main_test.go         # CLI单元测试
│       ├── integration_test.go  # 集成测试
│       └── functional_test.go   # 功能测试
├── internal/
│   ├── compression/             # 压缩服务
│   │   ├── compression.go       # 压缩策略和执行
│   │   └── compression_test.go
│   ├── config/                  # 配置管理
│   │   ├── config.go            # 配置加载和验证
│   │   ├── config_test.go
│   │   └── errors.go            # 配置错误定义
│   ├── coordinator/             # 协调器
│   │   ├── backup.go            # 备份协调器
│   │   ├── backup_test.go
│   │   ├── restore.go           # 恢复协调器
│   │   ├── restore_test.go
│   │   ├── timestamp.go         # 时间戳管理
│   │   ├── timestamp_test.go
│   │   └── errors.go            # 协调器错误定义
│   ├── detector/                # 增量检测器
│   │   ├── detector.go          # 文件变化检测
│   │   └── detector_test.go
│   ├── filesystem/              # 文件系统服务
│   │   ├── filesystem.go        # 文件系统操作抽象
│   │   └── filesystem_test.go
│   ├── index/                   # 文件索引服务
│   │   ├── index.go             # 索引数据结构
│   │   ├── index_test.go
│   │   ├── service.go           # 索引服务
│   │   └── service_test.go
│   ├── logger/                  # 日志记录器
│   │   └── logger.go            # 日志接口和实现
│   ├── performance/             # 性能测试
│   │   └── performance_test.go
│   └── sevenzip/                # 7zip包装器
│       ├── sevenzip.go          # 7zip命令封装
│       ├── sevenzip_test.go
│       └── errors.go            # 7zip错误定义
├── .kiro/                       # 项目规范文档
│   └── specs/
│       └── nas-backup-tool/
│           ├── requirements.md  # 需求文档
│           ├── design.md        # 设计文档
│           └── tasks.md         # 任务列表
├── config.example.yaml          # 配置文件示例
├── go.mod
├── go.sum
├── README.md
└── DEVELOPMENT.md
```

## 系统架构

zbak采用分层架构设计，从上到下分为以下层次：

```
┌─────────────────────────────────────────┐
│         命令行接口层 (CLI Layer)          │
│  - 参数解析                              │
│  - 子命令路由                            │
│  - 帮助信息                              │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│        业务逻辑层 (Business Layer)        │
│  - 备份协调器 (BackupCoordinator)        │
│  - 恢复协调器 (RestoreCoordinator)       │
│  - 增量检测器 (IncrementalDetector)      │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│        服务层 (Service Layer)            │
│  - 压缩服务 (CompressionService)         │
│  - 索引服务 (IndexService)               │
│  - 文件系统服务 (FileSystemService)      │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│        基础设施层 (Infrastructure)        │
│  - 配置管理 (ConfigManager)              │
│  - 7zip包装器 (7zipWrapper)              │
│  - 日志记录器 (Logger)                   │
│  - 并发池 (WorkerPool)                   │
└─────────────────────────────────────────┘
```

### 并发模型

系统使用工作池模式实现并发压缩：

```
                    ┌──────────────┐
                    │  Task Queue  │
                    └──────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ↓                  ↓                  ↓
   ┌─────────┐        ┌─────────┐       ┌─────────┐
   │ Worker 1│        │ Worker 2│  ...  │ Worker N│
   └─────────┘        └─────────┘       └─────────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           ↓
                    ┌──────────────┐
                    │ Index Update │
                    │  (Mutex Lock)│
                    └──────────────┘
```

## 核心组件

### 1. 命令行接口 (CLI)

**职责：** 解析命令行参数，路由到相应的业务逻辑

**接口：**
```go
type CLI interface {
    Run(args []string) error
}
```

**支持的命令：**
- `backup` - 执行备份操作
- `restore` - 执行恢复操作
- `help` / `--help` / `-h` - 显示帮助信息
- `version` / `--version` / `-v` - 显示版本信息

### 2. 配置管理器 (ConfigManager)

**职责：** 加载和验证YAML配置文件

**接口：**
```go
type ConfigManager interface {
    Load(path string) (*Config, error)
    Validate(config *Config) error
}

type Config struct {
    SourceDir    string `yaml:"source_dir"`
    TargetDir    string `yaml:"target_dir"`
    VolumeSize   int64  `yaml:"volume_size"`
    Password     string `yaml:"password"`
    Concurrency  int    `yaml:"concurrency"`
}
```

### 3. 文件索引服务 (IndexService)

**职责：** 管理文件索引的读写和更新

**数据结构：**
```go
type FileIndex struct {
    Files map[string]FileEntry `yaml:"files"`
}

type FileEntry struct {
    SourcePath      string    `yaml:"source_path"`
    Size            int64     `yaml:"size"`
    ModTime         time.Time `yaml:"mod_time"`
    ArchivePath     string    `yaml:"archive_path"`
    TimestampDir    string    `yaml:"timestamp_dir"`
    Deleted         bool      `yaml:"deleted"`
}
```

**线程安全：** 使用sync.Mutex保护并发访问

### 4. 增量检测器 (IncrementalDetector)

**职责：** 比较源目录和文件索引，识别需要备份的文件

**接口：**
```go
type IncrementalDetector interface {
    Detect(sourceDir string, index *FileIndex) (*ChangeSet, error)
}

type ChangeSet struct {
    NewFiles      []string
    ModifiedFiles []string
    DeletedFiles  []string
    UnchangedFiles []string
}
```

**检测逻辑：**
- 遍历源目录所有文件
- 对比文件大小和修改时间
- 标记新增、修改、删除和未变化的文件

### 5. 压缩服务 (CompressionService)

**职责：** 决定压缩策略并执行压缩操作

**压缩策略：**
```go
type CompressionStrategy int

const (
    StrategySmallDir CompressionStrategy = iota  // 小目录单文件压缩
    StrategyLargeNoSubdir                        // 大目录无子目录分卷压缩
    StrategyLargeWithSubdir                      // 大目录有子目录递归压缩
)
```

**策略选择逻辑：**
1. 计算目录总大小
2. 如果大小 < 分卷大小：使用StrategySmallDir
3. 如果大小 >= 分卷大小：
   - 检查是否有子目录
   - 无子目录：使用StrategyLargeNoSubdir
   - 有子目录：使用StrategyLargeWithSubdir

### 6. 7zip包装器 (7zipWrapper)

**职责：** 封装7zip命令行工具的调用

**接口：**
```go
type SevenZipWrapper interface {
    Detect() (string, error)
    Compress(params CompressParams) error
    Extract(params ExtractParams) error
}
```

**实现细节：**
- 检测7z或7za命令
- 构建7zip命令行参数
- 捕获标准输出和标准错误
- 处理命令执行失败

### 7. 备份协调器 (BackupCoordinator)

**职责：** 协调整个备份流程

**流程：**
1. 加载文件索引
2. 执行增量检测
3. 创建时间戳目录
4. 构建压缩任务
5. 启动工作池执行任务
6. 更新文件索引
7. 生成备份报告

### 8. 恢复协调器 (RestoreCoordinator)

**职责：** 协调整个恢复流程

**流程：**
1. 加载文件索引
2. 确定恢复范围（单个时间戳或范围）
3. 按时间戳排序
4. 依次解压每个时间戳的备份
5. 处理已删除文件
6. 生成恢复报告

## 开发环境设置

### 前置要求

- Go 1.21 或更高版本
- 7zip命令行工具（用于测试）
- Git

### 克隆仓库

```bash
git clone https://github.com/yourusername/zbak.git
cd zbak
```

### 安装依赖

```bash
go mod download
```

### 验证环境

```bash
# 检查Go版本
go version

# 检查7zip
7z --help  # 或 7za --help

# 运行测试
go test ./...
```

## 构建和测试

### 构建

```bash
# 构建可执行文件
go build -o zbak ./cmd/zbak

# 构建并安装到 $GOPATH/bin
go install ./cmd/zbak

# 交叉编译（Linux）
GOOS=linux GOARCH=amd64 go build -o zbak-linux ./cmd/zbak

# 交叉编译（Windows）
GOOS=windows GOARCH=amd64 go build -o zbak.exe ./cmd/zbak

# 交叉编译（macOS）
GOOS=darwin GOARCH=amd64 go build -o zbak-macos ./cmd/zbak
```

### 测试

```bash
# 运行所有测试
go test ./...

# 运行测试并显示详细输出
go test -v ./...

# 运行特定包的测试
go test ./internal/compression

# 运行特定测试
go test -run TestCompressDirectory ./internal/compression

# 运行测试并生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行性能测试
go test -bench=. ./internal/performance

# 运行功能测试（需要7zip）
go test -v ./cmd/zbak -run TestFunctional
```

### 测试分类

1. **单元测试** - 测试单个组件的功能
   - 位置：`*_test.go` 文件
   - 使用模拟对象隔离外部依赖
   - 快速执行，不依赖外部工具

2. **集成测试** - 测试组件间的集成
   - 位置：`cmd/zbak/integration_test.go`
   - 测试CLI与协调器的集成
   - 验证错误传播和退出码

3. **功能测试** - 端到端测试
   - 位置：`cmd/zbak/functional_test.go`
   - 使用真实的7zip工具
   - 测试完整的备份和恢复流程
   - 验证文件一致性

4. **性能测试** - 验证性能特性
   - 位置：`internal/performance/performance_test.go`
   - 测试内存使用、CPU利用率
   - 验证索引性能和日志性能

## 代码贡献指南

### 代码风格

- 遵循Go官方代码风格指南
- 使用`gofmt`格式化代码
- 使用`golint`检查代码质量
- 保持函数简洁，单一职责
- 添加适当的注释和文档

### 提交规范

提交信息格式：
```
<类型>: <简短描述>

<详细描述>

<相关issue>
```

类型：
- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `test`: 测试相关
- `refactor`: 代码重构
- `perf`: 性能优化
- `chore`: 构建/工具相关

示例：
```
feat: 添加并发压缩支持

实现了工作池模式，支持多个目录并发压缩。
使用sync.WaitGroup确保所有任务完成。

Closes #123
```

### Pull Request流程

1. Fork仓库并创建特性分支
2. 编写代码和测试
3. 确保所有测试通过
4. 更新相关文档
5. 提交Pull Request
6. 等待代码审查
7. 根据反馈修改代码
8. 合并到主分支

### 测试要求

- 新功能必须包含单元测试
- 测试覆盖率应保持在80%以上
- 功能测试应覆盖主要使用场景
- 性能测试应验证关键性能指标

### 文档要求

- 公开接口必须有文档注释
- 复杂逻辑需要添加说明注释
- 更新README.md（如果影响用户使用）
- 更新DEVELOPMENT.md（如果影响开发流程）

## 性能优化指南

### 内存优化

- 使用流式处理避免一次性加载大文件
- 及时释放不再使用的资源
- 使用对象池复用频繁创建的对象

### CPU优化

- 合理设置并发数（建议为CPU核心数）
- 避免不必要的计算和重复操作
- 使用高效的数据结构（如map而非slice查找）

### I/O优化

- 使用缓冲I/O减少系统调用
- 批量处理文件操作
- 避免频繁的小文件读写

## 故障排查

### 常见问题

1. **7zip未找到**
   - 确认7zip已安装
   - 检查PATH环境变量
   - 尝试使用绝对路径

2. **权限错误**
   - 检查源目录和目标目录的读写权限
   - 确认当前用户有足够的权限

3. **内存不足**
   - 减少并发数
   - 增加系统可用内存
   - 检查是否有内存泄漏

4. **压缩失败**
   - 检查磁盘空间是否充足
   - 验证密码是否正确
   - 查看日志文件获取详细错误信息

### 调试技巧

- 使用`-v`参数查看详细日志
- 检查日志文件（`backup-*.log`）
- 使用Go调试工具（delve）
- 添加临时日志输出定位问题

## 发布流程

1. 更新版本号（`cmd/zbak/main.go`中的`version`常量）
2. 更新CHANGELOG.md
3. 运行完整测试套件
4. 创建Git标签
5. 构建各平台二进制文件
6. 创建GitHub Release
7. 上传二进制文件到Release

## 许可证

本项目采用MIT许可证。详见LICENSE文件。
