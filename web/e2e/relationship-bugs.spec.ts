import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function setupVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Rel');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Rel Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For relationship testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
  await expect(page.getByRole('heading', { name: 'Rel Vault' })).toBeVisible({ timeout: 10000 });
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

/**
 * Navigate back to contact list and wait for table rows to be rendered.
 * This prevents race conditions where clicking a contact name before
 * the table has loaded causes navigation to fail.
 */
async function goBackToContacts(page: import('@playwright/test').Page) {
  await page.getByRole('button', { name: /back/i }).first().click();
  await expect(page).toHaveURL(/\/contacts$/, { timeout: 5000 });
  // Wait for the contact table to render at least one row
  await expect(page.locator('.ant-table-row').first()).toBeVisible({ timeout: 10000 });
}

/**
 * Click a contact row in the Ant Design table by matching the name text.
 * Uses table row locator to avoid matching text elsewhere on the page.
 */
async function clickContactInTable(page: import('@playwright/test').Page, contactName: string) {
  const row = page.locator('.ant-table-row').filter({ hasText: contactName });
  await row.first().click();
  await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 10000 });
}

/**
 * Open the "Add relationship" modal from the Social tab's Relationships card.
 */
async function openRelationshipModal(page: import('@playwright/test').Page) {
  const relCard = page.locator('.ant-card').filter({ hasText: /Relationships/ });
  await expect(relCard).toBeVisible({ timeout: 10000 });
  await relCard.locator('.ant-card-extra button').click();
  const modal = page.locator('.ant-modal').filter({ hasText: /relationship/i });
  await expect(modal).toBeVisible({ timeout: 5000 });
  return modal;
}

/**
 * Select an option from Ant Design virtual-scrolling Select dropdown by typing
 * into the search input to filter, then clicking the matching option.
 * Ant Design grouped Selects use virtual scroll; options not in viewport are not rendered.
 * Typing into the search box filters the list so the target option becomes visible.
 */
async function selectRelType(
  page: import('@playwright/test').Page,
  modal: import('@playwright/test').Locator,
  typeName: string,
) {
  const selects = modal.locator('.ant-select');
  const typeSelect = selects.nth(1);
  // Click to open the dropdown, then type to filter
  await typeSelect.click();
  await page.waitForTimeout(200);
  await typeSelect.locator('input').fill(typeName);
  await page.waitForTimeout(500);
  // Use getByTitle with exact match to avoid partial matches (e.g. "parent" vs "grand parent")
  await page.locator('.ant-select-dropdown:visible').getByTitle(typeName, { exact: true }).click();
}

/**
 * Fill the relationship modal: select a contact and a relationship type, then submit.
 * Seed data uses lowercase names: "parent", "child", "brother/sister", "friend", etc.
 */
async function addRelationship(
  page: import('@playwright/test').Page,
  modal: import('@playwright/test').Locator,
  contactName: string,
  typeName: string,
) {
  const selects = modal.locator('.ant-select');

  // Select contact
  await selects.first().click();
  await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: contactName }).click();

  // Dismiss first dropdown by clicking modal header
  await modal.locator('.ant-modal-header').click();
  await page.waitForTimeout(300);

  // Select relationship type via search-filter
  await selectRelType(page, modal, typeName);

  // Submit
  const responsePromise = page.waitForResponse(
    (resp) => resp.url().includes('/relationships') && resp.request().method() === 'POST'
  );
  await modal.getByRole('button', { name: /ok/i }).click();
  const resp = await responsePromise;
  expect(resp.status()).toBeLessThan(400);
}

