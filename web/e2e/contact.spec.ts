import { test, expect } from '@playwright/test';

async function setupVault(page: import('@playwright/test').Page) {
  const email = `contact-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Contact');
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
  await expect(page.getByRole('heading', { name: 'Test Vault' })).toBeVisible({ timeout: 10000 });
}

async function goToContacts(page: import('@playwright/test').Page) {
  await page.getByText('View all contacts').click();
  await expect(page).toHaveURL(/\/contacts/, { timeout: 5000 });
}

async function createContact(page: import('@playwright/test').Page, firstName: string, lastName: string) {
  await page.getByRole('button', { name: /add contact/i }).click();
  await page.getByPlaceholder('First name').fill(firstName);
  await page.getByPlaceholder('Last name').fill(lastName);
  await page.getByRole('button', { name: /create contact/i }).click();
  // Wait for navigation to contact detail page
  await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 10000 });
  await expect(page.getByText(`${firstName} ${lastName}`)).toBeVisible({ timeout: 5000 });
}

test.describe('Contacts', () => {
  test('should show empty contact list', async ({ page }) => {
    await setupVault(page);
    await goToContacts(page);
    await expect(page.getByText('No contacts yet')).toBeVisible();
  });

  test('should create a new contact', async ({ page }) => {
    await setupVault(page);
    await goToContacts(page);
    await createContact(page, 'John', 'Doe');
  });

  test('should view contact detail', async ({ page }) => {
    await setupVault(page);
    await goToContacts(page);
    await createContact(page, 'Jane', 'Smith');

    await page.getByText('Jane Smith').click();
    await expect(page.getByText('Jane Smith')).toBeVisible();
  });

  test('should search contacts', async ({ page }) => {
    await setupVault(page);
    await goToContacts(page);

    await createContact(page, 'Alice', 'Wonder');

    await page.getByRole('button', { name: /back to contacts/i }).click();
    await createContact(page, 'Bob', 'Builder');

    await page.getByRole('button', { name: /back to contacts/i }).click();
    await page.getByPlaceholder(/search/i).fill('Alice');
    await expect(page.getByText('Alice Wonder')).toBeVisible();
  });
});
