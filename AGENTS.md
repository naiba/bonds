# AGENTS.md — Bonds

个人关系管理器。Go + React 单仓库。

## 构建 / 测试命令

```bash
# --- Go 后端（工作目录：server/）---
cd server
go build ./...                            # 编译所有包
go test ./... -v -count=1                 # 运行所有后端测试
go test ./internal/services -run TestCreateNote -v -count=1  # 运行单个测试
go test ./internal/handlers -v -count=1   # 仅运行 handler 集成测试
go vet ./...                              # 静态检查

# --- React 前端（工作目录：web/）---
cd web
bun run build                             # 类型检查 (tsc -b) + vite 构建
bun run test                              # vitest run（所有单元测试）
bun run test -- src/test/Login.test.tsx    # 运行单个测试文件
bun run lint                              # eslint

# --- E2E 测试（工作目录：web/）---
cd web && bunx playwright test            # 运行所有 e2e 用例（自动启动 server + vite）
bunx playwright test e2e/auth.spec.ts     # 运行单个 e2e 文件

# --- Makefile 快捷方式（从项目根目录）---
make test                                 # 后端 + 前端测试
make test-server / make test-web / make test-e2e
make build                                # 后端 + 前端分别构建
make build-all                            # 构建内嵌前端的单二进制文件
make dev                                  # 开发模式同时启动前后端
make swagger                              # 生成 Swagger 文档（swag init）
make gen-api                              # swagger + 生成前端 TypeScript API client
make setup                                # 安装依赖（go mod download + bun install）
```

**Go 代理（中国网络必须）：** 始终使用 `GOPROXY=https://goproxy.cn,direct` 执行 `go mod download`。

**包管理器：** 使用 `bun`，禁止 `npm` 或 `yarn`。

### 代码生成管线

前端 TypeScript API 客户端从后端 OpenAPI/Swagger 规范**自动生成**，生成的文件不纳入 git 版本控制。

```
Go handlers (swag 注解) → make swagger → server/docs/swagger.json
                        → make gen-api → web/src/api/generated/  (gitignored)
                                       → web/src/api/index.ts    (入口，引用 generated/)
```

- **修改后端 API 后**必须运行 `make gen-api` 重新生成前端客户端
- CI/Dockerfile 会在构建前自动生成，无需手动提交
- 生成工具：`swagger-typescript-api`（dev 依赖），配置在 `web/package.json` 的 `gen:api` 脚本
- **禁止手动修改 `web/src/api/generated/` 目录下的任何文件**

## 项目结构

```
server/                    # Go 后端（模块：github.com/naiba/bonds）
  cmd/server/main.go       # 入口 — Echo + GORM + Cron 初始化 + SPA 服务 + 信号优雅关闭
  internal/
    calendar/               # 多历法抽象：Converter 接口 + 注册表，gregorian.go（直通）、lunar.go（农历，6tail/lunar-go）
    config/                 # 基于环境变量的配置加载（含 SMTP/OAuth/Telegram/Geocoding/Bleve/WebAuthn）
    cron/                   # Cron 调度器（robfig/cron v3），支持数据库锁防重复执行
    database/               # GORM Connect + AutoMigrate
    dav/                    # CardDAV/CalDAV 服务器（emersion/go-webdav），Basic Auth + Backend 接口实现
    frontend/               # 内嵌前端静态文件（go:embed dist）
    i18n/                   # 国际化：embed 加载 en.json/zh.json，中间件解析 Accept-Language
    models/                 # 85+ GORM 结构体，registry.go 列出所有迁移模型
      seed.go               # 全局种子：SeedCurrencies（货币表）
      seed_account.go       # 账户级种子：SeedAccountDefaults（注册时调用）
      seed_vault.go         # Vault 级种子：SeedVaultDefaults（创建 vault 时调用）
    dto/                    # 请求/响应结构体（json + validate 标签）
    search/                 # 全文搜索引擎（Bleve v2），CJK 中文分词，Engine 接口 + NoopEngine
    services/               # 业务逻辑，每个领域一个文件
      calendar_convert.go   # 共享历法转换辅助函数 applyCalendarFields()
    handlers/               # HTTP 处理器（Echo），routes.go 统一注册路由
    middleware/              # JWT 认证、CORS、locale、vault 权限校验
    testutil/               # SetupTestDB（内存 SQLite）、TestJWTConfig
  pkg/
    avatar/                 # 头像生成：首字母 + 确定性颜色 → PNG（纯 stdlib image）
    response/               # API 响应封装：OK、Created、Paginated、各种错误

web/                       # React 前端（Vite + TypeScript）
  src/
    api/                    # API 客户端（自动生成）
      generated/            # swagger-typescript-api 生成的模块（gitignored，禁止手动修改）
      index.ts              # API 入口：实例化 HttpClient + 所有生成模块
    components/             # 共享组件（Layout.tsx、SearchBar.tsx、CalendarDatePicker.tsx）
    locales/                # 前端 i18n：en.json、zh.json（react-i18next）
    pages/                  # 按领域组织的路由页面（含 TwoFactor/Invitations/AcceptInvite/OAuthCallback）
    stores/                 # AuthProvider 上下文 + ThemeProvider（dark/light/system）
    types/                  # TypeScript 类型声明（仅 lunar-javascript.d.ts），DTO 类型统一从 @/api 导入
    utils/                  # 工具函数（calendar.ts — 前端多历法抽象 + 注册表）
    test/                   # Vitest 单元测试 + setup.ts
    i18n.ts                 # react-i18next 初始化 + 语言检测
  e2e/                      # Playwright 测试用例

Dockerfile                 # 多阶段构建：bun build → go embed → 单二进制
docker-compose.yml         # 单容器部署
.github/workflows/
  test.yml                 # CI：任何 push / PR 触发
  release.yml              # CD：test.yml 成功 + v* tag 时 workflow_run 触发
```

