# 开发指南

## 适用范围
本文档面向本项目的开发、调试与测试，覆盖 CLI 与 UNO 模式的本地开发流程。

## 环境准备

### 必需依赖
- Go 1.21+
- LibreOffice（包含 soffice）
- Python 3（UNO 模式需要）

### 可选依赖
- golangci-lint
- docker / docker-compose（用于容器化开发验证）

## macOS 开发环境

### 安装 LibreOffice
- 推荐使用 Homebrew 安装：
  - `brew install --cask libreoffice`
- 确认 `soffice` 路径（示例）：
  - `/Applications/LibreOffice.app/Contents/MacOS/soffice`
  - Homebrew: `/opt/homebrew/bin/soffice`

### UNO 模式 Python 路径
- 建议使用 LibreOffice 自带 Python，避免 UNO 兼容性问题
- Homebrew 版本示例路径：
  - `/opt/homebrew/Cellar/libreoffice/<版本>/lib/libreoffice/program/python`
- 将该路径写入 `converter.uno.python_path`

## Linux 开发环境

### 安装 LibreOffice
- 使用发行版包管理器安装
- 确认 `soffice` 可执行路径在 PATH 中

### UNO 模式 Python 依赖
- 使用系统 Python 3
- 确保能够 `import uno`
- 如无法导入，请安装 LibreOffice 的 UNO Python 组件

## 配置说明

### 配置文件
- 从 `config.yaml.example` 复制为 `config.yaml`
- 生产环境配置禁止提交到版本控制

### 重要配置
- `converter.mode`：`cli` 或 `uno`
- `converter.libreoffice_path`：`soffice` 路径
- `converter.uno.python_path`：UNO 脚本使用的 Python
- `converter.uno.script_path`：UNO 脚本路径
- `converter.uno.pool_size`：UNO 实例数量

### 环境变量覆盖
- 前缀：`LIBREOFFICE_REST_API_`
- 示例：
  - `LIBREOFFICE_REST_API_CONVERTER_MODE=uno`
  - `LIBREOFFICE_REST_API_CONVERTER_UNO_PYTHON_PATH=/opt/homebrew/Cellar/libreoffice/<版本>/lib/libreoffice/program/python`

## 本地运行

### CLI 模式
1. 确认 `converter.mode=cli`
2. 配置 `converter.libreoffice_path`
3. 启动服务：
   - `make dev`
   - 或 `go run ./cmd/server --config config.yaml`

### UNO 模式
1. 设置 `converter.mode=uno`
2. 设置 `converter.uno.python_path` 与 `converter.uno.script_path`
3. 可选：设置 `converter.uno.pool_size`
4. 启动服务：
   - `make dev`
   - 或 `go run ./cmd/server --config config.yaml`

## UNO 开发注意事项
- UNO 模式仅支持主流输出格式：pdf、docx、xlsx、pptx
- UNO 实例异常退出时会自动重启，但当前请求仍可能失败
- 建议在开发环境通过日志文件定位 UNO 进程异常原因

## 常用命令

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
│   ├── SCRIPTS.md               # 脚本索引
│   ├── DEVELOPMENT.md           # 开发文档
│   ├── DEPLOMENT.md             # 部署文档
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

### 关键模块说明
- `cmd/server`：服务入口
- `internal/core/libreoffice`：CLI 与 UNO 执行器
- `internal/service`：业务流程编排
- `internal/infrastructure`：配置、日志、指标
- `scripts/uno_convert.py`：UNO 转换脚本

## 测试指南

### 单元测试
- 全量：`go test ./...`
- 指定包：`go test ./internal/core/libreoffice`

### 覆盖率
- `make coverage`

### 竞态检测
- `make race`

## 调试指南

### 常见问题
- UNO 模式报 `DisposedException`：通常是 UNO 进程异常退出
- UNO 无法启动：检查 `soffice` 路径与端口占用
- Python 无法导入 `uno`：需使用 LibreOffice 自带 Python 或安装 UNO 组件

### 日志建议
- 开发时将 `logger.level` 设为 `debug`
- 若需要排查 UNO 崩溃，建议保留 soffice stderr 输出

## 代码规范
- 所有注释必须使用中文
- 公共导出函数必须有中文文档注释
- 关键逻辑需添加中文行内注释

## 提交规范
- 语义化提交信息：`feat:`、`fix:`、`refactor:`、`test:`、`docs:`、`chore:`

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

