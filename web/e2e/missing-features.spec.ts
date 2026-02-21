import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function registerUser(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('MissFeat');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });
}

async function setupVault(page: import('@playwright/test').Page, prefix: string) {
  await registerUser(page, prefix);

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('MF Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For missing features');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
  await expect(page.getByRole('heading', { name: 'MF Vault' })).toBeVisible({ timeout: 10000 });
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

test.describe('Tasks - Completed Tasks Toggle', () => {
  test('should show toggle button for completed tasks', async ({ page }) => {
    await setupVault(page, 'task-toggle');
    await goToContacts(page);
    await createContact(page, 'TaskToggle', 'User');

    await navigateToTab(page, 'Information', true);

    const tasksCard = page.locator('.ant-card').filter({ hasText: /^Tasks/ });
    await expect(tasksCard).toBeVisible({ timeout: 10000 });
    await expect(tasksCard.getByText(/show completed/i)).toBeVisible({ timeout: 10000 });
  });

  test('should create task, toggle it, and see it in completed section', async ({ page }) => {
    await setupVault(page, 'task-complete');
    await goToContacts(page);
    await createContact(page, 'TaskComplete', 'User');

    await navigateToTab(page, 'Information', true);

    const tasksCard = page.locator('.ant-card').filter({ hasText: /^Tasks/ });
    await expect(tasksCard).toBeVisible({ timeout: 10000 });

    await tasksCard.getByRole('button', { name: /add/i }).click();
    const input = tasksCard.getByPlaceholder(/task/i);
    await expect(input).toBeVisible({ timeout: 5000 });
    await input.fill('Complete me');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/tasks') && resp.request().method() === 'POST'
    );
    await tasksCard.getByRole('button', { name: /save|add|ok/i }).first().click();
    await createResp;
    await expect(tasksCard.getByText('Complete me')).toBeVisible({ timeout: 10000 });

    const toggleResp = page.waitForResponse(
      (resp) => resp.url().includes('/toggle') && resp.request().method() === 'PUT'
    );
    await tasksCard.locator('.ant-checkbox').first().click();
    await toggleResp;

    await tasksCard.getByText(/show completed/i).click();
    await expect(tasksCard.getByText(/hide completed/i)).toBeVisible({ timeout: 10000 });
    await expect(tasksCard.getByText('Complete me').first()).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Preferences - Extended Fields', () => {
  test('should show all preference fields including new ones', async ({ page }) => {
    await registerUser(page, 'pref-fields');

    await page.goto('/settings/preferences');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'Preferences' })).toBeVisible({ timeout: 10000 });

    await expect(page.getByText('Name display order')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Date format')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Timezone')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Language')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Default map service')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Distance unit')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Number format')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Show help sections')).toBeVisible({ timeout: 5000 });
  });

  test('should save preference changes', async ({ page }) => {
    await registerUser(page, 'pref-save');

    await page.goto('/settings/preferences');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'Preferences' })).toBeVisible({ timeout: 10000 });

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/preferences') && resp.request().method() === 'PUT'
    );
    await page.getByRole('button', { name: /save changes/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);
  });
});

