import { test, expect } from "@playwright/test";
import { mkdir } from "node:fs/promises";
import { join } from "node:path";
import { apiUrl } from "./api-base-url";

for (const viewport of [
  { name: "desktop", width: 1440, height: 1000 },
  { name: "mobile", width: 390, height: 844 },
]) {
  test(`merge selected contacts after review on ${viewport.name}`, async ({
    page,
    request,
  }) => {
    await page.setViewportSize(viewport);
    const email = `contact-merge-${viewport.name}-${Date.now()}@example.test`;
    const registration = await request.post(apiUrl("/auth/register"), {
      data: {
        first_name: "Merge",
        last_name: "Tester",
        email,
        password: "Password123!",
      },
    });
    expect(registration.status()).toBe(201);
    const auth = (await registration.json()).data;
    const headers = { Authorization: `Bearer ${auth.token}` };
    const vaultResponse = await request.post(apiUrl("/vaults"), {
      headers,
      data: { name: "Synthetic contact merge" },
    });
    expect(vaultResponse.status()).toBe(201);
    const vaultId = (await vaultResponse.json()).data.id;
    const contacts: { id: string; first_name: string }[] = [];
    for (const data of [
      { first_name: "Alice", last_name: "Chen" },
      { first_name: "Alicia", last_name: "Chen", nickname: "Ally" },
      { first_name: "Alice", last_name: "Work", nickname: "Work contact" },
    ]) {
      const response = await request.post(
        apiUrl(`/vaults/${vaultId}/contacts`),
        { headers, data },
      );
      expect(response.status()).toBe(201);
      contacts.push((await response.json()).data);
    }
    const noteResponse = await request.post(
      apiUrl(`/vaults/${vaultId}/contacts/${contacts[1].id}/notes`),
      {
        headers,
        data: {
          title: "Shared project",
          body: "Met at the synthetic gardening club.",
        },
      },
    );
    expect(noteResponse.status()).toBe(201);
    const noteId = (await noteResponse.json()).data.id;
    await page.goto("/login");
    await page.getByPlaceholder("Email").fill(email);
    await page
      .getByPlaceholder("Password", { exact: true })
      .fill("Password123!");
    await page.getByRole("button", { name: "Sign in", exact: true }).click();
    await expect(page).toHaveURL(/\/vaults$/);
    await page.goto(`/vaults/${vaultId}/contacts`);
    const table = page.getByRole("table");
    await expect(table.getByText("Alice Chen", { exact: true })).toBeVisible();
    await table
      .getByRole("row")
      .filter({ hasText: "Alice Chen" })
      .getByRole("checkbox")
      .check();
    await expect(
      page.getByRole("button", { name: "Merge contacts", exact: true }),
    ).toHaveCount(0);
    for (const name of ["Alicia Chen", "Alice Work"]) {
      await table
        .getByRole("row")
        .filter({ hasText: name })
        .getByRole("checkbox")
        .check();
    }
    await page
      .getByRole("button", { name: "Merge contacts", exact: true })
      .click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByRole("radio")).toHaveCount(3);
    await dialog
      .getByRole("radio", { name: "Alicia Chen — Ally", exact: true })
      .check();
    await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(dialog).not.toBeVisible();
    const beforeMerge = await request.get(
      apiUrl(`/vaults/${vaultId}/contacts`),
      { headers },
    );
    expect((await beforeMerge.json()).data).toHaveLength(3);
    await page
      .getByRole("button", { name: "Merge contacts", exact: true })
      .click();
    await dialog
      .getByRole("radio", { name: "Alice Chen", exact: true })
      .check();
    await expect(
      dialog.getByText(
        "The other selected contacts will be removed. This cannot be undone.",
      ),
    ).toBeVisible();
    const evidenceDirectory = process.env.CONTACT_MERGE_SCREENSHOTS;
    if (evidenceDirectory) {
      await mkdir(evidenceDirectory, { recursive: true });
      await page.screenshot({
        path: join(
          evidenceDirectory,
          `contact-merge-${viewport.name}-review.png`,
        ),
        fullPage: true,
        animations: "disabled",
      });
    }
    if (viewport.name === "desktop") {
      await page.route(
        "**/contacts/merge",
        async (route) =>
          route.fulfill({
            status: 500,
            contentType: "application/json",
            body: JSON.stringify({
              success: false,
              error: { code: "INTERNAL_ERROR", message: "Merge unavailable" },
            }),
          }),
        { times: 1 },
      );
      await dialog
        .getByRole("button", { name: "Confirm merge", exact: true })
        .click();
      await expect(
        page.getByText("Merge unavailable", { exact: true }),
      ).toBeVisible();
      await expect(dialog).toBeVisible();
      const unchanged = await request.get(
        apiUrl(`/vaults/${vaultId}/contacts`),
        { headers },
      );
      expect((await unchanged.json()).data).toHaveLength(3);
    }
    await dialog
      .getByRole("button", { name: "Confirm merge", exact: true })
      .click();
    await expect(page).toHaveURL(new RegExp(`/contacts/${contacts[0].id}$`));
    await expect(
      page.getByText("Alice Chen", { exact: true }).first(),
    ).toBeVisible();
    const merged = await request.get(
      apiUrl(`/vaults/${vaultId}/contacts/${contacts[0].id}`),
      { headers },
    );
    expect(merged.status()).toBe(200);
    expect((await merged.json()).data).toMatchObject({
      id: contacts[0].id,
      first_name: "Alice",
      last_name: "Chen",
      nickname: "Ally",
    });
    const notes = await request.get(
      apiUrl(`/vaults/${vaultId}/contacts/${contacts[0].id}/notes`),
      { headers },
    );
    expect(notes.status()).toBe(200);
    const noteData = (await notes.json()).data;
    expect(noteData).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          id: noteId,
          body: "Met at the synthetic gardening club.",
        }),
      ]),
    );
    expect(
      noteData.filter((note: { title: string }) =>
        note.title.startsWith("Merged contact:"),
      ),
    ).toHaveLength(2);
    for (const contact of contacts.slice(1)) {
      const removed = await request.get(
        apiUrl(`/vaults/${vaultId}/contacts/${contact.id}`),
        { headers },
      );
      expect(removed.status()).toBe(404);
    }
    await page.goto(`/vaults/${vaultId}/contacts`);
    await expect(table.getByText("Alice Chen", { exact: true })).toBeVisible();
    await expect(table.getByRole("row")).toHaveCount(2);
    if (evidenceDirectory)
      await page.screenshot({
        path: join(
          evidenceDirectory,
          `contact-merge-${viewport.name}-result.png`,
        ),
        fullPage: true,
        animations: "disabled",
      });
  });
}
