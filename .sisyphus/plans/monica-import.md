# Monica 4.x JSON Data Import

## TL;DR

> **Quick Summary**: 为 Bonds 添加 Monica 4.x JSON 全量导出文件的导入功能，将 Monica 的联系人、笔记、通话、任务、提醒、地址、关系、标签、宠物、礼物、债务、生活事件、照片、文档等数据映射到 Bonds 模型中。
>
> **Deliverables**:
> - Go 后端：JSON 解析器 + MonicaImportService + Handler + DTO + Swagger
> - React 前端：Vault Settings 中的导入页面（上传 + 结果报告）
> - 测试：Service 单元测试 + Handler 集成测试 + Playwright E2E
> - i18n：中英文翻译
>
> **Estimated Effort**: Large
> **Parallel Execution**: YES - 4 waves
> **Critical Path**: Task 1 → Task 2 → Task 3 → Task 5 → Task 7 → Task 8 → Task 9 → F1-F4

---

## Context

### Original Request
GitHub Issue #72: Monica data import — 用户希望从 Monica CRM 迁移数据到 Bonds。

### Interview Summary
**Key Discussions**:
- 仅支持 Monica 4.x JSON 导出格式（`version: 1.0-preview.1`），v5 已移除 JSON 导出且用户极少
- Bonds 已有 VCard 导入覆盖 v5 和通用场景
- Monica 4.x 无 Vault 概念，1 个 Account 扁平映射到 1 个 Bonds Vault
- 入口放在 Vault Settings，仅 Manager（permission=100）可操作
- 所有导入数据的 author 归属当前登录用户
- 多用户 Account 场景忽略其他用户，仅导入联系人数据

**Research Findings**:
- Monica 4.x 有 75 个模型，JSON 导出覆盖 ~20 种实体类型
- Bonds 有 55 个模型，架构与 Monica v5 高度一致
- JSON 格式使用 UUID 交叉引用，需建立映射表
- Photos/Documents 嵌入 base64 dataUrl
- Activity 和 Conversation 在 Bonds 中无对应模型，降级为 Note

### Metis Review
**Identified Gaps** (addressed):
- 大文件内存问题 → v1 设置上传大小限制，同步处理
- UUID 交叉引用解析 → 4 阶段导入，先建立映射表
- Seed 类型不匹配 → 按名称/协议模糊匹配，未匹配的跳过并报告
- SQLite 大事务 → 按联系人分批提交（每个联系人一个子事务）
- 前端范围蔓延 → v1 仅上传+报告，无预览/选择性导入
- 重复导入 → 存储 Monica UUID 到 Contact.DistantUUID 用于检测

---

## Work Objectives

### Core Objective
实现 Monica 4.x JSON 全量导出文件的一键导入，让 Monica 用户可以无缝迁移到 Bonds。

### Concrete Deliverables
- `server/internal/services/monica_import.go` — JSON struct 定义 + 导入服务
- `server/internal/services/monica_import_test.go` — 服务层测试
- `server/internal/handlers/monica_import.go` — HTTP handler + swagger 注解
- `server/internal/dto/monica_import.go` — 请求/响应 DTO
- `server/internal/testdata/monica_export.json` — 测试用 JSON fixture
- `web/src/pages/vault/VaultSettings.tsx` 中新增 Monica Import tab — 前端导入 UI
- `web/e2e/monica-import.spec.ts` — E2E 测试
- `web/src/locales/en.json` + `zh.json` — i18n 键

### Definition of Done
- [ ] `go test ./internal/services -run TestMonicaImport -v -count=1` 全部通过
- [ ] `go test ./internal/handlers -v -count=1 -run TestMonicaImport` 全部通过
- [ ] `go build ./...` + `go vet ./...` 无错误
- [ ] `cd web && bun run build` 成功
- [ ] `cd web && bun run lint` 无错误
- [ ] `cd web && bun run test` 全部通过
- [ ] `cd web && bunx playwright test e2e/monica-import.spec.ts` 通过

### Must Have
- Contact 导入（name, nickname, job, company, starred, is_dead 等基础字段）
- Tag → Label 映射（去重，自动生成 slug）
- Gender 匹配（按名称匹配 seed 数据）
- SpecialDate → ContactImportantDate（birthdate, deceased_date）
- Note 导入
- Call 导入
- ContactTask 导入
- ContactReminder 导入（frequency_type 映射）
- Address + ContactAddress 导入
- ContactField → ContactInformation 导入（按 protocol/name 匹配类型）
- Pet 导入（按 category name 匹配 PetCategory）
- Gift 导入（status 映射）
- Debt → Loan 导入（in_debt 布尔→type 映射）
- LifeEvent + TimelineEvent 导入
- Relationship 导入（按 type name 匹配，UUID 交叉引用）
- Photo/Document 导入（base64 decode + 文件存储 + File 模型创建）
- Activity → Note 降级导入
- Conversation → Note 降级导入
- Monica UUID 存储到 Contact.DistantUUID
- Feed 记录（每个导入的 contact 一条 contact_created）
- Search 索引（导入的 contact 加入全文搜索）
- 导入结果报告（成功/跳过/失败 计数 + 错误详情）
- Manager 权限校验
- 前端上传 UI + 结果展示
- 中英文 i18n

### Must NOT Have (Guardrails)
- ❌ 不导入 `account.properties.journal_entries`（Journal 结构差异大，out of scope）
- ❌ 不导入 `account.properties.modules`（账户配置，非联系人数据）
- ❌ 不导入 `account.properties.reminder_rules`（配置项）
- ❌ 不导入 `account.properties.audit_logs`（Bonds 自动生成审计日志）
- ❌ 不新增任何 model 字段或数据库迁移（所有目标模型已存在）
- ❌ 不新增 Activity 或 Conversation 模型（降级为 Note）
- ❌ 不构建异步任务队列（v1 同步处理）
- ❌ 不构建预览/映射/选择性导入 UI（v1 火力全开直接导入）
- ❌ 不修改 `web/src/api/generated/` 下任何文件（自动生成）
- ❌ 不使用 `as any`、`@ts-ignore`
- ❌ 不添加不必要的 JSDoc 注释或过度抽象
- ❌ 不硬编码日期格式字符串（使用 `useDateFormat()` hook）

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES（Go: `testing` stdlib + `testutil.SetupTestDB`; React: vitest; E2E: Playwright）
- **Automated tests**: YES (TDD for Go service, Tests-after for handler/frontend)
- **Framework**: Go stdlib `testing` + vitest + Playwright
- **TDD**: Service 层每个 entity mapping 先写测试再实现

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Backend**: Use Bash (curl/go test) — run tests, assert status + response fields
- **Frontend/UI**: Use Playwright — navigate, interact, assert DOM, screenshot
- **Files**: Use Bash — verify file creation, check content

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — foundation):
├── Task 1: Monica JSON struct 定义 + 测试 fixture [quick]
├── Task 2: DTO 定义 [quick]
└── (2 tasks, foundation wave — small but blocks everything)

Wave 2 (After Wave 1 — core import logic, MAX PARALLEL):
├── Task 3: Contact 核心导入 + Tag/Gender/ImportantDate + tests [deep]
├── Task 4: Contact 子资源导入 (Note/Call/Task/Reminder/Address/Pet/ContactInfo/Gift/Loan/LifeEvent) + tests [deep]
└── (2 tasks — logically sequential within service but can be developed as separate test suites)

Wave 3 (After Wave 2 — cross-references + files + handler):
├── Task 5: Relationship 导入 + Activity/Conversation 降级 + tests [unspecified-high]
├── Task 6: Photo/Document 文件导入 + tests [unspecified-high]
├── Task 7: Handler + Route + Swagger + handler tests [unspecified-high]
└── (3 parallel tasks)

Wave 4 (After Wave 3 — frontend + integration):
├── Task 8: 前端导入页面 + i18n [visual-engineering]
├── Task 9: Feed 记录 + Search 索引 + 集成测试 [quick]
├── Task 10: E2E 测试 [unspecified-high]
└── (3 parallel tasks)

Wave FINAL (After ALL tasks — 4 parallel reviews, then user okay):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
→ Present results → Get explicit user okay