## Go 后端约定

### 架构：Handler → Service → DTO

每个功能遵循：**handler**（HTTP 层）→ **service**（业务逻辑）→ **dto**（请求/响应）→ **model**（GORM）。

- Handler 绑定请求、校验、委托给 Service，通过 `response.*` 辅助函数返回。
- Service 接收 DTO、返回 DTO。持有 `*gorm.DB`，负责所有查询逻辑。
- Model 是纯 GORM 结构体，不含业务逻辑。

### ID 类型

- `Account`、`User`、`Vault`、`Contact` 使用 **UUID string** 主键（`gorm:"primaryKey;type:text"` + `BeforeCreate` 钩子）。
- 其余所有模型使用 **自增 uint**（`gorm:"primaryKey;autoIncrement"`）。

### 错误处理

- Service 定义哨兵错误：`var ErrNoteNotFound = errors.New("note not found")`
- Handler 通过 `errors.Is(err, services.ErrXxxNotFound)` 判断 → `response.NotFound(c, "...")`
- 通用错误 → `response.InternalError(c, "...")`
- 禁止将原始数据库错误暴露给客户端。

### 响应封装

所有 API 响应使用 `pkg/response/response.go`：`{success, data, error, meta}`。
使用辅助函数：`response.OK`、`response.Created`、`response.Paginated`、`response.BadRequest`、`response.NotFound`、`response.InternalError`、`response.ValidationError`、`response.NoContent`。

### 命名规范

- 文件：`snake_case.go` — 每个领域一个文件（如 `note.go`、`important_date.go`）
- 类型：`PascalCase` — `NoteService`、`NoteHandler`、`CreateNoteRequest`、`NoteResponse`
- 构造函数：`NewXxxService(db *gorm.DB)`、`NewXxxHandler(svc *XxxService)`
- 响应转换器：私有 `toXxxResponse(model) dto.XxxResponse`，放在 service 文件内
- 哨兵错误：`var ErrXxxNotFound = errors.New("xxx not found")`

### 种子数据（Seed）

参照 Monica PHP `SetupAccount` Job 和 `CreateVault` Service，分三层种子：

**全局种子**（`seed.go`）— 应用启动时：
- `SeedCurrencies`：160+ 种货币

**账户级种子**（`seed_account.go`）— 用户注册时在事务内调用 `SeedAccountDefaults(tx, accountID, userID, email)`：
- Gender（3）、Pronoun（7）、AddressType（5）、PetCategory（10）
- ContactInformationType（12，含 email/phone 不可删除）
- RelationshipGroupType（4 组 17 种关系类型）
- CallReasonType（2 类 7 条原因）、Religion（9）、GroupType（5 + roles）
- Emotion（3）、GiftOccasion（5）、GiftState（5）、PostTemplate（2）
- Template（1 默认模板 + 5 TemplatePage）
- Module（24 个默认模块） + ModuleTemplatePage（模块绑定到模板页的 pivot）
- UserNotificationChannel（用户 email 通知）
- AccountCurrency（关联所有货币到账户）

**默认模板页 → 模块映射**（`seedDefaultModules`）：
- "Contact information" (slug: `contact`) → avatar, contact_names, family_summary, important_dates, gender_pronoun, labels, company, religions
- "Feed" (slug: `feed`) → feed
- "Social" (slug: `social`) → relationships, pets, groups, addresses, contact_information
- "Life & goals" (slug: `life-goals`) → life_events, goals
- "Information" (slug: `information`) → documents, photos, notes, reminders, loans, tasks, calls, posts

