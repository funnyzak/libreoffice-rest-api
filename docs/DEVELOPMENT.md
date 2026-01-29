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

## 项目结构说明
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

