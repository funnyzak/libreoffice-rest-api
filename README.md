# libreoffice-rest-api

[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.23%2B-00ADD8.svg)](go.mod)
[![GitHub Release](https://img.shields.io/github/v/release/funnyzak/libreoffice-rest-api)](https://github.com/funnyzak/libreoffice-rest-api/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/funnyzak/libreoffice-rest-api/release.yml)](https://github.com/funnyzak/libreoffice-rest-api/actions)
[![Image Size](https://img.shields.io/docker/image-size/funnyzak/libreoffice-rest-api)](https://hub.docker.com/r/funnyzak/libreoffice-rest-api/)

libreoffice-rest-api 是基于 Go 的轻量级文档转换服务，封装 LibreOffice 提供 REST API，支持同步与异步转换、任务管理、鉴权与格式校验。提供 CLI 与 UNO 双模式，UNO 通过常驻实例降低启动成本，适合部署为文档转换网关或微服务。

## 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/funnyzak/libreoffice-rest-api/main/scripts/install.sh | bash
```

**使用 Homebrew（推荐）**

macOS 用户首选的安装方式是使用 Homebrew：

```bash
# 添加 tap
brew tap funnyzak/libreoffice-rest-api

# 安装 libreoffice-rest-api
brew install libreoffice-rest-api

# 更新
brew update && brew upgrade libreoffice-rest-api
```

## 特性

- **双模式转换**：CLI 与 UNO 可切换，UNO 通过常驻实例降低启动耗时
- **同步与异步**：同步立即返回结果，异步进入任务队列
- **多格式输出**：支持 PDF、HTML、PNG 等常用格式
- **文档合并**：多文件合并为单个 PDF
- **灵活输入**：文件上传与 URL 远程下载
- **安全防护**：API Key、MIME 与扩展名白名单、大小限制、魔数检测
- **可观测性**：Prometheus 指标、健康检查、结构化日志
- **易用运维**：Swagger 文档自动生成、配置可环境变量覆盖
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
| 提交转换任务 | POST | `/api/v1/convert` | 必需 | 支持文件上传与 URL 下载，支持 sync/async，同步模式支持 binary 参数控制返回格式 |
| 合并文档 | POST | `/api/v1/merge` | 必需 | 多文件合并为 PDF，支持 sync/async，同步模式支持 binary 参数控制返回格式 |
| 查询任务状态 | GET | `/api/v1/tasks/:id` | 必需 | 返回任务状态与下载链接 |
| 下载转换结果 | GET | `/api/v1/files/:id/download` | 必需 | 返回转换后的文件流 |
| 健康检查 | GET | `/health` | 可选 | 返回服务与依赖健康状态 |
| 指标 | GET | `/metrics` | 可选 | Prometheus 指标（可配置认证） |
| Swagger 文档 | GET | `/swagger/index.html` | 可选 | API 文档界面（可配置认证） |

### 支持的输出格式

> 输出格式是否可用与输入文件类型及 LibreOffice 过滤器有关，以下为常用格式列表。

**CLI 模式**（通过命令行转换）：
- 文档：pdf, html, xhtml, htm, txt, odt, doc, docx, rtf, epub
- 表格：ods, xls, xlsx, csv, tsv, tab
- 演示：odp, ppt, pptx
- 绘图：odg, svg
- 图片：png, jpg, jpeg, webp

**UNO 模式**（通过 UNO 接口转换）：
- pdf：支持文本类、表格类、演示类输入
- docx：支持文本类输入
- xlsx：支持表格类输入
- pptx：支持演示类输入

> 注意：UNO 模式专注于主流格式，如需其他格式请使用 CLI 模式。

### 支持的输入格式

服务支持以下输入文件格式（需在安全白名单中配置）：

**文本类**：
- `.doc`, `.docx` - Microsoft Word 文档
- `.odt` - OpenDocument 文本文档
- `.rtf` - Rich Text Format
- `.txt` - 纯文本
- `.html`, `.htm` - HTML 网页
- `.pdf` - PDF 文档

**表格类**：
- `.xls`, `.xlsx` - Microsoft Excel 工作簿
- `.ods` - OpenDocument 表格
- `.csv` - 逗号分隔值
- `.tsv` - 制表符分隔值

**演示类**：
- `.ppt`, `.pptx` - Microsoft PowerPoint 演示文稿
- `.odp` - OpenDocument 演示文稿

### 安全配置白名单

为确保服务安全，生产环境需要配置文件类型白名单：

```yaml
security:
  # 允许的 MIME 类型
  allowed_mime_types:
    - "application/pdf"
    - "application/msword"
    - "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    - "application/vnd.oasis.opendocument.text"
    - "text/rtf"
    - "text/plain"
    - "text/html"
    - "application/vnd.ms-excel"
    - "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
    - "application/vnd.oasis.opendocument.spreadsheet"
    - "text/csv"
    - "text/tab-separated-values"
    - "application/vnd.ms-powerpoint"
    - "application/vnd.openxmlformats-officedocument.presentationml.presentation"
    - "application/vnd.oasis.opendocument.presentation"

  # 允许的文件扩展名
  allowed_extensions:
    - ".pdf"
    - ".doc"
    - ".docx"
    - ".odt"
    - ".rtf"
    - ".txt"
    - ".html"
    - ".htm"
    - ".xls"
    - ".xlsx"
    - ".ods"
    - ".csv"
    - ".tsv"
    - ".ppt"
    - ".pptx"
    - ".odp"
```

配置示例文件请参阅 `config.yaml.example`，部署时请根据实际需求调整白名单。

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

**返回文件流（默认行为）**:

```bash
curl -X POST http://localhost:30231/api/v1/convert \
  -H "X-API-Key: your-api-key" \
  -F "file=@document.docx" \
  -F "format=pdf" \
  -F "mode=sync" \
  --output result.pdf
```

**返回 JSON 格式**（binary=false）:

```bash
curl -X POST http://localhost:30231/api/v1/convert \
  -H "X-API-Key: your-api-key" \
  -F "file=@document.docx" \
  -F "format=pdf" \
  -F "mode=sync" \
  -F "binary=false"
```

响应示例：

```json
{
  "success": true,
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "download_url": "http://localhost:30231/api/v1/files/550e8400-e29b-41d4-a716-446655440000/download",
    "output_name": "document.pdf",
    "output_format": "pdf"
  },
  "message": "转换完成"
}
```

### 合并文档

#### 表单上传（异步）

```bash
curl -X POST http://localhost:30231/api/v1/merge \
  -H "X-API-Key: your-api-key" \
  -F "files=@part1.docx" \
  -F "files=@part2.docx" \
  -F "format=pdf" \
  -F "mode=async"
```

#### URL 合并

```bash
curl -X POST http://localhost:30231/api/v1/merge \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "urls": [
      "https://example.com/part1.docx",
      "https://example.com/part2.docx"
    ],
    "format": "pdf",
    "mode": "async"
  }'
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

如需开放下载链接，可在配置中将 `download.require_auth` 设为 false。

更多 API 详情请参阅 [API.md](docs/API.md)。

## 配置说明

配置文件为 `config.yaml`（从 `config.yaml.example` 复制生成），支持环境变量覆盖（前缀：`LIBREOFFICE_REST_API_`）。

环境变量配置示例（对应 config.yaml 的键）：

- server.port → LIBREOFFICE_REST_API_SERVER_PORT=30231
- server.public_base_url → LIBREOFFICE_REST_API_SERVER_PUBLIC_BASE_URL=https://api.example.com
- logger.file.path → LIBREOFFICE_REST_API_LOGGER_FILE_PATH=/var/log/app.log
- download.require_auth → LIBREOFFICE_REST_API_DOWNLOAD_REQUIRE_AUTH=false
- 列表类配置可用逗号分隔：LIBREOFFICE_REST_API_AUTH_API_KEYS=key1,key2

### 核心配置

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.port` | 服务端口 | 30231 |
| `server.public_base_url` | 对外返回下载链接使用的基础地址 | 空 |
| `auth.enabled` | 是否启用认证 | true |
| `auth.api_keys` | API Key 列表 | - |
| `download.require_auth` | 下载接口是否需要认证 | true |
| `storage.max_file_mb` | 最大文件大小（MB） | 50 |
| `worker.concurrency` | 并发工作数 | 4 |
| `converter.mode` | 转换模式（cli/uno） | cli |
| `converter.timeout_seconds` | 转换超时时间（秒） | 300 |
| `converter.uno.pool_size` | UNO 实例数量 | 1 |

## 生产部署

支持多种部署方式：Docker Compose、Systemd、二进制部署等。

**快速开始**：
- Docker Compose：`docker-compose up -d`
- Systemd：创建服务文件并启动

更多部署详情、配置说明和最佳实践，请参阅 [部署文档](docs/DEPLOMENT.md)。


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

本项目采用**六边形架构**（Hexagonal Architecture），确保关注点分离并便于测试。

**快速开始**：
```bash
make init-config  # 初始化配置
make build && make run  # 构建并运行
```

**常用命令**：
- `make build` - 构建当前平台二进制
- `make test` - 运行测试
- `make coverage` - 生成覆盖率报告
- `make fmt` - 格式化代码
- `make lint` - 代码检查

更多开发详情，包括项目结构、架构设计、测试指南、调试技巧等，请参阅 [开发文档](docs/DEVELOPMENT.md)。


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

欢迎贡献代码！请遵循以下流程：

1. 创建功能分支：`git checkout -b feature/your-feature`
2. 确保代码检查通过：`make fmt && make lint && make test`
3. 提交前补齐文档与测试，确保覆盖率满足项目要求

更多开发规范和贡献指南，请参阅 [开发文档](docs/DEVELOPMENT.md)。

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
