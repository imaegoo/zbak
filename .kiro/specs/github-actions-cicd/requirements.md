# Requirements Document

## Introduction

本文档定义了为 zbak（基于 7-zip 的 NAS 分卷加密备份工具）项目添加 GitHub Actions CI/CD 自动化流程的需求。该功能将实现代码提交时的自动测试和编译、发布时的多平台二进制包构建，以及多架构 Docker 镜像的自动构建和发布。

## Glossary

- **CI_Workflow**: 持续集成工作流，负责在代码提交到 main 分支时自动运行测试和编译
- **Release_Workflow**: 发布工作流，负责在创建 release 时构建多平台二进制包并上传到 release assets
- **Docker_Workflow**: Docker 镜像构建工作流，负责构建和发布多架构 Docker 镜像
- **zbak_Binary**: zbak 项目编译后的可执行文件
- **GitHub_Actions**: GitHub 提供的 CI/CD 自动化平台
- **Release_Asset**: GitHub Release 中附加的文件资源
- **Multi_Arch_Image**: 支持多种 CPU 架构的 Docker 镜像
- **Alpine_Base**: 基于 Alpine Linux 的轻量级 Docker 基础镜像
- **7z_Tool**: 7-Zip 命令行压缩工具

## Requirements

### Requirement 1: 持续集成工作流

**User Story:** 作为开发者，我希望在提交代码到 main 分支时自动运行测试和编译，以便及时发现代码问题。

#### Acceptance Criteria

1. WHEN 代码被推送到 main 分支时，THE CI_Workflow SHALL 自动触发执行
2. WHEN 创建针对 main 分支的 Pull Request 时，THE CI_Workflow SHALL 自动触发执行
3. THE CI_Workflow SHALL 使用 Go 1.25.0 或更高版本执行构建
4. THE CI_Workflow SHALL 下载项目依赖并执行 `go test ./...` 运行所有测试
5. THE CI_Workflow SHALL 执行 `go build ./cmd/zbak` 编译项目
6. IF 测试失败，THEN THE CI_Workflow SHALL 标记工作流为失败状态并报告错误
7. IF 编译失败，THEN THE CI_Workflow SHALL 标记工作流为失败状态并报告错误
8. THE CI_Workflow SHALL 在 Ubuntu 最新版本的运行环境中执行

### Requirement 2: 多平台二进制包发布

**User Story:** 作为项目维护者，我希望在发布 release 时自动构建多平台二进制包，以便用户可以直接下载适合其操作系统的可执行文件。

#### Acceptance Criteria

1. WHEN 创建新的 GitHub Release 时，THE Release_Workflow SHALL 自动触发执行
2. THE Release_Workflow SHALL 构建 Windows x64 平台的 zbak_Binary（GOOS=windows GOARCH=amd64）
3. THE Release_Workflow SHALL 构建 Linux x64 平台的 zbak_Binary（GOOS=linux GOARCH=amd64）
4. THE Release_Workflow SHALL 构建 Linux ARM 平台的 zbak_Binary（GOOS=linux GOARCH=arm）
5. THE Release_Workflow SHALL 构建 macOS x64 平台的 zbak_Binary（GOOS=darwin GOARCH=amd64）
6. THE Release_Workflow SHALL 构建 macOS ARM 平台的 zbak_Binary（GOOS=darwin GOARCH=arm64）
7. THE Release_Workflow SHALL 将 Windows 二进制文件命名为 `zbak-windows-amd64.exe`
8. THE Release_Workflow SHALL 将 Linux x64 二进制文件命名为 `zbak-linux-amd64`
9. THE Release_Workflow SHALL 将 Linux ARM 二进制文件命名为 `zbak-linux-arm`
10. THE Release_Workflow SHALL 将 macOS x64 二进制文件命名为 `zbak-darwin-amd64`
11. THE Release_Workflow SHALL 将 macOS ARM 二进制文件命名为 `zbak-darwin-arm64`
12. THE Release_Workflow SHALL 将所有构建的二进制文件上传为 Release_Asset
13. THE Release_Workflow SHALL 使用 Go 1.25.0 或更高版本执行构建
14. IF 任何平台的构建失败，THEN THE Release_Workflow SHALL 标记工作流为失败状态并报告错误

### Requirement 3: 多架构 Docker 镜像构建

**User Story:** 作为用户，我希望能够使用 Docker 运行 zbak 工具，并且支持我的 CPU 架构，以便在容器环境中使用该工具。

#### Acceptance Criteria

