# Import contacts from CSV

Bonds can import contacts from any comma-separated values (CSV) file — Google Contacts, Apple Contacts, Outlook, or any app that can export to CSV.

## How to import

1. Open your vault and go to **Vault Settings** (gear icon in the top navigation).
2. Click the **CSV Import** tab.
3. Drag and drop your CSV file onto the upload area, or click to browse. After uploading you will be taken to the column mapping screen before any data is imported.
4. For each contact field, select the CSV column that contains that data. Columns you do not want to import can be left as **— not mapped —**. Common column names are detected and mapped automatically.
5. Click **Import**. Bonds processes every row and shows a summary when finished.

## Supported fields

| Contact field | Notes |
|---------------|-------|
| First name | **Required.** Rows without a first name are skipped. |
| Last name | |
| Middle name | |
| Nickname | |
| Prefix | Mr., Dr., Prof., … |
| Suffix | Jr., Sr., MD, … |
| Gender | Matched to your account genders by name. Common values (Male/Female/Other and their translations) are recognised automatically. |
| Birthday | Multiple date formats are accepted — see below. |
| Email | Stored as a contact email address. |
| Phone | Stored as a contact phone number. |
| Company | Stored as a note on the contact ("Company: …"). Full company linking is not supported in CSV import. |
| Job title | |
| Tags | Comma-separated list of tag names inside the cell. Tags are created automatically if they do not exist. Example: `"Family, Friends"` |
| Groups | Comma-separated list of group names. **Groups must already exist in your vault** before importing. Example: `"Book club, Hiking"` |
| Notes | Free-text note attached to the contact. |
| Address — street | |
| Address — city | |
| Address — state / province | |
| Address — postal code | |
| Address — country | Imported as a "Home" address type. |

## Accepted birthday formats

| Format | Example |
|--------|---------|
| ISO 8601 | `1985-06-15` |
| European (DD/MM/YYYY) | `15/06/1985` |
| US (MM/DD/YYYY) | `06/15/1985` |
| With dashes | `15-06-1985` |
| Long form | `15 June 1985` or `June 15, 1985` |
| Short month | `15 Jun 1985` or `Jun 15, 1985` |

## Tags and groups with commas

Your CSV must quote any cell that contains a comma. Standard spreadsheet apps (Excel, Google Sheets, LibreOffice Calc) do this automatically when you export to CSV. Example row:

```
John,Doe,"Family, Friends","Book club"
```

## Column auto-detection

Bonds recognises common column names and maps them automatically. If your column headers use different names, you can adjust the mapping on the mapping screen.

Recognised names (case-insensitive, punctuation ignored):

| Field | Recognised headers |
|-------|--------------------|
| First name | First Name, FirstName, Given Name, Prénom |
| Last name | Last Name, LastName, Surname, Family Name, Nom |
| Email | Email, Email Address, Mail, Courriel |
| Phone | Phone, Phone Number, Mobile, Telephone, Tel |
| Birthday | Birthday, Birthdate, DOB, Date of Birth, Naissance |
| Company | Company, Organization, Organisation, Employer, Société |
| Tags | Tags, Labels, Categories |
| Groups | Groups, Groupes |

## Tips

- **Duplicate contacts are not detected.** Running the import twice will create duplicate contacts. Check your existing contacts before importing.
- **Imports are not reversible** from the UI. If you need to undo an import, restore a backup from **Vault Settings → Backups**.
- Large files (thousands of rows) may take a minute to process. Keep the page open until the result appears.
- **UTF-8 BOM** files (produced by Excel on Windows and some other apps) are handled automatically — the invisible byte-order mark is stripped before reading column headers.
