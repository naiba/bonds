import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function setupVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Module');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Module Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For module testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
  await expect(page.getByRole('heading', { name: 'Module Vault' })).toBeVisible({ timeout: 10000 });
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

async function navigateToTab(page: import('@playwright/test').Page, tabName: string, exact = false) {
  const tab = page.getByRole('tab', { name: tabName, exact });
  await tab.click();
  await page.waitForLoadState('networkidle');
}

test.describe('Contact Modules - Notes', () => {
  test('should create a note', async ({ page }) => {
    await setupVault(page, 'note-create');
    await goToContacts(page);
    await createContact(page, 'NoteTest', 'User');

    await navigateToTab(page, 'Information', true);

    const notesCard = page.locator('.ant-card').filter({ hasText: /^Notes/ });
    await expect(notesCard).toBeVisible({ timeout: 10000 });
    await notesCard.getByRole('button', { name: /add/i }).click();

    await notesCard.getByPlaceholder(/title/i).fill('Test Note Title');
    await notesCard.locator('textarea').fill('This is a test note body');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/notes') && resp.request().method() === 'POST'
    );
    await notesCard.getByRole('button', { name: /save/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(notesCard.getByText('Test Note Title')).toBeVisible({ timeout: 10000 });
  });

  test('should delete a note', async ({ page }) => {
    await setupVault(page, 'note-delete');
    await goToContacts(page);
    await createContact(page, 'NoteDelTest', 'User');

    await navigateToTab(page, 'Information', true);

    const notesCard = page.locator('.ant-card').filter({ hasText: /^Notes/ });
    await expect(notesCard).toBeVisible({ timeout: 10000 });

    await notesCard.getByRole('button', { name: /add/i }).click();
    await notesCard.getByPlaceholder(/title/i).fill('Delete Me');
    await notesCard.locator('textarea').fill('Note to delete');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/notes') && resp.request().method() === 'POST'
    );
    await notesCard.getByRole('button', { name: /save/i }).click();
    await createResp;

    await expect(notesCard.getByText('Delete Me')).toBeVisible({ timeout: 10000 });

    await notesCard.getByRole('button', { name: /delete/i }).first().click();
    await page.locator('.ant-popconfirm').getByRole('button', { name: /ok|yes/i }).click();

    await expect(notesCard.getByText('Delete Me')).not.toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Tasks', () => {
  test('should create and toggle a task', async ({ page }) => {
    await setupVault(page, 'task-create');
    await goToContacts(page);
    await createContact(page, 'TaskTest', 'User');

    await navigateToTab(page, 'Information', true);

    const tasksCard = page.locator('.ant-card').filter({ hasText: /^Tasks/ });
    await expect(tasksCard).toBeVisible({ timeout: 10000 });

    await tasksCard.getByRole('button', { name: /add/i }).click();

    const input = tasksCard.getByPlaceholder(/task/i);
    await expect(input).toBeVisible({ timeout: 5000 });
    await input.fill('Buy groceries');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/tasks') && resp.request().method() === 'POST'
    );
    await tasksCard.getByRole('button', { name: /save|add|ok/i }).first().click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(tasksCard.getByText('Buy groceries')).toBeVisible({ timeout: 10000 });

    const toggleResp = page.waitForResponse(
      (resp) => resp.url().includes('/toggle') && resp.request().method() === 'PUT'
    );
    await tasksCard.locator('.ant-checkbox').first().click();
    await toggleResp;
  });
});

test.describe('Contact Modules - Reminders', () => {
  test('should create a reminder', async ({ page }) => {
    await setupVault(page, 'reminder');
    await goToContacts(page);
    await createContact(page, 'ReminderTest', 'User');

    await navigateToTab(page, 'Information', true);

    const remindersCard = page.locator('.ant-card').filter({ hasText: /Reminders/ });
    await expect(remindersCard).toBeVisible({ timeout: 10000 });
    await remindersCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal').filter({ hasText: /reminder/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.getByLabel(/label/i).fill('Birthday reminder');

    await modal.locator('.ant-picker').click();
    const dateCell = page.locator('.ant-picker-dropdown:visible .ant-picker-cell:not(.ant-picker-cell-disabled):not(.ant-picker-cell-today)').first();
    await dateCell.click();

    await page.locator('.ant-modal .ant-select').click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').first().click();

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/reminders') && resp.request().method() === 'POST'
    );
    await page.locator('.ant-modal-footer .ant-btn-primary').click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(remindersCard.getByText('Birthday reminder')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Addresses', () => {
  test('should create an address', async ({ page }) => {
    await setupVault(page, 'address');
    await goToContacts(page);
    await createContact(page, 'AddressTest', 'User');

    await navigateToTab(page, 'Social');

    const addressCard = page.locator('.ant-card').filter({ hasText: /Addresses/ });
    await expect(addressCard).toBeVisible({ timeout: 10000 });
    await addressCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal');
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.getByLabel(/address line 1/i).fill('123 Main St');
    await modal.getByLabel(/city/i).fill('San Francisco');
    await modal.getByLabel(/country/i).fill('USA');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/addresses') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(addressCard.getByText('123 Main St')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Pets', () => {
  test('should create a pet', async ({ page }) => {
    await setupVault(page, 'pet');
    await goToContacts(page);
    await createContact(page, 'PetTest', 'User');

    await navigateToTab(page, 'Social');

    const petsCard = page.locator('.ant-card').filter({ hasText: /Pets/ });
    await expect(petsCard).toBeVisible({ timeout: 10000 });
    await petsCard.getByRole('button', { name: /add/i }).click();

    await petsCard.getByPlaceholder(/name/i).fill('Buddy');
    await petsCard.getByPlaceholder(/category/i).fill('1');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/pets') && resp.request().method() === 'POST'
    );
    await petsCard.getByRole('button', { name: /save/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(petsCard.getByText('Buddy')).toBeVisible({ timeout: 10000 });
  });
});
