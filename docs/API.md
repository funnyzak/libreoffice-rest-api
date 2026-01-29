# API 参考

## 认证

- 默认启用 API Key 认证。
- 支持 Header `X-API-Key` 或 `Authorization: Bearer <key>`。
- 下载接口是否需要认证可通过 `download.require_auth` 配置控制，默认开启。

## 统一响应格式

### 成功响应
```json
{
  "success": true,
  "data": {},
  "message": "操作成功"
}
```

### 错误响应
```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "Bad Request",
    "details": "详细错误信息"
  }
}
```

> 同步转换与文件下载为文件流响应，失败时仍返回统一错误格式。

## 接口列表

### 提交转换任务

`POST /api/v1/convert`

#### 表单上传
- Content-Type: `multipart/form-data`
- 字段说明:
  - `file`: 上传文件
  - `format`: 输出格式（常用：pdf/html/png/txt/odt/doc/docx/rtf/epub/xls/xlsx/ods/csv/ppt/pptx/odp/odg/svg/jpg/jpeg/webp）
  - `mode`: async/sync，默认 async
  - `binary`: 同步模式下是否直接返回二进制，true/false，默认 true
  - `url`: 可选，URL 模式时传入

示例响应（异步）：
```json
{
  "success": true,
  "data": {
    "task_id": "...",
    "download_url": "http://host/api/v1/files/{id}/download"
  },
  "message": "任务已提交"
}
```
> `download_url` 会优先使用 `server.public_base_url` 作为基础地址（如配置为空则使用请求 Host）。

#### URL 提交
- Content-Type: `application/json`

```json
{
  "url": "https://example.com/demo.docx",
  "format": "pdf",
  "mode": "sync",
  "binary": false
}
```

#### 同步模式返回
- **binary=true (默认)**: Content-Type: `application/octet-stream`，返回文件流
- **binary=false**: 返回 JSON 格式，包含 task_id 和 download_url

示例响应 (sync + binary=false):
```json
{
  "success": true,
  "data": {
    "task_id": "...",
    "download_url": "http://host/api/v1/files/{id}/download",
    "output_name": "document.pdf",
    "output_format": "pdf"
  },
  "message": "转换完成"
}
```

### 合并文档

`POST /api/v1/merge`

> 合并仅支持输出 PDF，且至少需要两个文件或 URL，可通过 `mode` 选择同步或异步。

#### 表单上传
- Content-Type: `multipart/form-data`
- 字段说明:
  - `files`: 上传文件（可多次传入）
  - `format`: 输出格式（仅支持 pdf）
  - `mode`: async/sync，默认 async
  - `binary`: 同步模式下是否直接返回二进制，true/false，默认 true
  - `urls`: 可选，URL 列表

示例响应（异步）：
```json
{
  "success": true,
  "data": {
    "task_id": "...",
    "download_url": "http://host/api/v1/files/{id}/download"
  },
  "message": "任务已提交"
}
```
> `download_url` 会优先使用 `server.public_base_url` 作为基础地址（如配置为空则使用请求 Host）。

#### URL 提交
- Content-Type: `application/json`

```json
{
  "urls": [
    "https://example.com/part1.docx",
    "https://example.com/part2.docx"
  ],
  "format": "pdf",
  "mode": "sync",
  "binary": false
}
```

#### 同步模式返回
- **binary=true (默认)**: Content-Type: `application/pdf`，返回文件流
- **binary=false**: 返回 JSON 格式，包含 task_id 和 download_url

示例响应 (sync + binary=false):
```json
{
  "success": true,
  "data": {
    "task_id": "...",
    "download_url": "http://host/api/v1/files/{id}/download",
    "output_name": "merged.pdf",
    "output_format": "pdf"
  },
  "message": "合并完成"
}
```

### 查询任务状态

`GET /api/v1/tasks/:id`

示例响应：
```json
{
  "success": true,
  "data": {
    "id": "...",
    "status": "success",
    "download_url": "http://host/api/v1/files/{id}/download",
    "output_format": "pdf",
    "error": ""
  },
  "message": "查询成功"
}
```
> `download_url` 会优先使用 `server.public_base_url` 作为基础地址（如配置为空则使用请求 Host）。

### 下载文件

`GET /api/v1/files/:id/download`

- 返回转换结果文件流
- 是否需要认证由 `download.require_auth` 控制

### 健康检查

`GET /health`

示例响应：
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "checks": {
      "database": "healthy",
      "disk": "healthy: free 10GB",
      "libreoffice": "healthy"
    },
    "time": "2026-01-28T00:00:00Z"
  },
  "message": "健康检查"
}
```

### 指标

`GET /metrics`

- Prometheus 指标，需在配置中启用，可通过 `metrics.require_auth` 控制认证。

### Swagger 文档

- 地址: `/swagger/index.html`（默认无需认证）
- 通过 `make swagger` 生成文档，可通过配置关闭或启用认证