1. WHEN 创建新的 GitHub Release 时，THE Docker_Workflow SHALL 自动触发执行
2. THE Docker_Workflow SHALL 使用 Alpine_Base 作为基础镜像
3. THE Docker_Workflow SHALL 构建支持 linux/386 架构的镜像（32位）
4. THE Docker_Workflow SHALL 构建支持 linux/amd64 架构的镜像（64位）
5. THE Docker_Workflow SHALL 构建支持 linux/arm64/v8 架构的镜像（ARM64 v8a）
6. THE Docker_Workflow SHALL 构建支持 linux/arm/v7 架构的镜像（armeabi）
7. THE Docker_Workflow SHALL 在镜像中安装 7z_Tool
8. THE Docker_Workflow SHALL 将 zbak_Binary 复制到镜像中的 `/usr/local/bin/zbak` 路径
9. THE Docker_Workflow SHALL 设置镜像的默认 CMD 为 zbak_Binary
10. THE Docker_Workflow SHALL 将镜像推送到 Docker Hub 或 GitHub Container Registry
11. THE Docker_Workflow SHALL 使用 release 版本号作为镜像标签
12. THE Docker_Workflow SHALL 同时创建 `latest` 标签指向最新版本
13. IF Docker 镜像构建失败，THEN THE Docker_Workflow SHALL 标记工作流为失败状态并报告错误

### Requirement 4: Docker 镜像功能验证

**User Story:** 作为用户，我希望 Docker 镜像包含完整的运行环境，以便可以直接使用容器执行备份任务。

#### Acceptance Criteria

1. WHEN 用户运行 Docker 镜像时，THE Docker 镜像 SHALL 默认执行 zbak_Binary
2. THE Docker 镜像 SHALL 包含可用的 7z_Tool 命令
3. WHEN 用户在容器中执行 `zbak --help` 时，THE zbak_Binary SHALL 显示帮助信息
4. WHEN 用户在容器中执行 `7z` 时，THE 7z_Tool SHALL 响应并显示版本信息
5. THE Docker 镜像 SHALL 支持通过卷挂载访问宿主机文件系统
6. THE zbak_Binary SHALL 能够在容器中调用 7z_Tool 执行压缩操作

### Requirement 5: 工作流配置和安全性

**User Story:** 作为项目维护者，我希望 CI/CD 工作流配置清晰且安全，以便于维护和保护敏感信息。

#### Acceptance Criteria

1. THE GitHub_Actions 工作流配置文件 SHALL 存储在 `.github/workflows/` 目录中
2. THE CI_Workflow 配置文件 SHALL 命名为 `ci.yml`
3. THE Release_Workflow 配置文件 SHALL 命名为 `release.yml`
4. THE Docker_Workflow 配置文件 SHALL 命名为 `docker.yml`
5. WHERE Docker 镜像需要推送到镜像仓库，THE Docker_Workflow SHALL 使用 GitHub Secrets 存储认证凭据
6. THE 工作流配置文件 SHALL 包含清晰的注释说明每个步骤的作用
7. THE 工作流 SHALL 使用官方或广泛信任的 GitHub Actions
8. IF 工作流需要写入权限，THEN THE 工作流配置 SHALL 明确声明所需的最小权限范围

### Requirement 6: 构建优化和缓存

**User Story:** 作为开发者，我希望 CI/CD 工作流执行速度快，以便快速获得反馈。

#### Acceptance Criteria

1. THE CI_Workflow SHALL 缓存 Go 模块依赖以加速后续构建
2. THE Release_Workflow SHALL 缓存 Go 模块依赖以加速后续构建
3. THE Docker_Workflow SHALL 使用 Docker 层缓存以加速镜像构建
4. WHEN Go 依赖未变化时，THE 工作流 SHALL 从缓存恢复依赖而不是重新下载
5. THE 工作流 SHALL 使用 `go.sum` 文件作为缓存键的一部分以确保缓存有效性

### Requirement 7: 构建产物命名和版本管理

**User Story:** 作为用户，我希望能够清楚地识别不同版本和平台的构建产物，以便下载正确的文件。

#### Acceptance Criteria

1. THE Release_Asset 文件名 SHALL 包含平台和架构信息
2. THE Docker 镜像标签 SHALL 包含 release 版本号
3. WHEN 发布新版本时，THE Release_Workflow SHALL 从 Git 标签中提取版本号
4. THE 版本号 SHALL 遵循语义化版本规范（例如 v1.0.0）
5. THE Docker 镜像 SHALL 同时推送版本标签和 latest 标签
6. THE 二进制文件 SHALL 在编译时嵌入版本信息（通过 -ldflags）

