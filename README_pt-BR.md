# Bonds

[![Test](https://github.com/naiba/bonds/actions/workflows/test.yml/badge.svg)](https://github.com/naiba/bonds/actions/workflows/test.yml)
[![Release](https://github.com/naiba/bonds/actions/workflows/release.yml/badge.svg)](https://github.com/naiba/bonds/actions/workflows/release.yml)
[![GitHub Release](https://img.shields.io/github/v/release/naiba/bonds)](https://github.com/naiba/bonds/releases)

📖 [Documentação](https://naiba.github.io/bonds/) | [中文文档](README_zh.md) | 💬 [Discord](https://discord.gg/faaEJyt4h)

<a href="https://www.producthunt.com/products/bonds?embed=true&amp;utm_source=badge-featured&amp;utm_medium=badge&amp;utm_campaign=badge-bonds" target="_blank" rel="noopener noreferrer"><img alt="Bonds - Remember everything about the people who matter. | Product Hunt" width="250" height="54" src="https://api.producthunt.com/widgets/embed-image/v1/featured.svg?post_id=1091729&amp;theme=light&amp;t=1772852214754"></a>

Um gerenciador de relacionamentos pessoais moderno, orientado pela comunidade, inspirado no [Monica](https://github.com/monicahq/monica), reconstruído com **Go** e **React**.

## Por que Bonds?

Monica é um CRM pessoal open-source muito querido com mais de 24k estrelas. Mas, como um projeto paralelo mantido por uma equipe pequena ([palavras deles](https://github.com/monicahq/monica/issues/6626)), o desenvolvimento desacelerou, deixando mais de 700 issues abertas e capacidade limitada.

**Bonds** continua de onde Monica parou:

- **Rápido e leve**: Binário único, inicializa em milissegundos, memória mínima.
- **Fácil de implantar**: Um binário + SQLite. Sem PHP, sem Composer, sem Node runtime.
- **Interface moderna**: React 19 + TypeScript, experiência SPA fluida.
- **Bem testado**: 1014 testes de backend, 129 testes de frontend, 180 testes E2E.
- **Comunidade em primeiro lugar**: Construído para contribuições e iteração rápida.

> **Créditos**: Bonds está sobre os ombros de [@djaiss](https://github.com/djaiss), [@asbiin](https://github.com/asbiin) e de toda a comunidade Monica. O Monica original permanece disponível sob AGPL-3.0 em [monicahq/monica](https://github.com/monicahq/monica).

## Funcionalidades

- **Contatos**: Gerenciamento completo do ciclo de vida com notas, tarefas, lembretes, presentes, empréstimos de dinheiro e itens, atividades, eventos de vida, animais de estimação e muito mais. Inclui uma flag de verificação necessária para manter seus dados atualizados.
- **Painel do Cofre**: Layout responsivo de 3 colunas com feed de atividades, eventos de vida, métricas de vida (contador +1), registro de humor, lembretes futuros e tarefas pendentes.
- **Cofres**: Isolamento de dados com múltiplos cofres e acesso baseado em funções (Gerente, Editor, Visualizador).
- **Lembretes**: Únicos e recorrentes (semanal, mensal, anual), com notificações por email e compatíveis com Shoutrrr.
- **Busca em Texto Completo**: Busca CJK alimentada por Bleve em contatos e notas.
- **CardDAV / CalDAV**: Sincronize contatos e calendários com Apple, Thunderbird e outros clientes DAV. Suporta Tokens de Acesso Pessoal.
- **Assinaturas de Sincronização DAV**: Assine e sincronize catálogos de endereços CardDAV externos diretamente em um cofre.
- **Tokens de Acesso Pessoal**: Gere tokens de API e sincronização seguros para acessar endpoints com segurança.
- **Acesso para Agentes de IA (MCP)**: Endpoint `/mcp` integrado para clientes MCP. Agentes podem descobrir capacidades, pesquisar dados do cofre, buscar recursos e executar operações existentes da `/api` com a mesma autenticação e permissões. Veja [Acesso para Agentes de IA](https://naiba.github.io/bonds/features/ai-agents).
- **Importação CSV**: Importe contatos de um arquivo CSV com mapeamento de colunas definido pelo usuário (nome, email, telefone, aniversário, endereço, tags, grupos, notas).
- **Importação Monica**: Migre contatos diretamente de uma instância Monica via API.
- **Importação/Exportação vCard**: Importe em lote arquivos `.vcf`, exporte contatos individuais ou todos.
- **Upload de Arquivos**: Mídia de contato com fotos e vídeos, anexos de documentos e avatares iniciais gerados. Limites de tamanho de armazenamento gerenciados diretamente pela interface.
- **Autenticação de Dois Fatores (TOTP)**: 2FA baseada em TOTP com códigos de recuperação.
- **WebAuthn / FIDO2**: Login por chave de acesso (chaves de hardware, biometria).
- **Login OAuth**: Login único com GitHub e Google.
- **Convites de Usuário**: Convide outros para sua conta via email com níveis de permissão.
- **Registro de Auditoria**: Feed de todas as alterações nos contatos.
- **Geocodificação**: Coordenadas de endereço via Nominatim (gratuito) ou LocationIQ.
- **Notificações Shoutrrr**: Entrega de lembretes via Telegram e outros canais compatíveis com Shoutrrr.
- **i18n**: Inglês, Chinês e Português, frontend e backend.

## Início Rápido

### Opção 1: Docker (Recomendado)

```bash
# Baixe o docker-compose.yml
curl -O https://raw.githubusercontent.com/naiba/bonds/main/docker-compose.yml

# Inicie o serviço
docker compose up -d
```

Abra **http://localhost:8080** e crie sua conta.

Para personalizar as configurações, edite `docker-compose.yml`:

```yaml
environment:
  - JWT_SECRET=your-secret-key-here   # ⚠️ Altere isto!
```

### Opção 2: Binário Pré-compilado

Baixe a versão mais recente dos [GitHub Releases](https://github.com/naiba/bonds/releases) e então:

```bash
export JWT_SECRET=your-secret-key-here
./bonds-server
```

O servidor inicia em **http://localhost:8080** com um frontend embutido e banco de dados SQLite.

### Opção 3: Compilar a Partir do Código Fonte

**Pré-requisitos**: Go 1.25+, [Bun](https://bun.sh) 1.x

```bash
git clone https://github.com/naiba/bonds.git
cd bonds

# Instale as dependências
make setup

# Compile um único binário (frontend embutido)
make build-all

# Execute
export JWT_SECRET=your-secret-key-here
./server/bin/bonds-server
```

## Configuração

Bonds usa uma abordagem de **configuração híbrida**:

- **Variáveis de ambiente**: Para configurações essenciais de infraestrutura (banco de dados, servidor, segurança).
- **Interface de administração**: Para todas as configurações em tempo de execução (SMTP, OAuth, Telegram, WebAuthn, limite de tamanho de armazenamento, etc.).

Na primeira inicialização, as variáveis de ambiente são semeadas no banco de dados. Após isso, gerencie as configurações a partir de **Admin > Configurações do Sistema** na interface web.

```bash
cp server/.env.example server/.env
```

### Variáveis de Ambiente (Obrigatórias)

| Variável | Padrão | Descrição |
|----------|---------|-------------|
| `DEBUG` | `false` | Ativa o modo de depuração: registro de requisições Echo, logs SQL do GORM, Swagger UI (ativo por padrão) |
| `JWT_SECRET` | — | **Obrigatório em produção.** Chave de assinatura para tokens de autenticação |
| `SETTINGS_ENC_KEY` | _(vazio)_ | Opcional. Ativa criptografia em repouso AES-256-GCM para segredos SMTP/OAuth/geocodificação. Veja [docs](https://naiba.github.io/bonds/guide/configuration#encrypting-sensitive-settings) |
| `SERVER_PORT` | `8080` | Porta em que o servidor escuta |
| `SERVER_HOST` | `0.0.0.0` | Endereço do host ao qual o servidor se vincula |
| `DB_DSN` | `bonds.db` | String de conexão do banco de dados. SQLite: caminho do arquivo; PostgreSQL: `host=... port=5432 user=... password=... dbname=... sslmode=disable` |
| `DB_DRIVER` | `sqlite` | Driver do banco de dados (`sqlite` ou `postgres`) |
| `APP_ENV` | `development` | Defina como `production` para uso em produção |
| `STORAGE_UPLOAD_DIR` | `uploads` | Diretório de upload de arquivos |
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | Diretório do índice de busca em texto completo |
| `BACKUP_DIR` | `data/backups` | Diretório para armazenar arquivos de backup |

### Configurações da Interface de Administração

As seguintes são gerenciadas a partir da página **Admin > Configurações do Sistema** após o login:

- **Aplicação**: Nome, URL, banner de anúncio.
- **Autenticação**: Alternar autenticação por senha, alternar registro de usuário.
- **JWT**: Expiração do token, janela de renovação.
- **SMTP**: Host, Porta, opcional Usuário/Senha, Email do remetente. Deixe ambos Usuário e Senha vazios para relays não autenticados.
- **OAuth / OIDC**: Credenciais GitHub, Google e OIDC/SSO.
- **WebAuthn**: ID do Relying Party, Nome de Exibição, Origens.
- **Telegram**: Token do bot para notificações.
- **Geocodificação**: Provedor (Nominatim/LocationIQ), chave da API.
- **Armazenamento**: Limite máximo de tamanho de upload.
- **Backup**: Agendamento Cron, dias de retenção.
- **Swagger**: Ativar ou desativar a interface de documentação da API.

## Desenvolvimento

```bash
# Instale as dependências
make setup

# Gere o cliente da API (necessário antes da primeira compilação)
make gen-api

# Inicie frontend e backend em modo de desenvolvimento
make dev
```

Isso executa o backend Go em `:8080` e o servidor de desenvolvimento Vite em `:5173`. O frontend faz proxy automático das requisições da API para o backend.

### Pipeline de Geração de Código

O cliente TypeScript da API do frontend é **gerado automaticamente** a partir do esquema OpenAPI do backend. Os arquivos gerados não são commitados no git. Eles são regenerados no CI e durante o desenvolvimento.

```
Go handlers (anotações swag)
    ↓  make swagger
server/docs/swagger.json
    ↓  make gen-api (ou bun run gen:api)
web/src/api/generated/   ← gitignored, regenerado sob demanda
    ↓
web/src/api/index.ts     ← ponto de entrada, importa módulos gerados
```

Após alterar qualquer API do backend (handlers, DTOs, rotas), execute:

```bash
make gen-api       # Regenera swagger.json + cliente TypeScript da API
```

### Comandos Úteis

```bash
make dev           # Inicia frontend + backend em modo de desenvolvimento
make build         # Compila backend + frontend separadamente
make build-all     # Compila binário único com frontend embutido
make test          # Executa todos os testes (backend + frontend)
make test-e2e      # Executa testes de ponta a ponta (Playwright)
make lint          # Executa linters (go vet + eslint)
make swagger       # Regenera apenas a documentação Swagger/OpenAPI
make gen-api       # Regenera docs Swagger + cliente TypeScript da API
make clean         # Limpa todos os artefatos de compilação + arquivos gerados
make setup         # Instala todas as dependências
```

### Documentação da API

Bonds fornece documentação OpenAPI/Swagger gerada automaticamente cobrindo todos os endpoints da API.

Para acessar o Swagger UI, ative o modo de depuração ou ative-o em Admin > Configurações > Swagger:
```bash
# Opção 1: Modo de depuração (Swagger ativado por padrão)
DEBUG=true ./bonds-server
# Opção 2: Ativar via interface de administração sem modo de depuração
# Vá para Admin > Configurações > Swagger > Ativar
```

Então abra http://localhost:8080/swagger/index.html

> O Swagger UI usa por padrão a flag `DEBUG`, mas pode ser ativado/desativado independentemente na página de Configurações do Admin.

## Relação com o Monica

Bonds é uma reescrita do zero inspirada pelo [Monica](https://github.com/monicahq/monica) (AGPL-3.0). Ele reimplementa o modelo de dados e o conjunto de funcionalidades do Monica usando uma pilha de tecnologia completamente diferente (Go + React em vez de PHP/Laravel + Vue). Não contém nenhum código do projeto original.

## Licença

[Business Source License 1.1](LICENSE) (BSL 1.1), Licença de Código Fonte Disponível com os seguintes termos:

- **Indivíduos**: Gratuito para qualquer uso não comercial.
- **Organizações**: O uso comercial requer uma licença paga do Licenciante.
- **Proibido**: Revender, sublicenciar ou oferecer como um serviço gerenciado/hospedado.
- **Data de Mudança**: 13 de junho de 2030, converte automaticamente para [AGPL-3.0](LICENSE) (mesma do Monica original).

Após a Data de Mudança, o software se torna totalmente open source sob AGPL-3.0.

Ao enviar código, documentação, traduções ou qualquer outra contribuição, você concorda com os [termos de contribuição](CONTRIBUTING.md), incluindo a renúncia de toda propriedade e outros direitos ou reivindicações sobre essa contribuição na extensão máxima permitida por lei.
