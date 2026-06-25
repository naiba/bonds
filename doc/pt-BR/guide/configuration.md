# Configuração

Bonds usa um modelo de configuração híbrido: as **configurações de infraestrutura** são definidas via variáveis de ambiente, enquanto as **configurações da aplicação** são gerenciadas com segurança através do painel de administração na interface web.

## Variáveis de Ambiente

Copie o arquivo de exemplo para começar:

```bash
cp server/.env.example server/.env
```

### Configurações Principais

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `DEBUG` | `false` | Ativar modo de depuração: logging de requisições, logging SQL, Swagger UI (padrão ativado) |
| `JWT_SECRET` | — | **Obrigatório em produção.** Chave de assinatura para tokens de autenticação |
| `SETTINGS_ENC_KEY` | _(vazio)_ | Opcional. Ativa criptografia AES-256-GCM em repouso para configurações sensíveis do sistema (senha SMTP, segredos de cliente OAuth, chaves de API de geocodificação). Veja [Criptografando Configurações Sensíveis](#criptografando-configuracoes-sensiveis) abaixo. |
| `SERVER_PORT` | `8080` | Porta em que o servidor escuta |
| `SERVER_HOST` | `0.0.0.0` | Endereço do host ao qual o servidor se vincula |
| `DB_DRIVER` | `sqlite` | Driver do banco de dados: `sqlite` ou `postgres` |
| `DB_DSN` | `bonds.db` | String de conexão do banco de dados |
| `APP_ENV` | `development` | Defina como `production` para uso em produção |

### Armazenamento e Busca

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `STORAGE_UPLOAD_DIR` | `uploads` | Diretório para arquivos enviados |
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | Diretório do índice de busca de texto completo |
| `BACKUP_DIR` | `data/backups` | Diretório para backups automáticos |

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

## Configurações de Administração (Interface Web)

A maioria das configurações da aplicação é definida através do painel de **Configurações de Administração**, acessível a usuários com privilégios de administrador. Estas incluem:

- **Geral**: Nome da aplicação, URL pública, banner de aviso.
- **Autenticação**: Alternar login por senha, alternar registro.
- **JWT**: Expiração do token, janela de renovação.
- **SMTP**: Host do servidor de e-mail, porta, credenciais opcionais, endereço do remetente. Deixe ambos usuário e senha vazios para pular SMTP AUTH em relays não autenticados.
- **OAuth**: Credenciais de cliente OAuth do GitHub e Google.
- **OIDC**: Provedor OpenID Connect para SSO (Authentik, Keycloak, etc.).
- **WebAuthn**: Configuração da Parte Confiável para autenticação por chave de acesso.
- **Telegram**: Token do bot para notificações no Telegram.
- **Geocodificação**: Provedor e chave de API para geocodificação de endereços.
- **Armazenamento**: Tamanho máximo de upload para arquivos e documentos (configurado na interface, não via variáveis de ambiente).
- **Backup**: Agendamento cron, período de retenção para backups automáticos.
- **Swagger**: Ativar ou desativar a interface de documentação da API independentemente do modo de depuração.

::: dica Migração de Variáveis de Ambiente
Na primeira inicialização, Bonds semeia estas configurações de administração a partir de variáveis de ambiente, se presentes. Depois disso, todas as alterações são feitas através do painel de administração. Variáveis de ambiente para estas configurações são usadas apenas como valores de semente iniciais.
:::

## Criptografando Configurações Sensíveis {#criptografando-configuracoes-sensiveis}

Por padrão, configurações sensíveis do sistema (senha SMTP, segredos de cliente OAuth, chaves de API de geocodificação) são armazenadas como texto simples no banco de dados. Qualquer pessoa que possa ler o arquivo do banco de dados ou um arquivo de backup recupera todas as credenciais da implantação.

Defina `SETTINGS_ENC_KEY` para ativar a criptografia AES-256-GCM em repouso para estes valores:

```bash
# Gere uma chave aleatória uma vez e armazene-a junto com outros segredos
SETTINGS_ENC_KEY="$(openssl rand -hex 32)"
```

Comportamento:

- A chave **nunca é escrita no banco de dados**, portanto um backup do BD roubado sozinho não pode recuperar texto simples.
- Linhas criptografadas são marcadas com o prefixo `enc:v1:`. Linhas já criptografadas são detectadas e ignoradas na re-criptografia.
- Na inicialização, quaisquer linhas de texto simples pré-existentes na lista de permissões de chave secreta são **automaticamente migradas** para texto cifrado (idempotente).
- Deixe a variável vazia para manter o comportamento legado de texto simples. Implantações de instância única não são forçadas a migrar.
- O endpoint **GET /api/admin/settings** do administrador sempre oculta chaves secretas como `***`. Enviar `***` no **PUT** mantém o valor existente intacto, para que interfaces de administração possam fazer edições não secretas de forma segura.

Atualmente criptografado em repouso quando a chave está definida:

| Campo | Armazenamento |
|-------|--------------|
| `system_settings.value` para `smtp.password`, `geocoding.api_key` e qualquer chave `secret.*` | AES-256-GCM |
| `oauth_providers.client_secret` (GitHub, Google, GitLab, Discord, OIDC) | AES-256-GCM |

::: warning Perdendo a chave
Se você definir `SETTINGS_ENC_KEY` e depois perdê-la, os segredos criptografados são irrecuperáveis. Trate esta chave como `JWT_SECRET` e faça backup dela fora do sistema.
:::

## Lista de Verificação para Produção

1. **Defina `JWT_SECRET`**: Use uma string forte e aleatória (32+ caracteres).
2. **Defina `SETTINGS_ENC_KEY`**: Recomendado para produção. Criptografa credenciais SMTP/OAuth/geocodificação em repouso.
3. **Defina `APP_ENV=production`**: Desativa funcionalidades de depuração.
4. **Defina `APP_URL`**: Sua URL pública, usada em e-mails e callbacks OAuth.
5. **Configure SMTP**: Necessário para notificações por e-mail e convites.
6. **Use HTTPS**: Necessário para WebAuthn; recomendado para todas as implantações.
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
