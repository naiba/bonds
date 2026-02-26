import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function registerAndLogin(page: import('@playwright/test').Page, prefix = 'vault') {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Vault');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
}

async function registerAndCreateVault(page: import('@playwright/test').Page, prefix: string) {
  await registerAndLogin(page, prefix);

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Test Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
}

function getVaultUrl(page: import('@playwright/test').Page): string {
  return page.url();
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
  await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 10000 });
  await expect(page.getByText(`${firstName} ${lastName}`).first()).toBeVisible({ timeout: 10000 });
}

test.describe('Vaults', () => {
  test('should show empty vault list after registration', async ({ page }) => {
    await registerAndLogin(page, 'vault-empty');
    await expect(page).toHaveURL(/\/vaults/);
  });

  test('should create a new vault', async ({ page }) => {
    await registerAndLogin(page, 'vault-create');
    await page.getByRole('button', { name: /new vault/i }).click();
    await page.getByPlaceholder(/e\.g\. family/i).fill('Personal');
    await page.getByPlaceholder(/what is this vault/i).fill('My personal contacts');
    await page.getByRole('button', { name: /create vault/i }).click();
    // Fix: nav now also shows vault name, so getByText matches 2 elements. Use heading role for precision.
    await expect(page.getByRole('heading', { name: 'Personal' })).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to vault detail', async ({ page }) => {
    await registerAndLogin(page, 'vault-detail');

    await page.getByRole('button', { name: /new vault/i }).click();
    await page.getByPlaceholder(/e\.g\. family/i).fill('Work');
    await page.getByPlaceholder(/what is this vault/i).fill('Work contacts');
    await page.getByRole('button', { name: /create vault/i }).click();
    await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });
  });
});

test.describe('Vault - Most Consulted', () => {
  test('most consulted contact link navigates correctly', async ({ page }) => {
    await registerAndCreateVault(page, 'vault-mc');
    const vaultUrl = getVaultUrl(page);

    // Create a contact
    await goToContacts(page);
    await createContact(page, 'MC', 'Test');

    // Extract the contact ID from the URL
    const contactUrl = page.url();
    const contactMatch = contactUrl.match(/\/contacts\/([a-f0-9-]+)$/);
    expect(contactMatch).toBeTruthy();
    const contactId = contactMatch![1];

    // Visit the contact detail page multiple times to increment view count
    for (let i = 0; i < 3; i++) {
      await page.goto(contactUrl);
      await page.waitForLoadState('networkidle');
      await expect(page.getByText('MC Test').first()).toBeVisible({ timeout: 10000 });
    }

    // Navigate back to vault dashboard
    await page.goto(vaultUrl);
    await page.waitForLoadState('networkidle');

    // Find the "Most Consulted" card
    const mostConsultedCard = page.locator('.ant-card').filter({ hasText: 'Most Consulted' });
    await expect(mostConsultedCard).toBeVisible({ timeout: 10000 });

    // Wait for the list to load and find the contact entry
    const contactEntry = mostConsultedCard.locator('.ant-list-item').filter({ hasText: 'MC Test' });
    await expect(contactEntry).toBeVisible({ timeout: 15000 });

    // Click the contact entry
    await contactEntry.click();

    // Verify navigation goes to a valid contact URL (NOT /contacts/undefined)
    await expect(page).toHaveURL(new RegExp(`/vaults/[a-f0-9-]+/contacts/${contactId}$`), { timeout: 10000 });

    // Verify the contact name is visible on the detail page
    await expect(page.getByText('MC Test').first()).toBeVisible({ timeout: 10000 });
  });
});