// ============================================================================
// Issue #35 Bug 1: Relationship type dropdown should show BOTH directions
// ============================================================================
test.describe('Issue #35 Bug 1: Relationship type selection shows both directions', () => {
  test('should show both parent and child in the relationship type dropdown', async ({ page }) => {
    await setupVault(page, 'rel-direction');
    await goToContacts(page);

    await createContact(page, 'Sarah', 'Johnson');
    await goBackToContacts(page);
    await createContact(page, 'Emily', 'Johnson');
    await goBackToContacts(page);

    // Go to Sarah's page
    await clickContactInTable(page, 'Sarah Johnson');

    await navigateToTab(page, 'Social');
    const modal = await openRelationshipModal(page);
    const typeSelect = modal.locator('.ant-select').nth(1);

    // Search "parent" in the type dropdown
    await typeSelect.click();
    await page.waitForTimeout(200);
    await typeSelect.locator('input').fill('parent');
    await page.waitForTimeout(500);
    const dropdown = page.locator('.ant-select-dropdown:visible');
    await expect(dropdown.getByTitle('parent', { exact: true })).toBeVisible({ timeout: 5000 });

    // Clear the search, close and reopen the dropdown to reset filter state
    await typeSelect.locator('input').fill('');
    await modal.locator('.ant-modal-header').click();
    await page.waitForTimeout(300);

    // Reopen and search "child"
    await typeSelect.click();
    await page.waitForTimeout(200);
    await typeSelect.locator('input').fill('child');
    await page.waitForTimeout(500);
    const dropdown2 = page.locator('.ant-select-dropdown:visible');
    await expect(dropdown2.getByTitle('child', { exact: true })).toBeVisible({ timeout: 5000 });

    // Close the dropdown before clicking Cancel to avoid "grand child" option intercepting the click
    await modal.locator('.ant-modal-header').click();
    await page.waitForTimeout(300);

    await modal.getByRole('button', { name: /cancel/i }).click();
  });

  test('should create Parent→Child relationship from child perspective correctly', async ({ page }) => {
    await setupVault(page, 'rel-child-dir');
    await goToContacts(page);

    await createContact(page, 'Mommy', 'Doe');
    await goBackToContacts(page);
    await createContact(page, 'Kiddo', 'Doe');

    // From Kiddo's page, add Mommy as parent
    await navigateToTab(page, 'Social');
    const modal = await openRelationshipModal(page);
    await addRelationship(page, modal, 'Mommy Doe', 'parent');

    // Kiddo's relationship list should show Mommy
    const relCard = page.locator('.ant-card').filter({ hasText: /Relationships/ }).first();
    await expect(relCard.getByText('Mommy Doe').first()).toBeVisible({ timeout: 10000 });

    // Navigate to Mommy's page — should show Kiddo with reverse type "child"
    await goBackToContacts(page);
    await clickContactInTable(page, 'Mommy Doe');
    await navigateToTab(page, 'Social');

    const mommyRelCard = page.locator('.ant-card').filter({ hasText: /Relationships/ }).first();
    await expect(mommyRelCard.getByText('Kiddo Doe').first()).toBeVisible({ timeout: 10000 });
    // The reverse relationship should display "child" (not "parent")
    await expect(mommyRelCard.locator('.ant-tag').filter({ hasText: /child/i }).first()).toBeVisible({ timeout: 5000 });
  });
});

// ============================================================================
// Issue #35 Bug 2: Reverse relationships cannot be edited/deleted
// ============================================================================
test.describe('Issue #35 Bug 2: Reverse relationship edit/delete', () => {
  test('should delete a relationship from the reverse contact page', async ({ page }) => {
    await setupVault(page, 'rel-rev-del');
    await goToContacts(page);

    await createContact(page, 'Alpha', 'Rev');
    await goBackToContacts(page);
    await createContact(page, 'Beta', 'Rev');

    // From Beta's page, create Beta→Alpha as parent
    await navigateToTab(page, 'Social');
    const modal = await openRelationshipModal(page);
    await addRelationship(page, modal, 'Alpha Rev', 'parent');

    // Go to Alpha's page — she should see the reverse relationship (child)
    await goBackToContacts(page);
    await clickContactInTable(page, 'Alpha Rev');
    await navigateToTab(page, 'Social');

    const relCard = page.locator('.ant-card').filter({ hasText: /Relationships/ }).first();
    await expect(relCard.getByText('Beta Rev').first()).toBeVisible({ timeout: 10000 });

    // Try to delete the relationship from Alpha's page
    const listItem = relCard.locator('.ant-list-item').filter({ hasText: 'Beta Rev' });
    await listItem.locator('.anticon-delete').click();

    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/relationships/') && resp.request().method() === 'DELETE'
    );
    await page.getByRole('button', { name: /ok|yes/i }).click();
    const resp = await deleteResp;
    // This should succeed (< 400), not return 404
    expect(resp.status()).toBeLessThan(400);

    await expect(relCard.getByText('Beta Rev')).not.toBeVisible({ timeout: 5000 });
  });

  test('should edit a relationship from the reverse contact page', async ({ page }) => {
    await setupVault(page, 'rel-rev-edit');
    await goToContacts(page);

    await createContact(page, 'Gamma', 'Rev');
    await goBackToContacts(page);
    await createContact(page, 'Delta', 'Rev');

    // From Delta's page, create Delta→Gamma as friend
    await navigateToTab(page, 'Social');
    const modal = await openRelationshipModal(page);
    await addRelationship(page, modal, 'Gamma Rev', 'friend');

    // Go to Gamma's page
    await goBackToContacts(page);
    await clickContactInTable(page, 'Gamma Rev');
    await navigateToTab(page, 'Social');

    const relCard = page.locator('.ant-card').filter({ hasText: /Relationships/ }).first();
    await expect(relCard.getByText('Delta Rev').first()).toBeVisible({ timeout: 10000 });

    // Try to edit the relationship from Gamma's page
    const listItem = relCard.locator('.ant-list-item').filter({ hasText: 'Delta Rev' });
    await listItem.locator('.anticon-edit').click();

    const editModal = page.locator('.ant-modal').filter({ hasText: /relationship/i });
    await expect(editModal).toBeVisible({ timeout: 5000 });

    // Change type to best friend via search
    await selectRelType(page, editModal, 'best friend');

    const editResp = page.waitForResponse(
      (resp) => resp.url().includes('/relationships/') && resp.request().method() === 'PUT'
    );
    await editModal.getByRole('button', { name: /ok/i }).click();
    const resp = await editResp;
    expect(resp.status()).toBeLessThan(400);
  });
});

