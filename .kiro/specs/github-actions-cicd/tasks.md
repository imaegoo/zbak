# Implementation Plan: GitHub Actions CI/CD

## Overview

本实现计划将为 zbak 项目创建完整的 GitHub Actions CI/CD 自动化流程，包括：
- CI 工作流：自动测试和编译验证
- Release 工作流：构建 5 个平台的二进制包
- Docker 工作流：构建和发布 4 个架构的 Docker 镜像
- 配置验证测试和属性测试

## Tasks

- [x] 1. 创建 CI 工作流配置文件
  - 创建 `.github/workflows/ci.yml` 文件
  - 配置触发条件：push 到 main 分支和 pull_request 到 main 分支
  - 配置 Go 1.25.0+ 环境和依赖缓存
  - 添加测试步骤（`go test ./...`）
  - 添加编译步骤（`go build ./cmd/zbak`）
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.8, 6.1_

- [x] 2. 创建 Release 工作流配置文件
  - 创建 `.github/workflows/release.yml` 文件
  - 配置触发条件：release published 事件
  - 配置构建矩阵，包含 5 个平台配置（Windows amd64, Linux amd64, Linux arm, macOS amd64, macOS arm64）
  - 添加版本号提取步骤（从 Git tag）
  - 添加跨平台编译步骤，使用 `-ldflags` 注入版本信息
  - 添加上传 Release Assets 步骤
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9, 2.10, 2.11, 2.12, 2.13, 6.2, 7.3, 7.6_

- [x] 3. 创建 Docker 工作流配置文件
  - 创建 `.github/workflows/docker.yml` 文件
  - 配置触发条件：release published 事件
  - 添加 QEMU 和 Docker Buildx 设置步骤
  - 添加镜像仓库登录步骤（使用 GitHub Secrets）
  - 配置多架构构建（linux/386, linux/amd64, linux/arm64/v8, linux/arm/v7）
  - 配置镜像标签（版本号 + latest）
  - _Requirements: 3.1, 3.3, 3.4, 3.5, 3.6, 3.10, 3.11, 3.12, 5.5, 6.3_

- [x] 4. 创建 Dockerfile
  - 创建项目根目录下的 `Dockerfile` 文件
  - 使用多阶段构建：第一阶段编译 Go 二进制，第二阶段创建运行时镜像
  - 使用 Alpine Linux 作为基础镜像
  - 安装 p7zip 包
  - 复制 zbak 二进制到 `/usr/local/bin/zbak`
  - 设置工作目录和默认 CMD
  - _Requirements: 3.2, 3.7, 3.8, 3.9_

- [~] 5. 修改 main.go 支持动态版本注入
  - 修改 `cmd/zbak/main.go` 中的 version 变量为可被 ldflags 覆盖
  - 将 `const version = "0.1.0"` 改为 `var version = "dev"`
  - 确保 `--version` 和 `--help` 命令正确显示版本信息
  - _Requirements: 7.6_

- [ ] 6. 创建工作流配置验证测试
  - [~] 6.1 创建测试文件和辅助函数
    - 创建 `internal/cicd/workflow_test.go` 文件
    - 实现 YAML 解析辅助函数
    - 定义工作流配置数据结构
    - _Requirements: 5.1, 5.2, 5.3, 5.4_

  - [~] 6.2 实现 CI 工作流配置验证测试
    - 验证 ci.yml 文件存在且可解析
    - 验证触发条件包含 push 和 pull_request 到 main 分支
    - 验证 Go 版本 >= 1.25.0
    - 验证包含测试和编译步骤
    - 验证运行环境为 ubuntu-latest
    - 验证启用了缓存
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.8, 6.1_

  - [~] 6.3 实现 Release 工作流配置验证测试
    - 验证 release.yml 文件存在且可解析
    - 验证触发条件为 release.published
    - 验证构建矩阵包含 5 个平台配置
    - 验证每个平台的 GOOS、GOARCH 和输出文件名正确
    - 验证包含版本号提取和 ldflags 注入
    - 验证包含上传 Release Assets 步骤
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9, 2.10, 2.11, 2.12, 2.13, 6.2, 7.3, 7.6_

  - [~] 6.4 实现 Docker 工作流配置验证测试
    - 验证 docker.yml 文件存在且可解析
    - 验证触发条件为 release.published
    - 验证包含 QEMU 和 Buildx 设置步骤
    - 验证包含登录步骤且使用 Secrets
    - 验证构建平台列表包含 4 个架构
    - 验证镜像标签包含版本号和 latest
    - _Requirements: 3.1, 3.3, 3.4, 3.5, 3.6, 3.10, 3.11, 3.12, 5.5, 6.3_

  - [~] 6.5 实现 Dockerfile 配置验证测试
    - 验证 Dockerfile 文件存在且可解析
    - 验证使用 Alpine 基础镜像
    - 验证包含安装 p7zip 的指令
    - 验证包含复制 zbak 二进制的指令
    - 验证 CMD 设置为 zbak
    - _Requirements: 3.2, 3.7, 3.8, 3.9_

- [ ]* 7. 创建属性测试
  - [ ]* 7.1 实现 Release Asset 命名规范属性测试
    - **Property 1: Release Asset 命名规范**
    - **Validates: Requirements 7.1**
    - 创建 `internal/cicd/naming_property_test.go` 文件
    - 使用属性测试库（gopter）生成随机的 GOOS 和 GOARCH 组合
    - 验证生成的文件名符合 `zbak-{goos}-{goarch}[.exe]` 格式
    - 验证 Windows 平台添加 .exe 后缀
    - 运行至少 100 次迭代

  - [ ]* 7.2 实现版本号语义化格式属性测试
    - **Property 2: 版本号语义化格式**
    - **Validates: Requirements 7.4**
    - 在 `internal/cicd/naming_property_test.go` 中添加测试
    - 使用属性测试库生成随机的版本号组件（major, minor, patch）
    - 验证生成的版本号匹配语义化版本规范 `v{major}.{minor}.{patch}`
    - 验证支持预发布版本格式
    - 运行至少 100 次迭代

- [~] 8. Checkpoint - 确保所有测试通过
  - 运行 `go test ./...` 确保所有配置验证测试通过
  - 确保工作流配置文件格式正确
  - 如有问题，询问用户

- [ ]* 9. 创建 Docker 镜像集成测试
  - 创建 `internal/cicd/docker_integration_test.go` 文件
  - 实现本地 Docker 镜像构建测试
  - 验证镜像可以成功构建
  - 验证容器中 `zbak --help` 命令可用
  - 验证容器中 `7z` 命令可用
  - 验证 zbak 可以在容器中调用 7z 执行压缩操作
  - 标记为集成测试（使用 build tag）
  - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.6_

- [~] 10. 最终检查点
  - 确保所有工作流配置文件已创建
  - 确保 Dockerfile 已创建
  - 确保 main.go 版本变量已修改
  - 确保所有测试通过
  - 如有问题，询问用户

## Notes

- 任务标记 `*` 为可选任务，可以跳过以加快 MVP 交付
- 每个任务都引用了具体的需求编号以确保可追溯性
- 检查点任务确保增量验证
- 属性测试验证通用的正确性属性
- 配置验证测试验证具体的配置内容
- 集成测试需要 Docker 环境，可以在本地或 CI 中运行
