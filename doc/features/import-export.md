# Import / Export

Bonds supports vCard-based import/export and Monica 4.x JSON import, making it easy to migrate data from other applications or create backups.

## Monica 4.x Import

If you're migrating from Monica CRM (version 4.x), Bonds can import your complete data including contacts, notes, calls, tasks, relationships, photos, and more.

### How to Import

1. In Monica, go to **Settings > Export** and download your JSON export file
2. In Bonds, navigate to **Vault Settings > Monica Import** tab
3. Upload the JSON file — all your data will be imported automatically

### What Gets Imported

| Monica Entity | Bonds Mapping |
|--------------|---------------|
| Contacts | Contact (name, gender, nickname, job, company, starred, is_dead) |
| Tags | Labels |
| Birthdate / Deceased date | Important Dates |
| Notes | Notes |
| Calls | Calls |
| Tasks | Tasks |
| Reminders | Reminders |
| Addresses | Addresses |
| Contact fields (email, phone, etc.) | Contact Information |
| Pets | Pets |
| Gifts | Gifts |
| Debts | Loans |
| Relationships | Relationships (matched by type name) |
| Life events | Life Events + Timeline |
| Photos & Documents | Files (base64 decoded and stored) |
| Activities | Notes (degraded with type prefix) |
| Conversations | Notes (formatted chat log) |

### Duplicate Detection

Re-importing the same file is safe — contacts are matched by their original Monica UUID and skipped if already imported.

### Permissions

Only Vault **Managers** can perform imports.

### Monica v5 Users

Monica v5 removed JSON export — only VCard is available. If you're on v5:
- Use VCard import for contact basics (name, phone, email, address)
- For full migration: export JSON from 4.x **before** upgrading to v5

## vCard Export

### Single Contact

Export any contact as a vCard 4.0 file:

```
GET /api/vaults/:vault_id/contacts/:contact_id/vcard
```

Returns a `.vcf` file with the contact's name, phone numbers, email addresses, and physical addresses.

### Bulk Export

Export all contacts in a vault at once:

```
GET /api/vaults/:vault_id/contacts/export
```

Returns a single `.vcf` file containing all contacts in the vault.

## vCard Import

Import contacts from a `.vcf` file (supports both single and multi-contact files):

```
POST /api/vaults/:vault_id/contacts/import
```

Upload a `.vcf` file as multipart form data. Bonds parses the vCard and creates contacts with the following field mapping:

| vCard Property | Bonds Field |
|---------------|-------------|
| `FN` | First name + Last name |
| `N` | Structured name (family, given, etc.) |
| `TEL` | Phone contact information |
| `EMAIL` | Email contact information |
| `ADR` | Address |

## Tips

- **Migrating from other apps**: Most contact management apps (Google Contacts, Apple Contacts, Outlook, Monica) can export contacts as `.vcf` files. Export from there, then import into Bonds.
- **Large imports**: Bonds handles multi-contact `.vcf` files, so you can import hundreds of contacts at once.
- **What's not imported**: Fields that don't have a direct mapping (like social media profiles in vCard `X-` extensions) are skipped. You can add those manually after import.

## Backup & Restore

For full data backups (not just contacts), use the built-in backup system available in the admin panel. See [Admin & Settings](/features/admin) for details.
