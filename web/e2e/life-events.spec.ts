import { test, expect } from '@playwright/test';
import { apiUrl } from './api-base-url';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

type ApiResponse<T> = {
  data: T;
};

type LifeEventCategory = {
  types?: Array<{ id?: number }>;
};

type ContactResponse = {
  id?: string;
};

type TimelineEventResponse = {
  id?: number;
};

async function setupVault(page: import('@playwright/test').Page) {
  const email = uniqueEmail('levt');
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
  const createVaultResp = page.waitForResponse(
    (resp) => resp.url().includes('/vaults') && resp.request().method() === 'POST' && resp.status() < 400
  );
  await page.getByRole('button', { name: /create vault/i }).click();
  await createVaultResp;
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');
}

async function getVaultId(page: import('@playwright/test').Page): Promise<string> {
  const match = page.url().match(/\/vaults\/([a-f0-9-]{36})/);
  if (!match) throw new Error(`Could not extract vault ID from URL: ${page.url()}`);
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
  const resp = await page.request.post(apiUrl(`/vaults/${vaultId}/contacts`), {
    headers: { Authorization: `Bearer ${token}` },
    data: { first_name: firstName, last_name: lastName },
  });
  expect(resp.ok()).toBeTruthy();
  const body = await resp.json() as ApiResponse<ContactResponse>;
  if (!body.data.id) throw new Error(`Contact create response missing id for ${firstName} ${lastName}`);
  return body.data.id;
}

async function getLifeEventTypeId(page: import('@playwright/test').Page, vaultId: string, token: string): Promise<number> {
  const resp = await page.request.get(apiUrl(`/vaults/${vaultId}/settings/lifeEventCategories`), {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.ok()).toBeTruthy();
  const body = await resp.json() as ApiResponse<LifeEventCategory[]>;
  const typeId = body.data.flatMap((category) => category.types ?? []).find((type) => type.id)?.id;
  if (!typeId) throw new Error('No seeded life event type found');
  return typeId;
}

async function createParticipantLifeEventViaAPI(
  page: import('@playwright/test').Page,
  vaultId: string,
  token: string,
  contactId: string,
  participantId: string,
  typeId: number,
) {
  const timelineResp = await page.request.post(apiUrl(`/vaults/${vaultId}/contacts/${contactId}/timelineEvents`), {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      started_at: '2026-06-15T00:00:00Z',
      label: 'Shared Summer Trip',
      participants: [participantId],
    },
  });
  expect(timelineResp.ok()).toBeTruthy();
  const timelineBody = await timelineResp.json() as ApiResponse<TimelineEventResponse>;
  const timelineId = timelineBody.data.id;
  if (!timelineId) throw new Error('Timeline create response missing id');

  const lifeEventResp = await page.request.post(apiUrl(`/vaults/${vaultId}/contacts/${contactId}/timelineEvents/${timelineId}/lifeEvents`), {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      life_event_type_id: typeId,
      happened_at: '2026-06-20T00:00:00Z',
      calendar_type: 'gregorian',
      summary: 'Shared festival memory',
      description: 'A participant-linked life event',
      participants: [participantId],
    },
  });
  expect(lifeEventResp.ok()).toBeTruthy();
}

async function navigateToContactLifeGoals(page: import('@playwright/test').Page, vaultId: string, contactId: string) {
  await page.goto(`/vaults/${vaultId}/contacts/${contactId}`);
  await page.waitForLoadState('networkidle');
  await page.locator('.ant-segmented-item-label').getByText('Full view', { exact: true }).click();
  const lifeGoalsTab = page.getByRole('tab', { name: 'Life & goals' });
  await expect(lifeGoalsTab).toBeVisible({ timeout: 10000 });
  await lifeGoalsTab.click();
  await page.waitForLoadState('networkidle');
}

async function expectParticipantLifeEventVisible(page: import('@playwright/test').Page, participantName: string) {
  const lifeEventsCard = page.locator('.ant-card').filter({ hasText: 'Life Events' });
  const timelinePanel = lifeEventsCard.locator('.ant-collapse-item').filter({ hasText: 'Shared Summer Trip' }).first();
  await expect(timelinePanel).toBeVisible({ timeout: 10000 });
  await expect(timelinePanel.getByText(participantName)).toBeVisible({ timeout: 10000 });

  const isExpanded = await timelinePanel.evaluate((element) => element.classList.contains('ant-collapse-item-active'));
  if (!isExpanded) {
    await timelinePanel.locator('.ant-collapse-header').click();
  }
  await expect(timelinePanel.getByText('Shared festival memory')).toBeVisible({ timeout: 10000 });
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

  test('should show participant life events on each participant contact timeline', async ({ page }) => {
    await setupVault(page);
    const vaultId = await getVaultId(page);
    const token = await getAuthToken(page);
    const contactAId = await createContactViaAPI(page, vaultId, token, 'AliceLife', 'Host');
    const contactBId = await createContactViaAPI(page, vaultId, token, 'BobLife', 'Participant');
    const typeId = await getLifeEventTypeId(page, vaultId, token);

    await createParticipantLifeEventViaAPI(page, vaultId, token, contactAId, contactBId, typeId);

    await navigateToContactLifeGoals(page, vaultId, contactAId);
    await expectParticipantLifeEventVisible(page, 'BobLife Participant');

    await navigateToContactLifeGoals(page, vaultId, contactBId);
    await expectParticipantLifeEventVisible(page, 'AliceLife Host');
  });
});
