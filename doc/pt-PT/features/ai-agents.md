# Acesso para Agentes de IA

Bonds expõe um endpoint interno do Model Context Protocol (MCP) em `/mcp`. Clientes de IA podem descobrir capacidades do Bonds, pesquisar dados do cofre, obter recursos e executar operações `/api` existentes através das mesmas permissões de backend usadas pela aplicação web.

Isto não é um CLI nem um processo de servidor MCP separado. O endpoint é executado dentro do backend Bonds existente e é implementado com o servidor Bonds normal.

## Endpoint

| Protocolo | URL | Transporte |
|-----------|-----|------------|
| MCP | `/mcp` | JSON-RPC sobre HTTP `POST` |

`GET /mcp` e `DELETE /mcp` retornam `405 Método Não Permitido`.

## Autenticação

Use a mesma autenticação Bearer da API REST:

```http
Authorization: Bearer <jwt-ou-token-de-acesso-pessoal>
```

Para integrações de agente de longa duração, crie um Token de Acesso Pessoal em **Definições > Tokens de API** e use-o como token Bearer. Tokens começam com `bonds_`.

O endpoint MCP requer um utilizador autenticado e ativado. Se a verificação de email estiver ativada, o utilizador também deve ter o email verificado. Chamadas de ferramenta mantêm a identidade do chamador; permissões de cofre, administrador de conta e administrador de instância são aplicadas pelo middleware existente do backend.

## Ferramentas

| Ferramenta | Propósito |
|------------|-----------|
| `get_current_context` | Retorna o utilizador autenticado e cofres acessíveis. |
| `discover_capabilities` | Lista ações `/api` registadas do Bonds que podem ser chamadas através de `execute_action`. |
| `describe_capability` | Retorna metadados para uma ação, incluindo método, caminho e parâmetros de caminho necessários. |
| `execute_action` | Executa uma ação `/api` registada através da pilha de rotas Echo existente e permissões. |
| `search_bonds` | Pesquisa dentro de um cofre usando consultas estruturadas mais o índice de texto completo Bleve existente. |
| `fetch_resource` | Lê recursos `bonds://...` suportados com verificações de permissão de Leitor. |

## Execução de Ações da API

Toda operação `/api` do backend registada no Echo é exposta como uma ação MCP. Rotas não-API como `/mcp` e Swagger não são alvos de ação.

`execute_action` não aceita URLs ou SQL arbitrários. O servidor constrói um registo de ações a partir das rotas `/api` registadas do backend e aceita apenas um `action_id` desse registo. Os metadados da ação incluem o método HTTP, modelo de caminho, parâmetros de caminho e se a operação é só de leitura ou destrutiva.

O pedido interno é encaminhado de volta através dos handlers de API existentes, portanto a validação de pedido normal e as permissões ainda se aplicam. Por exemplo, um Leitor pode descobrir uma ação de criação de contacto, mas executá-la ainda falha porque a rota original `/api/vaults/:vault_id/contacts` requer permissão de Editor.

Exemplo de formato:

```json
{
  "action_id": "post_vaults_by_vault_id_contacts",
  "path_params": {
    "vault_id": "cofre-uuid"
  },
  "body": {
    "first_name": "Alice",
    "last_name": "Example"
  }
}
```

## Pesquisa

`search_bonds` tem escopo de um único cofre e requer acesso de Leitor a esse cofre. Ele combina:

- o índice de texto completo Bleve existente para contactos e notas;
- consultas GORM fixas para contactos, informações de contacto, notas, tarefas, lembretes e datas importantes;
- filtros de linguagem natural estruturados como tarefas atrasadas, tarefas abertas, lembretes de hoje e aniversários do seguinte mês.

Bonds não usa embeddings ou pesquisa vetorial para MCP v1. A capacidade de pesquisa informa `semantic_vector_search: false`.

O cliente de IA nunca fornece SQL. SQL é produzido apenas por consultas GORM fixas do lado do servidor, e cada consulta tem escopo definido por verificações de permissão do cofre antes de retornar dados.

## Recursos

`fetch_resource` suporta estas formas de URI:

| Recurso | URI |
|---------|-----|
| Cofre | `bonds://vault/{id}` |
| Contacto | `bonds://contact/{id}` |
| Nota | `bonds://note/{id}` |
| Tarefa | `bonds://task/{id}` |
| Lembrete | `bonds://reminder/{id}` |
| Data importante | `bonds://important-date/{id}` |

Cada leitura de recurso verifica acesso de Leitor ao cofre proprietário. Contactos sombra não listados não são retornados por `fetch_resource`, e recursos anexados apenas a contactos sombra não listados são filtrados.

## Compatibilidade de Clientes

O endpoint MCP é coberto por um teste de integração usando o SDK Go oficial do MCP, `github.com/modelcontextprotocol/go-sdk/mcp`. O teste conecta-se via HTTP, inicializa a versão do protocolo `2025-06-18`, lista ferramentas, cria um contacto através de `execute_action`, encontra-o através de `search_bonds` e o lê de volta através de `fetch_resource`.

Para clientes que suportam MCP streamable HTTP, aponte-os para:

```text
https://bonds.example.com/mcp
```

e configure o cabeçalho de token Bearer mostrado acima.

## OpenAPI e CLI

O endpoint `/mcp` é intencionalmente separado do pipeline de geração de cliente OpenAPI REST. Ele não está incluído no cliente de API do frontend gerado e não requer `make gen-api` após alterações apenas no MCP.

MCP v1 não inclui um CLI, um binário MCP independente, pesquisa vetorial, portas de confirmação ou um log de auditoria específico do MCP. Os feeds existentes do Bonds e a validação do lado da API ainda se comportam normalmente quando ações são executadas através do MCP.
