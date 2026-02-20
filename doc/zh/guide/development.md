# 开发

## 前置条件

- **Go** 1.25+
- **[Bun](https://bun.sh)** 1.x（替代 npm/yarn）
- **Make**（GNU Make）

## 环境搭建

```bash
git clone https://github.com/naiba/bonds.git
cd bonds

# 安装所有依赖（Go 模块 + Bun 包）
make setup

# 生成 API 客户端（首次构建前必须执行）
make gen-api

# 同时启动前后端开发模式
make dev
```

Go 后端运行在 `:8080`，Vite 开发服务器运行在 `:5173`，前端自动将 API 请求代理到后端。

## 代码生成管线

前端 TypeScript API 客户端从后端 OpenAPI/Swagger 规范**自动生成**：

```
Go handlers（swag 注解）
    ↓  make swagger
server/docs/swagger.json
    ↓  make gen-api
web/src/api/generated/   ← gitignored
    ↓
web/src/api/index.ts     ← 入口文件
```

修改后端 API（handlers、DTOs、routes）后：

```bash
make gen-api   # 重新生成 swagger.json + TypeScript 客户端
```

::: warning
禁止手动修改 `web/src/api/generated/` 下的文件，每次生成都会覆盖。
:::

## 常用命令

| 命令 | 说明 |
|------|------|
| `make dev` | 同时启动前后端开发模式 |
| `make build` | 分别构建后端和前端 |
| `make build-all` | 构建内嵌前端的单二进制文件 |
| `make test` | 运行所有测试（后端 + 前端） |
| `make test-server` | 仅运行后端测试 |
| `make test-web` | 仅运行前端测试 |
| `make test-e2e` | 运行 Playwright E2E 测试 |
| `make lint` | 运行代码检查（`go vet` + ESLint） |
| `make swagger` | 重新生成 Swagger/OpenAPI 文档 |
| `make gen-api` | 重新生成 Swagger + TypeScript API 客户端 |
| `make clean` | 清理所有构建产物 |
| `make setup` | 安装所有依赖 |

## 项目结构

```
server/                    # Go 后端
  cmd/server/main.go       # 入口
  internal/
    handlers/               # HTTP 处理器 (Echo)
    services/               # 业务逻辑
    models/                 # GORM 模型
    dto/                    # 请求/响应结构体
    middleware/              # JWT 认证、CORS 等
    search/                 # Bleve 全文搜索
    dav/                    # CardDAV/CalDAV 服务器
    cron/                   # Cron 调度器
    i18n/                   # 后端国际化
  pkg/
    avatar/                 # 首字母头像生成
    response/               # API 响应封装

web/                       # React 前端
  src/
    api/                    # 自动生成的 API 客户端
    components/             # 共享组件
    pages/                  # 路由页面
    stores/                 # Auth + Theme 上下文
    locales/                # i18n（en.json、zh.json）
    utils/                  # 工具函数
  e2e/                      # Playwright 测试
```

## 后端架构

每个功能遵循：**Handler**（HTTP 层）→ **Service**（业务逻辑）→ **DTO**（请求/响应）→ **Model**（GORM）。

- Handler 绑定请求、校验、委托给 Service，通过 `response.*` 辅助函数返回
- Service 接收 DTO、返回 DTO，持有 `*gorm.DB` 执行查询
- Model 是纯 GORM 结构体，不含业务逻辑

## 测试

```bash
# 后端测试（内存 SQLite）
cd server && go test ./... -v -count=1

# 前端单元测试（Vitest）
cd web && bun run test

# E2E 测试（Playwright — 自动启动服务器）
cd web && bunx playwright test
```

## API 文档（Swagger）

Bonds 自动生成覆盖全部 286 个 API 端点的 OpenAPI 文档：

```bash
DEBUG=true ./bonds-server
# 打开 http://localhost:8080/swagger/index.html
```

Swagger UI 仅在 `DEBUG=true` 时可用。