Critical Path: Task 1 → Task 3 → Task 4 → Task 5 → Task 7 → Task 8 → Task 10 → F1-F4 → user okay
Parallel Speedup: ~50% faster than sequential
Max Concurrent: 3 (Waves 3 & 4)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | — | 2,3,4 | 1 |
| 2 | — | 7 | 1 |
| 3 | 1 | 4,5,6,7 | 2 |
| 4 | 1,3 | 5,6,7 | 2 |
| 5 | 3,4 | 7,10 | 3 |
| 6 | 3,4 | 7,10 | 3 |
| 7 | 2,3,4,5,6 | 8,10 | 3 |
| 8 | 7 | 10 | 4 |
| 9 | 3,4,5 | 10 | 4 |
| 10 | 7,8,9 | F1-F4 | 4 |
| F1-F4 | 10 | — | FINAL |

### Agent Dispatch Summary

- **Wave 1**: **2** — T1 → `quick`, T2 → `quick`
- **Wave 2**: **2** — T3 → `deep`, T4 → `deep`
- **Wave 3**: **3** — T5 → `unspecified-high`, T6 → `unspecified-high`, T7 → `unspecified-high`
- **Wave 4**: **3** — T8 → `visual-engineering`, T9 → `quick`, T10 → `unspecified-high`
- **FINAL**: **4** — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. Monica JSON Struct 定义 + 测试 Fixture

  **What to do**:
  - 定义完整的 Go struct 体系对应 Monica 4.x JSON 导出格式（`version: 1.0-preview.1`）
  - 顶层：`MonicaExport`（version, app_version, export_date, url, exported_by, account）
  - Account 层：`MonicaAccount`（uuid, data []MonicaCollection, properties, instance）
  - Collection 层：`MonicaCollection`（count, type, data []json.RawMessage）— type 字段区分 entity 类型
  - Entity struct：MonicaContact, MonicaNote, MonicaCall, MonicaTask, MonicaReminder, MonicaAddress, MonicaGift, MonicaDebt, MonicaPet, MonicaLifeEvent, MonicaConversation, MonicaRelationship, MonicaActivity, MonicaPhoto, MonicaDocument, MonicaContactField
  - Instance struct：MonicaGender, MonicaContactFieldType, MonicaActivityType, MonicaLifeEventType, MonicaLifeEventCategory
  - 每个 entity 都有 uuid, created_at, updated_at, properties 结构
  - 特殊嵌套：MonicaSpecialDate（用于 birthdate/deceased_date/first_met_date）、MonicaAvatar、MonicaMessage（Conversation 内）
  - 创建 `server/internal/testdata/monica_export.json` 真实 fixture 文件，包含：
    - 3 个联系人（含完整字段 + 部分字段 + minimal 字段各一个）
    - 每种子资源至少 1 条（note, call, task, reminder, address, pet, gift, debt, contact_field, life_event, conversation）
    - 2 条 relationship（联系人之间）
    - 1 个 activity（account 级别）
    - 1 个 photo + 1 个 document（小 base64 数据，如 1x1 PNG）
    - instance 数据：genders, contact_field_types, life_event_types, life_event_categories, activity_types
    - 所有 UUID 交叉引用正确
  - 写解析测试：
    - `TestParseMonicaExport_ValidFixture` — 解析 fixture 文件，验证 struct 各字段填充正确
    - `TestParseMonicaExport_InvalidJSON` — 无效 JSON 返回错误
    - `TestParseMonicaExport_WrongVersion` — 版本不匹配返回错误
    - `TestParseMonicaExport_EmptyContacts` — 空联系人数组可正常解析

  **Must NOT do**:
  - 不实现任何导入逻辑，此任务仅定义 struct + parser + fixture
  - 不使用第三方 JSON 库，标准库 `encoding/json` 足够

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 纯 struct 定义 + JSON fixture 创建，无复杂逻辑
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `playwright`: 无浏览器操作

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Task 2)
  - **Blocks**: Tasks 3, 4, 5, 6
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `/tmp/monica-4x/app/ExportResources/Account/Account.php` — 顶层 Account 导出结构
  - `/tmp/monica-4x/app/ExportResources/Contact/Contact.php` — Contact 导出结构（properties + data 嵌套）
  - `/tmp/monica-4x/app/Services/Account/Settings/JsonExportAccount.php` — 导出入口逻辑
  - `/tmp/monica-4x/tests/Unit/Services/Account/Settings/ExportAccountTest.php` — 测试 fixture 参考

  **API/Type References**:
  - 无（纯 Go struct 定义）

  **Test References**:
  - `server/internal/services/vcard_test.go` — 测试文件组织模式参考

  **WHY Each Reference Matters**:
  - Monica ExportResources 定义了 JSON 的精确结构，每个字段名和嵌套关系都从这里来
  - ExportAccountTest.php 包含真实数据样例，用于验证 fixture 的正确性
  - vcard_test.go 展示了 Bonds 项目中 service 测试的组织方式

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: 解析完整 fixture 文件成功
    Tool: Bash
    Preconditions: fixture 文件已创建在 server/internal/testdata/monica_export.json
    Steps:
      1. cd server && go test ./internal/services -run TestParseMonicaExport_ValidFixture -v -count=1
      2. 检查输出包含 "PASS"
    Expected Result: 测试通过，fixture 中 3 个 contact、2 个 relationship 等正确解析
    Failure Indicators: "FAIL" 或 "cannot unmarshal" 错误
    Evidence: .sisyphus/evidence/task-1-parse-fixture.txt

  Scenario: 拒绝无效 JSON
    Tool: Bash
    Preconditions: 同上
    Steps:
      1. cd server && go test ./internal/services -run TestParseMonicaExport_InvalidJSON -v -count=1
    Expected Result: 测试通过，返回解析错误
    Evidence: .sisyphus/evidence/task-1-invalid-json.txt

  Scenario: 拒绝错误版本号
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run TestParseMonicaExport_WrongVersion -v -count=1
    Expected Result: 测试通过
    Evidence: .sisyphus/evidence/task-1-wrong-version.txt
  ```

  **Commit**: YES
  - Message: `feat(server): add Monica 4.x JSON struct definitions and test fixture`
  - Files: `server/internal/services/monica_import.go`, `server/internal/testdata/monica_export.json`, `server/internal/services/monica_import_test.go`
  - Pre-commit: `cd server && go test ./internal/services -run TestParseMonicaExport -v -count=1`

- [x] 2. Monica Import DTO 定义

  **What to do**:
  - 创建 `server/internal/dto/monica_import.go`
  - 定义 `MonicaImportResponse` struct：
    ```go
    type MonicaImportResponse struct {
        ImportedContacts int      `json:"imported_contacts"`
        ImportedNotes    int      `json:"imported_notes"`
        ImportedCalls    int      `json:"imported_calls"`
        ImportedTasks    int      `json:"imported_tasks"`
        ImportedReminders int     `json:"imported_reminders"`
        ImportedAddresses int     `json:"imported_addresses"`
        ImportedRelationships int `json:"imported_relationships"`
        ImportedPhotos   int      `json:"imported_photos"`
        ImportedDocuments int     `json:"imported_documents"`
        SkippedCount     int      `json:"skipped_count"`
        Errors           []string `json:"errors"`
    }
    ```
  - 参考 `server/internal/dto/vcard.go` 的 `VCardImportResponse` 结构

  **Must NOT do**:
  - 不定义 request DTO（multipart form upload 不需要 request body struct）
  - 不添加过多字段，保持简洁

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 单文件 struct 定义
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Task 1)
  - **Blocks**: Task 7
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `server/internal/dto/vcard.go` — VCard 导入响应 DTO 模式

  **WHY Each Reference Matters**:
  - VCard DTO 是最近的同类实现，字段风格和 JSON tag 命名规范直接复用

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: DTO struct 编译通过
    Tool: Bash
    Steps:
      1. cd server && go build ./...
    Expected Result: 编译成功
    Evidence: .sisyphus/evidence/task-2-build.txt
  ```

  **Commit**: YES
  - Message: `feat(server): add Monica import DTO definitions`
  - Files: `server/internal/dto/monica_import.go`
  - Pre-commit: `cd server && go build ./...`

