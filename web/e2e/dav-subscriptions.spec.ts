import { test, expect } from '@playwright/test';

async function registerAndCreateVault(page: import('@playwright/test').Page) {
  const email = `dav-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('DAV');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('DAV Test');
  await page.getByPlaceholder(/what is this vault/i).fill('Test vault');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });

  const url = page.url();
  const vaultId = url.split('/vaults/')[1];
  return vaultId;
}

test.describe('DAV Subscriptions', () => {
  test('should show DAV subscriptions page', async ({ page }) => {
    const vaultId = await registerAndCreateVault(page);
    await page.goto(`/vaults/${vaultId}/dav-subscriptions`);
    await expect(page.getByRole('heading', { name: /CardDAV Subscriptions/i })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: /Add Subscription/i })).toBeVisible();
  });

  test('should open create subscription modal', async ({ page }) => {
    const vaultId = await registerAndCreateVault(page);
    await page.goto(`/vaults/${vaultId}/dav-subscriptions`);
    await expect(page.getByRole('heading', { name: /CardDAV Subscriptions/i })).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: /Add Subscription/i }).click();

    const modal = page.locator('.ant-modal');
    await expect(modal.getByText('Server URI')).toBeVisible({ timeout: 5000 });
    await expect(modal.getByText('Username')).toBeVisible();
    await expect(modal.getByText('Password')).toBeVisible();
    await expect(modal.getByText('Sync Direction')).toBeVisible();
    await expect(modal.getByText('Sync Frequency')).toBeVisible();
    await expect(modal.getByRole('button', { name: /Test Connection/i })).toBeVisible();
  });

  test('should create and display a subscription', async ({ page }) => {
    const vaultId = await registerAndCreateVault(page);
    await page.goto(`/vaults/${vaultId}/dav-subscriptions`);
    await expect(page.getByRole('heading', { name: /CardDAV Subscriptions/i })).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: /Add Subscription/i }).click();

    const modal = page.locator('.ant-modal');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.getByLabel('Server URI').fill('https://dav.example.com/contacts/');
    await modal.getByLabel('Username').fill('testuser');
    await modal.getByLabel('Password').fill('testpass');

    await modal.getByRole('button', { name: 'OK' }).click();

    await expect(page.getByRole('cell', { name: /dav\.example\.com/ })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('cell', { name: 'testuser' })).toBeVisible();
  });

  test('should delete a subscription', async ({ page }) => {
    const vaultId = await registerAndCreateVault(page);
    await page.goto(`/vaults/${vaultId}/dav-subscriptions`);
    await expect(page.getByRole('heading', { name: /CardDAV Subscriptions/i })).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: /Add Subscription/i }).click();
    const modal = page.locator('.ant-modal');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.getByLabel('Server URI').fill('https://delete-me.example.com/contacts/');
    await modal.getByLabel('Username').fill('deleteuser');
    await modal.getByLabel('Password').fill('deletepass');
    await modal.getByRole('button', { name: 'OK' }).click();
    // Wait for modal to close and table to update
    await expect(modal).not.toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('cell', { name: /delete-me\.example\.com/ })).toBeVisible({ timeout: 15000 });

    await page.getByRole('button', { name: 'delete' }).click();
    await page.getByRole('button', { name: 'OK' }).click();

    await expect(page.getByRole('cell', { name: /delete-me\.example\.com/ })).not.toBeVisible({ timeout: 10000 });
    await expect(page.getByText('No CardDAV subscriptions')).toBeVisible({ timeout: 10000 });
  });
});
