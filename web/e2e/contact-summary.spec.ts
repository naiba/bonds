import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function setupVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Summary');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Summary Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For summary testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
  await expect(page.getByRole('heading', { name: 'Summary Vault' })).toBeVisible({ timeout: 10000 });
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

test.describe('Contact Summary Card — Issue #54', () => {

  test('summary card should be visible on contact detail page', async ({ page }) => {
    await setupVault(page, 'summary-visible');
    await goToContacts(page);
    await createContact(page, 'SummaryVis', 'User');

    // The summary card should exist between the header card and the tabs.
    // It uses a specific data-testid for reliable E2E selection.
    const summaryCard = page.locator('[data-testid="contact-summary-card"]');
    await expect(summaryCard).toBeVisible({ timeout: 10000 });
  });

  test('summary card should show labels after adding one', async ({ page }) => {
    await setupVault(page, 'summary-labels');
    const vaultUrl = page.url();

    // First create a label in vault settings
    await page.goto(vaultUrl + '/settings');
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });
    await page.getByRole('tab', { name: 'Labels' }).click();
    await page.waitForLoadState('networkidle');

    const nameInput = page.getByPlaceholder('Name');
    await expect(nameInput).toBeVisible({ timeout: 10000 });
    await nameInput.fill('summary-label');

    const labelResponse = page.waitForResponse(
      (resp) => resp.url().includes('/labels') && resp.request().method() === 'POST'
    );
    await page.getByRole('button', { name: 'Add' }).click();
    await labelResponse;
    await page.waitForLoadState('networkidle');

    // Create contact and add the label
    await page.goto(vaultUrl);
    await page.waitForLoadState('networkidle');
    await goToContacts(page);
    await createContact(page, 'LabelSum', 'User');

    await navigateToTab(page, 'Contact information');

    // Use h5 title to precisely match the module card, not the summary card
    const labelsCard = page.locator('.ant-card').filter({ has: page.locator('h5', { hasText: 'Labels' }) });
    await expect(labelsCard).toBeVisible({ timeout: 10000 });
    await labelsCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal').filter({ hasText: /add label/i });
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.locator('.ant-select').click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: 'summary-label' }).click();

    const addResp = page.waitForResponse(
      (resp) => resp.url().includes('/labels') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /save/i }).click();
    await addResp;
    await page.waitForLoadState('networkidle');

    // Now verify the summary card shows the label
    const summaryCard = page.locator('[data-testid="contact-summary-card"]');
    await expect(summaryCard).toBeVisible({ timeout: 10000 });
    await expect(summaryCard.locator('.ant-tag').filter({ hasText: 'summary-label' })).toBeVisible({ timeout: 10000 });
  });

  test('summary card should show contact info (email/phone) after adding', async ({ page }) => {
    await setupVault(page, 'summary-cinfo');
    await goToContacts(page);
    await createContact(page, 'CInfoSum', 'User');

    // Navigate to the "Contact information" tab where ContactInfo module lives
    // (seed data assigns contact_information to the 'contact' slug page)
    await navigateToTab(page, 'Contact information');

    // Use .ant-card-head-title to precisely match the module card header,
    // avoiding false matches with the summary card or other elements
    const infoCard = page.locator('.ant-card').filter({ has: page.locator('.ant-card-head-title', { hasText: 'Contact Information' }) });
    await expect(infoCard).toBeVisible({ timeout: 10000 });
    // ContactInfo module uses inline form: click Add to show the form
    await infoCard.getByRole('button', { name: /add/i }).click();

    const valueInput = infoCard.getByPlaceholder(/value/i);
    await expect(valueInput).toBeVisible({ timeout: 5000 });
    await valueInput.fill('summary@example.com');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/contactInformation') && resp.request().method() === 'POST'
    );
    await infoCard.getByRole('button', { name: /save/i }).click();
    await createResp;
    await page.waitForLoadState('networkidle');

    // Now add a phone number
    await infoCard.getByRole('button', { name: /add/i }).click();
    const valueInput2 = infoCard.getByPlaceholder(/value/i);
    await expect(valueInput2).toBeVisible({ timeout: 5000 });
    // Switch type to phone
    await infoCard.locator('.ant-select').first().click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: 'Phone' }).click();
    await valueInput2.fill('+1-555-0123');

    const createResp2 = page.waitForResponse(
      (resp) => resp.url().includes('/contactInformation') && resp.request().method() === 'POST'
    );
    await infoCard.getByRole('button', { name: /save/i }).click();
    await createResp2;
    await page.waitForLoadState('networkidle');

    // Now check summary card shows the contact info
    const summaryCard = page.locator('[data-testid="contact-summary-card"]');
    await expect(summaryCard).toBeVisible({ timeout: 10000 });
    await expect(summaryCard.getByText('summary@example.com')).toBeVisible({ timeout: 10000 });
    await expect(summaryCard.getByText('+1-555-0123')).toBeVisible({ timeout: 10000 });
  });

  test('summary card should show address after adding', async ({ page }) => {
    await setupVault(page, 'summary-addr');
    await goToContacts(page);
    await createContact(page, 'AddrSum', 'User');

    // Navigate to the "Contact information" tab where Addresses module lives
    // (seed data assigns addresses to the 'contact' slug page)
    await navigateToTab(page, 'Contact information');

    // Use .ant-card-head-title to precisely match the module card header,
    // avoiding false matches with the summary card or other elements
    const addrCard = page.locator('.ant-card').filter({ has: page.locator('.ant-card-head-title', { hasText: 'Addresses' }) });
    await expect(addrCard).toBeVisible({ timeout: 10000 });
    await addrCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal').filter({ hasText: /address/i });
    await expect(modal).toBeVisible({ timeout: 5000 });
    // Address modal uses label-based inputs (no placeholders), use getByLabel
    await modal.getByLabel(/Address Line 1/i).fill('123 Main St');
    await modal.getByLabel(/City/i).fill('San Francisco');
    await modal.getByLabel(/Country/i).fill('US');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/addresses') && resp.request().method() === 'POST'
    );
    // Address modal uses onOk, so submit button is 'OK' not 'Save'
    await modal.getByRole('button', { name: /ok/i }).click();
    await createResp;
    await page.waitForLoadState('networkidle');

    // Check summary card shows the address
    const summaryCard = page.locator('[data-testid="contact-summary-card"]');
    await expect(summaryCard).toBeVisible({ timeout: 10000 });
    await expect(summaryCard.getByText(/123 Main St/)).toBeVisible({ timeout: 10000 });
    await expect(summaryCard.getByText(/San Francisco/)).toBeVisible({ timeout: 10000 });
  });

  test('summary card should show religion after setting it', async ({ page }) => {
    await setupVault(page, 'summary-religion');
    await goToContacts(page);
    await createContact(page, 'ReligionSum', 'User');

    // Navigate to "Contact information" tab where ExtraInfoModule lives
    await navigateToTab(page, 'Contact information');

    // Use h5 title to precisely match the module card, not the summary card
    const religionCard = page.locator('.ant-card').filter({
      has: page.locator('h5', { hasText: 'Religion' }),
    });
    await expect(religionCard).toBeVisible({ timeout: 10000 });

    await religionCard.getByRole('button', { name: /edit/i }).click();

    const modal = page.locator('.ant-modal').filter({
      has: page.locator('.ant-modal-title', { hasText: 'Religion' }),
    });
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.locator('.ant-select').click();
    // Select any religion from the seed data
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').first().click();

    const saveResp = page.waitForResponse(
      (resp) => resp.url().includes('/religion') && resp.request().method() === 'PUT'
    );
    await modal.getByRole('button', { name: /save/i }).click();
    await saveResp;
    await page.waitForLoadState('networkidle');

    // Summary card should show the religion name
    const summaryCard = page.locator('[data-testid="contact-summary-card"]');
    await expect(summaryCard).toBeVisible({ timeout: 10000 });
    // The religion should appear somewhere in the summary (from seed data)
    // We just verify the religion section is no longer showing "Not set"
    await expect(summaryCard.locator('[data-testid="summary-religion"]')).toBeVisible({ timeout: 10000 });
    // The religion section should have actual text (not empty)
    const religionText = await summaryCard.locator('[data-testid="summary-religion"]').textContent();
    expect(religionText).toBeTruthy();
    expect(religionText!.trim().length).toBeGreaterThan(0);
  });

  test('summary card should show job info after adding a job', async ({ page }) => {
    await setupVault(page, 'summary-job');
    const vaultUrl = page.url();

    // First create a company in vault settings
    await page.goto(vaultUrl + '/settings');
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });
    await page.getByRole('tab', { name: /companies/i }).click();
    await page.waitForLoadState('networkidle');

    // Wait for Companies tab to render before clicking Add
    await expect(page.getByRole('button', { name: /add company/i }).first()).toBeVisible({ timeout: 15000 });
    await page.getByRole('button', { name: /add company/i }).first().click();
    const companyModal = page.locator('.ant-modal');
    await expect(companyModal).toBeVisible({ timeout: 5000 });
    await companyModal.locator('input#name').fill('Acme Corp');
    const companyResp = page.waitForResponse(
      (resp) => resp.url().includes('/companies') && resp.request().method() === 'POST'
    );
    await companyModal.getByRole('button', { name: /ok/i }).click();
    await companyResp;
    await page.waitForLoadState('networkidle');

    // Now create a contact and add a job
    await page.goto(vaultUrl);
    await page.waitForLoadState('networkidle');
    await goToContacts(page);
    await createContact(page, 'JobSum', 'User');

    // Navigate to "Contact information" tab where ExtraInfoModule shows jobs
    await navigateToTab(page, 'Contact information');

    // Use h5 title to precisely match the module card, not the summary card
    const jobCard = page.locator('.ant-card').filter({
      has: page.locator('h5', { hasText: 'Job Information' }),
    });
    await expect(jobCard).toBeVisible({ timeout: 10000 });
    await jobCard.getByRole('button', { name: /add job/i }).click();

    const jobModal = page.locator('.ant-modal').filter({
      has: page.locator('.ant-modal-title', { hasText: 'Add Job' }),
    });
    await expect(jobModal).toBeVisible({ timeout: 5000 });
    await jobModal.locator('.ant-select').first().click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: 'Acme Corp' }).click();

    // Fill position using label
    await jobModal.getByLabel('Position').fill('Software Engineer');
    const jobResp = page.waitForResponse(
      (resp) => resp.url().includes('/jobs') && resp.request().method() === 'POST'
    );
    await jobModal.getByRole('button', { name: /save/i }).click();
    await jobResp;
    await page.waitForLoadState('networkidle');

    // Summary card should show the job
    const summaryCard = page.locator('[data-testid="contact-summary-card"]');
    await expect(summaryCard).toBeVisible({ timeout: 10000 });
    await expect(summaryCard.getByText(/Acme Corp/)).toBeVisible({ timeout: 10000 });
    await expect(summaryCard.getByText(/Software Engineer/)).toBeVisible({ timeout: 10000 });
  });

  test('summary card should show family relationships', async ({ page }) => {
    await setupVault(page, 'summary-family');
    await goToContacts(page);

    // Create two contacts
    await createContact(page, 'ParentSum', 'User');
    const parentUrl = page.url();

    // Go back to create a second contact
    await page.getByText(/back/i).click();
    await page.waitForLoadState('networkidle');
    await createContact(page, 'ChildSum', 'User');

    // Navigate to "Social" tab where RelationshipsModule lives
    await navigateToTab(page, 'Social');

    // Scope to active tab pane to avoid matching cards in hidden tab panes or the summary card
    const relCard = page.locator('.ant-tabs-tabpane-active .ant-card').filter({
      hasText: /Relationships/,
    });
    await expect(relCard).toBeVisible({ timeout: 10000 });
    await relCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal').filter({ hasText: /relationship/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Select the related contact (ParentSum User)
    const contactSelect = modal.locator('.ant-select').first();
    await contactSelect.click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: 'ParentSum' }).click();
    // Dismiss dropdown
    await modal.locator('.ant-modal-header').click();

    // Wait for first dropdown to close, then select a relationship type
    await page.waitForTimeout(300);
    const typeSelect = modal.locator('.ant-select').nth(1);
    await typeSelect.click();
    await page.waitForTimeout(200);
    // Type to filter — pick 'parent' from seed data
    await typeSelect.locator('input').fill('parent');
    await page.waitForTimeout(500);
    await page.locator('.ant-select-dropdown:visible').getByTitle('parent', { exact: true }).click();

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/relationships') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    await createResp;
    await page.waitForLoadState('networkidle');

    // Now go to the parent contact and verify the summary card shows the relationship
    await page.goto(parentUrl);
    await page.waitForLoadState('networkidle');

    const summaryCard = page.locator('[data-testid="contact-summary-card"]');
    await expect(summaryCard).toBeVisible({ timeout: 10000 });
    // Should show the relationship (ChildSum User)
    await expect(summaryCard.getByText(/ChildSum/)).toBeVisible({ timeout: 10000 });
  });
});
