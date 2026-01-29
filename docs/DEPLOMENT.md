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

### systemd 部署示例

```ini
[Unit]
Description=LibreOffice REST API
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/libreoffice-rest-api
ExecStart=/opt/libreoffice-rest-api/libreoffice-rest-api --config /opt/libreoffice-rest-api/config.yaml
Restart=on-failure
RestartSec=3
Environment=LIBREOFFICE_REST_API_AUTH_API_KEYS=REPLACE_WITH_API_KEY

[Install]
WantedBy=multi-user.target
```

启动服务：
- `systemctl enable libreoffice-rest-api`
- `systemctl start libreoffice-rest-api`

### Docker 部署
- 项目内提供 Dockerfile
- 推荐通过 docker-compose 挂载配置文件与持久化目录

## UNO 模式部署要点
- `converter.mode=uno`
- `converter.uno.python_path` 指向 LibreOffice 自带 Python
- `converter.uno.script_path` 指向 `scripts/uno_convert.py`
- `converter.uno.pool_size` 根据机器资源设置

### UNO 配置说明

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

### 最佳实践
- 生产环境建议启用 `restart_after_jobs`，降低长时间运行带来的内存增长风险
- 建议配置健康检查间隔为 10-30 秒，避免频繁探测造成额外负载
- 为 UNO 实例配置足够的文件描述符与进程数限制
- 日志文件与输出目录建议挂载到持久化磁盘
- 如遇 UNO 进程异常退出，优先检查字体包与 LibreOffice 版本兼容性
- 大文件或高并发场景建议增加 `converter.uno.pool_size` 并监控内存峰值

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