- [x] 3. Contact u6838u5fc3u5bfcu5165 + Tag/Gender/ImportantDate

  **What to do**:
  - u5728 `server/internal/services/monica_import.go` u4e2du5b9eu73b0 `MonicaImportService` structuff1a
    ```go
    type MonicaImportService struct {
        DB           *gorm.DB
        FeedRecorder *FeedRecorder    // u5f85 Task 9 u63a5u5165
        SearchEngine search.Engine    // u5f85 Task 9 u63a5u5165
    }
    func NewMonicaImportService(db *gorm.DB) *MonicaImportService
    func (s *MonicaImportService) Import(vaultID, userID string, data []byte) (*dto.MonicaImportResponse, error)
    ```
  - Import u65b9u6cd5u5185u90e8u6d41u7a0buff1a
    1. u89e3u6790 JSON u2192 `MonicaExport` structuff0cu6821u9a8c version u5b57u6bb5
    2. u89e3u6790 `account.instance` u2192 u5efau7acbu7c7bu578bu6620u5c04u8868uff08gender UUIDu2192name, contact_field_type UUIDu2192protocol+name u7b49uff09
    3. u904du5386 `account.data[type=contacts]` u521bu5efa Contactuff1a
       - u6620u5c04 first_name, last_name, middle_name, nickname, description, is_dead, job, company
       - `is_starred` u2192 u6682u65f6u5ffdu7565uff08Bonds u4e2du901au8fc7 ContactVaultUser.IsFavorite u8868u793auff09
       - `is_partial` u8054u7cfbu4eba u2192 u8bbeu7f6e `Listed=false`
       - u5b58u50a8 Monica UUID u5230 `Contact.DistantUUID`
       - Gender: u4ece instance.genders u67e5 UUIDu2192nameuff0cu518du5728 Bonds seed u4e2du6309 name u5339u914d Gender record
       - Tags: u4ece contact.properties.tags u6570u7ec4uff0cu4e3au6bcfu4e2au552fu4e00 tag name u521bu5efa Labeluff08u53bbu91cduff09uff0cu5efau7acb ContactLabel pivot
       - Birthdate: u4ece contact.properties.birthdate (SpecialDate) u2192 ContactImportantDateuff08type=birthdate seed lookupuff09
         - `is_year_unknown=true` u2192 Year=0
         - `is_age_based=true` u2192 u4ece date u4f30u7b97 Yearuff0cu5e76u5728 description u4e2du6807u6ce8
       - Deceased date: u540cu4e0auff0ctype=deceased_date
       - u521bu5efa `ContactVaultUser` pivotuff08is_favorite u5bf9u5e94 is_starreduff09
    4. u5efau7acb UUID u6620u5c04u8868uff1a`monicaContactUUID u2192 bondsContactID` (map[string]string)
  - u5148u5199u6d4bu8bd5uff08TDDuff09uff1a
    - `TestMonicaImportContacts_Basic` u2014 3 u4e2au8054u7cfbu4ebau5bfcu5165uff0cu9a8cu8bc1 name/distantUUID/listed
    - `TestMonicaImportContacts_Gender` u2014 gender u5339u914d seed u6570u636e
    - `TestMonicaImportContacts_Tags` u2014 tag u53bbu91cd + Label u521bu5efa + ContactLabel pivot
    - `TestMonicaImportContacts_Birthdate` u2014 birthdate + year_unknown + age_based u5904u7406
    - `TestMonicaImportContacts_DeceasedDate` u2014 deceased date u5bfcu5165
    - `TestMonicaImportContacts_IsPartial` u2014 partial contact u2192 Listed=false
    - `TestMonicaImportContacts_IsFavorite` u2014 is_starred u2192 ContactVaultUser.IsFavorite
    - `TestMonicaImportContacts_Duplicate` u2014 u76f8u540c DistantUUID u8df3u8fc7u4e0du91cdu590du5bfcu5165

  **Must NOT do**:
  - u4e0du5b9eu73b0u5b50u8d44u6e90u5bfcu5165uff08note/call/task u7b49uff09uff0cu7559u7ed9 Task 4
  - u4e0du5b9eu73b0 Relationship u5bfcu5165uff0cu7559u7ed9 Task 5
  - u4e0du5b9eu73b0 File u5bfcu5165uff0cu7559u7ed9 Task 6
  - u4e0du63a5u5165 FeedRecorder/SearchEngineuff0cu7559u7ed9 Task 9

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: u6838u5fc3u4e1au52a1u903bu8f91uff0cu6d89u53cau591au6a21u578bu6620u5c04u3001seed u6570u636eu67e5u627eu3001UUID u4ea4u53c9u5f15u7528uff0cu9700u8981u6df1u5ea6u7406u89e3
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 (sequential with Task 4)
  - **Blocks**: Tasks 4, 5, 6, 7, 9
  - **Blocked By**: Task 1

  **References**:

  **Pattern References**:
  - `server/internal/services/vcard.go:118-191` u2014 VCard u5bfcu5165u7684u4e8bu52a1 + u5faau73afu6a21u5f0fuff08u76f4u63a5u590du7528uff09
  - `server/internal/services/vcard.go:194-279` u2014 seed u7c7bu578bu67e5u627eu6a21u5f0fuff08phone/email/birthdate by internal identifieruff09
  - `server/internal/models/contact.go` u2014 Contact u6a21u578bu5b57u6bb5u5b9au4e49
  - `server/internal/models/label.go` u2014 Label + ContactLabel pivot u6a21u578b
  - `server/internal/models/contact_important_date.go` u2014 ImportantDate u6a21u578b + DateType seed
  - `server/internal/models/gender.go` u2014 Gender u6a21u578b

  **API/Type References**:
  - `server/internal/models/seed_account.go` u2014 SeedAccountDefaults u4e2d Gender/Pronoun seed u6570u636eu7684u5b9eu9645u540du79f0
  - `server/internal/models/seed_vault.go` u2014 SeedVaultDefaults u4e2d ContactImportantDateType seed

  **Test References**:
  - `server/internal/services/vcard_test.go` u2014 VCard u5bfcu5165u6d4bu8bd5u6a21u5f0fuff08setupTestDB + register + createVaultuff09
  - `server/internal/services/contact_test.go` u2014 Contact CRUD u6d4bu8bd5u6a21u5f0f

  **WHY Each Reference Matters**:
  - vcard.go u662fu6700u76f4u63a5u7684u53c2u8003u2014u2014u540cu6837u662fu5916u90e8u6570u636eu5bfcu5165uff0cu4e8bu52a1u6a21u5f0fu3001seed u67e5u627eu3001pivot u521bu5efau90fdu4e00u6837
  - seed u6587u4ef6u786eu5b9a Bonds u4e2d gender/date type u7684u5b9eu9645u540du79f0uff0cu7528u4e8eu6620u5c04 Monica u6570u636e
  - contact_test.go u5c55u793a setup u6d41u7a0buff08u6ce8u518cu7528u6237u89e6u53d1 seeduff0cu8ba1u6570u65f6u9700u8003u8651u5df2u6709 seed u6570u636euff09

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: u57fau672cu8054u7cfbu4ebau5bfcu5165
    Tool: Bash
    Preconditions: fixture u6587u4ef6u5b58u5728
    Steps:
      1. cd server && go test ./internal/services -run TestMonicaImportContacts_Basic -v -count=1
    Expected Result: 3 u4e2au8054u7cfbu4ebau6210u529fu521bu5efauff0cDistantUUID u5b58u50a8u6b63u786e
    Evidence: .sisyphus/evidence/task-3-contacts-basic.txt

  Scenario: u91cdu590du5bfcu5165u68c0u6d4b
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run TestMonicaImportContacts_Duplicate -v -count=1
    Expected Result: u7b2cu4e8cu6b21u5bfcu5165u76f8u540c UUID u7684u8054u7cfbu4ebau88abu8df3u8fc7
    Evidence: .sisyphus/evidence/task-3-duplicate.txt

  Scenario: u5168u90e8 contact u6d4bu8bd5u901au8fc7
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run TestMonicaImportContacts -v -count=1
    Expected Result: u6240u6709 TestMonicaImportContacts_* u6d4bu8bd5u901au8fc7
    Evidence: .sisyphus/evidence/task-3-all-contacts.txt
  ```

  **Commit**: YES
  - Message: `feat(server): implement Monica contact import with tag/gender/date mapping`
  - Files: `server/internal/services/monica_import.go`, `server/internal/services/monica_import_test.go`
  - Pre-commit: `cd server && go test ./internal/services -run TestMonicaImport -v -count=1`

- [x] 4. Contact u5b50u8d44u6e90u5bfcu5165

  **What to do**:
  - u5728 `MonicaImportService.Import` u65b9u6cd5u4e2duff0cu904du5386u6bcfu4e2a contact u7684 `data` u6570u7ec4uff0cu6309 type u5206u53d1u5bfcu5165uff1a

  - **Note** (`type=notes`)uff1a
    - `properties.body` u2192 `Note.Body`uff0c`properties.is_favorite` u2192 u5ffdu7565uff08Bonds u65e0 favorite noteuff09
    - `Note.Title` u7559u7a7auff08Monica note u65e0 titleuff09uff0c`AuthorID` u8bbeu4e3au5f53u524du7528u6237

  - **Call** (`type=calls`)uff1a
    - `properties.called_at` u2192 `Call.CalledAt`
    - `properties.content` u2192 `Call.Description`
    - `properties.contact_called` u2192 `Call.WhoInitiated`uff08true="contact", false="user"uff09
    - `properties.emotions` u2192 u5339u914d Bonds Emotion seed by nameuff0cu8bbeu7f6e `Call.EmotionID`uff08u53d6u7b2cu4e00u4e2auff09
    - `Call.Type` u9ed8u8ba4 "phone"uff0c`Call.Answered` u9ed8u8ba4 true

  - **ContactTask** (`type=tasks`)uff1a
    - `properties.title` u2192 `ContactTask.Label`
    - `properties.description` u2192 `ContactTask.Description`
    - `properties.completed` u2192 `ContactTask.Completed`
    - `properties.completed_at` u2192 `ContactTask.CompletedAt`

  - **ContactReminder** (`type=reminders`)uff1a
    - `properties.title` u2192 `ContactReminder.Label`
    - `properties.frequency_type` u6620u5c04uff1a"one_time"u2192"one_time", "week"u2192"recurring_week", "month"u2192"recurring_month", "year"u2192"recurring_year"
    - `properties.frequency_number` u2192 `ContactReminder.FrequencyNumber`
    - `properties.initial_date` u89e3u6790u4e3a Day/Month/Year u5b57u6bb5

  - **Address** (`type=addresses`)uff1a
    - `properties.street` u2192 `Address.Line1`uff0c`properties.city/province/postal_code/country` u76f4u63a5u6620u5c04
    - `properties.latitude/longitude` u76f4u63a5u6620u5c04
    - `properties.name` u2192 u67e5u627e AddressType by nameuff08Home/Work/Otheruff09uff0cu672au5339u914du5219u7528u7b2cu4e00u4e2a
    - u521bu5efa ContactAddress pivot

  - **ContactInformation** (`type=contact_fields`)uff1a
    - `properties.type` u662f Monica ContactFieldType UUID
    - u4ece instance.contact_field_types u67e5 UUIDu2192u83b7u53d6 protocol+name
    - u6309 protocol u5339u914d Bonds ContactInformationTypeuff08"mailto:"u2192email, "tel:"u2192phoneuff09uff0cu5426u5219u6309 name u6a21u7ccau5339u914d
    - `properties.data` u2192 `ContactInformation.Data`
    - u672au5339u914du7684 type u2192 u8df3u8fc7u5e76u8bb0u5f55u5230 errors

  - **Pet** (`type=pets`)uff1a
    - `properties.name` u2192 `Pet.Name`
    - `properties.category` u2192 u6309u540du79f0u5339u914d PetCategory seeduff08case-insensitiveuff09
    - u672au5339u914du7684 category u2192 u7528u7b2cu4e00u4e2au53efu7528 PetCategory

  - **Gift** (`type=gifts`)uff1a
    - `properties.name` u2192 `Gift.Name`
    - `properties.amount` u2192 `Gift.EstimatedPrice`uff08u8f6cu4e3au5206/u6574u6570uff09
    - `properties.comment` u2192 `Gift.Description`
    - `properties.status` u6620u5c04uff1a"idea"u2192u65e0u65e5u671fu7684 gift, "offered"u2192type="given", "received"u2192type="received"
    - `properties.date` u2192 `Gift.GivenAt` u6216 `Gift.ReceivedAt`

  - **Loan** (`type=debts`)uff1a
    - `properties.in_debt=true`uff08u6211u6b20u4ed6uff09u2192 `Loan.Type="borrowed_from"`uff0cLoanee=u5f53u524du7528u6237u5f71u5b50u8054u7cfbu4eba, Loaner=contact
    - `properties.in_debt=false`uff08u4ed6u6b20u6211uff09u2192 `Loan.Type="lent_to"`uff0cLoaner=u5f53u524du7528u6237u5f71u5b50u8054u7cfbu4eba, Loanee=contact
    - `properties.amount` u2192 `Loan.AmountLent`uff08u8f6cu4e3au5206uff09
    - `properties.currency` u2192 u67e5u627e Currency by Code
    - `properties.status="complete"` u2192 `Loan.Settled=true, Loan.SettledAt=now`

  - **LifeEvent** (`type=life_events`)uff1a
    - u4e3au6bcfu6b21u5bfcu5165u521bu5efau4e00u4e2a `TimelineEvent`uff08label="Monica Import"uff09
    - `properties.name` u2192 `LifeEvent.Summary`
    - `properties.happened_at` u2192 `LifeEvent.HappenedAt`
    - `properties.note` u2192 `LifeEvent.Description`
    - `properties.type` UUID u2192 u4ece instance.life_event_types u67e5u627euff0cu6309 name u5339u914d Bonds LifeEventType
    - u521bu5efa LifeEventParticipant pivot u5173u8054 contact

  - u6bcfu4e2a entity u7c7bu578bu5199u5bf9u5e94u6d4bu8bd5uff1a
    - `TestMonicaImportNotes`
    - `TestMonicaImportCalls`
    - `TestMonicaImportTasks`
    - `TestMonicaImportReminders`
    - `TestMonicaImportAddresses`
    - `TestMonicaImportContactInfo`
    - `TestMonicaImportPets`
    - `TestMonicaImportGifts`
    - `TestMonicaImportLoans`
    - `TestMonicaImportLifeEvents`

  **Must NOT do**:
  - u4e0du5b9eu73b0 Activity/Conversation u964du7ea7uff08Task 5uff09
  - u4e0du5b9eu73b0 Relationshipuff08Task 5uff09
  - u4e0du5b9eu73b0 Photo/Documentuff08Task 6uff09

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: u591au5b9eu4f53u6620u5c04u903bu8f91uff0cu6d89u53ca seed u67e5u627eu3001u7c7bu578bu8f6cu6362u3001pivot u8868u521bu5efa
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 (after Task 3)
  - **Blocks**: Tasks 5, 6, 7
  - **Blocked By**: Tasks 1, 3

  **References**:

  **Pattern References**:
  - `server/internal/services/vcard.go:194-279` u2014 seed u67e5u627eu6a21u5f0fuff08ContactInformationType by internal typeuff09
  - `server/internal/services/note.go` u2014 Note u521bu5efau6a21u5f0f
  - `server/internal/services/reminder.go` u2014 Reminder u521bu5efau6a21u5f0f + frequency u5904u7406
  - `server/internal/services/loan.go` u2014 Loan + ContactLoan pivot u521bu5efau6a21u5f0f
  - `server/internal/services/gift.go` u2014 Gift u521bu5efau6a21u5f0f
  - `server/internal/services/life_event.go` u2014 LifeEvent + TimelineEvent u521bu5efau6a21u5f0f
  - `server/internal/models/contact_information.go` u2014 ContactInformation + Type u5173u7cfb
  - `server/internal/models/address.go` u2014 Address + ContactAddress pivot
  - `server/internal/models/loan.go` u2014 Loan + ContactLoan pivot u7ed3u6784

  **API/Type References**:
  - `server/internal/models/seed_account.go` u2014 ContactInformationType seed u6570u636euff08email/phone u7684 protocol u548c type u503cuff09
  - `server/internal/models/seed_vault.go` u2014 LifeEventCategory/Type seed u6570u636e

  **WHY Each Reference Matters**:
  - u6bcfu4e2a service u6587u4ef6u5c55u793au8be5u5b9eu4f53u7684u6807u51c6u521bu5efau6d41u7a0buff0cu5305u62ecu54eau4e9bu5b57u6bb5u5fc5u586bu3001pivot u600eu4e48u5efa
  - seed u6587u4ef6u786eu5b9au53efu7528u7684u7c7bu578bu540du79f0u548cu5339u914du903bu8f91

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: u5168u90e8u5b50u8d44u6e90u5bfcu5165u6d4bu8bd5u901au8fc7
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run "TestMonicaImport(Notes|Calls|Tasks|Reminders|Addresses|ContactInfo|Pets|Gifts|Loans|LifeEvents)" -v -count=1
    Expected Result: u6240u6709 10 u4e2au5b50u8d44u6e90u6d4bu8bd5u901au8fc7
    Evidence: .sisyphus/evidence/task-4-sub-resources.txt

  Scenario: Debt u2192 Loan u7c7bu578bu6620u5c04u6b63u786e
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run TestMonicaImportLoans -v -count=1
    Expected Result: in_debt=true u2192 borrowed_from, in_debt=false u2192 lent_to
    Evidence: .sisyphus/evidence/task-4-loans.txt

  Scenario: u672au5339u914du7684 ContactFieldType u8df3u8fc7u5e76u62a5u544a
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run TestMonicaImportContactInfo -v -count=1
    Expected Result: u5df2u77e5u7c7bu578bu5bfcu5165u6210u529fuff0cu672au77e5u7c7bu578bu5728 errors u4e2du62a5u544a
    Evidence: .sisyphus/evidence/task-4-contact-info.txt
  ```

  **Commit**: YES
  - Message: `feat(server): implement Monica sub-resource import (note/call/task/reminder/address/pet/gift/loan/life-event)`
  - Files: `server/internal/services/monica_import.go`, `server/internal/services/monica_import_test.go`
  - Pre-commit: `cd server && go test ./internal/services -run TestMonicaImport -v -count=1`

