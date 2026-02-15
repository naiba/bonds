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
make setup                                # 安装依赖（go mod download + bun install）
```

**Go 代理（中国网络必须）：** 始终使用 `GOPROXY=https://goproxy.cn,direct` 执行 `go mod download`。

**包管理器：** 使用 `bun`，禁止 `npm` 或 `yarn`。

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
    api/                    # Axios API 客户端模块，每个领域一个文件（含 twofactor/search/invitations/vcard）
    components/             # 共享组件（Layout.tsx、SearchBar.tsx、CalendarDatePicker.tsx）
    locales/                # 前端 i18n：en.json、zh.json（react-i18next）
    pages/                  # 按领域组织的路由页面（含 TwoFactor/Invitations/AcceptInvite/OAuthCallback）
    stores/                 # AuthProvider 上下文
    types/                  # TypeScript 类型定义（含 lunar-javascript.d.ts）
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
- UserNotificationChannel（用户 email 通知）
- AccountCurrency（关联所有货币到账户）

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

- `src/api/` 下每个领域一个文件，导出 const 对象（如 `export const notesApi = { list, create, ... }`）。
- API 客户端在 `src/api/client.ts` — Axios，baseURL 为 `/api`，自动从 localStorage 附加 JWT。
- ID 参数类型为 `string | number`，因为 UUID 从路由参数中以字符串形式获取。

### 测试（Vitest）

- 测试文件在 `src/test/` 下，命名为 `Xxx.test.tsx`。
- 需要 auth 上下文的组件使用 `vi.mock("@/stores/auth", ...)` 模拟。
- 渲染包裹：`<ConfigProvider><AntApp><MemoryRouter>...</MemoryRouter></AntApp></ConfigProvider>`。
- 使用 `@testing-library/react` + `@testing-library/user-event`。
- Setup 文件：`src/test/setup.ts` — 为 Ant Design 填充 `matchMedia` polyfill。

### E2E（Playwright）

- 测试用例在 `web/e2e/` — `auth.spec.ts`、`vault.spec.ts`、`contact.spec.ts`、`calendar.spec.ts`、`search.spec.ts`、`settings.spec.ts`、`file-upload.spec.ts`。
- Playwright 自动启动 Go 服务器（端口 8080）和 Vite 开发服务器（端口 5173）。
- Ant Design 表单：使用 `page.getByPlaceholder(...)` 而非 `getByLabel(...)`。

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

- **禁止 `as any`、`@ts-ignore`、`@ts-expect-error`。**
- **禁止空 catch 块。**
- TypeScript 严格模式已开启：`noUnusedLocals`、`noUnusedParameters`、`noFallthroughCasesInSwitch`。
- Go：`go vet` 必须通过。没有正当理由禁止 `//nolint`。
- 格式化：Go 使用 `gofmt`，前端使用 Prettier。
- **提交前必须运行 `cd web && bun run lint`**（ESLint），CI 会检查。
- **React Hooks ESLint 规则**（`eslint-plugin-react-hooks`）：
  - `react-hooks/set-state-in-effect`：禁止在 `useEffect` 内同步调用 `setState`。如需同步外部 prop 到内部状态，使用纯受控模式（直接从 prop 派生）或用 `key` prop 重置组件。
  - `react-hooks/refs`：禁止在渲染期间读写 `ref.current`。Ref 只能在事件处理器或 effect 中访问。
  - 组件优先使用**受控模式**（状态由父组件通过 `value`/`onChange` 管理），避免内部 `useState` + `useEffect` 同步 prop 的反模式。

## 项目规模（供参考）

| 维度 | 数量 |
|------|------|
| Go Model 文件 | 49 |
| Go Handler 文件 | 39 |
| Go Service 文件 | 44 |
| Go DTO 文件 | 30 |
| API 路由 | ~143 |
| React 页面组件 | 43 |
| 前端 API 客户端 | 23 |
| i18n 翻译键 | ~478（en + zh 各一份） |

### 测试数量明细

| 层级 | 文件数 | 测试函数数 |
|------|--------|-----------|
| Go Service 测试 | 41 | ~254 |
| Go Handler 集成测试 | 1 | 73 |
| Go Cron 测试 | 1 | 7 |
| Go DAV 测试 | 2 | 20 |
| Go Search 测试 | 1 | 4 |
| Go Avatar 测试 | 1 | 7 |
| Go Calendar 测试 | 1 | 13 |
| **Go 后端总计** | **48** | **~378** |
| React Vitest | 21 | 88 |
| Playwright E2E | 7 | — |
| **全部总计** | **76** | **466+** |

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

## 环境变量完整列表

参见 `server/.env.example`，包含所有可配置项及默认值。分组：

- **Core**：`SERVER_PORT`、`DB_DSN`、`JWT_SECRET`、`APP_ENV`、`APP_URL`
- **SMTP**：`SMTP_HOST`、`SMTP_PORT`、`SMTP_USERNAME`、`SMTP_PASSWORD`、`SMTP_FROM`
- **Storage**：`STORAGE_UPLOAD_DIR`（默认 `uploads`）、`STORAGE_MAX_SIZE`（默认 10MB）
- **Search**：`BLEVE_INDEX_PATH`（默认 `data/bonds.bleve`）
- **Telegram**：`TELEGRAM_BOT_TOKEN`
- **OAuth**：`OAUTH_GITHUB_KEY/SECRET`、`OAUTH_GOOGLE_KEY/SECRET`
- **Geocoding**：`GEOCODING_PROVIDER`（nominatim/locationiq）、`GEOCODING_API_KEY`
- **WebAuthn**：`WEBAUTHN_RP_ID`、`WEBAUTHN_RP_DISPLAY_NAME`、`WEBAUTHN_RP_ORIGINS`
- **其他**：`ANNOUNCEMENT`（全局公告横幅文字）
