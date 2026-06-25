# Importação / Exportação

Bonds suporta importação/exportação baseada em vCard e importação JSON do Monica 4.x, facilitando a migração de dados de outros aplicativos ou a criação de backups.

## Importação Monica 4.x

Se você está migrando do Monica CRM (versão 4.x), Bonds pode importar seus dados completos, incluindo contatos, notas, chamadas, tarefas, relacionamentos, fotos e muito mais.

### Como Importar

1. No Monica, vá em **Configurações, Exportar** e baixe seu arquivo de exportação JSON.
2. No Bonds, navegue até a aba **Configurações do Cofre, Importação Monica**.
3. Envie o arquivo JSON. Todos os seus dados serão importados automaticamente.

### O Que é Importado

| Entidade Monica | Mapeamento Bonds |
|----------------|------------------|
| Contatos | Contato (nome, gênero, apelido, cargo, empresa, destacado, is_dead) |
| Tags | Etiquetas |
| Data de nascimento / Falecimento | Datas Importantes |
| Notas | Notas |
| Chamadas | Chamadas |
| Tarefas | Tarefas |
| Lembretes | Lembretes |
| Endereços | Endereços |
| Campos de contato (e-mail, telefone, etc.) | Informações de Contato |
| Animais de estimação | Animais de estimação |
| Presentes | Presentes |
| Dívidas | Empréstimos |
| Relacionamentos | Relacionamentos (correspondidos por nome do tipo) |
| Eventos da vida | Eventos da Vida + Linha do Tempo |
| Fotos e Documentos | Arquivos (decodificados de base64 e armazenados) |
| Atividades | Notas (rebaixadas com prefixo de tipo) |
| Conversas | Notas (registro de chat formatado) |

### Tratamento Especializado de Importação Monica

Bonds implementa estratégias de fallback robustas para lidar com diferenças entre a estrutura de dados do Monica e do Bonds:
- **Fallback de Gênero**: Monica exporta gêneros como campos de texto simples (ex.: nomes personalizados ou não-ingleses como "female", "male", "other", ou rótulos totalmente personalizados). Se um gênero importado não corresponder às opções padrão no banco de dados Bonds, o importador mapeia-o graciosamente para uma opção personalizada, recorre a registros de gênero padrão ou cria uma entrada personalizada correspondente, garantindo que a importação termine com sucesso.
- **Categorias de Animais**: Registros de animais do Monica são mapeados para objetos de animal do Bonds. O importador lida com categorias mapeando-as para categorias padrão apropriadas (como cachorro, gato, pássaro, etc.) e recorre a um tipo padrão se a categoria exata não puder ser resolvida.

### Detecção de Duplicatas

Reimportar o mesmo arquivo é seguro. Contatos são correspondidos por seu UUID original do Monica e ignorados se já importados.

### Permissões

Apenas **Gerentes** do cofre podem realizar importações.

### Usuários do Monica v5

Monica v5 removeu a exportação JSON. Apenas VCard está disponível. Se você está na v5:
- Use a importação VCard para o básico do contato (nome, telefone, e-mail, endereço).
- Para migração completa: exporte JSON da 4.x **antes** de atualizar para v5.

## Exportação vCard

### Contato Único

Exporte qualquer contato como um arquivo vCard 4.0:

```
GET /api/vaults/:vault_id/contacts/:contact_id/vcard
```

Retorna um arquivo `.vcf` com o nome, números de telefone, endereços de e-mail e endereços físicos do contato.

### Exportação em Massa

Exporte todos os contatos em um cofre de uma vez:

```
GET /api/vaults/:vault_id/contacts/export
```

Retorna um único arquivo `.vcf` contendo todos os contatos no cofre.

## Importação vCard

Importe contatos de um arquivo `.vcf` (suporta arquivos de contato único e múltiplo):

```
POST /api/vaults/:vault_id/contacts/import
```

Envie um arquivo `.vcf` como dados de formulário multipart. Bonds analisa o vCard e cria contatos com o seguinte mapeamento de campos:

| Propriedade vCard | Campo Bonds |
|-------------------|-------------|
| `FN` | Primeiro nome + Sobrenome |
| `N` | Nome estruturado (família, dado, etc.) |
| `TEL` | Informação de contato telefone |
| `EMAIL` | Informação de contato e-mail |
| `ADR` | Endereço |

## Dicas

- **Migrando de outros aplicativos**: A maioria dos aplicativos de gerenciamento de contatos (Google Contacts, Apple Contacts, Outlook, Monica) pode exportar contatos como arquivos `.vcf`. Exporte de lá e depois importe para o Bonds.
- **Importações grandes**: Bonds lida com arquivos `.vcf` de múltiplos contatos, então você pode importar centenas de contatos de uma vez.
- **O que não é importado**: Campos que não têm um mapeamento direto (como perfis de redes sociais em extensões `X-` do vCard) são ignorados. Você pode adicioná-los manualmente após a importação.

## Backup e Restauração

Para backups completos de dados (não apenas contatos), use o sistema de backup integrado disponível no painel de administração. Veja [Administração e Configurações](/pt-BR/features/admin) para detalhes.