**Vault 级种子**（`seed_vault.go`）— 创建 Vault 时在事务内调用 `SeedVaultDefaults(tx, vaultID)`：
- ContactImportantDateType（2：Birthdate、Deceased date，不可删除）
- MoodTrackingParameter（5 级，带 emoji + Tailwind 颜色）
- LifeEventCategory（4 类 20 种事件类型）
- VaultQuickFactsTemplate（2：Hobbies、Food preferences）

### 个性化设置（Personalize）

- API 路径：`/api/settings/personalize/:entity`
- entity key 使用 **kebab-case**：`genders`、`pronouns`、`address-types`、`pet-categories`、`contact-info-types`、`call-reasons`、`religions`、`gift-occasions`、`gift-states`、`group-types`、`post-templates`、`relationship-types`、`templates`、`modules`、`currencies`
- 后端 `entityConfigs` map 在 `services/personalize.go` 中定义 entity → 表名映射

### 测试

- **Service 测试** 在 `internal/services/xxx_test.go`（同包 — `package services`）
- **Handler 集成测试** 统一在 `internal/handlers/handlers_test.go`（包名 `handlers_test`）
- 每个测试文件有 `setupXxxTest(t)` 辅助函数，调用 `testutil.SetupTestDB(t)`，注册用户、创建 vault/contact，返回 service + ID
- 注册会触发 `SeedAccountDefaults`，创建 vault 会触发 `SeedVaultDefaults`，测试中需注意已有种子数据的计数
- 仅使用标准库 `testing`。不用 testify、gomock。
- 测试使用内存 SQLite：`testutil.SetupTestDB(t)`。

### SQLite 注意事项

- **必须禁用 PrepareStmt**（见 `database.go`），否则 SQLite 会报 "cannot commit transaction"。
- GORM `Create` 会跳过零值 bool 字段（`false` 视为零值）。配合 `gorm:"default:true"` 时，想设为 `false` 必须先 Create 再单独 `Update("field", false)`。种子数据中多处使用此技巧（`can_be_deleted`、`active` 等）。
- 中间表（pivot table）必须有显式的 `ID uint gorm:"primaryKey;autoIncrement"` 字段。

## React 前端约定

### 技术栈

React 19、TypeScript 严格模式、Vite 7、Ant Design v6、TanStack Query v5、React Router v7、Axios、react-i18next。

### 导入规范

- 内部导入统一使用 `@/` 路径别名（映射到 `src/`）。
- 仅类型导入必须用 `import type { X }`（由 `verbatimModuleSyntax` 强制）。

### 组件

- 每个页面使用默认导出：`export default function Login() { ... }`
- 页面在 App.tsx 中使用 `React.lazy()` + `Suspense` 实现代码分割。
- 测试中所有页面需包裹 `<ConfigProvider>` + `<App>`（Ant Design 上下文）。
- 所有用户可见文本使用 `t()` 函数（react-i18next），翻译键定义在 `src/locales/en.json` 和 `zh.json`。

### API 层

- `src/api/generated/` 下的文件由 `swagger-typescript-api` 从后端 OpenAPI 规范自动生成，**禁止手动修改**。
- `src/api/index.ts` 是 API 入口：创建 HttpClient（Axios，baseURL `/api`，JWT interceptor，401 重定向），实例化所有生成的 API 模块。
- 页面通过 `import { api } from "@/api"` 使用，如 `api.contacts.contactsList({ vaultId })`。
- 生成的方法直接返回解包后的响应体（`{success, data, error, meta}`），无需 `.data` 二次解包。

### 测试（Vitest）

- 测试文件在 `src/test/` 下，命名为 `Xxx.test.tsx`。
- 需要 auth 上下文的组件使用 `vi.mock("@/stores/auth", ...)` 模拟。
- 使用 `useTheme()` 的组件（Login/Register/AcceptInvite 等 auth 页面）需要 `vi.mock("@/stores/theme", ...)` 模拟，返回 `{ themeMode: "system", resolvedTheme: "light", setThemeMode: vi.fn() }`。
- 渲染包裹：`<ConfigProvider><AntApp><MemoryRouter>...</MemoryRouter></AntApp></ConfigProvider>`。
- 使用 `@testing-library/react` + `@testing-library/user-event`。
- Setup 文件：`src/test/setup.ts` — 为 Ant Design 填充 `matchMedia` polyfill。

### E2E（Playwright）

