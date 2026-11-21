import { test, expect, type Locator, type Page } from "@playwright/test";

// Calendar dates must not move when the browser is east of UTC. Every case here
// runs in Asia/Shanghai (UTC+8), where "local midnight" and "this morning" are
// still the previous day in UTC.
test.use({ timezoneId: "Asia/Shanghai" });

let counter = 0;

async function setupVault(page: Page, prefix: string) {
  const email = prefix + "-" + Date.now() + "-" + ++counter + "@example.com";
  await page.goto("/register");
  await page.getByPlaceholder("First name").fill("Calendar");
  await page.getByPlaceholder("Last name").fill("Tester");
  await page.getByPlaceholder("Email").fill(email);
  await page.getByPlaceholder(/password/i).fill("password123");
  await page.getByRole("button", { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole("button", { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill("Calendar Vault");
  await page.getByPlaceholder(/what is this vault/i).fill("Calendar dates east of UTC");
  await page.getByRole("button", { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState("networkidle");
  return page.url();
}

async function createContact(page: Page, vaultURL: string, firstName: string) {
  await page.goto(vaultURL.replace(/\/$/, "") + "/contacts");
  await page.getByRole("button", { name: /add contact/i }).click();
  await page.getByPlaceholder("First name").fill(firstName);
  await page.getByPlaceholder("Last name").fill("User");
  await page.getByRole("button", { name: /create contact/i }).click();
  await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 10000 });
  const section = page
    .getByRole("navigation", { name: "Contact sections" })
    .getByRole("button", { name: "Notes and records", exact: true });
  await expect(section).toBeVisible({ timeout: 10000 });
  await section.click();
  await page.waitForLoadState("networkidle");
}

const monthNames = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];
const monthAbbreviations = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

// The calendar date in Shanghai for an instant.
function shanghaiDate(instant: Date) {
  const shifted = new Date(instant.getTime() + 8 * 60 * 60 * 1000);
  return { year: shifted.getUTCFullYear(), month: shifted.getUTCMonth() + 1, day: shifted.getUTCDate() };
}

// The option lists are virtualised: later months and days are not in the page
// until the list is scrolled to them.
async function pickVirtualisedOption(page: Page, select: Locator, label: RegExp) {
  await select.click();
  const dropdown = page.locator(".ant-select-dropdown:visible").last();
  const option = dropdown.locator(".ant-select-item-option").filter({ hasText: label });
  await dropdown.hover();
  for (let scrolled = 0; scrolled < 40 && (await option.count()) === 0; scrolled++) {
    await page.mouse.wheel(0, 120);
    await page.waitForTimeout(50);
  }
  await option.first().click();
}

test("a loan's due date picked in the morning is the day that is shown", async ({ page }) => {
  // 07:30 in Shanghai: the same instant is 23:30 of the previous day in UTC.
  const today = shanghaiDate(new Date());
  const morningInShanghai = new Date(Date.UTC(today.year, today.month - 1, today.day, 7, 30) - 8 * 60 * 60 * 1000);
  await page.clock.setFixedTime(morningInShanghai);
  const vaultURL = await setupVault(page, "loan-due-date");
  await createContact(page, vaultURL, "LoanDueDate");

  const loansCard = page.locator(".ant-card").filter({ hasText: /^Loans/ });
  await expect(loansCard).toBeVisible({ timeout: 10000 });
  await loansCard.getByRole("button", { name: /add/i }).click();
  const modal = page.locator(".ant-modal").filter({ hasText: /add loan/i });
  await expect(modal).toBeVisible({ timeout: 5000 });
  await modal.locator(".ant-form-item").filter({ hasText: /category/i }).locator(".ant-select").click();
  await page.locator(".ant-select-dropdown:visible .ant-select-item-option").filter({ hasText: "Item" }).click();
  await modal.locator(".ant-form-item").filter({ hasText: /name/i }).first().locator("input").fill("Borrowed tent");
  await modal.locator(".ant-form-item").filter({ hasText: /item name/i }).locator("input").fill("Tent");
  await modal.locator(".ant-picker").click();
  await page.locator(".ant-picker-dropdown:visible .ant-picker-cell-today").click();
  const created = page.waitForResponse(
    (response) => response.url().includes("/loans") && response.request().method() === "POST",
  );
  await page.locator(".ant-modal-footer .ant-btn-primary").click();
  expect((await created).status()).toBeLessThan(400);

  const expected =
    "Due " + monthAbbreviations[today.month - 1] + " " + String(today.day).padStart(2, "0") + ", " + today.year;
  await expect(loansCard.getByText("Borrowed tent")).toBeVisible({ timeout: 10000 });
  await expect(loansCard.getByText(/Due [A-Z][a-z]{2} \d{1,2}, \d{4}/)).toHaveText(new RegExp(expected));

  // It is the stored value that is right, not only this render.
  await page.reload();
  await page
    .getByRole("navigation", { name: "Contact sections" })
    .getByRole("button", { name: "Notes and records", exact: true })
    .click();
  await expect(loansCard.getByText(/Due [A-Z][a-z]{2} \d{1,2}, \d{4}/)).toHaveText(new RegExp(expected), {
    timeout: 10000,
  });
});

test("a month and day reminder keeps its day on the vault dashboard", async ({ page }) => {
  // The day after tomorrow in Shanghai, so the reminder is upcoming.
  const target = shanghaiDate(new Date(Date.now() + 2 * 24 * 60 * 60 * 1000));
  const vaultURL = await setupVault(page, "reminder-day");
  await createContact(page, vaultURL, "ReminderDay");

  const remindersCard = page.locator(".ant-card").filter({ hasText: /Reminders/ });
  await expect(remindersCard).toBeVisible({ timeout: 10000 });
  await remindersCard.getByRole("button", { name: /add/i }).click();
  const modal = page.locator(".ant-modal").filter({ hasText: /reminder/i });
  await expect(modal).toBeVisible({ timeout: 5000 });
  await modal.getByLabel(/label/i).fill("Day that must not move");
  await modal.getByText("Month & day", { exact: true }).click();
  const compactSelects = modal.locator(".ant-space-compact .ant-select");
  await expect(compactSelects).toHaveCount(2, { timeout: 5000 });
  await pickVirtualisedOption(page, compactSelects.nth(0), new RegExp("^" + monthNames[target.month - 1] + "$"));
  await modal.locator(".ant-modal-header").click();
  await pickVirtualisedOption(page, compactSelects.nth(1), new RegExp("^0?" + target.day + "$"));
  await modal.locator(".ant-modal-header").click();
  await expect(compactSelects.nth(0)).toContainText(monthNames[target.month - 1]);
  await expect(compactSelects.nth(1)).toContainText(new RegExp("^0?" + target.day + "$"));
  const frequency = modal.locator(".ant-form-item").filter({ hasText: /frequency/i });
  await frequency.locator(".ant-select").click();
  await page
    .locator(".ant-select-dropdown")
    .last()
    .locator(".ant-select-item-option")
    .filter({ hasText: /yearly/i })
    .click();
  await modal.locator(".ant-modal-header").click();
  const created = page.waitForResponse(
    (response) => response.url().includes("/reminders") && response.request().method() === "POST",
  );
  await page.locator(".ant-modal-footer .ant-btn-primary").click();
  expect((await created).status()).toBeLessThan(400);

  await page.goto(vaultURL);
  // The dashboard lists the reminder as "label (contact)" with its date beside
  // it; the innermost element holding that label is the row.
  const reminderRow = page
    .locator("div")
    .filter({ has: page.getByText(/^Day that must not move \(/) })
    .last();
  await expect(reminderRow).toBeVisible({ timeout: 15000 });
  await expect(reminderRow).toContainText(
    new RegExp(monthAbbreviations[target.month - 1] + " 0?" + target.day + "(?!\\d)"),
  );
});
