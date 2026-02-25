import { test, expect } from '@playwright/test';

async function setupVault(page: import('@playwright/test').Page) {
  const email = `groups-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Groups');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Groups Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For groups testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');
}

function getVaultUrl(page: import('@playwright/test').Page): string {
  return page.url();
}

test.describe('Groups', () => {
  test('should navigate to groups page', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/groups');
    await expect(
      page.getByRole('heading', { level: 4 }).getByText('Groups')
    ).toBeVisible({ timeout: 10000 });
  });

  test('should show empty groups list', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/groups');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('No groups yet')).toBeVisible({ timeout: 10000 });
  });

  test('should show new group button', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/groups');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('button', { name: /new group/i }).first()).toBeVisible({ timeout: 10000 });
  });

  test('should create a new group', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/groups');
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: /new group/i }).first().click();

    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await expect(modal.getByText('New Group')).toBeVisible();

    await modal.locator('input#name').fill('Family');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/groups') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(page.locator('.ant-list').getByText('Family')).toBeVisible({ timeout: 15000 });
  });

  test('should navigate to group detail after clicking a group', async ({ page }) => {
    await setupVault(page);
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/groups');
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: /new group/i }).first().click();
    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.locator('input#name').fill('Friends');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/groups') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    await responsePromise;

    await expect(page.locator('.ant-list').getByText('Friends')).toBeVisible({ timeout: 15000 });
    await page.locator('.ant-list').getByText('Friends').click();
    await expect(page).toHaveURL(/\/groups\/\d+$/, { timeout: 10000 });
  });
});
