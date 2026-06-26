#!/usr/bin/env node

/**
 * Cross-checks that en.json, zh.json, and es.json all expose the same key set.
 * Exits with code 1 if any locale is missing or has extra keys compared to en.
 *
 * Usage: node scripts/check-i18n-keys.mjs
 */

import { readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const localesDir = join(__dirname, '..', 'src', 'locales');

function flattenKeys(obj, prefix = '') {
  const keys = [];
  for (const [k, v] of Object.entries(obj)) {
    const path = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      keys.push(...flattenKeys(v, path));
    } else {
      keys.push(path);
    }
  }
  return keys;
}

// Add or remove from this list when adding a locale bundle. SUPPORTED_LANGUAGES
// in web/src/i18n.ts is the user-facing source of truth — keep these in sync.
const locales = ['en', 'zh', 'es', 'fr', 'de', 'pt-BR', 'pt-PT'];

const flatByLocale = {};
for (const locale of locales) {
  const obj = JSON.parse(readFileSync(join(localesDir, `${locale}.json`), 'utf-8'));
  flatByLocale[locale] = new Set(flattenKeys(obj));
}

const en = flatByLocale.en;
let failed = false;

for (const locale of locales) {
  if (locale === 'en') continue;
  const target = flatByLocale[locale];
  const missing = [...en].filter((k) => !target.has(k)).sort();
  const extra = [...target].filter((k) => !en.has(k)).sort();
  if (missing.length > 0) {
    console.error(`❌ ${missing.length} key(s) in en.json but missing in ${locale}.json:`);
    missing.forEach((k) => console.error(`  - ${k}`));
    failed = true;
  }
  if (extra.length > 0) {
    console.error(`❌ ${extra.length} key(s) in ${locale}.json but missing in en.json:`);
    extra.forEach((k) => console.error(`  - ${k}`));
    failed = true;
  }
}

if (failed) {
  process.exit(1);
} else {
  console.log(
    `✅ i18n keys consistent across ${locales.join(', ')}: ${en.size} keys each`,
  );
}
