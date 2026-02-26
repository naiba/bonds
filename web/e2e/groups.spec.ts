import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function setupVault(page: import('@playwright/test').Page, prefix = 'groups') {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Groups');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Groups Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For groups testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
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

async function navigateToTab(page: import('@playwright/test').Page, tabName: string) {
  const tab = page.getByRole('tab', { name: tabName });
  await expect(tab).toBeVisible({ timeout: 15000 });
  await tab.click();
  await page.waitForLoadState('networkidle');
}

// Workaround: the NetworkGraph component crashes when the graph API returns null arrays.
// Intercept the graph/kinship endpoints to return empty arrays so the Social tab renders.
async function interceptGraphApi(page: import('@playwright/test').Page) {
  await page.route('**/relationships/graph', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { nodes: [], edges: [] } }),
    });
  });
  await page.route('**/relationships/kinship/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { degree: null, path: [] } }),
    });
  });
}

async function createGroup(page: import('@playwright/test').Page, vaultUrl: string, groupName: string) {
  await page.goto(vaultUrl + '/groups');
  await page.waitForLoadState('networkidle');

  await page.getByRole('button', { name: /new group/i }).first().click();
  const modal = page.getByRole('dialog');
  await expect(modal).toBeVisible({ timeout: 5000 });

  await modal.locator('input#name').fill(groupName);

  const responsePromise = page.waitForResponse(
    (resp) => resp.url().includes('/groups') && resp.request().method() === 'POST'
  );
  await modal.getByRole('button', { name: /ok/i }).click();
  const resp = await responsePromise;
  expect(resp.status()).toBeLessThan(400);

  await expect(page.locator('.ant-list').getByText(groupName)).toBeVisible({ timeout: 15000 });
}

function getVaultUrl(page: import('@playwright/test').Page): string {
  return page.url();
}

test.describe('Groups', () => {
  test('should navigate to groups page', async ({ page }) => {
    await setupVault(page, 'grp-nav');
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/groups');
    await expect(
      page.getByRole('heading', { level: 4 }).getByText('Groups')
    ).toBeVisible({ timeout: 10000 });
  });

  test('should show empty groups list', async ({ page }) => {
    await setupVault(page, 'grp-empty');
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/groups');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('No groups yet')).toBeVisible({ timeout: 10000 });
  });

  test('should show new group button', async ({ page }) => {
    await setupVault(page, 'grp-btn');
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/groups');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('button', { name: /new group/i }).first()).toBeVisible({ timeout: 10000 });
  });

  test('should create a new group', async ({ page }) => {
    await setupVault(page, 'grp-create');
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/groups');
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: /new group/i }).first().click();

    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await expect(modal.getByText('New Group')).toBeVisible();

    await modal.locator('input#name').fill('Family');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/groups') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(page.locator('.ant-list').getByText('Family')).toBeVisible({ timeout: 15000 });
  });

  test('should navigate to group detail after clicking a group', async ({ page }) => {
    await setupVault(page, 'grp-detail');
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/groups');
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: /new group/i }).first().click();
    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.locator('input#name').fill('Friends');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/groups') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    await responsePromise;

    await expect(page.locator('.ant-list').getByText('Friends')).toBeVisible({ timeout: 15000 });
    await page.locator('.ant-list').getByText('Friends').click();
    await expect(page).toHaveURL(/\/groups\/\d+$/, { timeout: 10000 });
  });
});

test.describe('Groups - Member Count', () => {
  test('member count updates after adding contact to group', async ({ page }) => {
    await setupVault(page, 'grp-memcount');
    const vaultUrl = getVaultUrl(page);

    // Create a group
    await createGroup(page, vaultUrl, 'Family');

    // Navigate back to vault, create a contact
    await page.goto(vaultUrl);
    await page.waitForLoadState('networkidle');
    await goToContacts(page);
    await createContact(page, 'Alice', 'Test');

    // Go to Social tab — wait for tabs to render
    await expect(page.locator('[role=tab]').first()).toBeVisible({ timeout: 15000 });
    await navigateToTab(page, 'Social');
    // Wait for tab content to load
    await page.waitForTimeout(1000);
    // Use h5 heading to precisely match the Groups card (avoid matching tab name)
    const groupsCard = page.locator('.ant-card').filter({
      has: page.locator('h5', { hasText: 'Groups' }),
    });
    await expect(groupsCard).toBeVisible({ timeout: 15000 });
    await groupsCard.getByRole('button', { name: /add/i }).click();

    // Wait for the modal with group selector
    const groupModal = page.locator('.ant-modal:visible');
    await expect(groupModal).toBeVisible({ timeout: 5000 });

    // Select the "Family" group from dropdown
    await groupModal.locator('.ant-select').click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: 'Family' }).click();

    // Save
    const addResp = page.waitForResponse(
      (resp) => resp.url().includes('/groups') && resp.request().method() === 'POST'
    );
    await groupModal.getByRole('button', { name: /save/i }).click();
    const resp = await addResp;
    expect(resp.status()).toBeLessThan(400);

    // Verify the tag appears
    await expect(groupsCard.locator('.ant-tag').filter({ hasText: 'Family' })).toBeVisible({ timeout: 10000 });

    // Navigate to groups list and verify member count
    await page.goto(vaultUrl + '/groups');
    await page.waitForLoadState('networkidle');

    const familyGroup = page.locator('.ant-list-item').filter({ hasText: 'Family' });
    await expect(familyGroup).toBeVisible({ timeout: 10000 });
    await expect(familyGroup.getByText(/1\s*member/i)).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Groups - Group Type', () => {
  test('create group form has group type selector', async ({ page }) => {
    await setupVault(page, 'grp-typsel');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/groups');
    await page.waitForLoadState('networkidle');

    // Open new group modal
    await page.getByRole('button', { name: /new group/i }).first().click();
    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Assert: the modal contains a "Group Type" label/field
    await expect(modal.locator('label').filter({ hasText: 'Group Type' })).toBeVisible({ timeout: 5000 });

    // Assert: there's a Select component for group type
    const groupTypeSelect = modal.locator('.ant-form-item').filter({ hasText: 'Group Type' }).locator('.ant-select');
    await expect(groupTypeSelect).toBeVisible({ timeout: 5000 });

    // Fill name only, leave group type empty, submit
    await modal.locator('input#name').fill('Test Group');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/groups') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    // Verify the group was created
    await expect(page.locator('.ant-list').getByText('Test Group')).toBeVisible({ timeout: 15000 });
  });
});
