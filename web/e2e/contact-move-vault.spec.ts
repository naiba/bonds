import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import { apiUrl } from './api-base-url';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function registerAndCreateTwoVaults(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Test');
  await page.getByPlaceholder('Last name').fill('User');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  // Create first vault
  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Work Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('Work contacts');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
  const firstVaultUrl = page.url();
  const firstVaultID = firstVaultUrl.match(/\/vaults\/([a-f0-9-]{36})$/)?.[1];
  if (!firstVaultID) throw new Error(`Could not extract first vault ID from ${firstVaultUrl}`);

  // Go back to vault list and create second vault
  await page.goto('/vaults');
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Friends Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('Friends contacts');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
  const secondVaultUrl = page.url();
  const secondVaultID = secondVaultUrl.match(/\/vaults\/([a-f0-9-]{36})$/)?.[1];
  if (!secondVaultID) throw new Error(`Could not extract second vault ID from ${secondVaultUrl}`);

  return { firstVaultUrl, firstVaultID, secondVaultID };
}

// Move contact to another vault: vault selector in Move modal should load properly
test.describe('Contact - Move to another vault', () => {
  test.setTimeout(90000);

  test('should load vault options in the Move Contact modal selector', async ({ page }) => {
    const { firstVaultUrl } = await registerAndCreateTwoVaults(page, 'move-vault');

    // 1. Create a contact in the first vault
    await page.goto(firstVaultUrl + '/contacts');
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /add contact/i }).click();
    await page.getByPlaceholder('First name').fill('MoveMe');
    await page.getByPlaceholder('Last name').fill('Please');
    await page.getByRole('button', { name: /create contact/i }).click();
    await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 15000 });
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('MoveMe Please').first()).toBeVisible({ timeout: 10000 });

    // 2. Open the three-dot dropdown menu and click "Move"
    const moreBtn = page.locator('button').filter({ has: page.locator('[aria-label="more"]') });
    await expect(moreBtn).toBeVisible({ timeout: 5000 });
    await moreBtn.click();
    await page.getByText('Move', { exact: true }).click();

    // 3. The Move Contact modal should appear
    const modal = page.locator('.ant-modal').filter({ hasText: /Move Contact/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    // 4. Click the Select to open dropdown — vault options should load (not spin forever)
    const select = modal.locator('.ant-select');
    await select.click();
    // Bug #58: selector stays loading with spinner, vault names never appear.
    // The other vault ("Friends Vault") should appear as an option.
    await expect(page.locator('.ant-select-item-option').filter({ hasText: 'Friends Vault' }))
      .toBeVisible({ timeout: 10000 });
  });

  test('uploaded avatar should display after moving contact to another vault', async ({ page }) => {
    const { firstVaultUrl, firstVaultID, secondVaultID } = await registerAndCreateTwoVaults(page, 'move-avatar');

    await page.goto(firstVaultUrl + '/contacts');
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /add contact/i }).click();
    await page.getByPlaceholder('First name').fill('AvatarMove');
    await page.getByPlaceholder('Last name').fill('Target');
    await page.getByRole('button', { name: /create contact/i }).click();
    await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 15000 });
    await page.waitForLoadState('networkidle');

    const contactID = page.url().match(/\/contacts\/([a-f0-9-]+)$/)?.[1];
    if (!contactID) throw new Error(`Could not extract contact ID from ${page.url()}`);

    const tmpDir = fs.mkdtempSync('/tmp/avatar-move-e2e-');
    const imgPath = path.join(tmpDir, 'avatar.png');
    const pngData = Buffer.from(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==',
      'base64',
    );
    fs.writeFileSync(imgPath, pngData);

    try {
      await page.locator('input[type="file"]').first().setInputFiles(imgPath);
      await expect(page.getByText(/avatar.*updated/i)).toBeVisible({ timeout: 10000 });
      await expect(page.locator('img[alt="Avatar"]').first()).toBeVisible({ timeout: 10000 });

      const token = await page.evaluate(() => localStorage.getItem('token'));
      if (!token) throw new Error('No auth token found in localStorage');

      const sourceAvatarBeforeMove = await page.request.get(apiUrl(`/vaults/${firstVaultID}/contacts/${contactID}/avatar`), {
        headers: { Authorization: `Bearer ${token}` },
      });
      expect(sourceAvatarBeforeMove.status()).toBe(200);

      const moveResp = await page.request.post(apiUrl(`/vaults/${firstVaultID}/contacts/${contactID}/move`), {
        headers: { Authorization: `Bearer ${token}` },
        data: { target_vault_id: secondVaultID },
      });
      expect(moveResp.ok()).toBeTruthy();

      const sourceAvatarAfterMove = await page.request.get(apiUrl(`/vaults/${firstVaultID}/contacts/${contactID}/avatar`), {
        headers: { Authorization: `Bearer ${token}` },
      });
      expect(sourceAvatarAfterMove.status()).toBe(404);

      const targetAvatarAfterMove = await page.request.get(apiUrl(`/vaults/${secondVaultID}/contacts/${contactID}/avatar`), {
        headers: { Authorization: `Bearer ${token}` },
      });
      expect(targetAvatarAfterMove.status()).toBe(200);

      await page.goto(`/vaults/${secondVaultID}/contacts/${contactID}`);
      await page.waitForLoadState('networkidle');
      await expect(page.getByText('AvatarMove Target').first()).toBeVisible({ timeout: 10000 });
      await expect(page.locator('img[alt="Avatar"]').first()).toBeVisible({ timeout: 10000 });
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});
