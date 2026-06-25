# Arquivos e Avatares

Bonds suporta upload seguro de arquivos para mídia de contatos, documentos e arquivos em nível de cofre.

## Upload de Arquivos

Envie arquivos para contatos ou cofres usando os seguintes endpoints:

| Endpoint | Propósito |
|----------|-----------|
| `POST /api/vaults/:vault_id/files` | Enviar arquivos em nível de cofre |
| `POST .../contacts/:contact_id/photos` | Enviar fotos ou vídeos de mídia do contato |
| `POST .../contacts/:contact_id/documents` | Enviar documentos do contato |

### Tipos de Arquivo Suportados

Bonds aplica uma lista de permissões estrita de tipos MIME:

- **Imagens**: JPEG, PNG, GIF, WebP.
- **Vídeos**: MP4, WebM, Ogg, QuickTime.
- **Documentos**: PDF.

### Limites de Tamanho

O tamanho máximo de upload é gerenciado dinamicamente na interface web em **Admin > Configurações do Sistema > Armazenamento**. Não é configurado via variáveis de ambiente.

### Armazenamento

Arquivos enviados são armazenados em disco dentro do diretório configurado pela variável de ambiente `STORAGE_UPLOAD_DIR` (padrão: `uploads`), organizados por data:

```
{STORAGE_UPLOAD_DIR}/{yyyy/MM/dd}/{uuid}{ext}
```

Por exemplo: `uploads/2026/02/20/a1b2c3d4-e5f6.jpg`

## Avatares

Cada contato tem um avatar exibido em listas e páginas de detalhes.

### Avatares Gerados

Se nenhuma foto for enviada, Bonds gera um **avatar de iniciais** automaticamente:

- Extrai a primeira letra do primeiro e último nome.
- Escolhe uma cor de fundo deterministicamente a partir do hash MD5 do nome.
- Renderiza como uma imagem PNG usando o pacote `image` padrão do Go.

O mesmo nome sempre produz a mesma cor, proporcionando consistência visual.

### Avatares Personalizados

Envie uma foto para um contato para substituir o avatar gerado. A foto enviada é servida diretamente. Se removida, Bonds volta ao avatar de iniciais gerado.

### API do Avatar

```
GET /api/vaults/:vault_id/contacts/:contact_id/avatar
```

Retorna a foto enviada se disponível, caso contrário gera e retorna um avatar de iniciais.

## Download de Arquivos

```
GET /api/vaults/:vault_id/files/:id/download
```

Baixa um arquivo pelo seu ID. O acesso é restrito a usuários que têm acesso ao cofre.
