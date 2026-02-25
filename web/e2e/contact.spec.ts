import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function setupVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Contact');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Test Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 20000 });
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
  await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 10000 });
  await expect(page.getByText(`${firstName} ${lastName}`).first()).toBeVisible({ timeout: 10000 });
}

test.describe('Contacts - CRUD', () => {
  test('should show empty contact list', async ({ page }) => {
    await setupVault(page, 'empty');
    await goToContacts(page);
    await expect(page.getByText('No contacts yet')).toBeVisible();
  });

  test('should create a new contact', async ({ page }) => {
    await setupVault(page, 'create');
    await goToContacts(page);
    await createContact(page, 'John', 'Doe');
  });

  test('should view contact detail', async ({ page }) => {
    await setupVault(page, 'detail');
    await goToContacts(page);
    await createContact(page, 'Jane', 'Smith');

    await page.getByText('Jane Smith').click();
    await expect(page.getByText('Jane Smith')).toBeVisible();
  });

  test('should search contacts', async ({ page }) => {
    await setupVault(page, 'search');
    await goToContacts(page);

    await createContact(page, 'Alice', 'Wonder');

    await page.getByRole('button', { name: /back to contacts/i }).click();
    await createContact(page, 'Bob', 'Builder');

    await page.getByRole('button', { name: /back to contacts/i }).click();
    await page.getByPlaceholder(/search/i).fill('Alice');
    await expect(page.getByText('Alice Wonder')).toBeVisible();
  });

  test('should edit a contact', async ({ page }) => {
    await setupVault(page, 'edit');
    await goToContacts(page);
    await createContact(page, 'EditTest', 'User');

    await page.getByRole('button', { name: 'Edit' }).first().click();

    const modal = page.locator('.ant-modal');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await expect(modal.getByText('Edit Contact')).toBeVisible();

    const firstNameInput = modal.locator('#first_name');
    await expect(firstNameInput).toBeVisible({ timeout: 5000 });

    await firstNameInput.clear();
    await firstNameInput.fill('UpdatedName');

    await modal.getByRole('button', { name: 'Save' }).click();

    await expect(modal).not.toBeVisible({ timeout: 15000 });
    await expect(page.getByText('UpdatedName User').first()).toBeVisible({ timeout: 10000 });
  });

  test('should show Move button on contact detail', async ({ page }) => {
    await setupVault(page, 'move');
    await goToContacts(page);
    await createContact(page, 'MoveTest', 'User');

    // Move button is now inside the More dropdown (icon-only button with MoreOutlined)
    // Find the more button by its icon aria-label
    const moreBtn = page.locator('button').filter({ has: page.locator('[aria-label="more"]') });
    await expect(moreBtn).toBeVisible({ timeout: 5000 });
    await moreBtn.click();
    await expect(page.getByText('Move', { exact: true })).toBeVisible({ timeout: 5000 });
  });
});

test.describe('Contacts - Feed Tab', () => {
  test('should show activity feed tab', async ({ page }) => {
    await setupVault(page, 'feed');
    await goToContacts(page);
    await createContact(page, 'FeedTest', 'User');

    const feedTab = page.getByRole('tab', { name: /feed/i });
    await feedTab.click();

    await expect(
      page.locator('.ant-card').filter({ hasText: /Activity Feed/i })
    ).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contacts - vCard', () => {
  test('should show import and export vCard buttons', async ({ page }) => {
    await setupVault(page, 'vcard');
    await goToContacts(page);

    await expect(page.getByRole('button', { name: 'Import vCard' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'Export All' })).toBeVisible({ timeout: 10000 });
  });
});
