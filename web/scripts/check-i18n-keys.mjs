#!/usr/bin/env node

/**
 * Checks that en.json and zh.json have exactly the same set of keys.
 * Exits with code 1 if any keys are missing from either file.
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

const en = JSON.parse(readFileSync(join(localesDir, 'en.json'), 'utf-8'));
const zh = JSON.parse(readFileSync(join(localesDir, 'zh.json'), 'utf-8'));

const enKeys = new Set(flattenKeys(en));
const zhKeys = new Set(flattenKeys(zh));

const missingInZh = [...enKeys].filter((k) => !zhKeys.has(k)).sort();
const missingInEn = [...zhKeys].filter((k) => !enKeys.has(k)).sort();

let failed = false;

if (missingInZh.length > 0) {
  console.error(`❌ ${missingInZh.length} key(s) in en.json but missing in zh.json:`);
  missingInZh.forEach((k) => console.error(`  - ${k}`));
  failed = true;
}

if (missingInEn.length > 0) {
  console.error(`❌ ${missingInEn.length} key(s) in zh.json but missing in en.json:`);
  missingInEn.forEach((k) => console.error(`  - ${k}`));
  failed = true;
}

if (failed) {
  process.exit(1);
} else {
  console.log(`✅ i18n keys consistent: ${enKeys.size} keys in both en.json and zh.json`);
}
