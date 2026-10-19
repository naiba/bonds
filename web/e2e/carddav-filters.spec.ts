import { expect, test } from "@playwright/test";
import { apiUrl } from "./api-base-url";

test("vault CardDAV filters persist, hide contacts without archiving and can be cleared", async ({
  page,
  request,
}, testInfo) => {
  const email = `carddav-filters-${Date.now()}@example.com`;
  const password = "synthetic-password123";
  const registration = await request.post(apiUrl("/auth/register"), {
    data: { first_name: "CardDAV", last_name: "Tester", email, password },
  });
  expect(registration.ok()).toBeTruthy();
  const auth = (await registration.json()).data;
  const headers = { Authorization: `Bearer ${auth.token}` };
  const vaultResponse = await request.post(apiUrl("/vaults"), {
    headers,
    data: { name: "CardDAV filter demonstration" },
  });
  expect(vaultResponse.ok()).toBeTruthy();
  const vault = (await vaultResponse.json()).data;
  const contactResponse = await request.post(
    apiUrl(`/vaults/${vault.id}/contacts`),
    {
      headers,
      data: { first_name: "Name only" },
    },
  );
  expect(contactResponse.ok()).toBeTruthy();
  const contact = (await contactResponse.json()).data;
  await page.goto("/login");
  await page.evaluate(
    (token: string) => localStorage.setItem("token", token),
    auth.token,
  );
  await page.goto(`/vaults/${vault.id}/settings`);
  await expect(page.getByText("CardDAV sync filters")).toBeVisible();
  await page.locator("#carddav_filter_fields").click();
  await page.getByText("Last name", { exact: true }).click();
  await page.getByText("CardDAV sync filters").click();
  const saveResponse = page.waitForResponse(
    (response) =>
      response.url().endsWith(`/vaults/${vault.id}/settings`) &&
      response.request().method() === "PUT",
  );
  await page
    .locator("form")
    .first()
    .getByRole("button", { name: /Save$/ })
    .click();
  expect((await saveResponse).status()).toBe(200);
  await page.reload();
  await expect(
    page.locator(".ant-select-selection-item").filter({ hasText: "Last name" }),
  ).toBeVisible();
  const davURL = new URL(apiUrl(`/vaults`));
  davURL.pathname = `/dav/addressbooks/${auth.user.id}/${vault.id}/${contact.id}.vcf`;
  const davHeaders = {
    Authorization: `Basic ${Buffer.from(`${email}:${password}`).toString("base64")}`,
  };
  const hidden = await request.get(davURL.toString(), { headers: davHeaders });
  expect(hidden.status()).toBe(404);
  const stillPresent = await request.get(
    apiUrl(`/vaults/${vault.id}/contacts/${contact.id}`),
    { headers },
  );
  expect(stillPresent.status()).toBe(200);
  expect((await stillPresent.json()).data.listed).toBe(true);
  await page.setViewportSize({ width: 1440, height: 1200 });
  await page.evaluate(() => window.scrollTo(0, 0));
  await page
    .locator("form")
    .first()
    .screenshot({
      path: testInfo.outputPath("carddav-filters.png"),
    });
  await page
    .locator(".ant-select-selection-item")
    .filter({ hasText: "Last name" })
    .locator(".ant-select-selection-item-remove")
    .click();
  const clearResponse = page.waitForResponse(
    (response) =>
      response.url().endsWith(`/vaults/${vault.id}/settings`) &&
      response.request().method() === "PUT",
  );
  await page
    .locator("form")
    .first()
    .getByRole("button", { name: /Save$/ })
    .click();
  expect((await clearResponse).status()).toBe(200);
  const restored = await request.get(davURL.toString(), {
    headers: davHeaders,
  });
  expect(restored.status()).toBe(200);
  expect(await restored.text()).toContain("FN:Name only");
});
