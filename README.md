# zbak - NAS备份工具

一个使用Go语言开发的命令行工具，用于将NAS中的文件进行加密分卷压缩备份，并支持增量备份和恢复功能。

## 项目结构

```
zbak/
├── cmd/
│   └── zbak/           # 主程序入口
│       └── main.go
├── internal/
│   ├── config/         # 配置管理
│   │   ├── config.go
│   │   └── errors.go
│   └── logger/         # 日志记录
│       └── logger.go
├── go.mod
├── go.sum
└── README.md
```

## 核心组件

### ConfigManager (配置管理器)
- 从YAML文件加载配置
- 验证配置参数的完整性和有效性
- 支持的配置项：
  - `source_dir`: 源目录路径
  - `target_dir`: 目标目录路径
  - `volume_size`: 分卷大小（字节）
  - `password`: 加密密码
  - `concurrency`: 并发数（最小值为1）

### Logger (日志记录器)
- 同时输出到标准输出和日志文件
- 日志文件命名格式：`backup-YYYY-MM-DD-HH-MM-SS.log`
- 支持三个日志级别：Info、Warn、Error

## 构建

```bash
go build -o zbak ./cmd/zbak
```

## 开发状态

当前已完成：
- ✅ 项目结构初始化
- ✅ 配置管理器实现
- ✅ 日志记录器实现

待实现：
- ⏳ 文件系统服务
- ⏳ 7zip包装器
- ⏳ 文件索引服务
- ⏳ 增量检测器
- ⏳ 压缩服务
- ⏳ 备份协调器
- ⏳ 恢复协调器
- ⏳ 命令行接口
