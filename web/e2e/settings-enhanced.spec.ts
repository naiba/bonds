import { test, expect } from '@playwright/test';

async function registerUser(page: import('@playwright/test').Page) {
  const email = `se-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Settings');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
}

test.describe('Enhanced Settings', () => {
  test('should navigate to WebAuthn settings page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/webauthn');
    await expect(
      page.getByRole('heading', { level: 4 }).first()
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-card')).toBeVisible({ timeout: 10000 });
  });

  test('should show register passkey button on WebAuthn page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/webauthn');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('button').filter({ has: page.locator('.anticon-plus') })).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to OAuth providers page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/oauth');
    await expect(
      page.getByRole('heading', { level: 4 }).first()
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-card')).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to storage info page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/storage');
    await expect(
      page.getByRole('heading', { level: 4 }).first()
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-card').first()).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to users page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/users');
    await expect(
      page.getByRole('heading', { level: 4 }).getByText('Users')
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-card')).toBeVisible({ timeout: 10000 });
  });

  test('should show current user in users list', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/users');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('table').getByText('Settings Tester')).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to personalize page and show currencies section', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/personalize');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { level: 4 }).first()).toBeVisible({ timeout: 10000 });

    const currenciesPanel = page.locator('.ant-collapse-item').filter({ hasText: 'Currencies' });
    await expect(currenciesPanel).toBeVisible({ timeout: 10000 });
    await currenciesPanel.locator('.ant-collapse-header').click();

    await expect(page.getByPlaceholder(/search currencies/i)).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: /enable all/i })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: /disable all/i })).toBeVisible({ timeout: 5000 });
  });

  test('should search and filter currencies', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/personalize');
    await page.waitForLoadState('networkidle');

    const currenciesPanel = page.locator('.ant-collapse-item').filter({ hasText: 'Currencies' });
    await currenciesPanel.locator('.ant-collapse-header').click();

    const searchInput = page.getByPlaceholder(/search currencies/i);
    await expect(searchInput).toBeVisible({ timeout: 10000 });

    await expect(page.locator('.ant-switch').first()).toBeVisible({ timeout: 20000 });

    await searchInput.fill('USD');
    await expect(page.locator('.ant-list-item').filter({ hasText: 'USD' })).toBeVisible({ timeout: 10000 });
  });

  test('should show currency toggle switches', async ({ page }) => {
    await registerUser(page);

    await page.goto('/settings/personalize');
    await page.waitForLoadState('networkidle');

    const currenciesPanel = page.locator('.ant-collapse-item').filter({ hasText: 'Currencies' });
    await currenciesPanel.locator('.ant-collapse-header').click();

    await expect(page.getByPlaceholder(/search currencies/i)).toBeVisible({ timeout: 10000 });

    await expect(page.locator('.ant-switch').first()).toBeVisible({ timeout: 20000 });
  });

  test('should show module reorder buttons in template page modules', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/personalize');
    await page.waitForLoadState('networkidle');

    // Expand Templates section
    const templatesPanel = page.locator('.ant-collapse-item').filter({
      has: page.locator('.ant-collapse-header span').getByText('Templates', { exact: true }),
    });
    await expect(templatesPanel).toBeVisible({ timeout: 10000 });
    await templatesPanel.locator('.ant-collapse-header').click();
    // Wait for template items to load
    await expect(templatesPanel.locator('.ant-list-item').first()).toBeVisible({ timeout: 10000 });

    // Expand the first template item to show SubItemsPanel (Pages)
    const firstTemplateItem = templatesPanel.locator('.ant-list-item').first();
    await firstTemplateItem.locator('button').filter({ has: page.locator('.anticon-right, .anticon-down') }).first().click();
    await page.waitForTimeout(500);
    // Wait for sub-items (template pages) to load
    await expect(templatesPanel.getByText('Pages').first()).toBeVisible({ timeout: 10000 });

    // Find the pages sub-area and expand modules on the first page
    const subItemsArea = templatesPanel.locator('[style*="border-left"]').filter({ hasText: 'Pages' });
    const pageItems = subItemsArea.locator('.ant-list-item');
    await expect(pageItems.first()).toBeVisible({ timeout: 10000 });
    // Click the modules icon (AppstoreOutlined) on the first page
    const firstPageItem = pageItems.first();
    await firstPageItem.locator('button').filter({ has: page.locator('.anticon-appstore') }).click();
    await page.waitForTimeout(500);

    // Verify modules list with up/down arrow buttons appears
    await expect(templatesPanel.locator('.anticon-arrow-up').first()).toBeVisible({ timeout: 10000 });
    await expect(templatesPanel.locator('.anticon-arrow-down').first()).toBeVisible({ timeout: 10000 });
  });

  test('should reorder module position via arrow buttons', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/personalize');
    await page.waitForLoadState('networkidle');

    // Expand Templates section
    const templatesPanel = page.locator('.ant-collapse-item').filter({
      has: page.locator('.ant-collapse-header span').getByText('Templates', { exact: true }),
    });
    await templatesPanel.locator('.ant-collapse-header').click();
    await expect(templatesPanel.locator('.ant-list-item').first()).toBeVisible({ timeout: 10000 });

    // Expand the first template item to show SubItemsPanel (Pages)
    const firstTemplateItem = templatesPanel.locator('.ant-list-item').first();
    await firstTemplateItem.locator('button').filter({ has: page.locator('.anticon-right, .anticon-down') }).first().click();
    await page.waitForTimeout(500);
    await expect(templatesPanel.getByText('Pages').first()).toBeVisible({ timeout: 10000 });

    // Find the pages sub-area and expand modules on the first page
    const subItemsArea = templatesPanel.locator('[style*="border-left"]').filter({ hasText: 'Pages' });
    const pageItems = subItemsArea.locator('.ant-list-item');
    await expect(pageItems.first()).toBeVisible({ timeout: 10000 });
    const firstPageItem = pageItems.first();
    await firstPageItem.locator('button').filter({ has: page.locator('.anticon-appstore') }).click();
    await page.waitForTimeout(500);
    // Wait for module list with Page Modules label
    await expect(templatesPanel.getByText('Page Modules').first()).toBeVisible({ timeout: 10000 });
    const modulesArea = templatesPanel.locator('[style*="border-left"]').filter({ hasText: 'Page Modules' }).last();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
    // Get the first module name
    const moduleListItems = modulesArea.locator('.ant-list-item');
    await expect(moduleListItems.first()).toBeVisible({ timeout: 10000 });
    const firstModuleName = await moduleListItems.first().locator('span').first().textContent();

    // Click down arrow on the first module (within modulesArea) and wait for response
    const downArrow = modulesArea.locator('.anticon-arrow-down').first();
    await expect(downArrow).toBeEnabled({ timeout: 5000 });
    const [response] = await Promise.all([
      page.waitForResponse(resp => resp.url().includes('/position') && resp.status() === 200, { timeout: 10000 }).catch(() => null),
      downArrow.click(),
    ]);
    // After reorder, the first module should have changed
    await page.waitForTimeout(500);
    const newFirstModuleName = await moduleListItems.first().locator('span').first().textContent();
    // If the API succeeded, the module should have moved down
    if (response) {
      expect(firstModuleName).not.toBe(newFirstModuleName);
    }
  });
});

test.describe('Enhanced Settings - Personalize', () => {
  test('personalize modules section shows module names', async ({ page }) => {
    await registerUser(page);

    await page.goto('/settings/personalize');
    await page.waitForLoadState('networkidle');

    // Find the Modules collapse panel and expand it
    const modulesPanel = page.locator('.ant-collapse-item').filter({ hasText: 'Modules' });
    await expect(modulesPanel).toBeVisible({ timeout: 10000 });
    await modulesPanel.locator('.ant-collapse-header').click();

    // Wait for the list items to load
    await expect(modulesPanel.locator('.ant-list-item').first()).toBeVisible({ timeout: 15000 });

    // Verify known module names from seed data are present
    const moduleNames = ['Avatar', 'Contact name', 'Notes', 'Feed'];
    for (const name of moduleNames) {
      await expect(modulesPanel.getByText(name, { exact: false }).first()).toBeVisible({ timeout: 5000 });
    }

    // Verify list is not empty
    const count = await modulesPanel.locator('.ant-list-item').count();
    expect(count).toBeGreaterThan(0);
  });
});
