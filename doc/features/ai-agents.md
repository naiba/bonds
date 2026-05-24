# AI Agent Access

Bonds exposes a built-in Model Context Protocol (MCP) endpoint at `/mcp`. AI clients can discover Bonds capabilities, search vault data, fetch resources, and execute existing `/api` operations through the same backend permissions used by the web app.

This is not a CLI and not a separate MCP server process. The endpoint runs inside the existing Bonds backend and is deployed with the normal Bonds server.

## Endpoint

| Protocol | URL | Transport |
|----------|-----|-----------|
| MCP | `/mcp` | JSON-RPC over HTTP `POST` |

`GET /mcp` and `DELETE /mcp` return `405 Method Not Allowed`.

## Authentication

Use the same Bearer authentication as the REST API:

```http
Authorization: Bearer <jwt-or-personal-access-token>
```

For long-running agent integrations, create a Personal Access Token under **Settings > API Tokens** and use it as the Bearer token. Tokens start with `bonds_`.

The MCP endpoint requires an authenticated, enabled user. If email verification is enabled, the user must also be email-verified. Tool calls keep the caller's identity; vault, account admin, and instance admin permissions are enforced by the existing backend middleware.

## Tools

| Tool | Purpose |
|------|---------|
| `get_current_context` | Returns the authenticated user and accessible vaults. |
| `discover_capabilities` | Lists registered Bonds `/api` actions that can be called through `execute_action`. |
| `describe_capability` | Returns metadata for one action, including method, path, and required path parameters. |
| `execute_action` | Executes a registered `/api` action through the existing Echo route stack and permissions. |
| `search_bonds` | Searches within one vault using structured queries plus the existing Bleve full-text index. |
| `fetch_resource` | Reads supported `bonds://...` resources with viewer permission checks. |

## API Action Execution

Every backend `/api` operation registered in Echo is exposed as an MCP action. Non-API routes such as `/mcp` and Swagger are not action targets.

`execute_action` does not accept arbitrary URLs or SQL. The server builds an action registry from the backend's registered `/api` routes and accepts only an `action_id` from that registry. Action metadata includes the HTTP method, path template, path parameters, and whether the operation is read-only or destructive.

The internal request is routed back through the existing API handlers, so normal request validation and permissions still apply. For example, a Viewer can discover a contact creation action, but executing it still fails because the original `/api/vaults/:vault_id/contacts` route requires Editor permission.

Example shape:

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

## Search

`search_bonds` is scoped to a single vault and requires Viewer access to that vault. It combines:

- the existing Bleve full-text index for contacts and notes;
- fixed GORM queries for contacts, contact information, notes, tasks, reminders, and important dates;
- structured natural-language filters such as overdue tasks, open tasks, today's reminders, and next-month birthdays.

Bonds does not use embeddings or vector search for MCP v1. The search capability reports `semantic_vector_search: false`.

The AI client never supplies SQL. SQL is produced only by fixed service-side GORM queries, and each query is scoped by vault permission checks before returning data.

## Resources

`fetch_resource` supports these URI forms:

| Resource | URI |
|----------|-----|
| Vault | `bonds://vault/{id}` |
| Contact | `bonds://contact/{id}` |
| Note | `bonds://note/{id}` |
| Task | `bonds://task/{id}` |
| Reminder | `bonds://reminder/{id}` |
| Important date | `bonds://important-date/{id}` |

Each resource read checks Viewer access to the owning vault. Unlisted shadow contacts are not returned by `fetch_resource`, and resources attached only to unlisted shadow contacts are filtered out.

## Client Compatibility

The MCP endpoint is covered by an integration test using the official Go MCP SDK, `github.com/modelcontextprotocol/go-sdk/mcp`. The test connects over HTTP, initializes protocol version `2025-06-18`, lists tools, creates a contact through `execute_action`, finds it through `search_bonds`, and reads it back through `fetch_resource`.

For clients that support MCP streamable HTTP, point them at:

```text
https://your-bonds.example.com/mcp
```

and configure the Bearer token header shown above.

## OpenAPI and CLI

The `/mcp` endpoint is intentionally separate from the REST OpenAPI client generation pipeline. It is not included in the generated frontend API client and does not require `make gen-api` after MCP-only changes.

MCP v1 does not include a CLI, a standalone MCP binary, vector search, confirmation gates, or an MCP-specific audit log. Existing Bonds feeds and API-side validation still behave normally when actions are executed through MCP.
