# 部署指南

## 适用范围
本文档用于生产环境部署，重点覆盖 Linux 环境。macOS 部署可参考开发指南中的 UNO 配置说明。

## 部署前准备

### 系统依赖
- Linux x86_64
- LibreOffice（提供 `soffice`）
- Python 3（UNO 模式需要）
- 中文字体包（避免字体缺失导致转换异常）

### 目录与权限
建议使用以下目录结构并授予写权限：
- `storage/temp`
- `storage/output`
- `storage/lo-profile`
- `logs`

## 配置准备

### 配置文件
1. 复制 `config.yaml.example` 为 `config.yaml`
2. 修改以下关键项：
   - `auth.api_keys`
   - `converter.mode`
   - `converter.libreoffice_path`
   - UNO 模式下的 `converter.uno.*`

### 环境变量
生产环境建议通过环境变量注入密钥：
- `LIBREOFFICE_REST_API_AUTH_API_KEYS=key1,key2`

## 部署方式

### 二进制部署
1. 构建二进制：
   - `make build`
2. 启动服务：
   - `./build/libreoffice-rest-api --config /path/to/config.yaml`

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

**注意**：如果使用环境变量配置 API Key，可以在 `[Service]` 部分添加：
```ini
Environment=LIBREOFFICE_REST_API_AUTH_API_KEYS=REPLACE_WITH_API_KEY
```

### Docker 部署
- 项目内提供 Dockerfile
- 推荐通过 docker-compose 挂载配置文件与持久化目录

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

### Docker Compose 示例（UNO 模式）

```yaml
version: "3.8"

services:
  libreoffice-rest-api:
    image: funnyzak/libreoffice-rest-api:latest
    container_name: libreoffice-rest-api
    restart: unless-stopped
    ports:
      - "30231:30231"
    volumes:
      - ./storage:/app/storage
      - ./logs:/app/logs
    environment:
      # 基础服务配置
      - LIBREOFFICE_REST_API_SERVER_HOST=0.0.0.0
      - LIBREOFFICE_REST_API_SERVER_PORT=30231
      - LIBREOFFICE_REST_API_SERVER_MODE=release
      - LIBREOFFICE_REST_API_SERVER_PUBLIC_BASE_URL=
      - LIBREOFFICE_REST_API_SERVER_READ_TIMEOUT_SECONDS=30
      - LIBREOFFICE_REST_API_SERVER_WRITE_TIMEOUT_SECONDS=30
      - LIBREOFFICE_REST_API_SERVER_SHUTDOWN_TIMEOUT_SECONDS=15
      - LIBREOFFICE_REST_API_SERVER_MAX_BODY_MB=50

      # 认证配置
      - LIBREOFFICE_REST_API_AUTH_ENABLED=true
      - LIBREOFFICE_REST_API_AUTH_API_KEYS=REPLACE_WITH_API_KEY  # 请务必替换自己的KEY

      # 存储配置
      - LIBREOFFICE_REST_API_STORAGE_TEMP_DIR=/app/storage/temp
      - LIBREOFFICE_REST_API_STORAGE_OUTPUT_DIR=/app/storage/output
      - LIBREOFFICE_REST_API_STORAGE_MAX_FILE_MB=50
      - LIBREOFFICE_REST_API_STORAGE_RETENTION_HOURS=24

      # 工作池配置
      - LIBREOFFICE_REST_API_WORKER_CONCURRENCY=4
      - LIBREOFFICE_REST_API_WORKER_QUEUE_SIZE=100

      # 转换器基础配置
      - LIBREOFFICE_REST_API_CONVERTER_MODE=uno
      - LIBREOFFICE_REST_API_CONVERTER_TIMEOUT_SECONDS=300
      - LIBREOFFICE_REST_API_CONVERTER_LIBREOFFICE_PATH=/usr/bin/soffice
      - LIBREOFFICE_REST_API_CONVERTER_USER_PROFILE_BASE_DIR=/app/storage/lo-profile

      # UNO 配置
      - LIBREOFFICE_REST_API_CONVERTER_UNO_ENABLED=true
      - LIBREOFFICE_REST_API_CONVERTER_UNO_HOST=127.0.0.1
      - LIBREOFFICE_REST_API_CONVERTER_UNO_BASE_PORT=2002
      - LIBREOFFICE_REST_API_CONVERTER_UNO_POOL_SIZE=1
      - LIBREOFFICE_REST_API_CONVERTER_UNO_LIBREOFFICE_PATH=/usr/bin/soffice
      - LIBREOFFICE_REST_API_CONVERTER_UNO_PYTHON_PATH=/usr/bin/python3
      - LIBREOFFICE_REST_API_CONVERTER_UNO_SCRIPT_PATH=/app/scripts/uno_convert.py
      - LIBREOFFICE_REST_API_CONVERTER_UNO_USER_PROFILE_BASE_DIR=/app/storage/lo-profile/uno
      - LIBREOFFICE_REST_API_CONVERTER_UNO_STARTUP_TIMEOUT_SECONDS=15
      - LIBREOFFICE_REST_API_CONVERTER_UNO_CONVERT_TIMEOUT_SECONDS=120
      - LIBREOFFICE_REST_API_CONVERTER_UNO_RESTART_AFTER_JOBS=200
      - LIBREOFFICE_REST_API_CONVERTER_UNO_HEALTHCHECK_INTERVAL_SECONDS=10

      # 数据库配置
      - LIBREOFFICE_REST_API_DATABASE_PATH=/app/storage/tasks.db

      # 日志配置
      - LIBREOFFICE_REST_API_LOGGER_LEVEL=info
      - LIBREOFFICE_REST_API_LOGGER_FORMAT=json
      - LIBREOFFICE_REST_API_LOGGER_OUTPUT=both
      - LIBREOFFICE_REST_API_LOGGER_FILE_ENABLE=true
      - LIBREOFFICE_REST_API_LOGGER_FILE_PATH=/app/logs/app.log
      - LIBREOFFICE_REST_API_LOGGER_FILE_MAX_SIZE_MB=50
      - LIBREOFFICE_REST_API_LOGGER_FILE_MAX_BACKUPS=7
      - LIBREOFFICE_REST_API_LOGGER_FILE_MAX_AGE_DAYS=14
      - LIBREOFFICE_REST_API_LOGGER_FILE_COMPRESS=true

      # 指标配置
      - LIBREOFFICE_REST_API_METRICS_ENABLED=true
      - LIBREOFFICE_REST_API_METRICS_PATH=/metrics
      - LIBREOFFICE_REST_API_METRICS_REQUIRE_AUTH=true

      # Swagger 配置
      - LIBREOFFICE_REST_API_SWAGGER_ENABLED=true
      - LIBREOFFICE_REST_API_SWAGGER_PATH=/swagger
      - LIBREOFFICE_REST_API_SWAGGER_REQUIRE_AUTH=false

      # 下载配置
      - LIBREOFFICE_REST_API_DOWNLOAD_REQUIRE_AUTH=true

      # 安全配置（支持 UNO 模式全部输入格式）
      - LIBREOFFICE_REST_API_SECURITY_ALLOWED_MIME_TYPES=application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.oasis.opendocument.text,text/rtf,text/plain,text/html,application/vnd.ms-excel,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.oasis.opendocument.spreadsheet,text/csv,text/tab-separated-values,application/vnd.ms-powerpoint,application/vnd.openxmlformats-officedocument.presentationml.presentation,application/vnd.oasis.opendocument.presentation
      - LIBREOFFICE_REST_API_SECURITY_ALLOWED_EXTENSIONS=.pdf,.doc,.docx,.odt,.rtf,.txt,.html,.htm,.xls,.xlsx,.ods,.csv,.tsv,.ppt,.pptx,.odp
      - LIBREOFFICE_REST_API_SECURITY_MAX_FILENAME_LENGTH=128

      # 健康检查配置
      - LIBREOFFICE_REST_API_HEALTH_MIN_FREE_GB=1
```

