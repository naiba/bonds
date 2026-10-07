import { test, expect } from "@playwright/test";
import type { Locator } from "@playwright/test";
import { mkdir } from "node:fs/promises";
import { join } from "node:path";
import { apiUrl } from "./api-base-url";

// React controls these radios. Assert the committed state with Playwright's
// retrying assertion instead of check()'s immediate post-click DOM inspection.
async function selectMergeOption(option: Locator) {
  await option.click();
  await expect(option).toBeChecked();
}

for (const viewport of [
  { name: "desktop", width: 1440, height: 1000 },
  { name: "mobile", width: 390, height: 844 },
]) {
  test(`merge selected contacts after review on ${viewport.name}`, async ({
    page,
    request,
  }) => {
    await page.setViewportSize(viewport);
    const pageErrors: string[] = [];
    page.on("pageerror", (error) => pageErrors.push(error.message));
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
    await expect(
      dialog
        .getByRole("radiogroup", { name: "Choose the contact to keep" })
        .getByRole("radio"),
    ).toHaveCount(3);
    await selectMergeOption(
      dialog
        .getByRole("radiogroup", { name: "Choose the contact to keep" })
        .getByRole("radio", { name: "Alicia Chen", exact: true }),
    );
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
    const targetIndex = viewport.name === "mobile" ? 1 : 0;
    const retainedId = contacts[targetIndex].id;
    await selectMergeOption(
      dialog
        .getByRole("radiogroup", { name: "Choose the contact to keep" })
        .getByRole("radio", {
          name: targetIndex === 0 ? "Alice Chen" : "Alicia Chen",
          exact: true,
        }),
    );
    const chooseFields = async () => {
      const confirm = dialog.getByRole("button", {
        name: "Confirm merge",
        exact: true,
      });
      await expect(confirm).toBeDisabled();
      await selectMergeOption(
        dialog
          .getByRole("radiogroup", { name: "First name", exact: true })
          .getByRole("radio", { name: "Alice (Alice Chen)", exact: true }),
      );
      await selectMergeOption(
        dialog
          .getByRole("radiogroup", { name: "Last name", exact: true })
          .getByRole("radio", {
            name:
              targetIndex === 0 ? "Chen (Alice Chen)" : "Chen (Alicia Chen)",
            exact: true,
          }),
      );
      await selectMergeOption(
        dialog
          .getByRole("radiogroup", { name: "Nickname", exact: true })
          .getByRole("radio", { name: "Ally (Alicia Chen)", exact: true }),
      );
      await expect(confirm).toBeDisabled();
      await dialog
        .getByRole("checkbox", {
          name: "I reviewed these changes and accept discarding unselected profile values.",
        })
        .check();
      await expect(confirm).toBeEnabled();
    };
    await chooseFields();
    await expect(
      dialog.getByText("Remove 2 source contact(s).", { exact: true }),
    ).toBeVisible();
    await expect(
      dialog.getByText(
        targetIndex === 0 ? "Move 1 note(s)." : "What will change",
        { exact: true },
      ),
    ).toBeVisible();
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
          `contact-merge-${viewport.name}-effects.png`,
        ),
        fullPage: false,
        animations: "disabled",
      });
      await dialog
        .getByRole("radiogroup", { name: "Choose the contact to keep" })
        .scrollIntoViewIfNeeded();
      await page.screenshot({
        path: join(
          evidenceDirectory,
          `contact-merge-${viewport.name}-review.png`,
        ),
        fullPage: false,
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
      await chooseFields();
    }
    await dialog
      .getByRole("button", { name: "Confirm merge", exact: true })
      .click();
    await expect(page).toHaveURL(new RegExp(`/contacts/${retainedId}$`));
    await expect(
      page.getByText("Alice Chen", { exact: true }).first(),
    ).toBeVisible();
    const merged = await request.get(
      apiUrl(`/vaults/${vaultId}/contacts/${retainedId}`),
      { headers },
    );
    expect(merged.status()).toBe(200);
    expect((await merged.json()).data).toMatchObject({
      id: retainedId,
      first_name: "Alice",
      last_name: "Chen",
      nickname: "Ally",
    });
    const notes = await request.get(
      apiUrl(`/vaults/${vaultId}/contacts/${retainedId}/notes`),
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
    ).toHaveLength(0);
    expect(noteData).toHaveLength(1);
    for (const contact of contacts.filter(
      (contact) => contact.id !== retainedId,
    )) {
      const removed = await request.get(
        apiUrl(`/vaults/${vaultId}/contacts/${contact.id}`),
        { headers },
      );
      expect(removed.status()).toBe(404);
    }
    await page.goto(`/vaults/${vaultId}/contacts`);
    await expect(table.getByText("Alice Chen", { exact: true })).toBeVisible();
    await expect(table.getByRole("row")).toHaveCount(2);
    expect(pageErrors).toEqual([]);
    if (evidenceDirectory)
      await page.screenshot({
        path: join(
          evidenceDirectory,
          `contact-merge-${viewport.name}-result.png`,
        ),
        fullPage: false,
        animations: "disabled",
      });
  });
}
