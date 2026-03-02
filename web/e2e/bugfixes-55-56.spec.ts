import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function registerAndCreateVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Test');
  await page.getByPlaceholder('Last name').fill('User');
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

function getVaultUrl(page: import('@playwright/test').Page): string {
  return page.url();
}

async function createContact(page: import('@playwright/test').Page, vaultUrl: string, firstName: string, lastName: string) {
  await page.goto(vaultUrl + '/contacts');
  await page.waitForLoadState('networkidle');
  await page.getByRole('button', { name: /add contact/i }).click();
  await page.getByPlaceholder('First name').fill(firstName);
  await page.getByPlaceholder('Last name').fill(lastName);
  await page.getByRole('button', { name: /create contact/i }).click();
  await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 15000 });
  await page.waitForLoadState('networkidle');
}

// =====================================================================================
// Issue #55: Vault Settings > Companies > Employees should be displayed in the list
// =====================================================================================
test.describe('Bug #55 - Companies Employees display in list', () => {
  test.setTimeout(90000);

  test('should display employee tags in the companies list table', async ({ page }) => {
    await registerAndCreateVault(page, 'bug55');
    const vaultUrl = getVaultUrl(page);

    // 1. Create a contact to later assign as employee
    await createContact(page, vaultUrl, 'Alice', 'Johnson');

    // 2. Navigate to Vault Settings > Companies tab
    await page.goto(vaultUrl + '/settings');
    await page.waitForLoadState('networkidle');
    await page.getByRole('tab', { name: /Companies/i }).click();
    await expect(page.getByRole('button', { name: /add company/i }).first()).toBeVisible({ timeout: 15000 });

    // 3. Create a company
    await page.getByRole('button', { name: /add company/i }).first().click();
    const modal = page.locator('.ant-modal');
    await expect(modal).toBeVisible({ timeout: 10000 });
    await modal.locator('input#name').fill('Bug55 Corp');
    await modal.locator('input#type').fill('Technology');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/companies') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    await createResp;
    await expect(page.getByText('Bug55 Corp')).toBeVisible({ timeout: 10000 });

    // 4. Click the company row to open the drawer
    await page.getByRole('row').filter({ hasText: 'Bug55 Corp' }).click();
    const drawer = page.locator('.ant-drawer');
    await expect(drawer).toBeVisible({ timeout: 10000 });

    // 5. In the drawer, click "Add Employee"
    await drawer.getByRole('button', { name: /add employee/i }).click();
    const employeeModal = page.locator('.ant-modal').filter({ hasText: /add employee|select contact/i });
    await expect(employeeModal).toBeVisible({ timeout: 10000 });

    // 6. Select the contact from dropdown (it should list all vault contacts)
    const select = employeeModal.locator('.ant-select');
    await select.click();
    await page.waitForTimeout(1000); // Wait for dropdown options to load
    // Type to filter — the contact should show "Alice Johnson" or "Johnson Alice"
    await select.locator('input').fill('Alice');
    await page.waitForTimeout(500);
    await page.locator('.ant-select-item-option').filter({ hasText: /Alice/ }).first().click();

    // 7. Optionally fill job position
    await employeeModal.locator('input#job_position').fill('Engineer');

    // 8. Save the employee association
    const addResp = page.waitForResponse(
      (resp) => resp.url().includes('/employees') && resp.request().method() === 'POST'
    );
    await employeeModal.getByRole('button', { name: /save/i }).click();
    await addResp;

    // 9. Close the drawer by navigating back to settings
    // (drawer close button may not fully dismiss in Playwright)
    await page.goto(page.url());
    await page.waitForLoadState('networkidle');
    await page.getByRole('tab', { name: /Companies/i }).click();
    await expect(page.getByText('Bug55 Corp')).toBeVisible({ timeout: 15000 });

    // 10. Verify the employee name appears in the companies list table as a Tag
    // BUG FIX (#55): Before fix, the Employees column was always empty in the list.
    // After fix, the company row should show "Alice Johnson" (or similar) as a tag.
    const companyRow = page.getByRole('row').filter({ hasText: 'Bug55 Corp' });
    await expect(companyRow.locator('.ant-tag').filter({ hasText: /Alice/ })).toBeVisible({ timeout: 10000 });
  });
});

