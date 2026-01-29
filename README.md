# libreoffice-rest-api

[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.23%2B-00ADD8.svg)](go.mod)
[![GitHub Release](https://img.shields.io/github/v/release/funnyzak/libreoffice-rest-api)](https://github.com/funnyzak/libreoffice-rest-api/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/funnyzak/libreoffice-rest-api/release.yml)](https://github.com/funnyzak/libreoffice-rest-api/actions)
[![Image Size](https://img.shields.io/docker/image-size/funnyzak/libreoffice-rest-api)](https://hub.docker.com/r/funnyzak/libreoffice-rest-api/)

> 基于 Go 的轻量级文档转换服务，通过 REST API 封装 LibreOffice 命令行能力

libreoffice-rest-api 是一款基于 Go 1.23+ 开发的轻量级跨平台文档转换服务。

## 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/funnyzak/libreoffice-rest-api/main/scripts/install.sh | bash
```

## 特性

- **多格式转换**：支持 PDF、HTML、PNG 等格式输出
- **双模式支持**：同步转换（立即返回结果）与异步转换（任务队列）
- **灵活输入**：文件上传与 URL 远程下载
- **安全认证**：API Key 认证与文件类型白名单
- **监控友好**：Prometheus 指标 + 健康检查
- **开箱即用**：Swagger API 文档自动生成
- **跨平台**：支持 Linux、macOS、Windows、ARM 等

## 安装方式

### 方式一：脚本安装（推荐）

使用交互式安装脚本，支持安装、更新、卸载等操作：

```bash
# 下载并运行安装脚本
curl -fsSL https://raw.githubusercontent.com/funnyzak/libreoffice-rest-api/main/scripts/install.sh | bash

# 或下载脚本后手动执行
curl -fsSL -o install.sh https://raw.githubusercontent.com/funnyzak/libreoffice-rest-api/main/scripts/install.sh
chmod +x install.sh
./install.sh
```

**脚本支持的命令**：

| 命令 | 说明 |
|------|------|
| `./install.sh install` | 安装（默认行为） |
| `./install.sh update` | 更新到最新版本 |
| `./install.sh uninstall` | 卸载 |
| `./install.sh check` | 检查安装状态和更新 |
| `./install.sh list` | 列出所有可用版本 |

**常用选项**：

```bash
# 安装指定版本
./install.sh install -v v1.0.0

# 安装到自定义目录
./install.sh install -d /opt/bin

# 强制重新安装
./install.sh install -f

# 跳过校验和验证
./install.sh install --skip-checksum
```

### 方式二：Docker 部署

```bash 
docker run -d \
  -p 30231:30231 \
  -v $(pwd)/storage:/app/storage \
  funnyzak/libreoffice-rest-api
```

说明：Dockerfile 基于 `linuxserver/libreoffice`，并在构建时安装开源中文字体包（如 Noto CJK 或文泉驿），以避免转换结果中文乱码。

### 方式三：二进制下载

从 [Releases](https://github.com/funnyzak/libreoffice-rest-api/releases) 下载对应平台的二进制文件：

```bash
# 解压
tar -xzf libreoffice-rest-api-linux-amd64.tar.gz

# 运行
./libreoffice-rest-api --config config.yaml
```

### 方式四：源码编译

```bash
# 克隆仓库
git clone https://github.com/funnyzak/libreoffice-rest-api.git
cd libreoffice-rest-api

# 初始化配置
make init-config

# 构建并运行
make build && make run
```

## 环境要求

- LibreOffice（需可执行 `soffice` 命令）
- 中文字体包（避免中文乱码）
- 开发环境额外需要：Go 1.23+、make

## 快速开始指南

1. 初始化配置文件：
```bash
make init-config
```

2. 设置 API Key（示例）：
```bash
export LIBREOFFICE_REST_API_AUTH_API_KEYS=your-api-key
```

3. 启动服务：
```bash
make build && make run
```

4. 校验服务状态：
```bash
curl http://localhost:30231/health
```

## 快速验证

服务默认监听 `0.0.0.0:30231`。

```bash
# 健康检查
curl http://localhost:30231/health

# 查看 Swagger 文档
open http://localhost:30231/swagger/index.html
```

## 使用示例

### 功能 API 说明

| 功能 | 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|------|
| 提交转换任务 | POST | `/api/v1/convert` | 必需 | 支持文件上传与 URL 下载，支持 sync/async |
| 查询任务状态 | GET | `/api/v1/tasks/:id` | 必需 | 返回任务状态与下载链接 |
| 下载转换结果 | GET | `/api/v1/files/:id/download` | 必需 | 返回转换后的文件流 |
| 健康检查 | GET | `/health` | 可选 | 返回服务与依赖健康状态 |
| 指标 | GET | `/metrics` | 可选 | Prometheus 指标（可配置认证） |
| Swagger 文档 | GET | `/swagger/index.html` | 可选 | API 文档界面（可配置认证） |

### 支持的输出格式

> 输出格式是否可用与输入文件类型及 LibreOffice 过滤器有关，以下为常用格式列表。

- 文档：pdf, html, xhtml, htm, txt, odt, doc, docx, rtf, epub
- 表格：ods, xls, xlsx, csv, tsv, tab
- 演示：odp, ppt, pptx
- 绘图：odg, svg
- 图片：png, jpg, jpeg, webp

### 常用 curl 示例

```bash
# 健康检查
curl http://localhost:30231/health

# Prometheus 指标（如启用认证）
curl -H "X-API-Key: your-api-key" http://localhost:30231/metrics
```

### 提交转换任务

#### 表单上传（异步）

```bash
curl -X POST http://localhost:30231/api/v1/convert \
  -H "X-API-Key: your-api-key" \
  -F "file=@document.docx" \
  -F "format=pdf" \
  -F "mode=async"
```

响应示例：

```json
{
  "success": true,
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "download_url": "http://localhost:30231/api/v1/files/550e8400-e29b-41d4-a716-446655440000/download"
  },
  "message": "任务已提交"
}
```

#### URL 转换

```bash
curl -X POST http://localhost:30231/api/v1/convert \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "url": "https://example.com/document.docx",
    "format": "pdf",
    "mode": "async"
  }'
```

#### 同步模式

```bash
curl -X POST http://localhost:30231/api/v1/convert \
  -H "X-API-Key: your-api-key" \
  -F "file=@document.docx" \
  -F "format=pdf" \
  -F "mode=sync" \
  --output result.pdf
```

### 查询任务状态

```bash
curl http://localhost:30231/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000 \
  -H "X-API-Key: your-api-key"
```

### 下载转换结果

```bash
curl http://localhost:30231/api/v1/files/550e8400-e29b-41d4-a716-446655440000/download \
  -H "X-API-Key: your-api-key" \
  --output result.pdf
```

更多 API 详情请参阅 [API.md](docs/API.md)。

## 配置说明

配置文件为 `config.yaml`（从 `config.yaml.example` 复制生成），支持环境变量覆盖（前缀：`LIBREOFFICE_REST_API_`）。

环境变量配置示例（对应 config.yaml 的键）：

- server.port → LIBREOFFICE_REST_API_SERVER_PORT=30231
- logger.file.path → LIBREOFFICE_REST_API_LOGGER_FILE_PATH=/var/log/app.log
- 列表类配置可用逗号分隔：LIBREOFFICE_REST_API_AUTH_API_KEYS=key1,key2

### 核心配置

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.port` | 服务端口 | 30231 |
| `auth.enabled` | 是否启用认证 | true |
| `auth.api_keys` | API Key 列表 | - |
| `storage.max_file_mb` | 最大文件大小（MB） | 50 |
| `worker.concurrency` | 并发工作数 | 4 |
| `converter.timeout_seconds` | 转换超时时间（秒） | 300 |

## 生产部署

### Docker Compose（推荐）

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  libreoffice-rest-api:
    image: funnyzak/libreoffice-rest-api:latest
    container_name: libreoffice-rest-api
    ports:
      - "30231:30231"
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./storage:/app/storage
      - ./logs:/app/logs
    environment:
      - LIBREOFFICE_REST_API_SERVER_PORT=30231
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:30231/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

启动服务：

```bash
docker-compose up -d
```

### Systemd 服务

创建 `/etc/systemd/system/libreoffice-rest-api.service`：

```ini
[Unit]
Description=LibreOffice REST API Service
After=network.target

[Service]
Type=simple
User=libreoffice
WorkingDirectory=/opt/libreoffice-rest-api
ExecStart=/opt/libreoffice-rest-api/libreoffice-rest-api --config /etc/libreoffice-rest-api/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl enable libreoffice-rest-api
sudo systemctl start libreoffice-rest-api
```

## 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.23+ |
| Web 框架 | Gin |
| CLI 工具 | Cobra |
| 配置管理 | Viper |
| 日志 | Zerolog |
| 数据库 | SQLite (GORM) |
| 文档 | Swagger |
| 构建 | Make + GitHub Actions |

## 开发指南

### 常用命令

| 命令 | 说明 |
|------|------|
| `make build` | 构建当前平台二进制 |
| `make build-all` | 交叉编译所有平台 |
| `make test` | 运行测试 |
| `make coverage` | 生成覆盖率报告（输出到 `coverage/coverage.html`）|
| `make fmt` | 格式化代码 |
| `make vet` | 静态分析 |
| `make lint` | 代码检查 |
| `make race` | 竞态检测 |
| `make swagger` | 生成 Swagger 文档 |
| `make version` | 查看构建版本信息 |

## 项目结构

本项目采用**六边形架构**（Hexagonal Architecture），确保关注点分离并便于测试。

### 目录结构

```
libreoffice-rest-api/
├── cmd/                         # 应用入口
│   └── server/
│       ├── main.go              # 主程序入口
│       └── version.go           # 版本信息
├── internal/                    # 私有代码（不可外部导入）
│   ├── adapter/                 # 接入层
│   │   └── http/                # HTTP 适配器
│   │       ├── handler.go       # 请求处理器
│   │       ├── middleware.go    # 中间件
│   │       ├── response.go      # 统一响应格式
│   │       └── router.go        # 路由配置
│   ├── service/                 # 业务层
│   │   ├── converter.go         # 转换服务
│   │   └── task_service.go      # 任务管理服务
│   ├── core/                    # 核心层
│   │   ├── libreoffice/         # LibreOffice 执行器
│   │   │   └── executor.go
│   │   └── workerpool/          # 工作池
│   │       └── pool.go
│   ├── repository/              # 持久层
│   │   ├── models.go            # 数据模型
│   │   ├── repository.go        # 存储接口
│   │   └── sqlite.go            # SQLite 实现
│   └── infrastructure/          # 基础设施
│       ├── config/              # 配置管理
│       ├── logger/              # 日志系统
│       ├── metrics/             # Prometheus 指标
│       └── cleaner/             # 文件清理器
├── pkg/                         # 公共库（可外部导入）
│   └── errors/                  # 领域错误定义
├── docs/                        # 项目文档
│   ├── API.md                   # API 接口说明
├── test/                        # 测试辅助
│   └── integration_test.go      # 集成测试
├── .github/                     # GitHub 配置
│   └── workflows/               # CI/CD 工作流
│       ├── build.yml            # 构建流程
│       └── release.yml          # 发布流程
├── storage/                     # 运行时存储
│   ├── temp/                    # 临时文件
│   ├── output/                  # 输出文件
│   └── lo-profile/              # LibreOffice 用户配置
├── logs/                        # 日志文件
├── build/                       # 当前平台构建产物
├── dist/                        # 全平台构建产物
├── config.yaml.example          # 配置文件模板
├── Makefile                     # 构建脚本
├── go.mod / go.sum              # Go 依赖管理
├── AGENTS.md                    # AI 开发指导
└── README.md                    # 项目说明
```

### 架构层次

```
┌─────────────────────────────────────────────────────┐
│                   接入层 (Adapter)                   │
│              HTTP Handler + Middleware               │
└────────────────────┬────────────────────────────────┘
                     │
┌────────────────────┴────────────────────────────────┐
│                   业务层 (Service)                   │
│         Converter Service + Task Service             │
└────────────────────┬────────────────────────────────┘
                     │
┌────────────────────┴────────────────────────────────┐
│                   核心层 (Core)                      │
│       LibreOffice Executor + Worker Pool             │
└────────────────────┬────────────────────────────────┘
                     │
┌────────────────────┴────────────────────────────────┐
│                 持久层 (Repository)                  │
│                  SQLite (GORM)                       │
└─────────────────────────────────────────────────────┘

              ┌───────────────────────┐
              │  基础设施 (Infrastructure) │
              │  Config / Logger / Metrics │
              └───────────────────────┘
```

### 模块职责

| 层次 | 目录 | 职责 |
|------|------|------|
| 接入层 | `internal/adapter/http` | HTTP 请求处理、路由、中间件、响应格式化 |
| 业务层 | `internal/service` | 业务逻辑编排、事务管理、领域错误处理 |
| 核心层 | `internal/core` | 文档转换、并发控制、独立于业务的通用功能 |
| 持久层 | `internal/repository` | 数据访问、模型定义、存储抽象 |
| 基础设施 | `internal/infrastructure` | 配置、日志、监控、清理等横切关注点 |


## 常见问题

### LibreOffice 未找到

确认 LibreOffice 已安装，或在 `converter.libreoffice_path` 指定完整路径：

```bash
# Linux
which soffice

# macOS
which soffice

# Windows
where soffice.exe
```

### 中文乱码

安装中文字体包并重启服务：

```bash
# Ubuntu/Debian
sudo apt-get install fonts-noto-cjk

# CentOS/RHEL
sudo yum install google-noto-sans-cjk-fonts

# macOS
brew install --cask font-noto-sans-cjk
```

### 401 未认证

确认 `auth.enabled` 为 true，且 `auth.api_keys` 配置正确。

### 转换超时

增大 `converter.timeout_seconds`，或减少并发与文件大小。

### 磁盘占用过高

降低 `storage.retention_hours`，并确保清理任务运行。

## 最佳实践

### 性能优化

- 根据 CPU 核数调整 `worker.concurrency`
- 将 `storage` 与 `logs` 放在高性能磁盘
- 生产环境日志级别建议使用 `info` 或 `warn`

### 安全加固

- 使用强 API Key，通过环境变量注入（禁止硬编码）
- 生产环境开启 `metrics.require_auth` 与 `swagger.require_auth`
- 配置文件类型白名单，限制上传文件大小
- 通过反向代理启用 HTTPS

## 贡献指南

1. 创建分支并进行开发：
```bash
git checkout -b feature/your-feature
```

2. 确保测试与检查通过：
```bash
make fmt
make lint
make test
```

3. 提交前补齐文档与测试，并确保覆盖率满足项目要求。

## 致谢

本项目基于以下开源项目：

- [Gin](https://github.com/gin-gonic/gin) - HTTP Web 框架
- [Cobra](https://github.com/spf13/cobra) - CLI 应用框架
- [Viper](https://github.com/spf13/viper) - 配置管理
- [Zerolog](https://github.com/rs/zerolog) - 结构化日志
- [GORM](https://github.com/go-gorm/gorm) - ORM 框架
- [LibreOffice](https://www.libreoffice.org/) - 文档转换引擎

## 许可证

本项目使用 [AGPL-3.0 许可证](LICENSE)。使用本服务的网络应用需要开源其修改。