- [x] 5. Relationship u5bfcu5165 + Activity/Conversation u964du7ea7

  **What to do**:
  - **Relationship u5bfcu5165**uff08`account.data[type=relationships]`uff09uff1a
    - u904du5386 relationship u6570u7ec4uff0cu6bcfu6761u6709 `properties.contact_is`u3001`properties.of_contact`u3001`properties.type`
    - u4ece UUID u6620u5c04u8868u67e5u627eu4e24u4e2a contact u7684 Bonds IDuff0cu4efbu4e00u672au627eu5230u5219u8df3u8fc7u5e76u8bb0u5f55 error
    - `properties.type` u662fu5173u7cfbu7c7bu578bu540du79f0uff08u5982 "partner", "son", "friend"uff09
    - u5728 Bonds u7684 RelationshipType u4e2du6309 Name u6216 NameReverseRelationship u6a21u7ccau5339u914duff08case-insensitiveuff09
    - u672au5339u914du7684 type u2192 u8df3u8fc7u5e76u8bb0u5f55 error
    - u521bu5efa Relationship recorduff08ContactID, RelatedContactID, RelationshipTypeIDuff09
    - u68c0u67e5u662fu5426u5df2u5b58u5728u76f8u540cu5173u7cfbu907fu514du91cdu590d

  - **Activity u964du7ea7u4e3a Note**uff08`account.data[type=activities]`uff09uff1a
    - u4ece `account.instance.activity_types` u5efbu7acb UUIDu2192name u6620u5c04
    - Activity u6709 `properties.summary`u3001`properties.description`u3001`properties.happened_at`u3001`properties.type`
    - u8f6cu4e3a Noteuff1a`Title = "[Activity: {type_name}] {summary}"`uff0c`Body = description`
    - Activity u662f account u7ea7u522bu4e14 many-to-many contactu2014u2014u4f46 JSON u4e2d contact.data[type=activities] u53eau5305u542b UUID u6570u7ec4
    - u5904u7406u65b9u5f0fuff1au5148u5efbu7acb activity UUID u2192 Note u5185u5bb9u7684u6620u5c04uff0cu7136u540eu5728u5904u7406u6bcfu4e2a contact u65f6u5c06u5f15u7528u7684 activity UUID u521bu5efau4e3au8be5 contact u7684 Note

  - **Conversation u964du7ea7u4e3a Note**uff08`contact.data[type=conversations]`uff09uff1a
    - u5c06 messages u6570u7ec4u683cu5f0fu5316u4e3au804au5929u8bb0u5f55uff1a
      ```
      [{written_at}] {written_by_me ? "Me" : "Contact"}: {content}
      ```
    - `Title = "Conversation ({happened_at})"`uff0c`Body = u683cu5f0fu5316u6d88u606f`

  - u6d4bu8bd5uff1a
    - `TestMonicaImportRelationships_Basic` u2014 u6b63u5e38u5173u7cfbu5bfcu5165
    - `TestMonicaImportRelationships_UnresolvedContact` u2014 u672au627eu5230 contact u65f6u8df3u8fc7
    - `TestMonicaImportRelationships_UnknownType` u2014 u672au77e5u7c7bu578bu65f6u8df3u8fc7
    - `TestMonicaImportActivities` u2014 activity u964du7ea7u4e3a note
    - `TestMonicaImportConversations` u2014 conversation u964du7ea7u4e3a note

  **Must NOT do**:
  - u4e0du521bu5efau65b0u7684 Activity u6216 Conversation u6a21u578b
  - u4e0du81eau52a8u521bu5efau53cdu5411u5173u7cfbuff08Monica u5bfcu51fau5df2u5305u542bu53ccu5411uff09

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: u6d89u53cau591au79cdu5b9eu4f53u6620u5c04u548cu964du7ea7u903bu8f91uff0cu4e2du7b49u590du6742u5ea6
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 6, 7)
  - **Blocks**: Tasks 7, 9, 10
  - **Blocked By**: Tasks 3, 4

  **References**:

  **Pattern References**:
  - `server/internal/services/relationship.go` u2014 Relationship u521bu5efau6a21u5f0f
  - `server/internal/models/relationship.go` u2014 Relationship + RelationshipType + RelationshipGroupType u6a21u578b
  - `server/internal/models/seed_account.go` u2014 RelationshipGroupType + RelationshipType seed u6570u636euff08u5b9eu9645u540du79f0uff09
  - `server/internal/services/note.go` u2014 Note u521bu5efau6a21u5f0f

  **WHY Each Reference Matters**:
  - relationship.go u5c55u793au53ccu5411u5173u7cfbu521bu5efau903bu8f91uff0cu4f46u6b64u5904u65e0u9700u81eau52a8u53cdu5411uff08Monica u5df2u5bfcu51fau53ccu5411uff09
  - seed u6570u636eu786eu5b9a Bonds u4e2du5b9eu9645u7684 RelationshipType u540du79f0u7528u4e8eu5339u914d

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Relationship u5bfcu5165 + u964du7ea7u6d4bu8bd5
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run "TestMonicaImport(Relationships|Activities|Conversations)" -v -count=1
    Expected Result: u6240u6709 5 u4e2au6d4bu8bd5u901au8fc7
    Evidence: .sisyphus/evidence/task-5-relationships.txt

  Scenario: u672au89e3u6790u7684 contact UUID u8df3u8fc7u4e0du5d29u6e83
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run TestMonicaImportRelationships_UnresolvedContact -v -count=1
    Expected Result: u8df3u8fc7u5e76u5728 errors u4e2du62a5u544a
    Evidence: .sisyphus/evidence/task-5-unresolved.txt
  ```

  **Commit**: YES
  - Message: `feat(server): implement Monica relationship import and activity/conversation degradation`
  - Files: `server/internal/services/monica_import.go`, `server/internal/services/monica_import_test.go`
  - Pre-commit: `cd server && go test ./internal/services -run TestMonicaImport -v -count=1`

- [x] 6. Photo/Document u6587u4ef6u5bfcu5165

  **What to do**:
  - u5904u7406 `account.data[type=photos]` u548c `account.data[type=documents]`uff1a
    - u89e3u6790 `properties.dataUrl` u7684 base64 u524du7f00uff1a`data:{mime};base64,{payload}`
    - u9a8cu8bc1 MIME u7c7bu578bu5728u5141u8bb8u5217u8868u5185uff08image/jpeg, image/png, image/gif, image/webp, application/pdfuff09
    - Base64 u89e3u7801u4e3au5b57u8282u6570u7ec4
    - u751fu6210u6587u4ef6u5b58u50a8u8defu5f84uff1a`{uploadDir}/{yyyy/MM/dd}/{uuid}{ext}`
    - u5199u5165u78c1u76d8
    - u521bu5efa `File` u8bb0u5f55uff08VaultID, Name=original_filename, MimeType, Size=filesize, Type="photo"/"document"uff09
    - Photo u4e0e Contact u7684u5173u8054uff1aContact.data[type=photos] u5305u542b photo UUID u6570u7ec4
      - u5efbu7acb monica photo UUID u2192 Bonds File ID u6620u5c04
      - u5c06u7b2cu4e00u5f20u7167u7247u8bbeu4e3a Contact.FileIDuff08u5934u50cfuff09
      - u5176u4f59u7167u7247u901au8fc7 File.FileableID=contactID, FileableType="contacts" u5173u8054
    - Document u4e0e Contact u7684u5173u8054uff1aContact.data[type=documents] u5305u542b document UUID u6570u7ec4
      - u540cu7406uff0cFileableType="contacts"uff0cType="document"
    - u5982u679c `dataUrl` u4e3au7a7au6216u89e3u6790u5931u8d25uff0cu8df3u8fc7u5e76u8bb0u5f55 error
    - u68c0u67e5 uploadDir u914du7f6eu662fu5426u5b58u5728uff0cu4e0du5b58u5728u5219u8df3u8fc7u6240u6709u6587u4ef6u5bfcu5165u5e76u62a5u544a

  - u6d4bu8bd5uff1a
    - `TestMonicaImportPhotos` u2014 base64 u89e3u7801 + u6587u4ef6u5199u5165 + File u8bb0u5f55u521bu5efa
    - `TestMonicaImportPhotos_Avatar` u2014 u7b2cu4e00u5f20u7167u7247u8bbeu4e3a Contact.FileID
    - `TestMonicaImportDocuments` u2014 document u5bfcu5165
    - `TestMonicaImportPhotos_InvalidBase64` u2014 u635fu574f base64 u8df3u8fc7
    - `TestMonicaImportPhotos_NoUploadDir` u2014 u65e0 uploadDir u65f6u8df3u8fc7u5168u90e8u6587u4ef6

  **Must NOT do**:
  - u4e0du5b9eu73b0u56feu7247u538bu7f29u6216u7f29u7565u56feu751fu6210
  - u4e0du5b9eu73b0 CDN u4e0au4f20uff08v1 u4ec5u672cu5730u5b58u50a8uff09

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: u6d89u53cau6587u4ef6 I/Ou3001base64 u89e3u7801u3001u591au6001u6587u4ef6u5173u8054
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 5, 7)
  - **Blocks**: Tasks 7, 10
  - **Blocked By**: Tasks 3, 4

  **References**:

  **Pattern References**:
  - `server/internal/handlers/contact_photos.go` u2014 u7167u7247u4e0au4f20 handleruff08MIME u6821u9a8cu3001u6587u4ef6u5b58u50a8u8defu5f84u751fu6210uff09
  - `server/internal/services/file.go` u2014 File u670du52a1uff08u521bu5efa File u8bb0u5f55uff09
  - `server/internal/models/file.go` u2014 File u6a21u578buff08u591au6001u5173u8054u5b57u6bb5uff09
  - `server/internal/handlers/vault_files.go` u2014 Vault u6587u4ef6u4e0au4f20u6a21u5f0f

  **WHY Each Reference Matters**:
  - contact_photos.go u5c55u793au4e86 MIME u6821u9a8cu903bu8f91u548cu5b58u50a8u8defu5f84u751fu6210u89c4u5219
  - file.go service u5c55u793a File u521bu5efau7684u6807u51c6u6d41u7a0b
  - file.go model u5c55u793au591au6001u5173u8054u5b57u6bb5uff08FileableID/FileableTypeuff09u600eu4e48u8bbeu7f6e

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Photo u5bfcu5165u6210u529f
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run TestMonicaImportPhotos -v -count=1
    Expected Result: base64 u89e3u7801u3001u6587u4ef6u5199u5165u3001File u8bb0u5f55u521bu5efau5168u90e8u6210u529f
    Evidence: .sisyphus/evidence/task-6-photos.txt

  Scenario: u65e0u6548 base64 u8df3u8fc7u4e0du5d29u6e83
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run TestMonicaImportPhotos_InvalidBase64 -v -count=1
    Expected Result: u8df3u8fc7u5e76u5728 errors u62a5u544a
    Evidence: .sisyphus/evidence/task-6-invalid-base64.txt
  ```

  **Commit**: YES
  - Message: `feat(server): implement Monica photo/document import with base64 decoding`
  - Files: `server/internal/services/monica_import.go`, `server/internal/services/monica_import_test.go`
  - Pre-commit: `cd server && go test ./internal/services -run TestMonicaImport -v -count=1`

