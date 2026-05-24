# AI Agent 接入

Bonds 在后端内置了 Model Context Protocol（MCP）端点 `/mcp`。AI 客户端可以通过它发现 Bonds 能力、搜索 Vault 数据、读取资源，并通过现有后端权限执行已有 `/api` 操作。

这不是 CLI，也不是独立运行的 MCP 服务进程。端点运行在现有 Bonds 后端内部，跟随正常 Bonds 服务一起部署。

## 端点

| 协议 | URL | 传输方式 |
|------|-----|----------|
| MCP | `/mcp` | HTTP `POST` 上的 JSON-RPC |

`GET /mcp` 和 `DELETE /mcp` 返回 `405 Method Not Allowed`。

## 认证

使用与 REST API 相同的 Bearer 认证：

```http
Authorization: Bearer <jwt-or-personal-access-token>
```

长期运行的 agent 集成建议在 **设置 > API 令牌** 中创建个人访问令牌，并把它作为 Bearer token 使用。令牌以 `bonds_` 开头。

MCP 端点要求用户已登录、账户未禁用；如果启用了邮箱验证，还要求邮箱已验证。工具调用会沿用调用者身份；Vault、账户管理员和实例管理员权限都由现有后端中间件执行。

## 工具

| 工具 | 用途 |
|------|------|
| `get_current_context` | 返回当前用户和可访问的 Vault。 |
| `discover_capabilities` | 列出可通过 `execute_action` 调用的 Bonds `/api` action。 |
| `describe_capability` | 返回某个 action 的方法、路径和必填 path 参数等元数据。 |
| `execute_action` | 通过现有 Echo 路由栈和权限执行已注册的 `/api` action。 |
| `search_bonds` | 在单个 Vault 内用结构化查询和现有 Bleve 全文索引搜索。 |
| `fetch_resource` | 在 Viewer 权限校验后读取支持的 `bonds://...` 资源。 |

## 执行 API 操作

后端 Echo 中注册的每个 `/api` operation 都会暴露为 MCP action。`/mcp` 自身和 Swagger 等非 API 路由不会成为 action 目标。

`execute_action` 不接受任意 URL 或 SQL。服务端会从已注册的后端 `/api` 路由生成 action registry，只允许调用 registry 中存在的 `action_id`。Action 元数据包含 HTTP 方法、路径模板、path 参数，以及是否只读或具有破坏性。

内部请求会重新进入现有 API handler，因此原本的请求校验和权限仍然生效。例如 Viewer 可以发现创建联系人的 action，但执行时仍会失败，因为原始 `/api/vaults/:vault_id/contacts` 路由要求 Editor 权限。

示例结构：

```json
{
  "action_id": "post_vaults_by_vault_id_contacts",
  "path_params": {
    "vault_id": "vault-uuid"
  },
  "body": {
    "first_name": "Alice",
    "last_name": "Example"
  }
}
```

## 搜索

`search_bonds` 限定在单个 Vault 内，并要求调用者至少拥有该 Vault 的 Viewer 权限。它结合了：

- 现有 Bleve 全文索引，用于联系人和笔记；
- 固定 GORM 查询，用于联系人、联系信息、笔记、任务、提醒和重要日期；
- 结构化自然语言过滤，例如逾期任务、未完成任务、今天的提醒和下个月生日。

Bonds MCP v1 不使用 embedding 或向量搜索。搜索能力会明确返回 `semantic_vector_search: false`。

AI 客户端不能提交 SQL。SQL 只由服务端固定 GORM 查询生成，并在返回数据前按 Vault 权限进行限制。

## 资源

`fetch_resource` 支持以下 URI：

| 资源 | URI |
|------|-----|
| Vault | `bonds://vault/{id}` |
| 联系人 | `bonds://contact/{id}` |
| 笔记 | `bonds://note/{id}` |
| 任务 | `bonds://task/{id}` |
| 提醒 | `bonds://reminder/{id}` |
| 重要日期 | `bonds://important-date/{id}` |

每次资源读取都会校验所属 Vault 的 Viewer 权限。未列出的用户影子联系人不会被 `fetch_resource` 返回；仅关联到影子联系人的资源也会被过滤。

## 客户端兼容性

`/mcp` 端点已有官方 Go MCP SDK（`github.com/modelcontextprotocol/go-sdk/mcp`）集成测试覆盖。测试通过 HTTP 连接，初始化协议版本 `2025-06-18`，列出工具，通过 `execute_action` 创建联系人，再通过 `search_bonds` 找到联系人，并用 `fetch_resource` 读回资源。

支持 MCP streamable HTTP 的客户端可连接：

```text
https://your-bonds.example.com/mcp
```

并配置上文所示的 Bearer token 请求头。

## OpenAPI 和 CLI

`/mcp` 端点刻意独立于 REST OpenAPI 客户端生成管线。它不会进入前端生成 API client；只修改 MCP 时不需要运行 `make gen-api`。

MCP v1 不包含 CLI、独立 MCP binary、向量搜索、confirmation gate，也不新增 MCP 专用 audit log。通过 MCP 执行动作时，已有的 Bonds feed 和 API 侧校验仍会正常工作。
