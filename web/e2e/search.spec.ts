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

async function setupVaultWithContacts(page: import('@playwright/test').Page) {
  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Search Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For search testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');
}

test.describe('Search', () => {
  test('search bar is visible when inside a vault', async ({ page }) => {
    const email = `search-${Date.now()}@example.com`;
    await registerUser(page, email);
    await setupVaultWithContacts(page);

    await page.getByText('View all contacts').click();
    await expect(page).toHaveURL(/\/contacts/, { timeout: 5000 });

    await expect(page.getByPlaceholder(/search/i)).toBeVisible({ timeout: 5000 });
  });

  test('search input accepts text', async ({ page }) => {
    const email = `search2-${Date.now()}@example.com`;
    await registerUser(page, email);
    await setupVaultWithContacts(page);

    await page.getByText('View all contacts').click();
    await expect(page).toHaveURL(/\/contacts/, { timeout: 5000 });

    const searchInput = page.getByPlaceholder(/search/i);
    await searchInput.fill('test query');
    await expect(searchInput).toHaveValue('test query');
  });
});
