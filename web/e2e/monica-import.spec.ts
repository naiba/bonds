import { test, expect } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function registerAndCreateVault(page: import('@playwright/test').Page) {
  const email = uniqueEmail('monica-import');
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Monica');
  await page.getByPlaceholder('Last name').fill('Test');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Test Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For import testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
}

function getVaultId(page: import('@playwright/test').Page): string {
  const url = page.url();
  const match = url.match(/\/vaults\/([a-f0-9-]{36})/);
  return match ? match[1] : '';
}

test.describe('Monica Import', () => {
  test('should import Monica JSON and show results', async ({ page }) => {
    await registerAndCreateVault(page);
    const vaultId = getVaultId(page);

    // Navigate to vault settings
    await page.goto(`/vaults/${vaultId}/settings`);
    await page.waitForLoadState('networkidle');

    // Click Monica Import tab (left-positioned tabs)
    await page.getByRole('tab', { name: /Monica Import/i }).click();
    await expect(page.getByText('Import from Monica')).toBeVisible({ timeout: 5000 });

    // Upload fixture file via Upload.Dragger's hidden file input
    const fixturePath = path.join(__dirname, 'fixtures/monica_export.json');
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(fixturePath);

    // Wait for import to complete — success Alert appears
    await expect(page.locator('.ant-alert-success')).toBeVisible({ timeout: 30000 });
    await expect(page.getByText('Import completed')).toBeVisible({ timeout: 5000 });

    // Verify imported counts are displayed
    await expect(page.getByText(/Contacts.*[1-9]/)).toBeVisible({ timeout: 5000 });
    await expect(page.getByText(/Notes.*[1-9]/)).toBeVisible({ timeout: 5000 });

    // Navigate to contacts list and verify imported contacts appear
    await page.goto(`/vaults/${vaultId}/contacts`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('John').first()).toBeVisible({ timeout: 10000 });
  });

  test('should show error for invalid JSON file', async ({ page }) => {
    await registerAndCreateVault(page);
    const vaultId = getVaultId(page);

    await page.goto(`/vaults/${vaultId}/settings`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('tab', { name: /Monica Import/i }).click();
    await expect(page.getByText('Import from Monica')).toBeVisible({ timeout: 5000 });

    // Upload invalid JSON content
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles({
      name: 'invalid.json',
      mimeType: 'application/json',
      buffer: Buffer.from('{"version": "invalid"}'),
    });

    // Should show error alert
    await expect(page.locator('.ant-alert-error')).toBeVisible({ timeout: 15000 });
  });
});
