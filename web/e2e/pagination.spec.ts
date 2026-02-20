import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function setupVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Pagination');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Test Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
  await expect(page.getByRole('heading', { name: 'Test Vault' })).toBeVisible({ timeout: 10000 });
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

async function createContactsViaAPI(
  page: import('@playwright/test').Page,
  vaultId: string,
  token: string,
  count: number,
) {
  for (let i = 1; i <= count; i++) {
    const name = `Contact${String(i).padStart(2, '0')}`;
    const resp = await page.request.post(`http://localhost:8080/api/vaults/${vaultId}/contacts`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { first_name: name, last_name: 'Test' },
    });
    expect(resp.ok()).toBeTruthy();
  }
}

async function goToContacts(page: import('@playwright/test').Page) {
  await page.getByText('View all contacts').click();
  await expect(page).toHaveURL(/\/contacts/, { timeout: 5000 });
  await page.waitForLoadState('networkidle');
}

test.describe('Contact List Pagination', () => {
  test('should show all contacts with pagination', async ({ page }) => {
    await setupVault(page, 'pag1');
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);

    await createContactsViaAPI(page, vaultId, token, 25);
    await goToContacts(page);

    await expect(page.getByText('25 contacts').first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-pagination')).toBeVisible({ timeout: 5000 });

    const rows = page.locator('.ant-table-tbody .ant-table-row');
    await expect(rows).toHaveCount(20, { timeout: 10000 });

    await expect(
      page.locator('.ant-pagination-item').filter({ hasText: '2' })
    ).toBeVisible({ timeout: 5000 });
  });

  test('should navigate to second page', async ({ page }) => {
    await setupVault(page, 'pag2');
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);

    await createContactsViaAPI(page, vaultId, token, 25);
    await goToContacts(page);

    await expect(page.getByText('25 contacts').first()).toBeVisible({ timeout: 10000 });

    const firstPageRows = page.locator('.ant-table-tbody .ant-table-row');
    await expect(firstPageRows).toHaveCount(20, { timeout: 10000 });
    const firstPageNames: string[] = [];
    const firstPageCount = await firstPageRows.count();
    for (let i = 0; i < firstPageCount; i++) {
      const text = await firstPageRows.nth(i).innerText();
      firstPageNames.push(text);
    }

    await page.locator('.ant-pagination-item').filter({ hasText: '2' }).click();
    await page.waitForLoadState('networkidle');

    const secondPageRows = page.locator('.ant-table-tbody .ant-table-row');
    await expect(secondPageRows).toHaveCount(5, { timeout: 10000 });

    const secondPageFirstRowText = await secondPageRows.first().innerText();
    expect(firstPageNames).not.toContain(secondPageFirstRowText);
  });

  test('should sort contacts by name', async ({ page }) => {
    await setupVault(page, 'pag3');
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);

    for (const name of ['Zara', 'Alice', 'Mike']) {
      const resp = await page.request.post(`http://localhost:8080/api/vaults/${vaultId}/contacts`, {
        headers: { Authorization: `Bearer ${token}` },
        data: { first_name: name, last_name: 'Sorttest' },
      });
      expect(resp.ok()).toBeTruthy();
    }

    await goToContacts(page);
    await expect(page.getByText('3 contacts').first()).toBeVisible({ timeout: 10000 });

    const rows = page.locator('.ant-table-tbody .ant-table-row');
    await expect(rows).toHaveCount(3, { timeout: 10000 });

    await expect(rows.nth(0)).toContainText('Alice');
    await expect(rows.nth(1)).toContainText('Mike');
    await expect(rows.nth(2)).toContainText('Zara');

    const sortSelect = page.locator('.ant-select').filter({ hasText: 'Name' }).first();
    await sortSelect.click();
    await page.locator('.ant-select-dropdown .ant-select-item-option-content').filter({ hasText: 'Last updated' }).click();
    await page.waitForLoadState('networkidle');

    const sortedRows = page.locator('.ant-table-tbody .ant-table-row');
    await expect(sortedRows).toHaveCount(3, { timeout: 10000 });

    const namesAfterSort: string[] = [];
    for (let i = 0; i < 3; i++) {
      namesAfterSort.push(await sortedRows.nth(i).innerText());
    }
    expect(namesAfterSort.some((text) => text.includes('Zara'))).toBeTruthy();
    expect(namesAfterSort.some((text) => text.includes('Alice'))).toBeTruthy();
    expect(namesAfterSort.some((text) => text.includes('Mike'))).toBeTruthy();
  });
});

// ---------- helpers for contact-level tests ----------

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

async function createNotesViaAPI(
  page: import('@playwright/test').Page,
  vaultId: string,
  contactId: string,
  token: string,
  count: number,
) {
  for (let i = 1; i <= count; i++) {
    const resp = await page.request.post(
      `http://localhost:8080/api/vaults/${vaultId}/contacts/${contactId}/notes`,
      {
        headers: { Authorization: `Bearer ${token}` },
        data: { title: `Note ${String(i).padStart(2, '0')}`, body: `Body of note ${i}` },
      },
    );
    expect(resp.ok()).toBeTruthy();
  }
}

async function createCallsViaAPI(
  page: import('@playwright/test').Page,
  vaultId: string,
  contactId: string,
  token: string,
  count: number,
) {
  for (let i = 1; i <= count; i++) {
    const resp = await page.request.post(
      `http://localhost:8080/api/vaults/${vaultId}/contacts/${contactId}/calls`,
      {
        headers: { Authorization: `Bearer ${token}` },
        data: {
          called_at: new Date(Date.now() - i * 60000).toISOString(),
          type: 'phone',
          who_initiated: 'me',
          duration: 60,
        },
      },
    );
    expect(resp.ok()).toBeTruthy();
  }
}