- [x] 7. Handler + Route + Swagger + Handler u6d4bu8bd5

  **What to do**:
  - u521bu5efa `server/internal/handlers/monica_import.go`uff1a
    - `MonicaImportHandler` structuff08u6301u6709 `*services.MonicaImportService`uff09
    - `NewMonicaImportHandler(svc *services.MonicaImportService) *MonicaImportHandler`
    - `Import(c echo.Context) error` u65b9u6cd5uff1a
      - u4ece context u83b7u53d6 vaultIDu3001userID
      - u8bfbu53d6 multipart form file
      - u8c03u7528 `svc.Import(vaultID, userID, fileBytes)`
      - u8fd4u56de `response.OK(c, result)`
    - Swagger u6ce8u89e3uff1a
      ```
      // @Summary Import Monica 4.x JSON data
      // @Tags Vault Settings
      // @Accept multipart/form-data
      // @Param vault_id path string true "Vault ID"
      // @Param file formData file true "Monica JSON export file"
      // @Success 200 {object} response.APIResponse{data=dto.MonicaImportResponse}
      // @Router /vaults/{vault_id}/import/monica [post]
      ```
  - u5728 `server/internal/handlers/routes.go` u4e2du6ce8u518cu8defu7531uff1a
    - u627eu5230 vaultSettings u7ec4uff08PermissionManager u4e2du95f4u4ef6uff09
    - u6dfbu52a0 `vaultSettings.POST("/import/monica", monicaImportHandler.Import)`
  - u5728 `routes.go` u4e2du521du59cbu5316 `MonicaImportService` u548c `MonicaImportHandler`
  - Handler u96c6u6210u6d4bu8bd5uff08u5728 `handlers_test.go` u6216u65b0u6587u4ef6uff09uff1a
    - `TestMonicaImport_Success` u2014 u4e0au4f20 fixture JSONuff0cu9a8cu8bc1 200 + u54cdu5e94u4f53
    - `TestMonicaImport_InvalidJSON` u2014 u4e0au4f20u5783u573eu6570u636euff0cu9a8cu8bc1 400
    - `TestMonicaImport_NoFile` u2014 u65e0u6587u4ef6uff0cu9a8cu8bc1 400
    - `TestMonicaImport_PermissionDenied` u2014 u975e Manager u7528u6237uff0cu9a8cu8bc1 403
  - u8fd0u884c `make swagger` u751fu6210 swagger u6587u6863

  **Must NOT do**:
  - u4e0du5b9eu73b0u5f02u6b65u5904u7406u6216 job queue
  - u4e0du6dfbu52a0u6587u4ef6u5927u5c0fu9650u5236u4e2du95f4u4ef6uff08u590du7528 Echo u9ed8u8ba4u9650u5236uff09

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Handler + u8defu7531u6ce8u518c + Swagger + u96c6u6210u6d4bu8bd5uff0cu4e2du7b49u590du6742u5ea6
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 5, 6)
  - **Blocks**: Tasks 8, 10
  - **Blocked By**: Tasks 2, 3, 4, 5, 6

  **References**:

  **Pattern References**:
  - `server/internal/handlers/vcard.go:92-116` u2014 VCard u5bfcu5165 handleruff08multipart file upload + swaggeruff09
  - `server/internal/handlers/routes.go` u2014 u8defu7531u6ce8u518cu4f4du7f6euff08vaultSettings u7ec4uff09
  - `server/internal/handlers/handlers_test.go` u2014 Handler u96c6u6210u6d4bu8bd5u6a21u5f0fuff08setupTestServeruff09

  **API/Type References**:
  - `server/internal/dto/monica_import.go` u2014 MonicaImportResponse DTO
  - `server/pkg/response/response.go` u2014 API u54cdu5e94u5c01u88c5

  **WHY Each Reference Matters**:
  - vcard.go handler u662fu6700u76f4u63a5u7684u53c2u8003uff0cu540cu6837u662f multipart file upload u7684u5bfcu5165u63a5u53e3
  - routes.go u786eu5b9au6ce8u518cu4f4du7f6eu548cu4e2du95f4u4ef6u7ec4
  - handlers_test.go u5c55u793a setupTestServer u548cu8bf7u6c42u6784u5efau6a21u5f0f

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Handler u96c6u6210u6d4bu8bd5u901au8fc7
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/handlers -run TestMonicaImport -v -count=1
    Expected Result: 4 u4e2au6d4bu8bd5u5168u90e8u901au8fc7
    Evidence: .sisyphus/evidence/task-7-handler-tests.txt

  Scenario: Swagger u751fu6210u6210u529f
    Tool: Bash
    Steps:
      1. cd server && swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
    Expected Result: u751fu6210u6210u529fuff0cu65e0u9519u8bef
    Evidence: .sisyphus/evidence/task-7-swagger.txt

  Scenario: u6743u9650u6821u9a8c
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/handlers -run TestMonicaImport_PermissionDenied -v -count=1
    Expected Result: u975e Manager u7528u6237u8fd4u56de 403
    Evidence: .sisyphus/evidence/task-7-permission.txt
  ```

  **Commit**: YES
  - Message: `feat(server): add Monica import handler with swagger annotations`
  - Files: `server/internal/handlers/monica_import.go`, `server/internal/handlers/routes.go`, `server/internal/handlers/handlers_test.go`
  - Pre-commit: `cd server && go test ./internal/handlers -run TestMonicaImport -v -count=1 && swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal`

- [ ] 8. u524du7aefu5bfcu5165u9875u9762 + i18n

  **What to do**:
  - u5148u8fd0u884c `make gen-api` u751fu6210u65b0u7684 TypeScript API u5ba2u6237u7aefuff08u5305u542bu65b0u7684 Monica import u7aefu70b9uff09
  - u5728 Vault Settings u9875u9762u4e2du65b0u589e "Monica Import" section u6216 tabuff1a
    - u67e5u770bu73b0u6709 Vault Settings u9875u9762u7ed3u6784uff0cu627eu5230u5408u9002u7684u4f4du7f6eu6dfbu52a0
    - UI u7ec4u4ef6uff1a
      - u6807u9898 + u8bf4u660eu6587u5b57uff08u89e3u91ca Monica JSON u5bfcu5165u662fu4ec0u4e48uff0cu5982u4f55u83b7u53d6u5bfcu51fau6587u4ef6uff09
      - Ant Design `Upload` u7ec4u4ef6uff08`Dragger` u98ceu683cuff0cu5141u8bb8u62d6u62fdu4e0au4f20uff09
      - u53eau5141u8bb8 `.json` u6587u4ef6
      - u4e0au4f20u4e2du663eu793a loading u72b6u6001
      - u6210u529fu540eu663eu793au5bfcu5165u62a5u544auff1a
        - u5404u7c7bu578bu5bfcu5165u6570u91cfuff08contacts, notes, calls, tasks, reminders, addresses, relationships, photos, documentsuff09
        - u8df3u8fc7u6570u91cf
        - u9519u8befu5217u8868uff08u5982u679cu6709uff09
      - u5931u8d25u663eu793au9519u8befu6d88u606f
    - u4f7fu7528 `api.vaultSettings.importMonica(vaultId, file)` u8c03u7528u540eu7aef
    - u6240u6709u6587u672cu4f7fu7528 `t()` u51fdu6570
  - i18n u952eu6dfbu52a0u5230 `web/src/locales/en.json` u548c `zh.json`uff1a
    - `vault_settings.monica_import.title` = "Monica Import" / "u5bfcu5165 Monica u6570u636e"
    - `vault_settings.monica_import.description` = u8bf4u660eu6587u5b57
    - `vault_settings.monica_import.upload_hint` = u4e0au4f20u63d0u793a
    - `vault_settings.monica_import.success` = u6210u529fu6d88u606f
    - `vault_settings.monica_import.imported_contacts` = "u8054u7cfbu4eba"
    - u7b49u7b49uff08u6bcfu4e2au8ba1u6570u5b57u6bb5 + u9519u8bef/u8df3u8fc7uff09
  - `bun run build` u548c `bun run lint` u786eu4fddu901au8fc7

  **Must NOT do**:
  - u4e0du6784u5efau9884u89c8/u6620u5c04/u9009u62e9u6027u5bfcu5165 UI
  - u4e0du6dfbu52a0u8fdbu5ea6u6761uff08v1 u540cu6b65u5904u7406uff0cu7b49u5f85u5373u53efuff09
  - u4e0du4f7fu7528 `as any` u6216 `@ts-ignore`
  - u4e0du786cu7f16u7801u65e5u671fu683cu5f0f

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: u524du7aef UI u7ec4u4ef6u5f00u53d1uff0cu6d89u53ca Ant Design u7ec4u4ef6u4f7fu7528
  - **Skills**: [`frontend-design`]
    - `frontend-design`: u786eu4fdd UI u8bbeu8ba1u8d28u91cf

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 9, 10)
  - **Blocks**: Task 10
  - **Blocked By**: Task 7

  **References**:

  **Pattern References**:
  - `web/src/pages/vault/VaultSettings.tsx` u2014 u73b0u6709 Vault Settings u9875u9762uff08Tabs u5e03u5c40uff0c929 u884cuff09
  - `web/src/api/index.ts` u2014 API u5ba2u6237u7aefu5165u53e3

  **API/Type References**:
  - `web/src/api/generated/` u2014 u81eau52a8u751fu6210u7684 API u7c7bu578buff08`make gen-api` u540euff09

  **Test References**:
  - `web/src/test/` u2014 u73b0u6709 vitest u6d4bu8bd5u6a21u5f0f

  **WHY Each Reference Matters**:
  - VaultSettings u9875u9762u786eu5b9au65b0 section u7684u653eu7f6eu4f4du7f6eu548cu98ceu683cu4e00u81f4u6027
  - API u5ba2u6237u7aefu786eu5b9au8c03u7528u65b9u5f0f

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: u524du7aefu6784u5efau6210u529f
    Tool: Bash
    Steps:
      1. cd web && bun run build
    Expected Result: exit 0
    Evidence: .sisyphus/evidence/task-8-build.txt

  Scenario: Lint u901au8fc7
    Tool: Bash
    Steps:
      1. cd web && bun run lint
    Expected Result: u65e0u9519u8bef
    Evidence: .sisyphus/evidence/task-8-lint.txt

  Scenario: i18n u952eu4e00u81f4u6027
    Tool: Bash
    Steps:
      1. cd web && bun run lint
    Expected Result: i18n u68c0u67e5u901au8fc7uff08en.json u548c zh.json u952eu4e00u81f4uff09
    Evidence: .sisyphus/evidence/task-8-i18n.txt
  ```

  **Commit**: YES
  - Message: `feat(web): add Monica import page in vault settings`
  - Files: `web/src/pages/VaultSettings/u76f8u5173u6587u4ef6`, `web/src/locales/en.json`, `web/src/locales/zh.json`
  - Pre-commit: `cd web && bun run build && bun run lint`

