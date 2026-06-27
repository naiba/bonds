# Mais Funcionalidades

## Log de Auditoria (Feed)

Bonds registra um feed de todas as alterações em contactos, fornecendo uma trilha de auditoria completa:

- Contacto criado, atualizado, excluído.
- Notas adicionadas, editadas, removidas.
- Lembretes criados, disparados.
- Tarefas, presentes, empréstimos, atividades e outras alterações de entidades.

O feed é acessível por cofre em `GET /api/vaults/:vault_id/feed`, mostrando quem fez qual alteração e quando.

## Tokens de Acesso Pessoal {#tokens-de-acesso-pessoal}

Bonds permite criar Tokens de Acesso Pessoal, também mostrados na interface como Tokens de API, para acesso seguro à API e sincronização DAV:

- **Localização**: Gira os seus tokens em **Definições > Tokens de API**.
- **Criação**: Especifique uma descrição personalizada e um período de expiração opcional.
- **Segurança**: O token é mostrado apenas uma vez no momento da criação. Certifique-se de copiá-lo imediatamente.
- **Uso**: Use o token como palavra-passe em integrações externas e clientes DAV. Quando a Autenticação de Dois Fatores está ativa, logins com palavra-passe padrão são desativados para sincronização CardDAV e CalDAV. Você deve usar um Token de Acesso Pessoal.
- **Agentes de IA**: Use um Token de Acesso Pessoal como token Bearer para o endpoint [`/mcp`](/pt-PT/features/ai-agents) integrado.
- **Formato**: Todos os Tokens de Acesso Pessoal são prefixados com `bonds_` para fácil identificação.

## Geocodificação

Bonds pode geocodificar endereços para obter coordenadas de latitude/longitude. Dois provedores são suportados:

| Provedor | Custo | Configuração |
|----------|-------|--------------|
| **Nominatim** | Gratuito (OSM) | Nenhuma chave de API necessária |
| **LocationIQ** | Freemium | Requer chave de API |

A geocodificação é executada de forma assíncrona quando um endereço é criado. Se falhar, o endereço ainda é salvo. As coordenadas simplesmente ficam vazias.

Configure o provedor e a chave de API no painel de administração.

## Notificações Shoutrrr / Telegram {#notificacoes-telegram}

Receba notificações de lembretes através de URLs compatíveis com Shoutrrr, incluindo Telegram:

### Configuração

1. Crie um bot do Telegram via [@BotFather](https://t.me/BotFather).
2. Obtenha o ID do chat de destino.
3. Em **Definições > Notificações**, adicione um canal de notificação Shoutrrr.
4. Use uma URL Shoutrrr do Telegram como `telegram://token@telegram?channels=123456`.
5. Escolha o horário de envio preferido para este canal.

### Obtendo Seu ID do Chat

Envie uma mensagem para seu bot e depois visite:
```
https://api.telegram.org/bot<SEU_TOKEN>/getUpdates
```
Procure pelo campo `chat.id` na resposta.

O horário de envio preferido é usado para novos lembretes, lembretes existentes preenchidos retroativamente e futuros agendamentos de lembretes recorrentes. Se deixado em branco ou inválido, Bonds usa `09:00` como fallback.

## Internacionalização (i18n)

Bonds suporta inglês, chinês, espanhol, francês, alemão e português tanto no frontend quanto no backend:

- **Frontend**: React i18next com ficheiros de localidade `en.json`, `zh.json`, `es.json`, `fr.json`, `de.json`, `pt-PT.json` e `pt-PT.json`.
- **Backend**: Ficheiros de localidade JSON incorporados, analisados a partir do cabeçalho `Accept-Language`, incluindo fallbacks regionais para português.

O idioma é detectado automaticamente a partir da configuração de idioma do navegador. Os utilizadores também podem alternar idiomas manualmente.

## Convites de Utilizador

Convide outros para entrar na sua conta com permissões controladas:

| Nível de Permissão | Valor | Acesso |
|--------------------|-------|--------|
| Gestor | 100 | Acesso total à gestão do cofre |
| Editor | 200 | Pode criar e editar contactos |
| Leitor | 300 | Acesso só de leitura |

Convites são enviados via email com um link de token único, válido por 7 dias.

## Moedas

Bonds inclui uma tabela de moedas abrangente (160+ moedas) para rastrear dados financeiros como empréstimos de dinheiro e presentes. As moedas são vinculadas a contas e podem ser geridas através das definições de personalização.

## Suporte a Calendários

Bonds suporta múltiplos sistemas de calendário:

- **Gregoriano**: Calendário padrão (padrão).
- **Lunar (Chinês)**: Calendário chinês tradicional baseado em `6tail/lunar-go`.

O sistema de calendário usa uma interface de conversor, tornando-o extensível para tipos adicionais de calendário.
