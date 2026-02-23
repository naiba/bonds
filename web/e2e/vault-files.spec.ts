import { test, expect } from '@playwright/test';

async function setupVault(page: import('@playwright/test').Page) {
  const email = `vfiles-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('VaultFiles');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Files Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For files testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');
}

function getVaultUrl(page: import('@playwright/test').Page): string {
  return page.url();
}

test.describe('Vault Files', () => {
  test('should navigate to files page', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/files');
    await expect(
      page.getByRole('heading', { level: 4 }).getByText('Files')
    ).toBeVisible({ timeout: 10000 });
  });

  test('should show filter segmented control', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/files');
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('All', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Photos', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Documents', { exact: true })).toBeVisible({ timeout: 10000 });
  });

  test('should switch between filter tabs', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/files');
    await page.waitForLoadState('networkidle');

    await page.getByText('Photos', { exact: true }).click();
    await expect(page.locator('.ant-segmented-item-selected').getByText('Photos')).toBeVisible({ timeout: 5000 });

    await page.getByText('Documents', { exact: true }).click();
    await expect(page.locator('.ant-segmented-item-selected').getByText('Documents')).toBeVisible({ timeout: 5000 });

    await page.getByText('All', { exact: true }).click();
    await expect(page.locator('.ant-segmented-item-selected').getByText('All')).toBeVisible({ timeout: 5000 });
  });

  test('should show empty files table', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/files');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('No files')).toBeVisible({ timeout: 10000 });
  });

  test('should show upload button', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/files');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('button', { name: /upload file/i })).toBeVisible({ timeout: 10000 });
  });

  test('should show back button', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/files');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('.anticon-arrow-left').first()).toBeVisible({ timeout: 10000 });
  });
});
