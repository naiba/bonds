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
test.describe('Company employees display in list', () => {
  test.setTimeout(90000);

  test('should display employee tags in the companies list table', async ({ page }) => {
    await registerAndCreateVault(page, 'co-employees');
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
// Issue #63: Life Metrics redesign — event log (+1 increment) pattern
// Replaces old contact-association model (#56) with Monica v5's per-click event log.
// Life Metrics are now on the Vault Dashboard "Life metrics" tab.
// =====================================================================================
test.describe('Life Metrics increment and stats on Dashboard', () => {
  test.setTimeout(90000);

  test('should create, increment, and display stats for life metrics on dashboard', async ({ page }) => {
    await registerAndCreateVault(page, 'life-metrics');

    // 1. We are on the vault dashboard. Switch to the "Life metrics" tab.
    //    The dashboard uses Ant Design Segmented component for tabs.
    const lifeMetricsTab = page.getByText(/life metrics/i);
    await lifeMetricsTab.click();
    await page.waitForLoadState('networkidle');

    // 2. Create a new life metric via the "Track a new metric" button
    await page.getByRole('button', { name: /track a new metric/i }).click();
    const createModal = page.locator('.ant-modal:visible');
    await expect(createModal).toBeVisible({ timeout: 5000 });
    await createModal.locator('input#label').fill('Exercise');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/lifeMetrics') && resp.request().method() === 'POST'
    );
    await createModal.getByRole('button', { name: /ok/i }).click();
    await createResp;
    await expect(page.getByText('Exercise')).toBeVisible({ timeout: 10000 });

    // 3. Click the "+1" button to increment the metric
    // The metric card has a +1 button that briefly shows 🤭 emoji
    const incrementResp = page.waitForResponse(
      (resp) => resp.url().includes('/increment') && resp.request().method() === 'POST'
    );
    await page.getByRole('button', { name: '+1' }).click();
    await incrementResp;

    // 4. After increment, stats badges should show updated counts
    // Weekly events should now show "1"
    await expect(page.getByText(/1\/.*week/i)).toBeVisible({ timeout: 10000 });

    // 5. Increment again to verify count increases
    // Wait for the emoji to reset back to "+1"
    await expect(page.getByRole('button', { name: '+1' })).toBeVisible({ timeout: 5000 });
    const incrementResp2 = page.waitForResponse(
      (resp) => resp.url().includes('/increment') && resp.request().method() === 'POST'
    );
    await page.getByRole('button', { name: '+1' }).click();
    await incrementResp2;

    // 6. Weekly events should now show "2"
    await expect(page.getByText(/2\/.*week/i)).toBeVisible({ timeout: 10000 });

    // 7. Click the stats badge to expand the monthly bar chart
    await page.locator('.ant-tag').filter({ hasText: /week/i }).click();
    // The bar chart area should appear with month labels
    await expect(page.getByText(/Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec/i).first()).toBeVisible({ timeout: 10000 });

    // 8. Delete the metric via the "···" dropdown menu
    await page.getByRole('button', { name: '···' }).click();
    await page.getByText(/delete/i).click();
    // Confirm deletion in the modal
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/lifeMetrics/') && resp.request().method() === 'DELETE'
    );
    await page.locator('.ant-modal-confirm').getByRole('button', { name: /delete/i }).click();
    await deleteResp;

    // 9. Verify the metric is gone
    await expect(page.getByText('Exercise')).not.toBeVisible({ timeout: 10000 });
  });
});