## UNO 模式部署要点
- `converter.mode=uno`
- `converter.uno.python_path` 指向 LibreOffice 自带 Python
- `converter.uno.script_path` 指向 `scripts/uno_convert.py`
- `converter.uno.pool_size` 根据机器资源设置

### UNO 配置说明

#### 支持的输入格式

UNO 模式支持以下输入文件格式：

##### 文本类
| 扩展名 | 说明 |
|--------|------|
| `.doc` | Microsoft Word 97-2003 文档 |
| `.docx` | Microsoft Word 2007+ 文档 |
| `.odt` | OpenDocument 文本文档 |
| `.rtf` | Rich Text Format |
| `.txt` | 纯文本 |
| `.html` | HTML 网页 |
| `.htm` | HTML 网页 |

##### 表格类
| 扩展名 | 说明 |
|--------|------|
| `.xls` | Microsoft Excel 97-2003 工作簿 |
| `.xlsx` | Microsoft Excel 2007+ 工作簿 |
| `.ods` | OpenDocument 表格 |
| `.csv` | 逗号分隔值 |
| `.tsv` | 制表符分隔值 |

##### 演示类
| 扩展名 | 说明 |
|--------|------|
| `.ppt` | Microsoft PowerPoint 97-2003 演示文稿 |
| `.pptx` | Microsoft PowerPoint 2007+ 演示文稿 |
| `.odp` | OpenDocument 演示文稿 |

#### 支持的输出格式

UNO 模式当前支持以下输出格式：

| 输出格式 | 适用输入类型 |
|----------|--------------|
| `pdf` | 文本类、表格类、演示类 |
| `docx` | 文本类 |
| `xlsx` | 表格类 |
| `pptx` | 演示类 |

> **注意**：如需输出为其他格式（如 txt、rtf 等），请使用 CLI 模式。

#### 安全配置白名单

