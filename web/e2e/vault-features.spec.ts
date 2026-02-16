import { test, expect } from '@playwright/test';

async function setupVault(page: import('@playwright/test').Page) {
  const email = `vf-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('VaultFeature');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Test Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');
}

function getVaultUrl(page: import('@playwright/test').Page): string {
  return page.url();
}

test.describe('Vault Features', () => {
  test('should navigate to vault settings page', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/settings');
    await expect(
      page.getByRole('heading').first()
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to companies page', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/companies');
    await expect(
      page.getByRole('heading', { level: 4 }).first()
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to reminders page', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/reminders');
    await expect(
      page.getByRole('heading', { level: 4 }).first()
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to life metrics page', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/life-metrics');
    await expect(
      page.getByRole('heading', { level: 4 }).first()
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 10000 });
  });

  test('should show vault settings tabs', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/settings');
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-tabs-tab')).toHaveCount(9, { timeout: 10000 });
  });

  test('should show companies page with table', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/companies');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 10000 });
  });

  test('should show create button on life metrics page', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/life-metrics');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('button').filter({ has: page.locator('.anticon-plus') })).toBeVisible({ timeout: 10000 });
  });

  test('should show back button on reminders page', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/reminders');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('.anticon-arrow-left').first()).toBeVisible({ timeout: 10000 });
  });
});
