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

async function setupVaultAndContact(page: import('@playwright/test').Page) {
  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Upload Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For upload testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');

  await page.getByText('View all contacts').click();
  await expect(page).toHaveURL(/\/contacts/, { timeout: 5000 });

  await page.getByRole('button', { name: /add contact/i }).click();
  await page.getByPlaceholder('First name').fill('Upload');
  await page.getByPlaceholder('Last name').fill('Test');
  await page.getByRole('button', { name: /create contact/i }).click();
  await expect(page.getByText('Upload Test')).toBeVisible({ timeout: 5000 });
}

test.describe('File Upload', () => {
  test('should show contact detail page with photo tab', async ({ page }) => {
    const email = `upload-${Date.now()}@example.com`;
    await registerUser(page, email);
    await setupVaultAndContact(page);

    // Contact detail page should show the contact name
    await expect(page.getByText('Upload Test')).toBeVisible({ timeout: 5000 });
  });

  test('should navigate to photos & docs tab', async ({ page }) => {
    const email = `upload2-${Date.now()}@example.com`;
    await registerUser(page, email);
    await setupVaultAndContact(page);

    const infoTab = page.getByRole('tab', { name: 'Information', exact: true });
    await expect(infoTab).toBeVisible({ timeout: 10000 });
    await infoTab.click();
    await expect(
      page.getByText('Photos', { exact: true }).first(),
    ).toBeVisible({ timeout: 5000 });
  });
});
