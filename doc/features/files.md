# Files & Avatars

Bonds supports file uploads for contact photos, documents, and vault-level files.

## File Upload

Upload files to contacts or vaults:

| Endpoint | Purpose |
|----------|---------|
| `POST /api/vaults/:vault_id/files` | Upload vault-level files |
| `POST .../contacts/:contact_id/photos` | Upload contact photos |
| `POST .../contacts/:contact_id/documents` | Upload contact documents |

### Supported File Types

Bonds enforces a MIME type whitelist:

- **Images**: JPEG, PNG, GIF, WebP
- **Documents**: PDF
- Additional types may be supported in future releases

### Size Limits

The default maximum upload size is **10 MB**, configurable via `STORAGE_MAX_SIZE` environment variable (value in bytes).

### Storage

Uploaded files are stored on disk in a date-organized directory structure:

```
{STORAGE_UPLOAD_DIR}/{yyyy/MM/dd}/{uuid}{ext}
```

For example: `uploads/2026/02/20/a1b2c3d4-e5f6.jpg`

## Avatars

Every contact has an avatar displayed in lists and detail pages.

### Generated Avatars

If no photo is uploaded, Bonds generates an **initials avatar** automatically:

- Extracts the first letter of the first and last name
- Picks a background color deterministically from the name's MD5 hash
- Renders as a PNG image using Go's standard `image` package

The same name always produces the same color, providing visual consistency.

### Custom Avatars

Upload a photo to a contact to override the generated avatar. The uploaded photo is served directly; if removed, Bonds falls back to the generated initials avatar.

### Avatar API

```
GET /api/vaults/:vault_id/contacts/:contact_id/avatar
```

Returns the uploaded photo if available, otherwise generates and returns an initials avatar.

## File Download

```
GET /api/vaults/:vault_id/files/:id/download
```

Downloads a file by its ID. Access is restricted to users who have access to the vault.
