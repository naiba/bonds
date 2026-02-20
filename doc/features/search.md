# Full-text Search

Bonds includes built-in full-text search powered by [Bleve](https://blevesearch.com/), supporting both English and Chinese (CJK) text.

## What's Indexed

| Entity | Fields |
|--------|--------|
| **Contacts** | First name, last name, nickname |
| **Notes** | Title, body text |

The search index is updated incrementally — whenever you create, update, or delete a contact or note, the index is updated automatically.

## CJK Support

Bonds uses a CJK analyzer as the default analyzer, which means Chinese, Japanese, and Korean text is tokenized correctly. You can search for Chinese names or note content alongside English text seamlessly.

## Search Isolation

Search results are scoped to the current vault. You will only see results from contacts and notes within the vault you're currently viewing, regardless of what other vaults you have access to.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | Directory where the search index is stored |

The search index is stored on disk and persists across server restarts. If you delete the index directory, it will be rebuilt automatically on next startup.

## Fallback

If Bleve is not configured or fails to initialize, Bonds falls back to a no-op search engine. The application continues to work normally, but search will return no results.
