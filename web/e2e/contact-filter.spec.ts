import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function setupVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Filter');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Filter Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For filter testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
  await expect(page.getByRole('heading', { name: 'Filter Vault' })).toBeVisible({ timeout: 10000 });
}

async function getVaultId(page: import('@playwright/test').Page): Promise<string> {
  const url = page.url();
  const match = url.match(/\/vaults\/([a-f0-9-]{36})/);
  if (!match) throw new Error('Could not extract vault ID from URL: ' + url);
  return match[1];
}

async function getAuthToken(page: import('@playwright/test').Page): Promise<string> {
  const token = await page.evaluate(() => localStorage.getItem('token'));
  if (!token) throw new Error('No auth token found in localStorage');
  return token;
}

async function createContactViaAPI(
  page: import('@playwright/test').Page,
  vaultId: string,
  token: string,
  firstName: string,
  lastName: string,
): Promise<string> {
  const resp = await page.request.post(`http://localhost:8080/api/vaults/${vaultId}/contacts`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { first_name: firstName, last_name: lastName },
  });
  expect(resp.ok()).toBeTruthy();
  const body = await resp.json();
  return body.data.id;
}

async function toggleFavoriteViaAPI(
  page: import('@playwright/test').Page,
  vaultId: string,
  token: string,
  contactId: string,
) {
  const resp = await page.request.put(
    `http://localhost:8080/api/vaults/${vaultId}/contacts/${contactId}/favorite`,
    { headers: { Authorization: `Bearer ${token}` } },
  );
  expect(resp.ok()).toBeTruthy();
}

async function toggleArchiveViaAPI(
  page: import('@playwright/test').Page,
  vaultId: string,
  token: string,
  contactId: string,
) {
  const resp = await page.request.put(
    `http://localhost:8080/api/vaults/${vaultId}/contacts/${contactId}/archive`,
    { headers: { Authorization: `Bearer ${token}` } },
  );
  expect(resp.ok()).toBeTruthy();
}

async function goToContacts(page: import('@playwright/test').Page) {
  await page.getByText('View all contacts').click();
  await expect(page).toHaveURL(/\/contacts/, { timeout: 5000 });
  await page.waitForLoadState('networkidle');
  // Wait for the contact list to load — either table rows or the "contacts" total text
  await expect(page.locator('.ant-table')).toBeVisible({ timeout: 15000 });
}

async function selectStatusFilter(page: import('@playwright/test').Page, value: string) {
  // Use data-testid to find the status filter Select
  const statusSelect = page.getByTestId('status-filter');
  await statusSelect.click();
  // Wait for dropdown and select the option
  const dropdown = page.locator('.ant-select-dropdown:visible');
  await expect(dropdown).toBeVisible({ timeout: 5000 });
  await dropdown.locator('.ant-select-item-option-content').getByText(value, { exact: true }).click();
  await page.waitForLoadState('networkidle');
}

test.describe('Contact List - Status Filter', () => {
  test('default Active filter hides archived contacts', async ({ page }) => {
    await setupVault(page, 'filter-active');
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);

    // Create two contacts via API
    const aliceId = await createContactViaAPI(page, vaultId, token, 'Alice', 'Active');
    const bobId = await createContactViaAPI(page, vaultId, token, 'Bob', 'Archived');

    // Archive Bob
    await toggleArchiveViaAPI(page, vaultId, token, bobId);

    // Go to contact list — default filter is "Active"
    await goToContacts(page);

    // Alice should be visible, Bob should not
    await expect(page.getByText('Alice Active')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Bob Archived')).not.toBeVisible();

    // Switch to "Archived" filter
    await selectStatusFilter(page, 'Archived');

    // Now Bob should be visible, Alice should not
    await expect(page.getByText('Bob Archived')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Alice Active')).not.toBeVisible();
  });

  test('All filter shows both active and archived contacts', async ({ page }) => {
    await setupVault(page, 'filter-all');
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);

    const aliceId = await createContactViaAPI(page, vaultId, token, 'AllAlice', 'Smith');
    const bobId = await createContactViaAPI(page, vaultId, token, 'AllBob', 'Jones');

    // Archive Bob
    await toggleArchiveViaAPI(page, vaultId, token, bobId);

    await goToContacts(page);

    // Default: only Alice visible
    await expect(page.getByText('AllAlice Smith')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('AllBob Jones')).not.toBeVisible();

    // Switch to "All"
    await selectStatusFilter(page, 'All');

    // Both should be visible
    await expect(page.getByText('AllAlice Smith')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('AllBob Jones')).toBeVisible({ timeout: 10000 });
  });

  test('Favorites filter shows only favorited contacts', async ({ page }) => {
    await setupVault(page, 'filter-fav');
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);

    const aliceId = await createContactViaAPI(page, vaultId, token, 'FavAlice', 'Star');
    const bobId = await createContactViaAPI(page, vaultId, token, 'FavBob', 'Normal');

    // Favorite Alice
    await toggleFavoriteViaAPI(page, vaultId, token, aliceId);

    await goToContacts(page);

    // Default "Active" — both visible (favorites are still active)
    await expect(page.getByText('FavAlice Star')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('FavBob Normal')).toBeVisible({ timeout: 10000 });

    // Switch to "Favorites"
    await selectStatusFilter(page, 'Favorites');

    // Only Alice should be visible
    await expect(page.getByText('FavAlice Star')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('FavBob Normal')).not.toBeVisible();
  });

  test('favorites are sorted first in Active view', async ({ page }) => {
    await setupVault(page, 'filter-sort');
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);

    // Create contacts — "Zzz" sorts after "Aaa" alphabetically
    const zzzId = await createContactViaAPI(page, vaultId, token, 'Zzz', 'Later');
    const aaaId = await createContactViaAPI(page, vaultId, token, 'Aaa', 'First');

    // Favorite Zzz (would normally sort last alphabetically)
    await toggleFavoriteViaAPI(page, vaultId, token, zzzId);

    await goToContacts(page);

    // Both should be visible
    await expect(page.getByText('Zzz Later')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Aaa First')).toBeVisible({ timeout: 10000 });

    // Zzz (favorited) should appear before Aaa in the table
    const rows = page.locator('.ant-table-tbody .ant-table-row');
    const firstRowText = await rows.first().textContent();
    expect(firstRowText).toContain('Zzz Later');
  });

  test('archived contacts are excluded from Favorites filter', async ({ page }) => {
    await setupVault(page, 'filter-excl');
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);

    const aliceId = await createContactViaAPI(page, vaultId, token, 'ExclAlice', 'Fav');
    const bobId = await createContactViaAPI(page, vaultId, token, 'ExclBob', 'FavArch');

    // Favorite both
    await toggleFavoriteViaAPI(page, vaultId, token, aliceId);
    await toggleFavoriteViaAPI(page, vaultId, token, bobId);

    // Archive Bob (he is both favorited and archived)
    await toggleArchiveViaAPI(page, vaultId, token, bobId);

    await goToContacts(page);

    // Switch to "Favorites"
    await selectStatusFilter(page, 'Favorites');

    // Only Alice should be visible — Bob is archived, excluded from favorites filter
    await expect(page.getByText('ExclAlice Fav')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('ExclBob FavArch')).not.toBeVisible();
  });
});
