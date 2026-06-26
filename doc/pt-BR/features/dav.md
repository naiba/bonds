# CardDAV / CalDAV

Bonds inclui um servidor CardDAV e CalDAV integrado. Isso permite sincronizar contatos e calendários com aplicativos externos como Apple Contacts, Thunderbird, GNOME Contacts e outros clientes compatíveis com DAV.

## Endpoints

| Protocolo | URL | Descrição |
|-----------|-----|-----------|
| CardDAV | `/dav/` | Sincronização de contatos |
| CalDAV | `/dav/` | Sincronização de calendário (datas importantes, tarefas) |
| Descoberta | `/.well-known/carddav` | Redirecionamento de descoberta automática |
| Descoberta | `/.well-known/caldav` | Redirecionamento de descoberta automática |

## Autenticação

Clientes DAV usam **HTTP Basic Auth** (não JWT), porque a maioria dos clientes DAV não suporta autenticação baseada em token.

| Cenário | Credenciais |
|---------|-------------|
| 2FA **desativado** | E-mail + senha, ou e-mail + Token de Acesso Pessoal |
| 2FA **ativado** | E-mail + Token de Acesso Pessoal **apenas** (senha é bloqueada) |

### Tokens de Acesso Pessoal para Sincronização DAV

Tokens de Acesso Pessoal são altamente recomendados para clientes DAV mesmo quando a Autenticação de Dois Fatores está desativada.

- Você pode gerar tokens em **Configurações > Tokens de API**.
- Forneça uma descrição clara e um período de expiração opcional.
- Copie o token no momento da geração, pois ele é exibido apenas uma vez.
- Em seu cliente DAV, insira seu **endereço de e-mail** como nome de usuário e o token gerado (prefixado com `bonds_`) como senha.

::: warning
Quando você ativa o 2FA, quaisquer clientes DAV usando sua senha pararão de sincronizar. Atualize-os para usar um Token de Acesso Pessoal.
:::

## O Que é Sincronizado

### CardDAV (Contatos para vCard 4.0)

| Campo Bonds | Propriedade vCard |
|-------------|-------------------|
| Primeiro + Último nome | `FN`, `N` |
| Números de telefone | `TEL` |
| Endereços de e-mail | `EMAIL` |
| Endereços | `ADR` |

### CalDAV

| Entidade Bonds | Tipo iCal | Observações |
|----------------|-----------|-------------|
| Datas importantes | `VEVENT` | Com `RRULE=YEARLY` para datas recorrentes |
| Tarefas | `VTODO` | Datas de vencimento e status das tarefas |

## Assinaturas de Sincronização DAV

Além de expor Bonds como um servidor DAV, cada cofre pode assinar catálogos de endereços CardDAV externos a partir da página de **Sincronização DAV** do cofre.

- **Crie uma assinatura** com a URI do servidor remoto, nome de usuário, senha, caminho opcional do catálogo de endereços, direção da sincronização e frequência.
- **Testar Conexão** verifica o servidor remoto e descobre catálogos de endereços disponíveis antes de salvar. Se um catálogo de endereços for encontrado, Bonds o seleciona automaticamente.
- **Direções de sincronização**: Apenas Importar importa contatos remotos para o cofre, Apenas Exportar envia alterações de contatos locais para o catálogo de endereços remoto, e Bidirecional faz ambos.
- **Agendamento e execuções manuais**: a frequência padrão é 180 minutos, com opções de 30 minutos a 24 horas. Use **Sincronizar Agora** para disparar uma execução imediata.
- **Registros de sincronização** registram eventos de criação, atualização, exclusão, envio, pulados e erros para cada assinatura.
- Senhas remotas são criptografadas em repouso usando uma chave derivada de `JWT_SECRET`.

## Configuração de Clientes

### Apple Contacts / Calendar (macOS / iOS)

1. Vá para **Ajustes > Contas > Adicionar Conta > Outra**.
2. Escolha **Adicionar Conta CardDAV** ou **Adicionar Conta CalDAV**.
3. Insira:
   - Servidor: `https://seu-dominio-bonds.com`
   - Nome de usuário: seu e-mail
   - Senha: sua senha (se o 2FA estiver ativado, use um Token de Acesso Pessoal)

As URLs well-known (`/.well-known/carddav`, `/.well-known/caldav`) permitem descoberta automática.

### Thunderbird

1. Abra **Catálogo de Endereços > Novo > Catálogo de Endereços CardDAV**.
2. Insira a URL: `https://seu-dominio-bonds.com/dav/`
3. Autentique-se com suas credenciais Bonds.

## ETags

Bonds usa o timestamp `UpdatedAt` (época Unix) como ETag para todos os recursos DAV. Clientes usam ETags para detectar alterações e sincronizar eficientemente.
