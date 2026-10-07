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

Adding records and merging share the same contact transaction boundary. A completed addition is included in the next merge review; if the merge finishes first, additions to the deleted source are rejected. Refresh the retained contact before retrying. A file upload rejected for this reason is removed from storage. Activity-feed records commit with their actions. CSV and Monica imports report individual records that could not be attached; already imported vault files remain available in Files.

Select 2–50 contacts in one vault and choose **Merge contacts**. Choose the contact whose identity, visibility and layout should remain. The review shows every nonempty profile value, conflicting fields, important dates and counts of affected records. Choose one value for each conflict and acknowledge the consequences. Changes since review require another review. Editors and managers can merge; a protected contact can only be retained.

Empty fields are filled automatically. Company/position and first-met date components stay together. Notes, phone numbers, addresses, files, jobs, tasks, activities and reminders move in place; similar individual records are not deleted. Identical memberships and relationships are combined, and relationships within the selection are removed as self-links. Additional birthdays/death dates become ordinary dates with their reminder IDs and schedules intact. Conflicting group roles and introduced-by references unavailable in the current vault must be resolved before merging. Updating or deleting incoming relationships or introduced-by references requires Editor access to **every affected owning vault**. An unchanged incoming relationship to the retained contact does not require external editing permission.

Journal posts keep their section IDs, formatting and attachment references. Inline contact mentions are redirected to the retained contact in the same transaction as their associations; display names and surrounding text stay unchanged. Refresh a post opened before merging before saving it. Concurrent post edits or deletion are serialized with merging; changed sections require a fresh merge review. Legacy post associations pointing into another vault must be resolved in their owning journal before merging, since journal mentions only support contacts in that journal's vault.

No source-profile copies, backup notes or undo tables are created. Unselected single-value fields are discarded from the active profile after explicit confirmation. Sources retain the existing soft-delete lifecycle; the audit feed stores only the merge count. The operation cannot be undone. Monica imports skip deleted source UUIDs. File-based vCard imports remain append-only and may create duplicates on reimport without overwriting the merged contact.

Contacts linked to CardDAV cannot be merged, including paused and read-only subscriptions and legacy remote paths: a later pull could replace the combined fields, and deleting remote sources cannot be committed atomically with a local merge. Merge those contacts at their remote source instead. Active outgoing DAV synchronization must be paused before merging otherwise-unlinked local contacts. Raw unprocessed vCard profiles must be resolved first. These restrictions are checked again when confirming, without deleting remote records or dropping synchronization mappings.

For Bonds' own CardDAV address book, deleted source paths return not found on reads and reject subsequent writes. Conditional target writes enforce the current ETag, so stale conditional updates cannot overwrite a merge. Clients that intentionally write without a precondition retain the standard unconditional update behavior.

DAV updates reconcile individual records. Reordering phone parameters or editing an email preserves unrelated phone rows and their types. Adding, changing or removing the projected Gregorian birthday leaves other important dates intact; editing it retains its ID and linked reminders, reschedules pending notifications and keeps delivery history. Removing that birthday removes its associated reminders, as in the Bonds editor. Partial dates absent from the vCard are preserved. Editing a projected lunar birthday or separately synchronized calendar event requires the Bonds editor.

Unchanged addresses retain their IDs, coordinates and history even if the client reorders them or normalizes case/spacing. Multiple retained associations to a shared address remain separate without creating duplicate address records on DAV round trips. Other addresses can be added or edited independently. Removing a plain shared address only detaches this contact. Replacing an address with unrepresented details or history returns a conflict with instructions to edit it in Bonds. These rules apply equally to Bonds CardDAV and subscription pulls, including pause → merge unlinked contacts → resume. A rejected update is rolled back and logged; its sync checkpoint is not advanced, so it can be retried after resolving that specific record. Successful pulls in a partially failed batch do not count as local edits on retry; actual local edits retain their conflict protection. No remote deletion or sync identity reassignment is performed by a merge.

Only dates explicitly assigned the Birthdate type are projected as DAV birthdays. On upgrade, the first birthday with both month and day identified only by a Birthday/Birthdate label is classified once using the existing birthday type, preserving its row, label and reminders. Earlier incomplete dates are skipped; if none have both month and day, all remain ordinary dates without a DAV birthday. Ordinary dates do not become birthdays after primary deletion, repeated synchronization or restart; select the Birthdate type in Bonds to make an ordinary date the primary birthday.

Concurrent profile or detail edits cannot recreate merged sources or move records back to them. Refresh the surviving profile if an edit reports that its original record is unavailable. Incoming activity payer references require Editor access to the activity's owning vault, including historical references left after a contact move.

Addresses retain each residence period separately, even when periods share an address. API address responses include `contact_address_id`; pass it as the `contact_address_id` query parameter when updating or deleting a specific period. Existing address-ID calls continue to work for a single period and return 409 for an ambiguous selection. Deletion removes only that period; the shared address is removed when no periods reference it. Address text remains shared, while residence dates and past/current status belong to each period.
