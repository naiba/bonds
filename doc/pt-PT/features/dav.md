# CardDAV / CalDAV

Bonds inclui um servidor CardDAV e CalDAV integrado. Isso permite sincronizar contactos e calendários com aplicaçãos externos como Apple Contacts, Thunderbird, GNOME Contacts e outros clientes compatíveis com DAV.

## Endpoints

| Protocolo | URL | Descrição |
|-----------|-----|-----------|
| CardDAV | `/dav/` | Sincronização de contactos |
| CalDAV | `/dav/` | Sincronização de calendário (datas importantes, tarefas) |
| Descoberta | `/.well-known/carddav` | Redirecionamento de descoberta automática |
| Descoberta | `/.well-known/caldav` | Redirecionamento de descoberta automática |

## Autenticação

Clientes DAV usam **HTTP Basic Auth** (não JWT), porque a maioria dos clientes DAV não suporta autenticação baseada em token.

| Cenário | Credenciais |
|---------|-------------|
| 2FA **desativado** | Email + palavra-passe, ou email + Token de Acesso Pessoal |
| 2FA **ativado** | Email + Token de Acesso Pessoal **apenas** (palavra-passe é bloqueada) |

### Tokens de Acesso Pessoal para Sincronização DAV

Tokens de Acesso Pessoal são altamente recomendados para clientes DAV mesmo quando a Autenticação de Dois Fatores está desativada.

- Pode gerar tokens em **Definições > Tokens de API**.
- Forneça uma descrição clara e um período de expiração opcional.
- Copie o token no momento da geração, pois ele é exibido apenas uma vez.
- Em seu cliente DAV, insira seu **endereço de email** como nome de utilizador e o token gerado (prefixado com `bonds_`) como palavra-passe.

::: warning
Quando ativa o 2FA, quaisquer clientes DAV usando a sua palavra-passe pararão de sincronizar. Atualize-os para usar um Token de Acesso Pessoal.
:::

## O Que é Sincronizado

### CardDAV (Contactos para vCard 3.0 compatível)

O servidor CardDAV usa vCard 3.0 em UTF-8 para maior compatibilidade com os Contactos da Apple e outros clientes DAV. As exportações manuais e através da API REST continuam a usar vCard 4.0.

| Campo Bonds | Propriedade vCard |
|-------------|-------------------|
| Primeiro + Último nome | `FN`, `N` |
| Números de telefone | `TEL` |
| Endereços de email | `EMAIL` |
| Endereços | `ADR` |

### CalDAV

| Entidade Bonds | Tipo iCal | Observações |
|----------------|-----------|-------------|
| Datas importantes | `VEVENT` | Com `RRULE=YEARLY` para datas recorrentes |
| Tarefas | `VTODO` | Datas de vencimento e status das tarefas |

## Assinaturas de Sincronização DAV

Além de expor Bonds como um servidor DAV, cada cofre pode assinar catálogos de endereços CardDAV externos a partir da página de **Sincronização DAV** do cofre.

- **Crie uma assinatura** com a URI do servidor remoto, nome de utilizador, palavra-passe, caminho opcional do catálogo de endereços, direção da sincronização e frequência.
- **Testar Conexão** verifica o servidor remoto e descobre catálogos de endereços disponíveis antes de salvar. Se um catálogo de endereços for encontrado, Bonds o seleciona automaticamente.
- **Direções de sincronização**: Apenas Importar importa contactos remotos para o cofre, Apenas Exportar envia alterações de contactos locais para o catálogo de endereços remoto, e Bidirecional faz ambos.
- **Agendamento e execuções manuais**: a frequência padrão é 180 minutos, com opções de 30 minutos a 24 horas. Use **Sincronizar Agora** para disparar uma execução imediata.
- **Registos de sincronização** registam eventos de criação, atualização, exclusão, envio, pulados e erros para cada assinatura.
- Palavra-passes remotas são encriptadas em repouso usando uma chave derivada de `JWT_SECRET`.

## Configuração de Clientes

### Apple Contacts / Calendar (macOS / iOS)

1. Vá para **Ajustes > Contas > Adicionar Conta > Outra**.
2. Escolha **Adicionar Conta CardDAV** ou **Adicionar Conta CalDAV**.
3. Insira:
   - Servidor: `https://bonds.example.com`
   - Nome de utilizador: seu email
   - Palavra-passe: a sua palavra-passe (se o 2FA estiver ativado, use um Token de Acesso Pessoal)

As URLs well-known (`/.well-known/carddav`, `/.well-known/caldav`) permitem descoberta automática.

### Thunderbird

1. Abra **Catálogo de Endereços > Novo > Catálogo de Endereços CardDAV**.
2. Insira a URL: `https://bonds.example.com/dav/`
3. Autentique-se com suas credenciais Bonds.

## EEtiquetas

Os objetos de endereço CardDAV utilizam uma ETag derivada do vCard serializado completo, pelo que alterações a números de telefone, endereços de correio eletrónico e outros dados relacionados acionam a sincronização. Os restantes recursos DAV continuam a utilizar o respetivo timestamp `UpdatedAt`. Os clientes utilizam estas ETags para detetar alterações e sincronizar de forma eficiente.
