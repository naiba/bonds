import { test, expect } from "@playwright/test";

async function registerUser(page: import("@playwright/test").Page) {
  const email = `se-${Date.now()}@example.com`;
  await page.goto("/register");
  await page.getByPlaceholder("First name").fill("Settings");
  await page.getByPlaceholder("Last name").fill("Tester");
  await page.getByPlaceholder("Email").fill(email);
  await page.getByPlaceholder(/password/i).fill("password123");
  await page.getByRole("button", { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
}

async function createVaultAndOpenContactLayouts(
  page: import("@playwright/test").Page,
) {
  await registerUser(page);
  await page.getByRole("button", { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill("Layout Vault");
  await page
    .getByPlaceholder(/what is this vault/i)
    .fill("Contact layout testing");
  await page.getByRole("button", { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.goto(`${page.url()}/settings`);
  await page.waitForLoadState("networkidle");
  await expect(
    page.getByText("Contact view layouts", { exact: true }),
  ).toBeVisible({ timeout: 20000 });
  await expect(page.locator('input[value="Profile and contact"]')).toBeVisible({
    timeout: 20000,
  });
}

function contactLayoutSectionCard(
  page: import("@playwright/test").Page,
  sectionName: string,
) {
  return page
    .locator(`input[value="${sectionName}"]`)
    .locator(
      "xpath=ancestor::div[contains(concat(' ', normalize-space(@class), ' '), ' ant-card ')][1]",
    );
}

test.describe("Settings - WebAuthn and Modules", () => {
  test("should navigate to WebAuthn settings page", async ({ page }) => {
    await registerUser(page);
    await page.goto("/settings/webauthn");
    await expect(page.getByRole("heading", { level: 4 }).first()).toBeVisible({
      timeout: 10000,
    });
    await expect(page.locator(".ant-card")).toBeVisible({ timeout: 10000 });
  });

  test("should show register passkey button on WebAuthn page", async ({
    page,
  }) => {
    await registerUser(page);
    await page.goto("/settings/webauthn");
    await page.waitForLoadState("networkidle");
    await expect(
      page.getByRole("button").filter({ has: page.locator(".anticon-plus") }),
    ).toBeVisible({ timeout: 10000 });
  });

  test("should navigate to OAuth providers page", async ({ page }) => {
    await registerUser(page);
    await page.goto("/settings/oauth");
    await expect(page.getByRole("heading", { level: 4 }).first()).toBeVisible({
      timeout: 10000,
    });
    await expect(page.locator(".ant-card")).toBeVisible({ timeout: 10000 });
  });

  test("should navigate to storage info page", async ({ page }) => {
    await registerUser(page);
    await page.goto("/settings/storage");
    await expect(page.getByRole("heading", { level: 4 }).first()).toBeVisible({
      timeout: 10000,
    });
    await expect(page.locator(".ant-card").first()).toBeVisible({
      timeout: 10000,
    });
  });

  test("should navigate to users page", async ({ page }) => {
    await registerUser(page);
    await page.goto("/settings/users");
    await expect(
      page.getByRole("heading", { level: 4 }).getByText("Users"),
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator(".ant-card")).toBeVisible({ timeout: 10000 });
  });

  test("should show current user in users list", async ({ page }) => {
    await registerUser(page);
    await page.goto("/settings/users");
    await page.waitForLoadState("networkidle");
    await expect(
      page.getByRole("table").getByText("Settings Tester"),
    ).toBeVisible({ timeout: 10000 });
  });

  test("should navigate to personalize page and show currencies section", async ({
    page,
  }) => {
    await registerUser(page);
    await page.goto("/settings/personalize");
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("heading", { level: 4 }).first()).toBeVisible({
      timeout: 10000,
    });

    const currenciesPanel = page
      .locator(".ant-collapse-item")
      .filter({ hasText: "Currencies" });
    await expect(currenciesPanel).toBeVisible({ timeout: 10000 });
    await currenciesPanel.locator(".ant-collapse-header").click();

    await expect(page.getByPlaceholder(/search currencies/i)).toBeVisible({
      timeout: 10000,
    });
    await expect(page.getByRole("button", { name: /enable all/i })).toBeVisible(
      { timeout: 5000 },
    );
    await expect(
      page.getByRole("button", { name: /disable all/i }),
    ).toBeVisible({ timeout: 5000 });
  });

  test("should search and filter currencies", async ({ page }) => {
    await registerUser(page);
    await page.goto("/settings/personalize");
    await page.waitForLoadState("networkidle");

    const currenciesPanel = page
      .locator(".ant-collapse-item")
      .filter({ hasText: "Currencies" });
    await currenciesPanel.locator(".ant-collapse-header").click();

    const searchInput = page.getByPlaceholder(/search currencies/i);
    await expect(searchInput).toBeVisible({ timeout: 10000 });

    await expect(page.locator(".ant-switch").first()).toBeVisible({
      timeout: 20000,
    });

    await searchInput.fill("USD");
    await expect(
      page.locator(".ant-list-item").filter({ hasText: "USD" }),
    ).toBeVisible({ timeout: 10000 });
  });

  test("should show currency toggle switches", async ({ page }) => {
    await registerUser(page);

    await page.goto("/settings/personalize");
    await page.waitForLoadState("networkidle");

    const currenciesPanel = page
      .locator(".ant-collapse-item")
      .filter({ hasText: "Currencies" });
    await currenciesPanel.locator(".ant-collapse-header").click();

    await expect(page.getByPlaceholder(/search currencies/i)).toBeVisible({
      timeout: 10000,
    });

    await expect(page.locator(".ant-switch").first()).toBeVisible({
      timeout: 20000,
    });
  });

  test("should show stable modules and reorder controls in a Vault contact layout", async ({
    page,
  }) => {
    await createVaultAndOpenContactLayouts(page);
    const contactCard = contactLayoutSectionCard(page, "Profile and contact");

    for (const moduleName of [
      "Important dates",
      "Labels",
      "Quick Facts",
      "Religion",
      "Job information",
      "Addresses",
    ]) {
      await expect(
        contactCard.getByText(moduleName, { exact: true }),
      ).toBeVisible();
    }
    await expect(
      contactCard.getByRole("button", { name: "Move up" }).first(),
    ).toBeVisible();
    await expect(
      contactCard.getByRole("button", { name: "Move down" }).first(),
    ).toBeVisible();
  });

  test("should reorder a module and persist the complete Vault layout", async ({
    page,
  }) => {
    await createVaultAndOpenContactLayouts(page);
    const contactCard = contactLayoutSectionCard(page, "Profile and contact");
    const firstModule = contactCard
      .locator(".ant-card-body .ant-typography")
      .first();
    const firstModuleName = await firstModule.textContent();
    await contactCard
      .getByRole("button", { name: "Move down" })
      .first()
      .click();
    await expect(firstModule).not.toHaveText(firstModuleName ?? "");

    const saveResponse = page.waitForResponse(
      (response) =>
        response.url().includes("/contact-layout/templates/") &&
        response.url().endsWith("/layout") &&
        response.request().method() === "PUT" &&
        response.status() < 400,
    );
    await page.getByRole("button", { name: /save/i }).last().click();
    await saveResponse;
    await expect(
      page.getByText("Contact view saved", { exact: true }),
    ).toBeVisible();
  });

  test("should remove Religion without removing Job information", async ({
    page,
  }) => {
    await createVaultAndOpenContactLayouts(page);
    const contactCard = contactLayoutSectionCard(page, "Profile and contact");
    const religionRow = contactCard
      .getByText("Religion", { exact: true })
      .locator("xpath=parent::div");
    await religionRow.getByRole("button", { name: "Delete" }).click();

    await expect(
      contactCard.getByText("Religion", { exact: true }),
    ).toHaveCount(0);
    await expect(
      contactCard.getByText("Job information", { exact: true }),
    ).toBeVisible();

    const saveResponse = page.waitForResponse(
      (response) =>
        response.url().endsWith("/layout") &&
        response.request().method() === "PUT" &&
        response.status() < 400,
    );
    await page.getByRole("button", { name: /save/i }).last().click();
    const response = await saveResponse;
    const payload = response.request().postDataJSON() as {
      pages: Array<{ modules: Array<{ key: string }> }>;
    };
    const keys = payload.pages.flatMap((layoutPage) =>
      layoutPage.modules.map((module) => module.key),
    );
    expect(keys).not.toContain("religion");
    expect(keys).toContain("jobs");
  });

  test("should hide and restore a template page without deleting it", async ({
    page,
  }) => {
    await createVaultAndOpenContactLayouts(page);
    const networkCard = contactLayoutSectionCard(page, "Relationship network");
    const visibilitySwitch = networkCard.getByRole("switch");
    await expect(visibilitySwitch).toBeChecked();

    await visibilitySwitch.click();
    await expect(visibilitySwitch).not.toBeChecked();
    const hideResponse = page.waitForResponse(
      (response) =>
        response.url().endsWith("/layout") &&
        response.request().method() === "PUT" &&
        response.status() < 400,
    );
    await page.getByRole("button", { name: /save/i }).last().click();
    await hideResponse;
    await expect(
      contactLayoutSectionCard(page, "Relationship network").getByRole(
        "switch",
      ),
    ).not.toBeChecked();

    const restoredSwitch = contactLayoutSectionCard(
      page,
      "Relationship network",
    ).getByRole("switch");
    await restoredSwitch.click();
    const restoreResponse = page.waitForResponse(
      (response) =>
        response.url().endsWith("/layout") &&
        response.request().method() === "PUT" &&
        response.status() < 400,
    );
    await page.getByRole("button", { name: /save/i }).last().click();
    await restoreResponse;
    await expect(
      contactLayoutSectionCard(page, "Relationship network").getByRole(
        "switch",
      ),
    ).toBeChecked();
  });

  test("should add a layout section when randomUUID is unavailable", async ({
    page,
  }) => {
    await page.addInitScript(() => {
      Object.defineProperty(globalThis.crypto, "randomUUID", {
        configurable: true,
        value: undefined,
      });
    });
    await createVaultAndOpenContactLayouts(page);

    await page.getByRole("button", { name: "Add section" }).click();
    const modal = page
      .locator(".ant-modal")
      .filter({ hasText: /add section/i });
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.getByPlaceholder("Section name").fill("Memories");
    await modal.getByRole("button", { name: "OK" }).click();

    await expect(
      page.locator('input[aria-label="Section name"][value="Memories"]'),
    ).toBeVisible({
      timeout: 5000,
    });
    const saveResponse = page.waitForResponse(
      (response) =>
        response.url().endsWith("/layout") &&
        response.request().method() === "PUT" &&
        response.status() < 400,
    );
    await page.getByRole("button", { name: /save/i }).last().click();
    await saveResponse;
    await page.reload();
    await expect(
      page.locator('input[aria-label="Section name"][value="Memories"]'),
    ).toBeVisible({
      timeout: 10000,
    });
  });
});

test.describe("Settings - Personalize", () => {
  test("personalize excludes Vault-owned contact layouts", async ({ page }) => {
    await registerUser(page);

    await page.goto("/settings/personalize");
    await page.waitForLoadState("networkidle");
    await expect(
      page.getByRole("heading", { name: "Shared account data" }),
    ).toBeVisible();
    await expect(
      page.locator(".ant-collapse-item").filter({ hasText: /^Templates/ }),
    ).toHaveCount(0);
    await expect(
      page.locator(".ant-collapse-item").filter({ hasText: /^Modules/ }),
    ).toHaveCount(0);
  });
});
