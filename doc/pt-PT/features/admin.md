# Administração e Definições

Bonds inclui um painel de administração para configuração de todo o sistema e gestão de utilizadores.

## Tornando-se Administrador

O primeiro utilizador a registar-se numa nova instância Bonds recebe automaticamente privilégios de administrador da instância. Depois disso, administradores existentes podem promover outros utilizadores a partir da página de Gestão de Utilizadores no painel de administração.

## Painel de Administração

Utilizadores com privilégios de administrador podem acessar o painel de administração para configurar:

- **Definições do sistema**: Todas as definições em nível de aplicação armazenadas na base de dados.
- **Gestão de utilizadores**: Visualizar e gerir todos os utilizadores registados.
- **Backup**: Configurar backups automáticos e disparar backups manuais.

## Definições do Sistema

A maioria das definições é armazenada na base de dados. O painel de administração fornece uma interface web para configurar:

| Categoria | Definições |
|-----------|---------------|
| **Geral** | Nome da aplicação, URL, banner de aviso |
| **SMTP** | Host do servidor de email, porta, credenciais opcionais, endereço do remetente. Utilizador e palavra-passe vazios pulam SMTP AUTH. |
| **OAuth** | Credenciais de cliente OAuth do GitHub e Google |
| **OIDC** | Provedor OpenID Connect para SSO |
| **WebAuthn** | ID da Parte Confiável, nome de exibição, origens permitidas |
| **Telegram** | Token do bot para notificações |
| **Geocodificação** | Seleção de provedor e chave de API |
| **Armazenamento** | Tamanho máximo de carregamento (gerido aqui, não via variáveis de ambiente) |
| **Backup** | Agendamento cron, período de retenção |
| **Swagger** | Ativar ou desativar a interface de documentação da API |

::: dica
No primeiro arranque, estas definições são semeadas a partir de variáveis de ambiente, se presentes. Depois disso, as alterações são feitas exclusivamente através do painel de administração.
:::

### Encriptação em Repouso

Quando `SETTINGS_ENC_KEY` está configurada (veja [Configuração, A encriptar Definições Sensíveis](/pt-PT/guide/configuration#criptografando-configuracoes-sensiveis)), os seguintes campos são encriptados com AES-256-GCM na base de dados:

- `smtp.password`, `geocoding.api_key` e qualquer chave `secret.*` em **system_settings**
- `client_secret` para cada entrada em **oauth_providers** (GitHub, Google, GitLab, Discord, OIDC)

O endpoint **GET /admin/settings** do administrador sempre oculta valores secretos como `***` independentemente de a encriptação estar ativada. Navegadores de administração e logs de auditoria nunca veem credenciais em texto simples. Enviar `***` na atualização mantém o valor existente intacto, para que a interface possa fazer edições não secretas sem apagar credenciais.

Linhas de texto simples existentes são migradas de forma transparente na primeira arranque após a chave ser definida. A migração é idempotente e segura para reexecutar.

## Personalização {#personalizacao}

Proprietários de conta podem personalizar muitos aspectos do Bonds através das definições de personalização em `/api/settings/personalize/:entity`.

Várias tabelas de personalização suportam reordenação. Pode mover itens, páginas de modelo, seções de modelo de post, funções de grupo ou módulos dentro de páginas de modelo para cima ou para baixo usando os botões da interface.

| Entidade | O Que Você Pode Personalizar | Escopo | Reordenável |
|----------|------------------------------|--------|-------------|
| `genders` | Opções de género | Conta | Sim |
| `pronouns` | Opções de pronomes | Conta | Sim |
| `address-types` | Rótulos de tipo de endereço | Conta | Sim |
| `pet-categories` | Tipos de categoria de animal | Conta | Sim |
| `contact-info-types` | Tipos de informação de contacto | Conta | Sim |
| `relationship-types` | Definições de tipo de relacionamento | Conta | Sim (Sub-tipos) |
| `templates` | Modelos de página de contacto | Conta | Sim (Páginas) |
| `modules` | Configuração de módulo | Conta | Sim (Módulos de página) |
| `currencies` | Preferências de moeda | Conta | Não (Interruptores de ativação) |
| `religions` | Opções de religião | Conta | Sim |
| `call-reasons` | Categorias de motivo de chamada | Conta | Sim (Sub-motivos) |
| `gift-occasions` | Tipos de ocasião de presente | Conta | Sim |
| `gift-states` | Estados de rastreamento de presente | Conta | Sim |
| `post-templates` | Modelos de postagem de diário | Conta | Sim (Seções) |
| `group-types` | Tipos de grupo de contacto | Conta | Sim (Funções) |
| `task-statuses` | Status de tarefa personalizados | Conta | Sim |

Alguns itens incorporados, como tipos de informação de contacto de email e telefone, não podem ser excluídos.

## Convites de Utilizador

Convide outras pessoas para a sua conta via email:

1. Vá em Definições, Convites.
2. Insira o email do destinatário e escolha um nível de permissão.
3. Um email é enviado com um link de convite, válido por 7 dias.
4. O destinatário cria uma conta e é automaticamente associado à sua conta.

Níveis de permissão: **Gestor** (100), **Editor** (200), **Leitor** (300).

## Sistema de Backup

Bonds inclui um sistema de backup automático:

- **Backups agendados**: Configure um agendamento cron no painel de administração (formato de 6 campos com segundos, ex.: `0 0 2 * * *` para 2h diariamente).
- **Backups manuais**: Dispare um backup sob demanda a partir do painel de administração.
- **Retenção**: Backups antigos são limpos automaticamente após um número configurável de dias (padrão: 30).
- **Armazenamento**: Backups são armazenados na diretoria configurada por `BACKUP_DIR` (padrão: `data/backups`).

## Agendador Cron

Entrega de lembretes, sincronização CardDAV/CalDAV e backups automáticos são executados através de um agendador cron interno:

- Implementações SQLite de processo único funcionam imediatamente. Cada tarefa é executada no máximo uma vez por tick agendado.
- **Implementações PostgreSQL com múltiplas réplicas** (ex.: Kubernetes Deployments com `replicas: 2`, Docker Compose `deploy.replicas`, pods com balanceamento de carga) também são seguras. Cada tarefa é protegida por um `pg_try_advisory_xact_lock` mais um `UPDATE` condicional atômico na tabela `crons`. Duas réplicas disparando a mesma tarefa no mesmo instante não podem ambas executá-la.
- Réplicas que falham não podem travar uma tarefa. O lock consultivo é liberado automaticamente quando a transação que o mantém termina.

Nenhuma configuração é necessária. O agendador escolhe a estratégia correta com base no driver de base de dados ativo.