// ============================================================================
// Issue #35 Bug 3: Symmetric relationships show duplicates
// ============================================================================
test.describe('Issue #35 Bug 3: Symmetric relationship deduplication', () => {
  test('symmetric relationship should appear only once per contact', async ({ page }) => {
    await setupVault(page, 'rel-dedup');
    await goToContacts(page);

    await createContact(page, 'JacobX', 'Sib');
    await goBackToContacts(page);
    await createContact(page, 'LilyX', 'Sib');
    await goBackToContacts(page);

    // Go to JacobX's page
    await clickContactInTable(page, 'JacobX Sib');

    await navigateToTab(page, 'Social');
    const modal = await openRelationshipModal(page);
    await addRelationship(page, modal, 'LilyX Sib', 'brother/sister');

    // JacobX's page should show LilyX ONCE, not twice
    const relCard = page.locator('.ant-card').filter({ hasText: /Relationships/ }).first();
    const lilyEntries = relCard.locator('.ant-list-item').filter({ hasText: 'LilyX Sib' });
    await expect(lilyEntries.first()).toBeVisible({ timeout: 10000 });
    await expect(lilyEntries).toHaveCount(1);

    // Go to LilyX's page — should also show JacobX ONCE
    await goBackToContacts(page);
    await clickContactInTable(page, 'LilyX Sib');
    await navigateToTab(page, 'Social');

    const lilyRelCard = page.locator('.ant-card').filter({ hasText: /Relationships/ }).first();
    const jacobEntries = lilyRelCard.locator('.ant-list-item').filter({ hasText: 'JacobX Sib' });
    await expect(jacobEntries.first()).toBeVisible({ timeout: 10000 });
    await expect(jacobEntries).toHaveCount(1);
  });

  test('asymmetric relationship should appear once per contact with correct type', async ({ page }) => {
    await setupVault(page, 'rel-asym');
    await goToContacts(page);

    await createContact(page, 'Padre', 'Smith');
    await goBackToContacts(page);
    await createContact(page, 'Hijo', 'Smith');

    // From Hijo's page, create Hijo→Padre as parent
    await navigateToTab(page, 'Social');
    const modal = await openRelationshipModal(page);
    await addRelationship(page, modal, 'Padre Smith', 'parent');

    // Hijo's page: should show Padre with "parent" tag, exactly once
    const hijoRelCard = page.locator('.ant-card').filter({ hasText: /Relationships/ }).first();
    const padreEntries = hijoRelCard.locator('.ant-list-item').filter({ hasText: 'Padre Smith' });
    await expect(padreEntries.first()).toBeVisible({ timeout: 10000 });
    await expect(padreEntries).toHaveCount(1);
    await expect(padreEntries.first().locator('.ant-tag').filter({ hasText: /parent/i })).toBeVisible();

    // Padre's page: should show Hijo with "child" tag, exactly once
    await goBackToContacts(page);
    await clickContactInTable(page, 'Padre Smith');
    await navigateToTab(page, 'Social');

    const padreRelCard = page.locator('.ant-card').filter({ hasText: /Relationships/ }).first();
    const hijoEntries = padreRelCard.locator('.ant-list-item').filter({ hasText: 'Hijo Smith' });
    await expect(hijoEntries.first()).toBeVisible({ timeout: 10000 });
    await expect(hijoEntries).toHaveCount(1);
    await expect(hijoEntries.first().locator('.ant-tag').filter({ hasText: /child/i })).toBeVisible();
  });
});
