# Configuração

Bonds usa um modelo de configuração híbrido: as **definições de infraestrutura** são definidas via variáveis de ambiente, enquanto as **definições da aplicação** são geridas com segurança através do painel de administração na interface web.

## Variáveis de Ambiente

Copie o ficheiro de exemplo para começar:

```bash
cp server/.env.example server/.env
```

### Definições Principais

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `DEBUG` | `false` | Ativar modo de depuração: logging de pedidos, logging SQL, Swagger UI (padrão ativado) |
| `JWT_SECRET` | — | **Obrigatório em produção.** Chave de assinatura para tokens de autenticação |
| `SETTINGS_ENC_KEY` | _(vazio)_ | Opcional. Ativa encriptação AES-256-GCM em repouso para definições sensíveis do sistema (palavra-passe SMTP, segredos de cliente OAuth, chaves de API de geocodificação). Veja [A encriptar Definições Sensíveis](#criptografando-configuracoes-sensiveis) abaixo. |
| `SERVER_PORT` | `8080` | Porta em que o servidor escuta |
| `SERVER_HOST` | `0.0.0.0` | Endereço do host ao qual o servidor se vincula |
| `DB_DRIVER` | `sqlite` | Driver da base de dados: `sqlite` ou `postgres` |
| `DB_DSN` | `bonds.db` | String de conexão da base de dados |
| `APP_ENV` | `development` | Defina como `production` para uso em produção |

### Armazenamento e Pesquisa

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `STORAGE_UPLOAD_DIR` | `uploads` | Diretoria para ficheiros carregados |
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | Diretoria do índice de pesquisa de texto completo |
| `BACKUP_DIR` | `data/backups` | Diretoria para backups automáticos |

### Conexão com o Banco de Dados

**SQLite** (padrão, zero configuração):
```bash
DB_DRIVER=sqlite
DB_DSN=bonds.db
```

**PostgreSQL**:
```bash
DB_DRIVER=postgres
DB_DSN="host=localhost port=5432 user=bonds password=secret dbname=bonds sslmode=disable"
```

## Definições de Administração (Interface Web)

A maioria das definições da aplicação é definida através do painel de **Definições de Administração**, acessível a utilizadores com privilégios de administrador. Estas incluem:

- **Geral**: Nome da aplicação, URL pública, banner de aviso.
- **Autenticação**: Alternar login por palavra-passe, alternar registo.
- **JWT**: Expiração do token, janela de renovação.
- **SMTP**: Host do servidor de email, porta, credenciais opcionais, endereço do remetente. Deixe ambos utilizador e palavra-passe vazios para ignorar SMTP AUTH em relays não autenticados.
- **OAuth**: Credenciais de cliente OAuth do GitHub e Google.
- **OIDC**: Provedor OpenID Connect para SSO (Authentik, Keycloak, etc.).
- **WebAuthn**: Configuração da Parte Confiável para autenticação por chave de acesso.
- **Telegram**: Token do bot para notificações no Telegram.
- **Geocodificação**: Provedor e chave de API para geocodificação de endereços.
- **Armazenamento**: Tamanho máximo de carregamento para ficheiros e documentos (configurado na interface, não via variáveis de ambiente).
- **Backup**: Agendamento cron, período de retenção para backups automáticos.
- **Swagger**: Ativar ou desativar a interface de documentação da API independentemente do modo de depuração.

::: dica Migração de Variáveis de Ambiente
No primeiro arranque, Bonds semeia estas definições de administração a partir de variáveis de ambiente, se presentes. Depois disso, todas as alterações são feitas através do painel de administração. Variáveis de ambiente para estas definições são usadas apenas como valores de semente iniciais.
:::

## A encriptar Definições Sensíveis {#criptografando-configuracoes-sensiveis}

Por padrão, definições sensíveis do sistema (palavra-passe SMTP, segredos de cliente OAuth, chaves de API de geocodificação) são armazenadas como texto simples na base de dados. Qualquer pessoa que possa ler o ficheiro da base de dados ou um ficheiro de backup recupera todas as credenciais da implementação.

Defina `SETTINGS_ENC_KEY` para ativar a encriptação AES-256-GCM em repouso para estes valores:

```bash
# Gere uma chave aleatória uma vez e armazene-a junto com outros segredos
SETTINGS_ENC_KEY="$(openssl rand -hex 32)"
```

Comportamento:

- A chave **nunca é escrita na base de dados**, portanto um backup do BD roubado sozinho não pode recuperar texto simples.
- Linhas encriptadas são marcadas com o prefixo `enc:v1:`. Linhas já encriptadas são detetadas e ignoradas na re-encriptação.
- No arranque, quaisquer linhas de texto simples pré-existentes na lista de permissões de chave secreta são **automaticamente migradas** para texto cifrado (idempotente).
- Deixe a variável vazia para manter o comportamento legado de texto simples. Implementações de instância única não são forçadas a migrar.
- O endpoint **GET /api/admin/settings** do administrador sempre oculta chaves secretas como `***`. Enviar `***` no **PUT** mantém o valor existente intacto, para que interfaces de administração possam fazer edições não secretas de forma segura.

Atualmente encriptado em repouso quando a chave está definida:

| Campo | Armazenamento |
|-------|--------------|
| `system_settings.value` para `smtp.password`, `geocoding.api_key` e qualquer chave `secret.*` | AES-256-GCM |
| `oauth_providers.client_secret` (GitHub, Google, GitLab, Discord, OIDC) | AES-256-GCM |

::: warning Perdendo a chave
Se definir `SETTINGS_ENC_KEY` e depois perdê-la, os segredos encriptados são irrecuperáveis. Trate esta chave como `JWT_SECRET` e faça uma cópia de segurança dela fora do sistema.
:::

## Lista de Verificação para Produção

1. **Defina `JWT_SECRET`**: Use uma string forte e aleatória (32+ caracteres).
2. **Defina `SETTINGS_ENC_KEY`**: Recomendado para produção. Criptografa credenciais SMTP/OAuth/geocodificação em repouso.
3. **Defina `APP_ENV=production`**: Desativa funcionalidades de depuração.
4. **Defina `APP_URL`**: A sua URL pública, usada em emails e callbacks OAuth.
5. **Configure SMTP**: Necessário para notificações por email e convites.
6. **Use HTTPS**: Necessário para WebAuthn; recomendado para todas as implementações.
7. **Backup**: Configure backups automáticos através do painel de administração.

## Exemplo de Ambiente Docker

```yaml
services:
  bonds:
    image: ghcr.io/naiba/bonds:latest
    ports:
      - "8080:8080"
    environment:
      - JWT_SECRET=change-me-to-a-random-string
      - SETTINGS_ENC_KEY=change-me-to-another-random-string
      - APP_ENV=production
      - APP_URL=https://bonds.example.com
      - DB_DSN=/data/bonds.db
      - STORAGE_UPLOAD_DIR=/data/uploads
      - BLEVE_INDEX_PATH=/data/bonds.bleve
      - BACKUP_DIR=/data/backups
    volumes:
      - bonds-data:/data

volumes:
  bonds-data:
```
