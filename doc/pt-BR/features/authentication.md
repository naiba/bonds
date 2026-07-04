# Autenticação

Bonds suporta múltiplos métodos de autenticação, desde login tradicional com senha até chaves de acesso modernas.

## Senha, JWT e Verificação de E-mail

O fluxo de autenticação padrão é:

1. Registre-se com e-mail e senha.
2. **Verificação de E-mail**: Um link de verificação é enviado para o endereço de e-mail registrado. Os usuários devem clicar no link para verificar sua conta antes de acessar seus cofres.
   - Se o e-mail de verificação for perdido ou expirar, os usuários podem clicar no botão de reenvio na tela de verificação para disparar um novo link.
   - A verificação de e-mail também é necessária ao atualizar seu endereço de e-mail nas configurações da conta.
3. O login retorna um token JWT.
4. O token é enviado no cabeçalho `Authorization: Bearer <token>`.
5. Os tokens expiram após 24 horas (configurável via `JWT_EXPIRY_HRS`).
6. Os tokens podem ser renovados dentro de 7 dias (configurável via `JWT_REFRESH_HRS`).

## Autenticação de Dois Fatores (TOTP)

Adicione uma camada extra de segurança com 2FA baseado em TOTP:

1. **Ativar**: Vá em Configurações, Segurança, Ativar 2FA.
2. **Escanear código QR**: Use qualquer aplicativo autenticador (Google Authenticator, Authy, 1Password, etc.).
3. **Salvar códigos de recuperação**: 8 códigos de recuperação de uso único são gerados. Armazene-os com segurança.
4. **Confirmar**: Insira um código TOTP para ativar.

### Login com 2FA

Quando o 2FA está ativado, o login é um processo de duas etapas:

1. Insira e-mail e senha. O servidor retorna `requires_two_factor: true` e um token temporário.
2. Insira o código TOTP (ou um código de recuperação). O servidor retorna o JWT completo.

### Códigos de Recuperação

8 códigos aleatórios de 8 caracteres são gerados quando o 2FA é ativado. Cada código só pode ser usado uma vez. Use-os se você perder o acesso ao seu aplicativo autenticador.

### 2FA e Sincronização DAV

Quando o 2FA está ativado, a **autenticação DAV baseada em senha é bloqueada**. Clientes DAV (CardDAV/CalDAV) devem usar um [Token de Acesso Pessoal](/pt-BR/features/more#tokens-de-acesso-pessoal) em vez de sua senha. Isso segue o mesmo modelo de segurança do Google, Apple e Fastmail. Uma senha roubada sozinha não pode contornar o 2FA para acessar seus dados através de endpoints DAV.

Para configurar seu cliente DAV após ativar o 2FA:

1. Vá para **Configurações > Tokens de API** e crie um novo token.
2. Em seu cliente DAV, use seu **e-mail** como nome de usuário e o **token** (começando com `bonds_`) como senha.

## WebAuthn / FIDO2

Bonds suporta login sem senha via WebAuthn:

- **Chaves de hardware**: YubiKey, Titan Security Key, etc.
- **Biometria**: Touch ID, Face ID, Windows Hello.
- **Chaves de acesso**: iCloud Keychain, chaves de acesso do Android.

### Configuração

1. Vá para Configurações, Segurança, Registrar uma nova chave de acesso.
2. Siga o prompt do seu navegador para criar uma credencial.
3. A chave de acesso agora está vinculada à sua conta.

### Requisitos

- HTTPS é **obrigatório** (exceto `localhost` para desenvolvimento).
- Configure as configurações de WebAuthn no painel de administração:
  - **RP ID**: Seu domínio (ex.: `bonds.example.com`).
  - **RP Display Name**: Mostrado aos usuários durante a autenticação.
  - **RP Origins**: Origens permitidas (ex.: `https://bonds.example.com`).

### Atualizando da v0.12.5 ou anterior

Chaves de acesso registradas antes da persistência do indicador de backup não podem recuperar seu valor original de elegibilidade de backup, então chaves de acesso sincronizadas (iCloud Keychain, Google Password Manager, 1Password, etc.) continuarão falhando ao fazer login até serem re-registradas. Exclua e registre a chave de acesso novamente em Configurações, Segurança.

## Login OAuth

Bonds suporta login único via:

| Provedor | Configuração |
|----------|--------------|
| **GitHub** | ID e segredo do cliente OAuth App |
| **Google** | ID e segredo do cliente OAuth |

Configure estes no painel de administração. Quando ativados, botões "Entrar com GitHub" ou "Entrar com Google" aparecem na página de login.

Se o e-mail do OAuth corresponder a uma conta Bonds existente, as contas são vinculadas automaticamente.

### Fluxo de Callback OAuth

```
GET /api/auth/:provider -> Redirecionar para provedor OAuth
GET /api/auth/:provider/callback -> JWT -> Redirecionar para /auth/callback?token=xxx
```

## OIDC (OpenID Connect)

Bonds suporta provedores OIDC genéricos para SSO empresarial:

| Configuração | Descrição |
|-------------|-----------|
| **Client ID** | ID do cliente OIDC |
| **Client Secret** | Segredo do cliente OIDC |
| **Discovery URL** | URL de descoberta do provedor |
| **Display Name** | Rótulo do botão na página de login (padrão: "SSO") |

Configure a URL de callback do seu IdP como `https://{your-bonds-url}/api/auth/{provider-name}/callback`. O segmento `{provider-name}` é o **Name** / slug do provedor configurado no Bonds, e não o nome de exibição mostrado na página de login. Por exemplo, se a URL do seu Bonds for `https://bonds.domain.com` e o Name do provedor estiver configurado como `nextcloud-sso`, a URL de callback deve ser `https://bonds.domain.com/api/auth/nextcloud-sso/callback`.

Compatível com Authentik, Keycloak, Azure AD, Okta e outros provedores compatíveis com OIDC. Configure no painel de administração.
