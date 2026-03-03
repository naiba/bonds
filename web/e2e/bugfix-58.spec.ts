import { test, expect } from '@playwright/test';

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

  // Go back to vault list and create second vault
  await page.goto('/vaults');
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Friends Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('Friends contacts');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');

  return firstVaultUrl;
}

// =====================================================================================
// Issue #58: "Move contact to another vault" selector stays loading forever
// =====================================================================================
test.describe('Bug #58 - Move contact vault selector does not load', () => {
  test.setTimeout(90000);

  test('should load vault options in the Move Contact modal selector', async ({ page }) => {
    const firstVaultUrl = await registerAndCreateTwoVaults(page, 'bug58');

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
});