- [ ] 9. Feed u8bb0u5f55 + Search u7d22u5f15

  **What to do**:
  - u5728 `MonicaImportService` u4e2du63a5u5165 `FeedRecorder` u548c `search.Engine`uff1a
    - u4feeu6539 `NewMonicaImportService` u63a5u53d7 `feedRecorder *FeedRecorder` u548c `searchEngine search.Engine` u53c2u6570
    - u5728u6bcfu4e2a Contact u521bu5efau540eu8c03u7528 `feedRecorder.Record(contactID, userID, ActionContactCreated, "", contactID, "contacts")`
    - u6ce8u610fuff1au53eau8bb0u5f55 contact_createduff0cu4e0du4e3au6bcfu4e2au5b50u8d44u6e90u8bb0u5f55uff08u907fu514d feed u5237u5c4fuff09
    - u5728u6240u6709 Contact u521bu5efau5b8cu6210u540eu8c03u7528 `searchEngine.IndexContact(contact)` u5efau7acbu641cu7d22u7d22u5f15
  - u4feeu6539 `routes.go` u4e2d MonicaImportService u7684u521du59cbu5316uff0cu4f20u5165 feedRecorder u548c searchEngine
  - u6d4bu8bd5uff1a
    - `TestMonicaImport_FeedRecords` u2014 u5bfcu5165 3 u4e2au8054u7cfbu4ebau540euff0cu67e5u8be2 ContactFeedItem u8868uff0cu9a8cu8bc1u6709 3 u6761 contact_created u8bb0u5f55
    - `TestMonicaImport_SearchIndex` u2014 u5bfcu5165u540eu641cu7d22u8054u7cfbu4ebau540du79f0u53efu627eu5230

  **Must NOT do**:
  - u4e0du4e3au5b50u8d44u6e90uff08note/call/task u7b49uff09u751fu6210 feed u8bb0u5f55
  - u4e0du4feeu6539 FeedRecorder u6216 SearchEngine u7684u63a5u53e3

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: u63a5u7ebfu5df2u6709u670du52a1uff0cu4ee3u7801u91cfu5c0f
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 8, 10)
  - **Blocks**: Task 10
  - **Blocked By**: Tasks 3, 4, 5

  **References**:

  **Pattern References**:
  - `server/internal/services/contact.go` u2014 Contact u521bu5efau540eu7684 FeedRecorder.Record u8c03u7528u6a21u5f0f
  - `server/internal/services/note.go` u2014 Note u521bu5efau540eu7684 FeedRecorder u548c SearchEngine u8c03u7528
  - `server/internal/search/` u2014 SearchEngine u63a5u53e3u548c IndexContact u65b9u6cd5

  **WHY Each Reference Matters**:
  - contact.go u5c55u793au4e86 FeedRecorder.Record u7684u6b63u786eu8c03u7528u53c2u6570u548c action u5e38u91cf
  - search/ u5305u786eu5b9a IndexContact u7684u8c03u7528u7b7eu540d

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Feed u8bb0u5f55u751fu6210
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run TestMonicaImport_FeedRecords -v -count=1
    Expected Result: u6bcfu4e2au5bfcu5165u7684 contact u6709u4e00u6761 feed u8bb0u5f55
    Evidence: .sisyphus/evidence/task-9-feed.txt

  Scenario: u641cu7d22u7d22u5f15u5efau7acb
    Tool: Bash
    Steps:
      1. cd server && go test ./internal/services -run TestMonicaImport_SearchIndex -v -count=1
    Expected Result: u5bfcu5165u7684u8054u7cfbu4ebau53efu901au8fc7u641cu7d22u627eu5230
    Evidence: .sisyphus/evidence/task-9-search.txt
  ```

  **Commit**: YES
  - Message: `feat(server): wire feed recording and search indexing for Monica import`
  - Files: `server/internal/services/monica_import.go`, `server/internal/services/monica_import_test.go`, `server/internal/handlers/routes.go`
  - Pre-commit: `cd server && go test ./internal/services -run TestMonicaImport -v -count=1`

- [ ] 10. Playwright E2E u6d4bu8bd5

  **What to do**:
  - u521bu5efa `web/e2e/monica-import.spec.ts`uff1a
    - u521bu5efau6d4bu8bd5u7528 fixture u6587u4ef6 `web/e2e/fixtures/monica_export.json`uff08u7b80u5316u7248uff0c2 u4e2au8054u7cfbu4eba + u5c11u91cfu5b50u8d44u6e90uff0cu65e0 base64 u6587u4ef6uff09
    - u6d4bu8bd5u6d41u7a0buff1a
      1. u6ce8u518c + u767bu5f55 + u521bu5efa Vault
      2. u5bfcu822au5230 Vault Settings
      3. u627eu5230 Monica Import section
      4. u4e0au4f20 fixture JSON u6587u4ef6
      5. u7b49u5f85u5bfcu5165u5b8cu6210
      6. u9a8cu8bc1u6210u529fu6d88u606fu663eu793auff0cu5305u542bu5bfcu5165u8ba1u6570
      7. u5bfcu822au5230u8054u7cfbu4ebau5217u8868uff0cu9a8cu8bc1u5bfcu5165u7684u8054u7cfbu4ebau51fau73b0
      8. u70b9u51fbu5bfcu5165u7684u8054u7cfbu4ebauff0cu9a8cu8bc1u5b50u8d44u6e90u5b58u5728uff08u5982 notes tab u6709u5185u5bb9uff09
    - u9519u8befu573au666fuff1a
      - u4e0au4f20u975e JSON u6587u4ef6uff0cu9a8cu8bc1u9519u8befu63d0u793a

  **Must NOT do**:
  - u4e0du6d4bu8bd5u5927u6587u4ef6u4e0au4f20uff08E2E u4e2du4e0du9002u5408uff09
  - u4e0du6d4bu8bd5 base64 u6587u4ef6u5bfcu5165uff08E2E fixture u592au5927uff09

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Playwright E2E u6d4bu8bd5u7f16u5199uff0cu9700u8981u7406u89e3u524du540eu7aefu4ea4u4e92
  - **Skills**: [`playwright`]
    - `playwright`: Playwright u6d4bu8bd5u7f16u5199u4e13u7528 skill

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (after Tasks 8, 9)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 7, 8, 9

  **References**:

  **Pattern References**:
  - `web/e2e/settings.spec.ts` u2014 Settings u9875u9762 E2E u6d4bu8bd5u6a21u5f0f
  - `web/e2e/vault.spec.ts` u2014 Vault u64cdu4f5c E2E u6d4bu8bd5u6a21u5f0f
  - `web/e2e/contact.spec.ts` u2014 u8054u7cfbu4ebau521bu5efa/u67e5u770b E2E u6a21u5f0f
  - `web/playwright.config.ts` u2014 Playwright u914du7f6euff08u7aefu53e3u3001u81eau52a8u542fu52a8u670du52a1u5668uff09

  **WHY Each Reference Matters**:
  - settings.spec.ts u5c55u793au5982u4f55u5bfcu822au5230u8bbeu7f6eu9875u9762u548cu64cdu4f5cu8868u5355
  - vault.spec.ts u5c55u793a vault u521bu5efau6d41u7a0buff08E2E u524du7f6eu6b65u9aa4uff09
  - contact.spec.ts u5c55u793au8054u7cfbu4ebau5217u8868u548cu8be6u60c5u9875u7684u9a8cu8bc1u65b9u5f0f

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: E2E u5bfcu5165u6d41u7a0b
    Tool: Bash
    Steps:
      1. cd web && bunx playwright test e2e/monica-import.spec.ts
    Expected Result: u6240u6709u6d4bu8bd5u7528u4f8bu901au8fc7
    Failure Indicators: "failed" u6216 timeout
    Evidence: .sisyphus/evidence/task-10-e2e.txt
  ```

  **Commit**: YES
  - Message: `test(e2e): add Monica import E2E test`
  - Files: `web/e2e/monica-import.spec.ts`, `web/e2e/fixtures/monica_export.json`
  - Pre-commit: `cd web && bunx playwright test e2e/monica-import.spec.ts`

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `cd server && go vet ./...` + `cd web && bun run lint` + `cd web && bun run build` + `cd server && go test ./... -count=1` + `cd web && bun run test`. Review all changed files for: `as any`/`@ts-ignore`, empty catches, console.log in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high` (+ `playwright` skill)
  Start from clean state. Execute EVERY QA scenario from EVERY task. Test cross-task integration: upload Monica JSON → verify all entity types imported → check contact list → check contact detail → check relationships. Save screenshots to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1. Check "Must NOT do" compliance. Detect cross-task contamination. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| Commit | Scope | Message | Pre-commit Check |
|--------|-------|---------|------------------|
| 1 | T1 | `feat(server): add Monica 4.x JSON struct definitions and test fixture` | `go build ./...` |
| 2 | T2 | `feat(server): add Monica import DTO definitions` | `go build ./...` |
| 3 | T3 | `feat(server): implement Monica contact import with tag/gender/date mapping` | `go test ./internal/services -run TestMonicaImport -count=1` |
| 4 | T4 | `feat(server): implement Monica sub-resource import (note/call/task/reminder/address/pet/gift/loan/life-event)` | `go test ./internal/services -run TestMonicaImport -count=1` |
| 5 | T5 | `feat(server): implement Monica relationship import and activity/conversation degradation` | `go test ./internal/services -run TestMonicaImport -count=1` |
| 6 | T6 | `feat(server): implement Monica photo/document import with base64 decoding` | `go test ./internal/services -run TestMonicaImport -count=1` |
| 7 | T7 | `feat(server): add Monica import handler with swagger annotations` | `go test ./internal/handlers -run TestMonicaImport -count=1` |
| 8 | T8 | `feat(web): add Monica import page in vault settings` | `cd web && bun run build && bun run lint` |
| 9 | T9 | `feat(server): wire feed recording and search indexing for Monica import` | `go test ./internal/services -run TestMonicaImport -count=1` |
| 10 | T10 | `test(e2e): add Monica import E2E test` | `cd web && bunx playwright test e2e/monica-import.spec.ts` |

---

## Success Criteria

### Verification Commands
```bash
cd server && go build ./...           # Expected: no errors
cd server && go vet ./...              # Expected: no errors
cd server && go test ./... -count=1    # Expected: all PASS
cd web && bun run build                # Expected: exit 0
cd web && bun run lint                 # Expected: no errors
cd web && bun run test                 # Expected: all PASS
cd web && bunx playwright test e2e/monica-import.spec.ts  # Expected: PASS
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All Go tests pass
- [ ] All frontend tests pass
- [ ] E2E test passes
- [ ] i18n keys present in both en.json and zh.json
- [ ] Swagger annotations correct (`make swagger` succeeds)
- [ ] `make gen-api` generates updated TypeScript client