- 测试用例在 `web/e2e/` — `auth.spec.ts`、`vault.spec.ts`、`contact.spec.ts`、`calendar.spec.ts`、`search.spec.ts`、`settings.spec.ts`、`settings-enhanced.spec.ts`、`file-upload.spec.ts`、`groups.spec.ts`、`life-events.spec.ts`、`new-features.spec.ts`、`vault-features.spec.ts`、`vault-files.spec.ts`、`bugfixes.spec.ts`、`contact-modules.spec.ts`、`contact-modules-extended.spec.ts`、`missing-features.spec.ts`、`vault-extended.spec.ts`。
- Playwright 自动启动 Go 服务器（端口 8080）和 Vite 开发服务器（端口 5173）。
- Ant Design 表单：使用 `page.getByPlaceholder(...)` 而非 `getByLabel(...)`。
- **E2E 自动清理旧 DB**：`playwright.config.ts` 的 `webServer.command` 会在启动 Go 服务器前自动删除 `server/bonds.db*`。CI 环境下始终生效；本地 `reuseExistingServer=true` 时跳过（复用已运行的服务器）。

### Cron 调度器

- 使用 `robfig/cron/v3`，支持秒级精度（`cron.WithSeconds()`）
- `Scheduler.RegisterJob(spec, name, fn)` 自动数据库锁 + panic 恢复
- 集成到 `main.go`，优雅关闭：SIGINT/SIGTERM → cron 停止（30s） → echo shutdown（10s）
- 当前注册的 cron 任务：`process_reminders`（每分钟扫描到期提醒）

### 提醒/通知系统

- `ReminderSchedulerService.ProcessDueReminders()` 每分钟运行
- 扫描 `ContactReminderScheduled` WHERE `scheduled_at <= now AND triggered_at IS NULL`
- 根据 `UserNotificationChannel.Type` 分发（email/telegram）
- 失败计数：`UserNotificationChannel.Fails` 递增，达到 10 次自动禁用
- 重复提醒：根据 `ContactReminder.Type`（one_time/recurring_week/recurring_month/recurring_year）自动调度下一次
- `UserNotificationSent` 记录每次发送结果（含错误信息）

### 全文搜索（Bleve）

- `internal/search/` 包定义 `Engine` 接口 + `BleveEngine` 实现 + `NoopEngine` 降级
- CJK 分析器作为默认分析器，支持中英文混合搜索
- 索引实体：Contact（名字、昵称）、Note（标题、正文）
- 增量索引：Service 层在 CRUD 操作后自动更新索引
- 搜索权限隔离：查询强制按 vault_id 过滤
- 配置：`BLEVE_INDEX_PATH`，默认 `data/bonds.bleve`

### CardDAV/CalDAV 服务器

- 使用 `emersion/go-webdav` 库，实现 `carddav.Backend` 和 `caldav.Backend` 接口
- 路由挂载在 `/dav/`，使用 Basic Auth（非 JWT，因 DAV 客户端不支持）
- CardDAV：联系人 → vCard 4.0（姓名、电话、邮箱、地址）
- CalDAV：重要日期 → VEVENT（RRULE=YEARLY）、任务 → VTODO
- ETag 基于 `UpdatedAt` Unix 时间戳
- Well-known 发现：`/.well-known/carddav` 和 `/.well-known/caldav` 重定向到 `/dav/`

### 文件上传

- 上传端点：`POST /api/vaults/:vault_id/files`、`POST .../contacts/:contact_id/photos`、`POST .../documents`
- MIME 白名单：image/jpeg, image/png, image/gif, image/webp, application/pdf 等
- 大小限制：10MB（可配置 `STORAGE_MAX_SIZE`）
- 存储结构：`{uploadDir}/{yyyy/MM/dd}/{uuid}{ext}`
- 下载：`GET /api/vaults/:vault_id/files/:id/download`

### 头像系统

- `pkg/avatar/` 包：纯 Go stdlib `image` 包生成首字母 PNG
- 确定性颜色：基于名字 MD5 哈希选择预定义色板
- `GET /api/vaults/:vault_id/contacts/:contact_id/avatar` — 有上传头像则返回文件，否则生成 initials

### 2FA TOTP

- 使用 `pquerna/otp` 库，Issuer 为 "Bonds"
- 启用流程：Enable → 返回 secret + recovery codes → Confirm（TOTP 验证）→ TwoFactorConfirmedAt 设置
- 登录流程两步：密码验证后若 2FA 启用 → 返回 `requires_two_factor: true` + temp token → 提交 TOTP code → 返回正式 JWT
- Recovery codes：8 个 8 字符随机码，使用后消耗
- API：`/api/settings/2fa/{enable,confirm,disable,status}`