// =====================================================================================
// Issue #56: Life Metrics > Add contact should show linked contacts
// =====================================================================================
test.describe('Bug #56 - Life Metrics contacts display and removal', () => {
  test.setTimeout(90000);

  test('should display and remove linked contacts in life metrics', async ({ page }) => {
    await registerAndCreateVault(page, 'bug56');
    const vaultUrl = getVaultUrl(page);

    // 1. Create a contact
    await createContact(page, vaultUrl, 'Bob', 'Smith');

    // 2. Navigate to Life Metrics page
    await page.goto(vaultUrl + '/life-metrics');
    await expect(page).toHaveURL(/\/life-metrics$/, { timeout: 10000 });
    await expect(
      page.getByRole('heading', { level: 4 }).filter({ hasText: 'Life Metrics' })
    ).toBeVisible({ timeout: 10000 });

    // 3. Create a life metric
    await page.getByRole('button', { name: /add metric/i }).click();
    const createModal = page.locator('.ant-modal').filter({ hasText: /add metric|life metric/i });
    await expect(createModal).toBeVisible({ timeout: 5000 });
    await createModal.locator('input#label').fill('Bug56 Metric');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/lifeMetrics') && resp.request().method() === 'POST'
    );
    await createModal.getByRole('button', { name: /ok/i }).click();
    await createResp;
    await expect(page.getByText('Bug56 Metric')).toBeVisible({ timeout: 10000 });

    // 4. Click the "Add Contact" button in the metric row
    const metricRow = page.getByRole('row').filter({ hasText: 'Bug56 Metric' });
    await metricRow.locator('button').filter({ has: page.locator('.anticon-user-add') }).click();

    // 5. In the "Add Contact" modal, search for and select the contact
    const contactModal = page.locator('.ant-modal').filter({ hasText: /add contact/i });
    await expect(contactModal).toBeVisible({ timeout: 10000 });

    const contactSelect = contactModal.locator('.ant-select');
    await contactSelect.click();
    // Search for the contact using the search functionality
    await contactSelect.locator('input').fill('Bob');
    await page.waitForTimeout(1000); // Wait for search results
    await page.locator('.ant-select-item-option').filter({ hasText: /Bob/ }).first().click();

    const addContactResp = page.waitForResponse(
      (resp) => resp.url().includes('/contacts') && resp.request().method() === 'POST' && resp.url().includes('/lifeMetrics/')
    );
    await contactModal.getByRole('button', { name: /ok/i }).click();
    await addContactResp;

    // 6. Verify the contact appears as a tag in the life metric row
    // BUG FIX (#56): Before fix, contacts were never returned by the List API,
    // so the contacts column was always empty after adding a contact.
    await expect(metricRow.locator('.ant-tag').filter({ hasText: /Bob/ })).toBeVisible({ timeout: 10000 });

    // 7. Remove the contact by clicking the close button on the tag
    // BUG FIX (#56): Before fix, the DELETE endpoint didn't exist,
    // so removing a contact would fail silently.
    const contactTag = metricRow.locator('.ant-tag').filter({ hasText: /Bob/ });
    const removeResp = page.waitForResponse(
      (resp) => resp.url().includes('/contacts/') && resp.request().method() === 'DELETE' && resp.url().includes('/lifeMetrics/')
    );
    await contactTag.locator('.anticon-close').click();
    await removeResp;

    // 8. Verify the contact tag is removed
    await expect(metricRow.locator('.ant-tag').filter({ hasText: /Bob/ })).not.toBeVisible({ timeout: 10000 });
  });
});
