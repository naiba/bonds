import { test, expect } from '@playwright/test';

async function registerUser(page: import('@playwright/test').Page, email: string) {
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Test');
  await page.getByPlaceholder('Last name').fill('User');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
}

test.describe('Settings', () => {
  test('should show settings page', async ({ page }) => {
    const email = `settings-${Date.now()}@example.com`;
    await registerUser(page, email);

    await page.goto('/settings');
    await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to 2FA settings', async ({ page }) => {
    const email = `2fa-${Date.now()}@example.com`;
    await registerUser(page, email);

    await page.goto('/settings/2fa');
    await expect(page.getByRole('heading', { name: /two-factor/i })).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to invitations page', async ({ page }) => {
    const email = `invite-${Date.now()}@example.com`;
    await registerUser(page, email);

    await page.goto('/settings/invitations');
    await expect(page.getByRole('heading', { name: /invitations/i })).toBeVisible({ timeout: 10000 });
  });

  test('should show user account info', async ({ page }) => {
    const email = `acct-${Date.now()}@example.com`;
    await registerUser(page, email);

    await page.goto('/settings');
    await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('rowgroup').getByText('Test User')).toBeVisible();
    await expect(page.getByText(email)).toBeVisible();
  });
});
