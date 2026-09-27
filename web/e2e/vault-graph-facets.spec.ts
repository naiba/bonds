import { test, expect } from '@playwright/test';

test('graph facet tags keep a stable width after selection, including on narrow screens', async ({ page }) => {
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Graph');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(`graph-facet-${Date.now()}@example.com`);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/);

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Graph Vault');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/);

  // Keep this a browser layout test, independent of unrelated graph data.
  await page.route('**/api/vaults/*/relationships/graph?*', async (route) => {
    await route.fulfill({
      json: {
        success: true,
        data: {
          nodes: [], edges: [], components: 0, isolated_contacts: 0,
          external_relationships: 0, filtered_out: 0, truncated: false,
          facets: [{
            key: 'label',
            values: [
              { value: '1', label: 'Family', count: 12 },
              { value: '2', label: 'San Francisco', count: 12 },
            ],
          }],
        },
      },
    });
  });
  await page.goto(`${page.url()}/graph`);

  const facet = page.getByLabel('Labels');
  await expect(facet).toBeVisible();
  const select = facet.locator('xpath=ancestor::div[contains(concat(" ", normalize-space(@class), " "), " ant-select ")][1]');
  await facet.click();
  await page.getByTitle('Family (12)').click();
  await expect(select).toContainText('Family (12)');

  const widthSamples: number[] = [];
  const tagSamples: string[] = [];
  for (let sample = 0; sample < 20; sample++) {
    widthSamples.push((await select.boundingBox())?.width ?? 0);
    tagSamples.push(await select.innerText());
    await page.waitForTimeout(50);
  }
  expect(Math.max(...widthSamples) - Math.min(...widthSamples)).toBeLessThan(1);
  expect(new Set(tagSamples).size).toBe(1);
  expect(widthSamples[0]).toBe(220);

  await page.setViewportSize({ width: 320, height: 720 });
  const box = await select.boundingBox();
  expect(box).not.toBeNull();
  expect(box!.x + box!.width).toBeLessThanOrEqual(320);
});
