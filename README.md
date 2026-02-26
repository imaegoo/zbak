# zbak - 基于7-zip的NAS分卷加密备份工具

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

一个高效、安全的NAS备份工具，支持增量备份、加密压缩和灵活恢复。

## 解决痛点

1. NAS 上的文件非常零碎，数十万数百万文件，而绝大多数网盘对大量文件都不友好，上传慢下载更慢
2. 巨型文件难以备份，网盘限制单文件最大大小
3. 直接上传文件，没有压缩，没有加密，占用空间，安全性低

我使用过 Duplicati，确实解决了以上三个问题，但仍有以下问题：

1. Duplicati 生成的备份文件，必须由 Duplicati 读取
2. 备份文件名完全随机，难以从备份中快速找到某个文件

对此我提出了以下备份方案：

1. 总体思想：保持目录结构的同时，大文件分卷，小文件合并，使用 7z 进行加密压缩
2. 递归 NAS 目录，如果目录小于指定的分卷大小，直接整目录压缩为一个文件
3. 如果目录大于指定的分卷大小，没有子目录，直接整目录分卷压缩为多个文件
4. 如果目录大于指定的分卷大小，有子目录，对子目录执行以上判断，对子文件执行分卷压缩

这样当我需要从网盘取回某个指定文件时，只需要下载对应的压缩文件，用 7z 解压，输入密码即可，是不是非常方便！

## ✨ 特性

- 🔄 **增量备份** - 仅备份变化的文件，节省存储空间和时间
- 🔐 **加密压缩** - 使用7zip进行AES-256加密和分卷压缩
- 🚀 **并发处理** - 支持多目录并发压缩，充分利用系统资源
- 📦 **智能压缩策略** - 根据目录大小和结构自动选择最优压缩方案
- 🕐 **版本管理** - 每次备份创建独立的时间戳目录，支持多版本保留
- 🎯 **灵活恢复** - 支持完整恢复或选择性恢复特定时间点的备份
- 🌍 **跨平台** - 支持Linux、Windows和macOS

## 📋 系统要求

- Go 1.21 或更高版本
- 7zip命令行工具（`7z` 或 `7za`）

### 安装7zip

**Linux (Debian/Ubuntu):**
```bash
sudo apt-get install p7zip-full
```

**Linux (RHEL/CentOS):**
```bash
sudo yum install p7zip p7zip-plugins
```

**macOS:**
```bash
brew install p7zip
```

**Windows:**
从 [7-Zip官网](https://www.7-zip.org/) 下载并安装，确保 `7z.exe` 在系统PATH中。

## 🚀 快速开始

### 安装

```bash
# 克隆仓库
git clone https://github.com/yourusername/zbak.git
cd zbak

# 构建
go build -o zbak ./cmd/zbak

# 或直接安装到 $GOPATH/bin
go install ./cmd/zbak
```

### 配置

创建配置文件 `config.yaml`：

```yaml
source_dir: /path/to/nas/source    # 源目录路径
target_dir: /path/to/backup/target # 目标目录路径
volume_size: 4294967296            # 分卷大小（4GB，单位：字节）
password: your_secure_password     # 加密密码
concurrency: 4                     # 并发数
```

### 使用

**执行备份：**
```bash
zbak backup --config config.yaml
```

**完整恢复：**
```bash
zbak restore --config config.yaml
```

**恢复指定时间戳的备份：**
```bash
zbak restore --config config.yaml --timestamp 2024-01-15-10-30-00
```

**恢复时间范围内的备份：**
```bash
zbak restore --config config.yaml --from 2024-01-15-10-30-00 --to 2024-01-16-14-20-00
```

**查看帮助：**
```bash
zbak --help
zbak backup --help
zbak restore --help
```

## 📖 工作原理

### 备份流程

1. **增量检测** - 扫描源目录，比对文件索引，识别新增、修改和删除的文件
2. **智能压缩** - 根据目录大小和结构选择压缩策略：
   - 小目录（< 分卷大小）：单文件压缩
   - 大目录无子目录：分卷压缩
   - 大目录有子目录：递归处理
3. **并发执行** - 使用工作池并发处理多个压缩任务
4. **索引更新** - 更新文件索引，记录备份信息
5. **时间戳管理** - 将备份文件存储在时间戳目录中

### 恢复流程

1. **发现备份** - 扫描时间戳目录，识别所有压缩文件
2. **按序恢复** - 按时间顺序依次解压备份文件
3. **覆盖更新** - 新版本文件覆盖旧版本
4. **删除处理** - 删除索引中标记为已删除的文件

### 目录结构

```
target_dir/
├── index.yaml                      # 文件索引
├── backup-2024-01-15-10-30-00.log # 备份日志
├── 2024-01-15-10-30-00/           # 时间戳目录
│   ├── dir1.7z.001                # 小目录压缩
│   ├── dir2/
│   │   ├── files.7z.001           # 目录中的文件
│   │   ├── files.7z.002
│   │   └── subdir1.7z.001         # 子目录压缩
│   └── dir3/
│       ├── subdir1.7z.001
│       └── subdir2.7z.001
└── 2024-01-16-14-20-00/           # 另一个时间戳目录
    └── ...
```

## 🔧 配置说明

| 配置项 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `source_dir` | string | 是 | 需要备份的源目录路径 |
| `target_dir` | string | 是 | 存储备份文件的目标目录路径 |
| `volume_size` | int64 | 是 | 分卷大小（字节），建议4GB（4294967296） |
| `password` | string | 是 | 加密密码，用于7zip AES-256加密 |
| `concurrency` | int | 是 | 并发压缩任务数，建议设置为CPU核心数 |

## 📊 性能特点

- **内存稳定** - 处理大量小文件时保持稳定的内存使用
- **高效索引** - 使用哈希表实现O(1)查找性能
- **并发优化** - 合理利用CPU资源，支持多任务并发处理
- **缓冲写入** - 日志记录使用缓冲写入，提高I/O性能

## 🤝 贡献

欢迎贡献代码、报告问题或提出建议！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📝 开发文档

查看 [DEVELOPMENT.md](DEVELOPMENT.md) 了解：
- 项目架构和组件设计
- 开发环境设置
- 测试指南
- 代码贡献规范

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [7-Zip](https://www.7-zip.org/) - 强大的压缩工具
- Go社区 - 优秀的工具和库支持