部署时需要根据 UNO 支持的格式配置安全白名单，请在 `config.yaml` 或环境变量中添加完整的扩展名和 MIME 类型：

```yaml
security:
  allowed_extensions:
    # 文本类
    - ".pdf"
    - ".doc"
    - ".docx"
    - ".odt"
    - ".rtf"
    - ".txt"
    - ".html"
    - ".htm"
    # 表格类
    - ".xls"
    - ".xlsx"
    - ".ods"
    - ".csv"
    - ".tsv"
    # 演示类
    - ".ppt"
    - ".pptx"
    - ".odp"
  allowed_mime_types:
    # 文本类
    - "application/pdf"
    - "application/msword"
    - "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    - "application/vnd.oasis.opendocument.text"
    - "text/rtf"
    - "text/plain"
    - "text/html"
    # 表格类
    - "application/vnd.ms-excel"
    - "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
    - "application/vnd.oasis.opendocument.spreadsheet"
    - "text/csv"
    - "text/tab-separated-values"
    # 演示类
    - "application/vnd.ms-powerpoint"
    - "application/vnd.openxmlformats-officedocument.presentationml.presentation"
    - "application/vnd.oasis.opendocument.presentation"
```

#### 基础配置
- `converter.uno.host`：UNO 监听地址，建议使用 `127.0.0.1`
- `converter.uno.base_port`：UNO 监听基础端口，实例会使用 `base_port + index`
- `converter.uno.pool_size`：UNO 实例数量，决定并发上限
- `converter.uno.user_profile_base_dir`：每个实例的 UserInstallation 目录根路径
- `converter.uno.libreoffice_path`：UNO 模式使用的 soffice 路径（为空则沿用 `converter.libreoffice_path`）

#### 脚本与运行时
- `converter.uno.python_path`：执行 UNO 脚本的 Python 路径
- `converter.uno.script_path`：UNO 脚本路径，建议使用绝对路径

#### 超时与稳定性
- `converter.uno.startup_timeout_seconds`：实例启动超时
- `converter.uno.convert_timeout_seconds`：单次转换超时
- `converter.uno.restart_after_jobs`：处理多少任务后重启实例，0 表示禁用
- `converter.uno.healthcheck_interval_seconds`：健康检查间隔

### 配置注意事项
- API Key 必须通过环境变量配置
- `converter.uno.python_path` 建议使用 LibreOffice 自带 Python，避免 UNO 兼容性问题
- `converter.uno.script_path` 若使用相对路径，需确保 WorkingDirectory 一致
- `converter.uno.base_port` 不可被占用，多实例时需预留连续端口
- `converter.uno.user_profile_base_dir` 必须可写，且不同实例目录必须隔离
- `converter.uno.pool_size` 建议与机器 CPU 核心数与内存容量匹配
- UNO 模式仅支持主流输出格式（pdf、docx、xlsx、pptx），其他格式请使用 CLI 模式

### UNO 模式生产部署最佳实践

- **配置与密钥**：API Key 必须通过环境变量注入，严禁明文写入配置文件
- **重启与稳定性**：推荐在生产环境启用 `converter.uno.restart_after_jobs`（如设置 100~500），防止 UNO 实例长时间运行导致内存泄漏或资源碎片累积
- **健康检查**：合理设置 `converter.uno.healthcheck_interval_seconds`（建议 10~30 秒），在保证发现异常的前提下降低资源消耗
- **资源与并发**：根据实际 CPU 核心数和内存，配置足够的 `converter.uno.pool_size`，并为每个 UNO 实例提升文件描述符和进程数 ulimit
- **目录与持久化**：日志文件和输出目录应挂载至持久化磁盘，防止容器重启后数据丢失
- **字体与兼容性**：如出现 UNO 进程异常退出，需首先排查字体包是否完整、LibreOffice 及 Python 版本的兼容性
- **高并发与大文件处理**：大文件或高并发场景应适当增加 `converter.uno.pool_size`，并持续监控系统内存、CPU 和磁盘 I/O 峰值

> 请确保 UNO 相关环境依赖齐全（如字体、libreoffice、Python），并定期巡检日志与 Prometheus 指标，及时发现潜在风险。



## 运行验证

### 健康检查
- `GET /health`
- 正常响应为 HTTP 200

### 转换验证
- 通过 `POST /api/v1/convert` 进行小文件转换验证
- UNO 模式下建议先验证 docx -> pdf

## 日志与监控
- 日志路径由 `logger.file.path` 控制
- 若启用 Prometheus，使用 `GET /metrics` 获取指标

## 常见问题

### UNO 实例频繁重启
- 检查 LibreOffice 版本与系统依赖
- 检查字体包是否齐全
- 若运行在容器内，请确保 UNO 依赖完整

### UNO 转换失败
- 优先检查 UNO Python 路径是否正确
- 检查 `scripts/uno_convert.py` 是否可执行

## 安全建议
- 生产环境必须启用 API Key 认证
- 不要在代码中硬编码密钥
- 仅向可信网络开放服务端口
