# Import / Export

Bonds supports vCard-based import and export of contacts, making it easy to migrate data from other applications or create backups.

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
