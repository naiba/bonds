# Lembretes

Bonds tem um sistema de lembretes embutido que notifica você sobre datas e eventos importantes via e-mail ou canais compatíveis com Shoutrrr, como Telegram.

## Tipos

| Tipo | Comportamento |
|------|---------------|
| **Único** | Dispara uma vez na data/hora agendada, depois concluído |
| **Semanal** | Repete toda semana |
| **Mensal** | Repete todo mês |
| **Anual** | Repete todo ano (ideal para aniversários) |

## Como Funciona

1. **Crie um lembrete** em um contato — escolha a data, hora, tipo e um rótulo
2. **O agendador cron** é executado a cada minuto, procurando lembretes pendentes
3. **As notificações são enviadas** através dos canais de notificação configurados no horário de envio preferido de cada canal
4. **Para lembretes recorrentes**, a próxima ocorrência é agendada automaticamente com base no horário agendado anterior (não no horário atual, para evitar desvio)

## Canais de Notificação

Bonds suporta e-mail mais canais de notificação compatíveis com Shoutrrr:

### E-mail

Configure as configurações SMTP no painel de administração. Notificações por e-mail são ativadas por padrão quando um usuário registra — um canal de notificação é criado automaticamente com o endereço de e-mail do usuário.

### Shoutrrr / Telegram

Adicione uma URL Shoutrrr nas configurações de notificação do usuário, como uma URL do Telegram (`telegram://token@telegram?channels=123456`). Canais Shoutrrr ficam ativos imediatamente após a criação e podem usar qualquer serviço Shoutrrr suportado.

Cada canal tem um **horário de envio preferido**. Novos lembretes, preenchimento retroativo de lembretes existentes e reagendamentos de lembretes recorrentes usam esse horário local. Valores vazios ou inválidos usam `09:00` como fallback.

Veja [Notificações Shoutrrr / Telegram](/pt-BR/features/more#notificacoes-telegram) para detalhes de configuração.

## Confiabilidade do Canal

Cada canal de notificação rastreia um contador de falhas. Se um canal falhar **10 vezes consecutivas**, ele é automaticamente desativado para evitar spam. Você pode reativá-lo manualmente nas configurações do usuário após corrigir o problema subjacente.

## Histórico de Notificações

Cada tentativa de notificação é registrada em `UserNotificationSent`, incluindo:
- Status de entrega (sucesso/falha)
- Mensagem de erro (se falhou)
- Timestamp
