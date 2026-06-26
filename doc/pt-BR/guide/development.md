# Desenvolvimento

## Pré-requisitos

- **Go** 1.25+
- **[Bun](https://bun.sh)** 1.x (usado em vez de npm/yarn)
- **Make** (GNU Make)

## Configuração

```bash
git clone https://github.com/naiba/bonds.git
cd bonds

# Instalar todas as dependências (módulos Go + pacotes Bun)
make setup

# Gerar cliente da API (necessário antes do primeiro build)
make gen-api

# Iniciar frontend e backend em modo de desenvolvimento
make dev
```

Isso executa o backend Go na `:8080` e o servidor de desenvolvimento Vite na `:5173`. O frontend faz proxy das requisições de API para o backend automaticamente.

## Pipeline de Geração de Código

O cliente TypeScript da API do frontend é **gerado automaticamente** a partir da especificação OpenAPI/Swagger do backend:

```
Handlers Go (anotações swag)
    ↓  make swagger
server/docs/swagger.json
    ↓  make gen-api
web/src/api/generated/   ← gitignorado
    ↓
web/src/api/index.ts     ← ponto de entrada
```

Após alterar qualquer API do backend (handlers, DTOs, rotas):

```bash
make gen-api   # Regenerar swagger.json + cliente TypeScript
```

::: warning
Nunca edite manualmente arquivos em `web/src/api/generated/`. Eles são sobrescritos a cada geração.
:::

## Comandos Úteis

| Comando | Descrição |
|---------|-----------|
| `make dev` | Iniciar frontend + backend em modo de desenvolvimento |
| `make build` | Compilar backend + frontend separadamente |
| `make build-all` | Compilar binário único com frontend embarcado |
| `make test` | Executar todos os testes (backend + frontend) |
| `make test-server` | Executar apenas testes do backend |
| `make test-web` | Executar apenas testes do frontend |
| `make test-e2e` | Executar testes E2E Playwright |
| `make lint` | Executar linters (`go vet` + ESLint) |
| `make swagger` | Regenerar documentação Swagger/OpenAPI |
| `make gen-api` | Regenerar Swagger + cliente TypeScript da API |
| `make clean` | Limpar todos os artefatos de build |
| `make setup` | Instalar todas as dependências |

## Estrutura do Projeto

```
server/                    # Backend Go
  cmd/server/main.go       # Ponto de entrada
  internal/
    handlers/               # Handlers HTTP (Echo)
    services/               # Lógica de negócios
    models/                 # Modelos GORM
    dto/                    # Estruturas de requisição/resposta
    middleware/              # Autenticação JWT, CORS, etc.
    search/                 # Busca de texto completo Bleve
    dav/                    # Servidor CardDAV/CalDAV
    cron/                   # Agendador cron
    i18n/                   # Internacionalização do backend
  pkg/
    avatar/                 # Geração de avatar com iniciais
    response/               # Auxiliares de resposta da API

web/                       # Frontend React
  src/
    api/                    # Cliente da API gerado automaticamente
    components/             # Componentes compartilhados
    pages/                  # Páginas de rotas
    stores/                 # Contextos de autenticação + tema
    locales/                # Internacionalização (en.json, zh.json, es.json, fr.json, de.json, pt-BR.json, pt-PT.json)
    utils/                  # Funções utilitárias
  e2e/                      # Testes Playwright
```

## Arquitetura do Backend

Cada funcionalidade segue: **Handler** (camada HTTP) → **Service** (lógica de negócios) → **DTO** (requisição/resposta) → **Model** (GORM).

- Handlers vinculam requisições, validam, delegam a serviços e retornam via auxiliares `response.*`
- Services recebem DTOs, retornam DTOs e mantêm `*gorm.DB` para consultas
- Models são estruturas GORM puras sem lógica de negócios

## Testes

```bash
# Testes do backend (SQLite em memória)
cd server && go test ./... -v -count=1

# Testes unitários do frontend (Vitest)
cd web && bun run test

# Testes E2E (Playwright — inicia servidores automaticamente)
cd web && bunx playwright test
```

## Documentação da API (Swagger)
Bonds gera automaticamente documentação OpenAPI cobrindo todos os 286 endpoints da API.

Para acessar o Swagger UI, ative o modo de depuração ou alterne em Admin > Configurações > Swagger:
```bash
# Opção 1: Modo de depuração (Swagger ativado por padrão)
DEBUG=true ./bonds-server
# Opção 2: Ativar via interface de administração sem modo de depuração
# Vá para Admin > Configurações > Swagger > Ativar
```

Depois abra http://localhost:8080/swagger/index.html

> O Swagger UI usa por padrão a flag `DEBUG`, mas pode ser ativado/desativado independentemente na página de Configurações de Administração.

## Endpoint MCP

O acesso para agentes de IA é servido pelo endpoint [`/mcp`](/pt-BR/features/ai-agents) nativo do backend. Ele é intencionalmente separado do pipeline de geração Swagger/OpenAPI: alterações apenas no MCP não exigem `make gen-api`, e `/mcp` não está incluído no cliente gerado da API do frontend.