### OAuth 登录

- 使用 `markbates/goth` 库，支持 GitHub + Google
- 流程：`GET /api/auth/:provider` → 跳转 OAuth → `GET /api/auth/:provider/callback` → JWT → 重定向前端 `/auth/callback?token=xxx`
- 账户关联：同邮箱自动绑定已有账户
- 配置：`OAUTH_GITHUB_KEY/SECRET`、`OAUTH_GOOGLE_KEY/SECRET`

### WebAuthn/FIDO2

- 使用 `go-webauthn/webauthn` 库
- `WebAuthnCredential` 模型存储公钥凭证
- 注册：`/api/settings/webauthn/register/{begin,finish}`
- 登录：`/api/auth/webauthn/login/{begin,finish}`
- 需要 HTTPS（localhost 除外）
- 配置：`WEBAUTHN_RP_ID`、`WEBAUTHN_RP_DISPLAY_NAME`、`WEBAUTHN_RP_ORIGINS`

### 用户邀请

- `Invitation` 模型：Token（UUID）、Permission（100=Manager/200=Editor/300=Viewer）、7 天过期
- 发送邀请 → 邮件含链接 `{APP_URL}/accept-invite?token=xxx`
- 接受：公开端点 `POST /api/invitations/accept` → 创建用户，关联到同一 Account
- API：`/api/settings/invitations`（CRUD）

### vCard Import/Export

- 使用 `emersion/go-vcard` 库
- 导出：`GET /api/vaults/:vault_id/contacts/:contact_id/vcard` → text/vcard
- 批量导出：`GET /api/vaults/:vault_id/contacts/export`
- 导入：`POST /api/vaults/:vault_id/contacts/import` → multipart .vcf 文件
- 映射：FN↔FirstName+LastName、TEL↔phone ContactInfo、EMAIL↔email ContactInfo、ADR↔Address

### Telegram 通知

- 使用 `go-telegram-bot-api/telegram-bot-api/v5`
- `TelegramService.SendReminder(chatID, contactName, label)` 发送格式化消息
- 配置：`TELEGRAM_BOT_TOKEN`，未配置则降级为不可用
- 与 `UserNotificationChannel.Type="telegram"` 集成

### 地理编码（Geocoding）

- `Geocoder` 接口 + `NominatimGeocoder`（免费 OSM）+ `LocationIQGeocoder`（API key）
- 地址创建时异步编码，失败不影响主流程
- 配置：`GEOCODING_PROVIDER`（nominatim/locationiq）、`GEOCODING_API_KEY`

### 审计日志（Feed）

- `FeedRecorder.Record(contactID, authorID, action, description, feedableID, feedableType)`
- 操作常量：`ActionContactCreated`、`ActionNoteCreated`、`ActionReminderCreated` 等 15 种
- 集成到 ContactService、NoteService、ReminderService 等，CRUD 操作后自动记录
- 通过 `GET /api/vaults/:vault_id/feed` 查看

## 代码质量规则

以下规则由工具链自动强制执行（ESLint、TypeScript 严格模式、CI），违反时构建/lint 会直接失败，无需人工检查：

- `as any`、`@ts-ignore`、`@ts-expect-error`、空 catch 块 → ESLint 报错（`src/api/*.ts` 生成代码除外）
- `noUnusedLocals`、`noUnusedParameters`、`noFallthroughCasesInSwitch` → TypeScript 编译报错
- React Hooks 规则（`set-state-in-effect`、`refs`）→ ESLint 报错
- i18n key 一致性（en.json 与 zh.json）→ `bun run lint` 自动检查
- Go `go vet` → CI 强制

**非工具强制的设计指导：**

- 格式化：Go 使用 `gofmt`，前端使用 Prettier。
- 组件优先使用**受控模式**（状态由父组件通过 `value`/`onChange` 管理），避免内部 `useState` + `useEffect` 同步 prop 的反模式。

## 项目规模（供参考）

| 维度 | 数量 |
|------|------|
| Go Model 文件 | 49 |
| Go Handler 文件 | 67（含 swag 注解） |
| Go Service 文件 | 89（非测试） |
| Go DTO 文件 | 44（含 example 标签） |
| API 路由（Swagger 统计） | 194 paths / 286 operations |
| React 页面组件 | 43 |
| 前端 API 客户端 | 23 |
| i18n 翻译键 | ~576（en + zh 各一份） |

### 测试数量明细

