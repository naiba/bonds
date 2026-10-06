# Contacts

Contacts are the core entity in Bonds. Each contact lives inside a [Vault](/features/vaults) and can hold rich, structured information about a person in your life.

## Contact Information

Each contact supports:

- **Names**: First name, last name, nickname, maiden name.
- **Contact methods**: Email addresses, phone numbers, social media links (12 built-in types).
- **Addresses**: Multiple addresses with types (Home, Work, etc.), with optional geocoding.
- **Company**: Job title and company association.
- **Gender & Pronouns**: Customizable gender and pronoun options.
- **Religion**: Optional religious affiliation.
- **Verification Flag**: Mark a contact as needing verification when their details might be out of date. This acts as a visual indicator to review and confirm their details periodically.

## Modules

Contact detail pages open in **View mode** by default. This mode shows populated, human-readable details first, including summary fields, quick facts, and notes, without exposing editing controls. Switch to **Edit mode** to access the full template tabs and module CRUD tools.

Contact edit pages are built from **modules**, which are configurable building blocks displayed on template pages. Default modules include:

| Module | Description |
|--------|-------------|
| Contact names | Name fields and nickname |
| Important dates | Birthdays, anniversaries, custom dates |
| Relationships | Family members, partners, friends |
| Notes | Free-text notes attached to a contact |
| Tasks | To-do items linked to a contact |
| Reminders | Scheduled notifications |
| Calls | Phone call logs with notes |
| Gifts | Gift ideas and tracking |
| Loans | Money or items lent and borrowed, including quantity, loan date, due date, and returned status |
| Activities | Important activities and milestones, including shared activities with other contacts in the same vault |
| Pets | Pets with names and categories (categories are managed at the account level) |
| Groups | Contact group membership |
| Documents | Uploaded files (PDF, images) |
| Media | Photo and video gallery |
| Posts | Journal-style entries with templates |
| Goals | Personal goals tracking |
| Feed | Activity timeline |

## Templates

Templates control the layout of contact detail pages. Each template has **pages** (tabs), and each page displays a set of modules. The default template includes:

1. **Contact information**: Avatar, names, important dates, gender, labels, company, religions.
2. **Feed**: Activity timeline.
3. **Social**: Relationships, pets, groups, addresses, contact methods.
4. **Activities**: Important activities and milestones.
5. **Goals**: Personal goals and progress.
6. **Information**: Documents, media, notes, reminders, loans, tasks, calls, posts.

Templates and module assignments are customizable through the [personalization settings](/features/admin#personalization).

## Labels

Labels are tags you can assign to contacts for organization and filtering. Create custom labels to categorize contacts however you like.

## Avatar

Each contact has an avatar. If no photo is uploaded, Bonds auto-generates an **initials avatar**, which is a colored circle with the contact's first and last initials. The color is deterministic (based on the name hash), so the same name always gets the same color.

## Pets

You can add pets to contacts. Pet categories are account-scoped, allowing you to select from a predefined list of categories such as Dog, Cat, or Bird. You can manage these categories in Settings, under the Personalize tab.

## Look Up Contacts by Identity

Integrations and AI assistants frequently need to answer the question: _"which contact owns this email address / phone number?"_ without paginating through every contact in a vault.

Bonds exposes a vault-scoped lookup endpoint:

```
GET /api/vaults/{vault_id}/contactInformation/by-identity?data=<value>&type_id=<n>
```

- `data` (required): the identity value to search for. Matching is **case-insensitive**.
- `type_id` (optional): restrict the match to a single `ContactInformationType` (e.g. only emails).

The response is an array of matches. Each match includes the `contact_id`, the contact's name, and the full `ContactInformationResponse` object. Searches are scoped to a single vault and require Viewer permission on it.

Example:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "$APP_URL/api/vaults/$VAULT_ID/contactInformation/by-identity?data=alice@example.com"
```

## Relationships

Define relationships between contacts, including parent, child, partner, friend, colleague, and more. Relationship types are organized into groups:

- **Love**: Partner, spouse, significant other.
- **Family**: Parent, child, sibling.
- **Friend**: Close friend, acquaintance.
- **Work**: Colleague, mentor, boss.

### Cross-Vault Relationships

Relationships can span across vaults. When adding a relationship, the contact selector shows contacts from **all vaults** you have access to, grouped by vault name.

- If you have **Editor** permission on the target vault, a **bidirectional** relationship is created automatically (both contacts see the relationship).
- If you only have **Viewer** permission, a **one-way** relationship is created, with a hint in the UI explaining why.
- Deleting a cross-vault relationship automatically cleans up the reverse record on the other side.

## Merge duplicate contacts

Select two to fifty contacts in the same vault, then choose **Merge contacts**. Review the selection and choose the contact to keep before confirming. Editors and managers can merge contacts, including archived contacts selected using the list filters.

The retained contact keeps its ID, visibility and layout. Empty profile fields are filled from the other contacts in selection order; the latest conversation date is kept. Original source profiles are saved as notes, so conflicting names and other profile values remain available. Notes, contact information, files, addresses, jobs, quick facts, tasks, activities, reminders and other records move to the retained contact. Shared memberships and identical relationships are combined, and incoming references point to the retained contact. Conflicting birthdays or deceased dates are retained as ordinary important dates with their reminders intact.

The other contacts are removed from the list. This operation cannot be undone. Merging does not automatically find duplicates or deduplicate independent notes, phone numbers or reminders. CardDAV changes use the existing asynchronous synchronization; a read-only remote address book or a later reimport can recreate a source contact.