// ---------- Vault Feed ----------

test.describe('Vault Feed Pagination', () => {
  test('should load more feed items', async ({ page }) => {
    await setupVault(page, 'pag-feed');
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);

    // Create a contact (generates a feed entry)
    const contactId = await createContactViaAPI(page, vaultId, token, 'FeedTest', 'User');

    // Create 16 notes → each generates a feed entry
    await createNotesViaAPI(page, vaultId, contactId, token, 16);

    // Navigate to vault feed
    await page.goto(`/vaults/${vaultId}/feed`);
    await page.waitForLoadState('networkidle');

    // Feed page title should be visible
    await expect(page.getByText('Activity Feed').first()).toBeVisible({ timeout: 10000 });

    // Should have items rendered (List.Item elements)
    const items = page.locator('.ant-list-item');
    const initialCount = await items.count();
    expect(initialCount).toBeGreaterThan(0);

    // "Load more" button should be visible (15 per page, we have 17+ feed items)
    const loadMoreBtn = page.getByRole('button', { name: /Load more/i });
    await expect(loadMoreBtn).toBeVisible({ timeout: 5000 });

    // Click load more
    await loadMoreBtn.click();
    await page.waitForLoadState('networkidle');

    // After loading more, total items should increase
    await expect(async () => {
      const newCount = await items.count();
      expect(newCount).toBeGreaterThan(initialCount);
    }).toPass({ timeout: 10000 });
  });
});

// ---------- Notes Module ----------

test.describe('Notes Module Pagination', () => {
  test('should paginate notes', async ({ page }) => {
    await setupVault(page, 'pag-notes');
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);

    const contactId = await createContactViaAPI(page, vaultId, token, 'NotesTest', 'User');

    // Create 20 notes (page size is 15)
    await createNotesViaAPI(page, vaultId, contactId, token, 20);

    // Navigate to contact detail
    await page.goto(`/vaults/${vaultId}/contacts/${contactId}`);
    await page.waitForLoadState('networkidle');

    // Go to "Information" tab (exact match to avoid "Contact information")
    await page.getByRole('tab', { name: 'Information', exact: true }).click();
    await page.waitForLoadState('networkidle');

    // Find the Notes card by its header title to avoid matching cards containing "NotesTest"
    const notesCard = page.locator('.ant-card').filter({ has: page.locator('.ant-card-head-title', { hasText: 'Notes' }) });
    await expect(notesCard).toBeVisible({ timeout: 10000 });

    // Notes list items should be present
    const noteItems = notesCard.locator('.ant-list-item');
    await expect(noteItems.first()).toBeVisible({ timeout: 10000 });
    const firstPageCount = await noteItems.count();
    expect(firstPageCount).toBeGreaterThan(0);
    expect(firstPageCount).toBeLessThanOrEqual(15);

    // Ant Design Pagination should be visible (20 notes > 15 per page)
    const pagination = notesCard.locator('.ant-pagination');
    await expect(pagination).toBeVisible({ timeout: 5000 });

    // Page 2 should exist
    const page2 = pagination.locator('.ant-pagination-item').filter({ hasText: '2' });
    await expect(page2).toBeVisible({ timeout: 5000 });
  });
});

// ---------- Calls Module ----------

test.describe('Calls Module Pagination', () => {
  test('should load more calls', async ({ page }) => {
    await setupVault(page, 'pag-calls');
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);

    const contactId = await createContactViaAPI(page, vaultId, token, 'CallsTest', 'User');

    // Create 16 calls (page size is 15)
    await createCallsViaAPI(page, vaultId, contactId, token, 16);

    // Navigate to contact detail
    await page.goto(`/vaults/${vaultId}/contacts/${contactId}`);
    await page.waitForLoadState('networkidle');

    // Go to "Information" tab (exact match)
    await page.getByRole('tab', { name: 'Information', exact: true }).click();
    await page.waitForLoadState('networkidle');

    // Find the Calls card by its header title to avoid matching cards containing "CallsTest"
    const callsCard = page.locator('.ant-card').filter({ has: page.locator('.ant-card-head-title', { hasText: 'Calls' }) });
    await expect(callsCard).toBeVisible({ timeout: 10000 });

    // Call list items should be present
    const callItems = callsCard.locator('.ant-list-item');
    await expect(callItems.first()).toBeVisible({ timeout: 10000 });
    const initialCount = await callItems.count();
    expect(initialCount).toBeGreaterThan(0);

    // "Load more" button should be visible inside the calls card
    const loadMoreBtn = callsCard.getByRole('button', { name: /Load more/i });
    await expect(loadMoreBtn).toBeVisible({ timeout: 5000 });

    // Click load more
    await loadMoreBtn.click();
    await page.waitForLoadState('networkidle');

    // After loading more, total items should increase
    await expect(async () => {
      const newCount = await callItems.count();
      expect(newCount).toBeGreaterThan(initialCount);
    }).toPass({ timeout: 10000 });
  });
});

// ---------- Vault Files ----------

test.describe('Vault Files Page', () => {
  test('page loads correctly', async ({ page }) => {
    await setupVault(page, 'pag-files');
    const vaultId = await getVaultId(page);

    await page.goto(`/vaults/${vaultId}/files`);
    await page.waitForLoadState('networkidle');

    // Title "Files" should be visible
    await expect(page.getByText('Files').first()).toBeVisible({ timeout: 10000 });

    // Filter segmented control should be present
    await expect(page.locator('.ant-segmented')).toBeVisible({ timeout: 5000 });

    // Table should be present (even if empty)
    await expect(page.locator('.ant-table').first()).toBeVisible({ timeout: 5000 });
  });
});