| 层级 | 文件数 | 测试函数数 |
|------|--------|-----------|
| Go Service 测试 | 84 | ~425 |
| Go Handler 集成测试 | 1 | 109 |
| Go Cron 测试 | 1 | 7 |
| Go DAV 测试 | 2 | 20 |
| Go Search 测试 | 1 | 4 |
| Go Avatar 测试 | 1 | 7 |
| Go Calendar 测试 | 1 | 13 |
| **Go 后端总计** | **91** | **~585** |
| React Vitest | 24 | 82 |
| Playwright E2E | 18 | 104 |
| **全部总计** | **133** | **771+** |

## 已知坑和注意事项

### GORM + SQLite

1. **必须禁用 PrepareStmt**（见 `database.go`），否则 SQLite 报 "cannot commit transaction"。
2. **零值 bool 字段陷阱**：GORM `Create` 跳过 `false`（视为零值）。配合 `gorm:"default:true"` 时，想设为 `false` 必须先 Create 再 `Update("field", false)`。种子数据中 `can_be_deleted`、`active` 等多处使用此技巧。
3. **中间表必须有 ID**：pivot table 必须有显式 `ID uint gorm:"primaryKey;autoIncrement"` 字段，否则 GORM 行为异常。

### 测试基础设施

1. **`setupTestServer(t)`**（`handlers_test.go`）— 标准 handler 集成测试，内存 SQLite + NoopMailer + NoopSearchEngine。不配置 Storage.UploadDir、Bleve、SMTP、WebAuthn。
2. **`setupTestServerWithStorage(t)`** — 同上但预配置 `Storage.UploadDir = t.TempDir()`，避免 `RegisterRoutes` 重复调用。文件上传测试**必须**用这个。
3. **`testutil.SetupTestDB(t)`** — 创建内存 GORM DB + AutoMigrate 所有模型。Service 测试直接用这个。
4. **种子数据对测试的影响**：注册用户触发 `SeedAccountDefaults`，创建 vault 触发 `SeedVaultDefaults`。测试中做计数断言时必须考虑已有种子数据。
5. **NoopMailer**：`type NoopMailer struct{}` 实现 `Mailer` 接口（`Send(to, subject, htmlBody string) error` + `Close()`），测试中用于替代真实邮件发送。
6. **NoopSearchEngine**：`search.NoopEngine{}` 实现 `search.Engine` 接口，测试中用于替代 Bleve。

### Handler 测试模式

```go
// 标准 handler 测试流程
srv, cleanup := setupTestServer(t)  // 或 setupTestServerWithStorage(t)
defer cleanup()
// 1. 注册用户 → POST /api/auth/register
// 2. 从响应提取 JWT token
// 3. 创建 vault → POST /api/vaults（带 Authorization header）
// 4. 调用被测 API
// 5. 断言响应状态码 + JSON body
```

### Flaky Test 经验

- `TestFileUpload_Success` 曾间歇性失败 — 原因是 `setupTestServer` 已调用 `RegisterRoutes`，测试中又手动调用了一次，导致 Echo 路由重复注册。解决：创建 `setupTestServerWithStorage()` 在路由注册前就配置好 Storage。

### DAV 注意事项

- DAV 客户端使用 Basic Auth（非 JWT），需要单独的认证层。
- `emersion/go-webdav` 是目前唯一成熟的 Go CardDAV/CalDAV 库。
- ETag 基于 `UpdatedAt` Unix 时间戳。

### 提醒调度器

- `maxChannelFails = 10` — 通知渠道失败次数达到 10 后自动禁用（`active = false`）。
- 重复提醒调度下一次时，基于**当前 scheduled_at** 计算而非当前时间，防止漂移。

### Swagger / OpenAPI

- `server/docs/` 在 `.gitignore` 中，**不纳入版本控制**。
- Go 代码中 `routes.go` 有 `_ "github.com/naiba/bonds/docs"` 空导入，构建前**必须先运行** `swag init`。
- 本地开发：`cd server && swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal`（或 `make swagger`）
- CI（`.github/workflows/test.yml`）在 `go vet` / `go build` 之前执行 `swag init`，否则构建失败。
- Dockerfile 同样在 `go build` 前执行 `swag init`（行 20）。
- 安装 swag：`go install github.com/swaggo/swag/cmd/swag@latest`
 Swagger UI 默认在 `DEBUG=true` 时启用，也可通过管理后台 **Admin → Settings → Swagger** 的 `swagger.enabled` 开关控制（运行时生效，无需重启）。