test.describe('Notifications - Channel Verification', () => {
  test('should show seed notification channel with verification status', async ({ page }) => {
    await registerUser(page, 'notif-verify');

    await page.goto('/settings/notifications');
    await page.waitForLoadState('networkidle');

    await expect(
      page.getByRole('heading', { level: 4 }).getByText('Notifications')
    ).toBeVisible({ timeout: 10000 });

    const channelItem = page.locator('.ant-list-item').first();
    await expect(channelItem).toBeVisible({ timeout: 10000 });
    await expect(
      channelItem.getByText(/Verified|Unverified/)
    ).toBeVisible({ timeout: 5000 });
  });

  test('should show verify button for unverified channel and open modal', async ({ page }) => {
    await registerUser(page, 'notif-modal');

    await page.goto('/settings/notifications');
    await page.waitForLoadState('networkidle');

    await page.getByRole('button').filter({ has: page.locator('.anticon-plus') }).click();

    const createModal = page.locator('.ant-modal').filter({ hasText: /add notification/i });
    await expect(createModal).toBeVisible({ timeout: 5000 });

    await createModal.getByLabel(/label/i).fill('Test Email Channel');
    await createModal.getByLabel(/destination/i).fill('test-verify@example.com');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/notifications') && resp.request().method() === 'POST'
    );
    await createModal.getByRole('button', { name: /ok/i }).click();
    await createResp;

    await expect(createModal).not.toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('networkidle');

    const newChannel = page.locator('.ant-list-item').filter({ hasText: 'Test Email Channel' });
    await expect(newChannel).toBeVisible({ timeout: 10000 });
    await expect(newChannel.getByText('Unverified')).toBeVisible({ timeout: 5000 });

    const verifyBtn = newChannel.getByRole('button').filter({ has: page.locator('.anticon-safety-certificate') });
    await expect(verifyBtn).toBeVisible({ timeout: 5000 });

    await verifyBtn.click();

    const verifyModal = page.locator('.ant-modal').filter({ hasText: /verify notification/i });
    await expect(verifyModal).toBeVisible({ timeout: 5000 });
    await expect(verifyModal.getByPlaceholder(/verification token/i)).toBeVisible({ timeout: 3000 });

    await verifyModal.getByRole('button', { name: /cancel/i }).click();
    await expect(verifyModal).not.toBeVisible({ timeout: 5000 });
  });

  test('should create and delete a notification channel', async ({ page }) => {
    await registerUser(page, 'notif-crud');

    await page.goto('/settings/notifications');
    await page.waitForLoadState('networkidle');

    await page.getByRole('button').filter({ has: page.locator('.anticon-plus') }).click();

    const modal = page.locator('.ant-modal').filter({ hasText: /add notification/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.getByLabel(/label/i).fill('Delete Me Channel');
    await modal.getByLabel(/destination/i).fill('delete@example.com');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/notifications') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    await createResp;

    await expect(modal).not.toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('networkidle');

    const newChannel = page.locator('.ant-list-item').filter({ hasText: 'Delete Me Channel' });
    await expect(newChannel).toBeVisible({ timeout: 10000 });

    await newChannel.getByRole('button').filter({ has: page.locator('.anticon-delete') }).click();
    await page.locator('.ant-popconfirm').getByRole('button', { name: /ok|yes/i }).click();

    await expect(newChannel).not.toBeVisible({ timeout: 10000 });
  });

  test('should edit a notification channel', async ({ page }) => {
    await registerUser(page, 'notif-edit');

    await page.goto('/settings/notifications');
    await page.waitForLoadState('networkidle');

    await page.getByRole('button').filter({ has: page.locator('.anticon-plus') }).click();

    const createModal = page.locator('.ant-modal').filter({ hasText: /add notification/i });
    await expect(createModal).toBeVisible({ timeout: 5000 });

    await createModal.getByLabel(/label/i).fill('Edit Me Channel');
    await createModal.getByLabel(/destination/i).fill('edit-me@example.com');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/notifications') && resp.request().method() === 'POST'
    );
    await createModal.getByRole('button', { name: /ok/i }).click();
    await createResp;

    await expect(createModal).not.toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('networkidle');

    const channelItem = page.locator('.ant-list-item').filter({ hasText: 'Edit Me Channel' });
    await expect(channelItem).toBeVisible({ timeout: 10000 });

    await channelItem.getByRole('button').filter({ has: page.locator('.anticon-edit') }).click();

    const editModal = page.locator('.ant-modal').filter({ hasText: /edit notification/i });
    await expect(editModal).toBeVisible({ timeout: 5000 });

    const typeSelect = editModal.locator('.ant-select');
    await expect(typeSelect).toHaveClass(/ant-select-disabled/, { timeout: 3000 });

    await editModal.getByLabel(/label/i).clear();
    await editModal.getByLabel(/label/i).fill('Renamed Channel');

    const updateResp = page.waitForResponse(
      (resp) => resp.url().includes('/notifications/') && resp.request().method() === 'PUT' && !resp.url().includes('/toggle')
    );
    await editModal.getByRole('button', { name: /ok/i }).click();
    await updateResp;

    await expect(editModal).not.toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('networkidle');

    await expect(
      page.locator('.ant-list-item').filter({ hasText: 'Renamed Channel' })
    ).toBeVisible({ timeout: 10000 });
  });
});

