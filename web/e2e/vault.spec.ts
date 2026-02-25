import { test, expect } from '@playwright/test';

async function registerAndLogin(page: import('@playwright/test').Page) {
  const email = `vault-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Vault');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
}

test.describe('Vaults', () => {
  test('should show empty vault list after registration', async ({ page }) => {
    await registerAndLogin(page);
    await expect(page).toHaveURL(/\/vaults/);
  });

  test('should create a new vault', async ({ page }) => {
    await registerAndLogin(page);
    await page.getByRole('button', { name: /new vault/i }).click();
    await page.getByPlaceholder(/e\.g\. family/i).fill('Personal');
    await page.getByPlaceholder(/what is this vault/i).fill('My personal contacts');
    await page.getByRole('button', { name: /create vault/i }).click();
    // Fix: nav now also shows vault name, so getByText matches 2 elements. Use heading role for precision.
    await expect(page.getByRole('heading', { name: 'Personal' })).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to vault detail', async ({ page }) => {
    await registerAndLogin(page);

    await page.getByRole('button', { name: /new vault/i }).click();
    await page.getByPlaceholder(/e\.g\. family/i).fill('Work');
    await page.getByPlaceholder(/what is this vault/i).fill('Work contacts');
    await page.getByRole('button', { name: /create vault/i }).click();
    await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });
  });
});