- 使用 `echo-swagger` **v1.4.1**（对应 Echo v4）。v1.5.0+ 依赖 Echo v5，不兼容。
- **swag 类型解析陷阱**：handler 文件中的 `@Success ... dto.XxxResponse` 注解要求该文件能解析到 `dto` 包。如果 handler 的 Go 代码本身不 import `dto`（如 `currencies.go`、`vault_files.go`），swag 会报 `cannot find type definition`。解决方法：在文件中添加 `import "github.com/naiba/bonds/internal/dto"` + `var _ dto.XxxResponse`（类型锚点，防止 unused import 编译错误）。当前已有此模式的文件：`currencies.go`、`storage_info.go`、`user_management_extra.go`、`webauthn.go`、`avatar.go`、`calendar.go`、`companies.go`、`contact_photos.go`、`feed.go`、`post_photos.go`、`reports.go`、`vault_files.go`、`vault_tasks.go`、`vcard.go`。
- 全局注解（`@title`、`@BasePath`、`@securityDefinitions`）在 `cmd/server/main.go` 的 `func main()` 上方。
- 当前统计：194 paths、286 operations、184 definitions。

### 前端 i18n 注意事项

- 翻译文件为嵌套 JSON 结构（`src/locales/en.json`、`zh.json`），使用点号路径访问（如 `t("vault.companies.title")`）。
- 新增页面**必须同时**在 en.json 和 zh.json 中添加对应翻译键，否则 UI 上会显示原始键路径。（`bun run lint` 自动检查，CI 强制）
- Vitest 单元测试中 i18n 会被真实加载（非 mock），因此测试断言应匹配**翻译后的文本**（如 `"Vault Settings"`），而非键路径（如 `"vault_settings.title"`）。

### Vitest 本地 vs CI 差异（重要）

- **jsdom v28 + undici 平台行为差异**：同一版本的 jsdom 在 macOS 和 Linux 上行为不同。macOS/bun 环境下，未 mock 的 HTTP 请求会静默失败；CI Ubuntu 环境下，undici 会严格校验并抛出 `InvalidArgumentError: invalid onError method`，导致 unhandled rejection。
- **Vitest unhandled rejection = exit code 1**：即使所有测试用例都通过（85/85 passed），vitest 遇到 unhandled rejection 仍会以 exit code 1 退出，导致 CI 红灯。
- **解决方案**：测试中必须 mock 所有可能发出真实 HTTP 请求的模块。不仅要 mock `@tanstack/react-query`（覆盖 `useQuery`/`useMutation`），还要 mock `@/api`（覆盖直接使用 `httpClient.instance.get()` 的组件，如 `AvatarImageLoader`）。
- **规则**：新增组件测试时，检查组件及其子组件是否有绕过 react-query 直接使用 `httpClient` 的代码路径。如有，必须 mock `@/api` 模块的 `httpClient.instance`。

### 前端新增依赖必须加入 package.json

- 本地 `node_modules` 可能有全局或其他项目安装的包，本地测试通过但 CI 失败。
- 新引入第三方包时**必须** `bun add <package>` 确保写入 `package.json` 和 `bun.lock`。
- 已踩坑的包：`filesize`、`@simplewebauthn/browser` — 本地存在但未加入 `package.json`，导致 CI 构建失败。

### Playwright E2E 测试经验

- Ant Design 组件在 Playwright strict mode 下容易因多个元素匹配而失败（如 `.ant-card` 匹配多个卡片、`getByText` 在导航栏和内容区同时匹配）。解决：使用 `.first()`、`getByRole('table').getByText(...)` 等更精确的选择器。
- 联系人创建后会自动跳转到详情页，测试中应先 `await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/)` 等待导航完成，再断言页面内容。
- **Ant Design Select 下拉遮挡**：多个 Select 紧挨时，前一个 Select 的 dropdown 可能遮挡后一个。选完后点击 `modal.locator('.ant-modal-header').click()` 让 dropdown 失焦关闭。**不要用 `Escape`**——它会关闭整个 Modal。
- **表单 auto-fill 与 E2E 操作顺序**：`ImportantDatesModule` 选择 `internal_type=true` 的 date type 时会自动覆写 label 字段。E2E 测试中必须**先选 type 再填 label**，否则用户输入的 label 会被 auto-fill 覆盖。类似的 auto-fill 逻辑在其他模块中也可能存在，写 E2E 时需注意表单字段间的联动副作用。
- **Contact Detail 动态 Tab 名称**：Tab 名来自后端 seed 数据的 `TemplatePage.Name`（"Contact information"、"Feed"、"Social"、"Life & goals"、"Information"），与前端 fallback tabs 的 i18n 翻译名不同。E2E 选择 tab 时必须用 seed 数据中的名称，且注意 "Contact information" 和 "Information" 两个 tab 共存，用 `{ name: 'Information', exact: true }` 精确匹配。
- **Contact Detail 页面多个同名按钮**：动态 tabs 加载后，第一个 tab 默认展开所有模块，每个模块可能有 "Edit" 按钮。选择顶部操作栏的 Edit 按钮时用 `.first()`。

