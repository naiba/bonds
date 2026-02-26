import { test, expect } from '@playwright/test';
import * as path from 'path';
import * as fs from 'fs';

async function registerAndSetup(page: import('@playwright/test').Page, prefix: string) {
  const email = `${prefix}-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('AvatarFix');
  await page.getByPlaceholder('Last name').fill('Test');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Avatar Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('test');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });

  await page.getByText('View all contacts').click();
  await expect(page).toHaveURL(/\/contacts/, { timeout: 5000 });

  await page.getByRole('button', { name: /add contact/i }).click();
  await page.getByPlaceholder('First name').fill(prefix);
  await page.getByPlaceholder('Last name').fill('Contact');
  await page.getByRole('button', { name: /create contact/i }).click();
  await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 10000 });
}

test.describe('Avatar Display (#11)', () => {
  test('avatar GET request should not have double /api prefix', async ({ page }) => {
    const avatarGetUrls: string[] = [];
    page.on('request', (req) => {
      if (req.url().includes('/avatar') && req.method() === 'GET') {
        avatarGetUrls.push(req.url());
      }
    });

    await registerAndSetup(page, 'urlcheck');
    await expect(page.locator('img[alt="Avatar"]')).toBeVisible({ timeout: 10000 });

    for (const url of avatarGetUrls) {
      expect(url).not.toContain('/api/api/');
    }
    expect(avatarGetUrls.length).toBeGreaterThan(0);
    await expect(page.locator('img[alt="Avatar"]')).toBeVisible({ timeout: 5000 });
  });

  test('uploaded avatar should display after upload', async ({ page }) => {
    await registerAndSetup(page, 'upload');
    await expect(page.locator('img[alt="Avatar"]')).toBeVisible({ timeout: 10000 });

    const tmpDir = fs.mkdtempSync('/tmp/avatar-e2e-');
    const imgPath = path.join(tmpDir, 'avatar.png');
    const pngData = Buffer.from(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==',
      'base64',
    );
    fs.writeFileSync(imgPath, pngData);

    const fileInput = page.locator('input[type="file"]').first();
    await fileInput.setInputFiles(imgPath);
    await expect(page.getByText(/avatar.*updated/i)).toBeVisible({ timeout: 10000 });
    await expect(page.locator('img[alt="Avatar"]')).toBeVisible({ timeout: 10000 });

    fs.rmSync(tmpDir, { recursive: true });
  });
});

test.describe('Avatar Display - Contact Detail', () => {
  test('avatar area is visible on contact detail and endpoint returns 200', async ({ page }) => {
    await registerAndSetup(page, 'avatar-detail');

    // The avatar area renders either an <img> (backend generates initials PNG) or
    // a <span> with text initials. Check for either one inside the circular container.
    const avatarImg = page.locator('img[alt="Avatar"]').first();
    const avatarInitials = page.locator('div').filter({ hasText: /^AT$/ }).first();
    await expect(avatarImg.or(avatarInitials)).toBeVisible({ timeout: 10000 });

    // Verify the avatar endpoint returns a 200 response via direct API call
    const contactUrl = page.url();
    const match = contactUrl.match(/\/vaults\/([^/]+)\/contacts\/([^/]+)/);
    expect(match).toBeTruthy();
    const [, vid, cid] = match!;

    const token = await page.evaluate(() => localStorage.getItem('token'));
    const resp = await page.request.get(`http://localhost:8080/api/vaults/${vid}/contacts/${cid}/avatar`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.status()).toBe(200);
  });
});
