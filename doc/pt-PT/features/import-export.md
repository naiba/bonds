# Importação / Exportação

Bonds suporta importação/exportação baseada em vCard e importação JSON do Monica 4.x, facilitando a migração de dados de outros aplicaçãos ou a criação de backups.

## Importação Monica 4.x

Se está a migrar do Monica CRM (versão 4.x), Bonds pode importar os seus dados completos, incluindo contactos, notas, chamadas, tarefas, relacionamentos, fotos e muito mais.

### Como Importar

1. No Monica, vá em **Definições, Exportar** e transfira seu ficheiro de exportação JSON.
2. No Bonds, navegue até a aba **Definições do Cofre, Importação Monica**.
3. Envie o ficheiro JSON. Todos os seus dados serão importados automaticamente.

### O Que é Importado

| Entidade Monica | Mapeamento Bonds |
|----------------|------------------|
| Contactos | Contacto (nome, género, apelido, cargo, empresa, destacado, is_dead) |
| Etiquetas | Etiquetas |
| Data de nascimento / Falecimento | Datas Importantes |
| Notas | Notas |
| Chamadas | Chamadas |
| Tarefas | Tarefas |
| Lembretes | Lembretes |
| Endereços | Endereços |
| Campos de contacto (email, telefone, etc.) | Informações de Contacto |
| Animais de estimação | Animais de estimação |
| Presentes | Presentes |
| Dívidas | Empréstimos |
| Relacionamentos | Relacionamentos (correspondidos por nome do tipo) |
| Eventos da vida | Eventos da Vida + Linha do Tempo |
| Fotos e Documentos | Ficheiros (decodificados de base64 e armazenados) |
| Atividades | Notas (rebaixadas com prefixo de tipo) |
| Conversas | Notas (registo de chat formatado) |

### Tratamento Especializado de Importação Monica

Bonds implementa estratégias de fallback robustas para lidar com diferenças entre a estrutura de dados do Monica e do Bonds:
- **Fallback de Género**: Monica exporta géneros como campos de texto simples (ex.: nomes personalizados ou não-ingleses como "female", "male", "other", ou rótulos totalmente personalizados). Se um género importado não corresponder às opções padrão na base de dados Bonds, o importador mapeia-o graciosamente para uma opção personalizada, recorre a registos de género padrão ou cria uma entrada personalizada correspondente, garantindo que a importação termine com sucesso.
- **Categorias de Animais**: Registos de animais do Monica são mapeados para objetos de animal do Bonds. O importador lida com categorias mapeando-as para categorias padrão apropriadas (como cachorro, gato, pássaro, etc.) e recorre a um tipo padrão se a categoria exata não puder ser resolvida.

### Detecção de Duplicatas

Reimportar o mesmo ficheiro é seguro. Contactos são correspondidos por seu UUID original do Monica e ignorados se já importados.

### Permissões

Apenas **Gestores** do cofre podem realizar importações.

### Utilizadores do Monica v5

Monica v5 removeu a exportação JSON. Apenas VCard está disponível. Se está na v5:
- Use a importação VCard para o básico do contacto (nome, telefone, email, endereço).
- Para migração completa: exporte JSON da 4.x **antes** de atualizar para v5.

## Exportação vCard

### Contacto Único

Exporte qualquer contacto como um ficheiro vCard 4.0:

```
GET /api/vaults/:vault_id/contacts/:contact_id/vcard
```

Retorna um ficheiro `.vcf` com o nome, números de telefone, endereços de email e endereços físicos do contacto.

### Exportação em Massa

Exporte todos os contactos num cofre de uma vez:

```
GET /api/vaults/:vault_id/contacts/export
```

Retorna um único ficheiro `.vcf` contendo todos os contactos no cofre.

## Importação vCard

Importe contactos de um ficheiro `.vcf` (suporta ficheiros de contacto único e múltiplo):

```
POST /api/vaults/:vault_id/contacts/import
```

Envie um ficheiro `.vcf` como dados de formulário multipart. Bonds analisa o vCard e cria contactos com o seguinte mapeamento de campos:

| Propriedade vCard | Campo Bonds |
|-------------------|-------------|
| `FN` | Primeiro nome + Sobrenome |
| `N` | Nome estruturado (família, dado, etc.) |
| `TEL` | Informação de contacto telefone |
| `EMAIL` | Informação de contacto email |
| `ADR` | Endereço |

## Dicas

- **Migrando de outros aplicaçãos**: A maioria dos aplicaçãos de gestão de contactos (Google Contacts, Apple Contacts, Outlook, Monica) pode exportar contactos como ficheiros `.vcf`. Exporte de lá e depois importe para o Bonds.
- **Importações grandes**: Bonds lida com ficheiros `.vcf` de múltiplos contactos, então pode importar centenas de contactos de uma vez.
- **O que não é importado**: Campos que não têm um mapeamento direto (como perfis de redes sociais em extensões `X-` do vCard) são ignorados. Pode adicioná-los manualmente após a importação.

## Backup e Restauração

Para backups completos de dados (não apenas contactos), use o sistema de backup integrado disponível no painel de administração. Veja [Administração e Definições](/pt-PT/features/admin) para detalhes.