### 项目规模（供参考）

| 维度 | 数量 |
|------|------|
| Go Model 文件 | 49 |
| Go Handler 文件 | 67（含 swag 注解） |
| Go Service 文件 | 89（非测试） |
| Go DTO 文件 | 44（含 example 标签） |
| API 路由（Swagger 统计） | 194 paths / 286 operations |
| React 页面组件 | 43 |
| 前端 API 客户端 | 23 |
| i18n 翻译键 | ~576（en + zh 各一份） |

## 关键依赖版本

### Go 后端（go 1.25.2）

| 依赖 | 版本 | 用途 |
|------|------|------|
| `labstack/echo/v4` | v4.15.0 | HTTP 框架 |
| `gorm.io/gorm` | v1.31.1 | ORM |
| `gorm.io/driver/sqlite` | v1.6.0 | SQLite 驱动 |
| `blevesearch/bleve/v2` | v2.5.7 | 全文搜索 |
| `emersion/go-webdav` | v0.7.0 | CardDAV/CalDAV |
| `emersion/go-vcard` | latest | vCard 解析 |
| `emersion/go-ical` | latest | iCal 解析 |
| `go-webauthn/webauthn` | v0.15.0 | FIDO2/WebAuthn |
| `markbates/goth` | v1.82.0 | OAuth |
| `pquerna/otp` | v1.5.0 | TOTP 2FA |
| `robfig/cron/v3` | v3.0.1 | Cron 调度器 |
| `jordan-wright/email` | v4.0.1 | SMTP 发送 |
| `golang-jwt/jwt/v5` | v5.3.1 | JWT |
| `golang.org/x/crypto` | v0.48.0 | bcrypt 等 |
| `swaggo/swag` | v1.16.6 | Swagger 文档生成 |
| `swaggo/echo-swagger` | v1.4.1 | Swagger UI 中间件（必须 v1.4.x，v1.5+ 依赖 Echo v5） |
| `6tail/lunar-go` | v1.4.6 | 农历转换 |

### React 前端

| 依赖 | 版本 | 用途 |
|------|------|------|
| `react` | ^19.2.0 | UI 框架 |
| `antd` | ^6.3.0 | 组件库 |
| `@tanstack/react-query` | ^5.90.21 | 数据请求 |
| `react-router-dom` | ^7.13.0 | 路由 |
| `axios` | ^1.13.5 | HTTP 客户端 |
| `i18next` + `react-i18next` | ^25.8.7 / ^16.5.4 | 国际化 |
| `vite` | ^7.3.1 | 构建工具 |
| `vitest` | ^4.0.18 | 测试框架 |
| `@playwright/test` | ^1.58.2 | E2E 测试 |
| `typescript` | ~5.9.3 | 类型系统 |
| `filesize` | ^11.0.13 | 文件大小格式化 |
| `@simplewebauthn/browser` | ^13.2.2 | WebAuthn 客户端 |

## 环境变量完整列表

参见 `server/.env.example`，包含所有可配置项及默认值。分组：

 **Debug**：`DEBUG`（默认 `false`）— 启用 Echo 请求日志、GORM SQL 日志、Swagger UI（默认开启，也可通过管理后台 `swagger.enabled` 独立控制）
- **Core**：`SERVER_PORT`、`DB_DSN`、`JWT_SECRET`、`APP_ENV`、`APP_URL`
- **SMTP**：`SMTP_HOST`、`SMTP_PORT`、`SMTP_USERNAME`、`SMTP_PASSWORD`、`SMTP_FROM`
- **Storage**：`STORAGE_UPLOAD_DIR`（默认 `uploads`）、`STORAGE_MAX_SIZE`（默认 10MB）
- **Search**：`BLEVE_INDEX_PATH`（默认 `data/bonds.bleve`）
- **Telegram**：`TELEGRAM_BOT_TOKEN`
- **OAuth**：`OAUTH_GITHUB_KEY/SECRET`、`OAUTH_GOOGLE_KEY/SECRET`
- **Geocoding**：`GEOCODING_PROVIDER`（nominatim/locationiq）、`GEOCODING_API_KEY`
- **WebAuthn**：`WEBAUTHN_RP_ID`、`WEBAUTHN_RP_DISPLAY_NAME`、`WEBAUTHN_RP_ORIGINS`
- **其他**：`ANNOUNCEMENT`（全局公告横幅文字）
