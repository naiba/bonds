# Contacts

Contacts are the core entity in Bonds. Each contact lives inside a [Vault](/features/vaults) and can hold rich, structured information about a person in your life.

## Contact Information

Each contact supports:

- **Names** — First name, last name, nickname, maiden name
- **Contact methods** — Email addresses, phone numbers, social media links (12 built-in types)
- **Addresses** — Multiple addresses with types (Home, Work, etc.), with optional geocoding
- **Company** — Job title and company association
- **Gender & Pronouns** — Customizable gender and pronoun options
- **Religion** — Optional religious affiliation

## Modules

Contact detail pages are built from **modules** — configurable building blocks displayed on template pages. Default modules include:

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
| Debts / Loans | Money lent or borrowed |
| Activities | Shared activities and events |
| Life events | Major milestones (graduation, marriage, etc.) |
| Pets | Pets with names and categories |
| Groups | Contact group membership |
| Documents | Uploaded files (PDF, images) |
| Photos | Photo gallery |
| Posts | Journal-style entries with templates |
| Goals | Personal goals tracking |
| Feed | Activity timeline |

## Templates

Templates control the layout of contact detail pages. Each template has **pages** (tabs), and each page displays a set of modules. The default template includes:

1. **Contact information** — Avatar, names, important dates, gender, labels, company, religions
2. **Feed** — Activity timeline
3. **Social** — Relationships, pets, groups, addresses, contact methods
4. **Life & goals** — Life events, goals
5. **Information** — Documents, photos, notes, reminders, loans, tasks, calls, posts

Templates and module assignments are customizable through the [personalization settings](/features/admin#personalization).

## Labels

Labels are tags you can assign to contacts for organization and filtering. Create custom labels to categorize contacts however you like.

## Avatar

Each contact has an avatar. If no photo is uploaded, Bonds auto-generates an **initials avatar** — a colored circle with the contact's first and last initials. The color is deterministic (based on the name hash), so the same name always gets the same color.

## Relationships

Define relationships between contacts — parent, child, partner, friend, colleague, and more. Relationship types are organized into groups:

- **Love** — Partner, spouse, significant other
- **Family** — Parent, child, sibling
- **Friend** — Close friend, acquaintance
- **Work** — Colleague, mentor, boss
