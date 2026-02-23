import { test, expect } from '@playwright/test';

async function setupVault(page: import('@playwright/test').Page) {
  const email = `levt-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('LifeEvent');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('LE Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For life event testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');
}

function getVaultUrl(page: import('@playwright/test').Page): string {
  return page.url();
}

async function navigateToLifeEventsTab(page: import('@playwright/test').Page) {
  const vaultUrl = getVaultUrl(page);
  await page.goto(vaultUrl + '/settings');
  await page.waitForLoadState('networkidle');
  await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });
  await page.getByRole('tab', { name: /life events/i }).click();
  await page.waitForLoadState('networkidle');
}

test.describe('Vault Settings - Life Events', () => {
  test('should navigate to vault settings life events tab', async ({ page }) => {
    await setupVault(page);
    await navigateToLifeEventsTab(page);
    await expect(page.getByText('Add Category')).toBeVisible({ timeout: 10000 });
  });

  test('should show add category input and button', async ({ page }) => {
    await setupVault(page);
    await navigateToLifeEventsTab(page);

    const addCard = page.locator('.ant-card').filter({ hasText: 'Add Category' });
    await expect(addCard).toBeVisible({ timeout: 10000 });
    await expect(addCard.getByPlaceholder(/name/i)).toBeVisible({ timeout: 5000 });
    await expect(addCard.getByRole('button', { name: /add/i })).toBeVisible({ timeout: 5000 });
  });

  test('should reorder life event categories via arrow buttons', async ({ page }) => {
    await setupVault(page);
    await navigateToLifeEventsTab(page);

    // Wait for collapse panels (categories) to load
    await expect(page.locator('.ant-collapse-item').first()).toBeVisible({ timeout: 10000 });

    // Get the second category panel
    const secondPanel = page.locator('.ant-collapse-item').nth(1);
    await expect(secondPanel).toBeVisible({ timeout: 5000 });

    // Find the UP arrow button in the second category's extra area
    const upArrow = secondPanel.locator('button', { has: page.locator('.anticon-arrow-up') });
    await expect(upArrow).toBeVisible({ timeout: 5000 });

    // Click up arrow and wait for position API response
    const [posResp] = await Promise.all([
      page.waitForResponse(
        (resp) => resp.url().includes('/lifeEventCategories') && resp.url().includes('/position') && resp.request().method() === 'POST' && resp.status() < 400
      ),
      upArrow.click(),
    ]);
    const posBody = await posResp.json();
    expect(posBody.success).toBe(true);

    // Wait for refetch to complete
    await page.waitForResponse(
      (resp) => resp.url().includes('/lifeEventCategories') && resp.request().method() === 'GET' && resp.status() < 400,
      { timeout: 10000 }
    ).catch(() => null);
    await page.waitForLoadState('networkidle');

    // Verify: the UP arrow on the NEW first panel (previously second) should be disabled
    // because it's now at index 0
    const firstPanelUpArrow = page.locator('.ant-collapse-item').first().locator('button', { has: page.locator('.anticon-arrow-up') });
    await expect(firstPanelUpArrow).toBeDisabled({ timeout: 5000 });
  });

  test('should reorder life event types within a category via arrow buttons', async ({ page }) => {
    await setupVault(page);
    await navigateToLifeEventsTab(page);
    const lifeEventsCard = page.locator('.ant-card').filter({ hasText: 'Life Events' });
    await expect(lifeEventsCard).toBeVisible({ timeout: 10000 });
    const collapseItems = lifeEventsCard.locator('.ant-collapse-item');
    await expect(collapseItems.first()).toBeVisible({ timeout: 10000 });
    const firstPanel = collapseItems.first();
    await firstPanel.locator('.ant-collapse-header').click();

    const typeItems = firstPanel.locator('.ant-list-item');
    await expect(typeItems.first()).toBeVisible({ timeout: 15000 });
    const secondType = typeItems.nth(1);
    await expect(secondType).toBeVisible({ timeout: 5000 });
    const upArrow = secondType.locator('button', { has: page.locator('.anticon-arrow-up') });
    await expect(upArrow).toBeVisible({ timeout: 5000 });
    const [posResp] = await Promise.all([
      page.waitForResponse(
        (resp) => resp.url().includes('/position') && resp.request().method() === 'POST' && resp.status() < 400
      ),
      upArrow.click(),
    ]);
    const posBody = await posResp.json();
    expect(posBody.success).toBe(true);
    await page.waitForResponse(
      (resp) => resp.url().includes('/lifeEventCategories') && resp.request().method() === 'GET' && resp.status() < 400,
      { timeout: 10000 }
    ).catch(() => null);
    await page.waitForLoadState('networkidle');
    // After refetch, panel may have closed — re-expand if needed
    const isExpanded = await firstPanel.evaluate(el => el.classList.contains('ant-collapse-item-active'));
    if (!isExpanded) {
      await firstPanel.locator('.ant-collapse-header').click();
      await expect(firstPanel.locator('.ant-list-item').first()).toBeVisible({ timeout: 15000 });
    }

    const refreshedFirstType = firstPanel.locator('.ant-list-item').first();
    const firstTypeUpArrow = refreshedFirstType.locator('button', { has: page.locator('.anticon-arrow-up') });
    await expect(firstTypeUpArrow).toBeDisabled({ timeout: 5000 });
  });
});
