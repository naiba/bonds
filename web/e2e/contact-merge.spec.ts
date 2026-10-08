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
    const categoriesResponse = await request.get(
      apiUrl(`/vaults/${vaultId}/settings/activityCategories`),
      { headers },
    );
    expect(categoriesResponse.status()).toBe(200);
    const categories: { types?: { id: number }[] }[] = (
      await categoriesResponse.json()
    ).data;
    const activityTypeId = categories.flatMap(
      (category) => category.types ?? [],
    )[0]?.id;
    expect(activityTypeId).toBeDefined();
    const activityResponse = await request.post(
      apiUrl(`/vaults/${vaultId}/activities`),
      {
        headers,
        data: {
          title: "Synthetic gardening memory",
          activity_type_id: activityTypeId,
          primary_contact_id: contacts[1].id,
          description: `Walked with @[Alicia Chen](contact:${contacts[1].id}).`,
          description_format: "markdown",
        },
      },
    );
    expect(activityResponse.status()).toBe(201);
    const activityId = (await activityResponse.json()).data.id;
    const journalResponse = await request.post(
      apiUrl(`/vaults/${vaultId}/journals`),
      {
        headers,
        data: { name: "Synthetic merge memories" },
      },
    );
    expect(journalResponse.status()).toBe(201);
    const journalId = (await journalResponse.json()).data.id;
    const postPath = `/vaults/${vaultId}/journals/${journalId}/posts`;
    const postContentFormat =
      viewport.name === "desktop" ? "markdown" : "plain";
    const postResponse = await request.post(apiUrl(postPath), {
      headers,
      data: {
        title: "Garden walk before merging",
        written_at: "2026-01-02T00:00:00Z",
        sections: [
          {
            position: 1,
            label: "Memory",
            content: `Walked with @[Alice Work](contact:${contacts[2].id}).`,
            content_format: postContentFormat,
          },
        ],
      },
    });
    expect(postResponse.status()).toBe(201);
    const postId = (await postResponse.json()).data.id;
    if (viewport.name === "desktop") {
      const associationOnlyEdit = await request.put(
        apiUrl(`${postPath}/${postId}`),
        {
          headers,
          data: { title: "Garden walk before merging", contact_ids: [] },
        },
      );
      expect(associationOnlyEdit.status()).toBe(200);
      expect((await associationOnlyEdit.json()).data.contacts).toEqual([
        expect.objectContaining({ id: contacts[2].id }),
      ]);
    } else {
      // Moving away removes the vault-scoped pivot but keeps the original
      // journal text. Returning the contact must make that text mergeable.
      const travelVault = await request.post(apiUrl("/vaults"), {
        headers,
        data: { name: "Synthetic contact travel" },
      });
      expect(travelVault.status()).toBe(201);
      const travelVaultId = (await travelVault.json()).data.id;
      for (const [from, to] of [
        [vaultId, travelVaultId],
        [travelVaultId, vaultId],
      ]) {
        const move = await request.post(
          apiUrl(`/vaults/${from}/contacts/${contacts[2].id}/move`),
          {
            headers,
            data: { target_vault_id: to },
          },
        );
        expect(move.status()).toBe(200);
      }
    }
    const beforePostMerge = await request.get(apiUrl(`${postPath}/${postId}`), {
      headers,
    });
    expect(beforePostMerge.status()).toBe(200);
    const originalArticle = (await beforePostMerge.json()).data;
    expect(originalArticle.sections[0].content).toBe(
      `Walked with @[Alice Work](contact:${contacts[2].id}).`,
    );
    expect(originalArticle.contacts ?? []).toHaveLength(
      viewport.name === "desktop" ? 1 : 0,
    );
    await page.goto("/login");
    await page.getByPlaceholder("Email").fill(email);
    await page
      .getByPlaceholder("Password", { exact: true })
      .fill("Password123!");
    await page.getByRole("button", { name: "Sign in", exact: true }).click();
    await expect(page).toHaveURL(/\/vaults$/);
    // Exercise the real Vditor completion: it consumes @ and saves a bare
    // Markdown link, unlike the legacy mention editor. API fixtures missed this.
    await page.goto(`/vaults/${vaultId}/contacts/${contacts[1].id}`);
    if (viewport.name === "mobile") {
      await page.getByRole("combobox", { name: "Jump to section" }).click();
      await page
        .locator(".ant-select-dropdown")
        .getByText("Notes and records", { exact: true })
        .click();
    } else {
      await page
        .getByRole("navigation", { name: "Contact sections" })
        .getByRole("button", { name: "Notes and records", exact: true })
        .click();
    }
    const notesCard = page.locator(".ant-card").filter({ hasText: /^Notes/ });
    await notesCard.getByRole("button", { name: /add/i }).click();
    const noteDialog = page.getByRole("dialog");
    await noteDialog.getByPlaceholder(/title/i).fill("Shared project");
    // Save code examples and a real completion together. Unknown and source
    // UUIDs inside code must neither block saving nor change during the merge.
    const literalText = `[Unknown](contact:550e8400-e29b-41d4-a716-446655440000) [Alicia Chen](contact:${contacts[1].id})`;
    const encodedLinks = (id: string, escapedID = false) =>
      `[Encoded colon](contact\\:${id}) [Encoded entity](contact&#58;${id}) [Encoded ID](contact:${escapedID ? id.replaceAll("-", "\\-") : id})`;
    // These decoded addresses resemble internal parser markers but are not
    // contact links. They must not make either code example require a contact.
    const unrelatedDestinations = `[Unrelated](bonds&#45;contact-reference-0-contact:550e8400-e29b-41d4-a716-446655440000) [Another](bonds&#45;contact-reference-1-contact:${contacts[1].id})`;
    const notePrefix = `Literal examples: \`${literalText}\` `;
    await noteDialog
      .getByRole("textbox", { name: /write your note/i })
      .pressSequentially(`${notePrefix}Met @Alicia`, { delay: 10 });
    await page
      .locator(".vditor-hint")
      .getByRole("button", { name: "Alicia Chen", exact: true })
      .click();
    // Insert the encoded Markdown as a single pasted text input. Vditor IR
    // reparses entity keystrokes mid-token and can move the typing caret.
    await noteDialog.getByRole("textbox", { name: /write your note/i })
      .press("ControlOrMeta+End");
    await page.keyboard.insertText(` ${encodedLinks(contacts[1].id, true)} ${unrelatedDestinations}`);
    const noteSaved = page.waitForResponse(
      (response) =>
        response.url().endsWith(`/contacts/${contacts[1].id}/notes`) &&
        response.request().method() === "POST",
    );
    await noteDialog.getByRole("button", { name: "Save", exact: true }).click();
    const noteResponse = await noteSaved;
    expect(noteResponse.status()).toBe(201);
    const savedNote = (await noteResponse.json()).data;
    const noteId = savedNote.id;
    expect(savedNote.body.trim()).toBe(
      `${notePrefix}Met [Alicia Chen](contact:${contacts[1].id}) ${encodedLinks(contacts[1].id, true)} ${unrelatedDestinations}`,
    );
    expect(savedNote.rendered_body.match(/data-bonds-contact=/g)).toHaveLength(
      4,
    );
    await expect(
      notesCard.getByRole("link", { name: "Alicia Chen", exact: true }),
    ).toHaveAttribute("href", `/vaults/${vaultId}/contacts/${contacts[1].id}`);
    await expect(notesCard.locator("code")).toHaveText(literalText);
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
          body: savedNote.body
            .replace(encodedLinks(contacts[1].id, true), encodedLinks(retainedId, retainedId === contacts[1].id))
            .replace(
            `Met [Alicia Chen](contact:${contacts[1].id})`,
            `Met [Alicia Chen](contact:${retainedId})`,
          ),
        }),
      ]),
    );
    expect(
      noteData.filter((note: { title: string }) =>
        note.title.startsWith("Merged contact:"),
      ),
    ).toHaveLength(0);
    expect(noteData).toHaveLength(1);
    expect(
      noteData[0].rendered_body.match(/data-bonds-contact=/g),
    ).toHaveLength(4);
    for (const contact of contacts.filter(
      (contact) => contact.id !== retainedId,
    )) {
      const removed = await request.get(
        apiUrl(`/vaults/${vaultId}/contacts/${contact.id}`),
        { headers },
      );
      expect(removed.status()).toBe(404);
    }
    // Both migrated ownership and rendered content links must resolve to the survivor.
    if (viewport.name === "mobile") {
      await page.getByRole("combobox", { name: "Jump to section" }).click();
      // Ant Select virtualizes its accessibility options; choose the visible item.
      await page
        .locator(".ant-select-dropdown")
        .getByText("Notes and records", { exact: true })
        .click();
    } else {
      await page
        .getByRole("navigation", { name: "Contact sections" })
        .getByRole("button", { name: "Notes and records", exact: true })
        .click();
    }
    for (const name of ["Encoded colon", "Encoded entity", "Encoded ID"]) {
      const encodedLink = page.getByRole("link", { name, exact: true });
      await expect(encodedLink).toHaveAttribute(
        "href", `/vaults/${vaultId}/contacts/${retainedId}`,
      );
      await encodedLink.click();
      await expect(page).toHaveURL(`/vaults/${vaultId}/contacts/${retainedId}`);
      const resolved = await request.get(
        apiUrl(`/vaults/${vaultId}/contacts/${retainedId}`), { headers },
      );
      expect(resolved.status()).toBe(200);
    }
    const noteLink = page
      .getByRole("link", { name: "Alicia Chen", exact: true })
      .first();
    await expect(noteLink).toBeVisible();
    await noteLink.scrollIntoViewIfNeeded();
    await expect(noteLink).toBeInViewport();
    await expect(noteLink).toHaveAttribute(
      "href",
      `/vaults/${vaultId}/contacts/${retainedId}`,
    );
    if (evidenceDirectory)
      await page.screenshot({
        path: join(
          evidenceDirectory,
          `contact-merge-${viewport.name}-note.png`,
        ),
        animations: "disabled",
      });
    await noteLink.click();
    await expect(page).toHaveURL(new RegExp(`/contacts/${retainedId}$`));
    const migratedActivityResponse = await request.get(
      apiUrl(`/vaults/${vaultId}/activities/${activityId}`),
      { headers },
    );
    expect(migratedActivityResponse.status()).toBe(200);
    const migratedActivity = (await migratedActivityResponse.json()).data;
    expect(migratedActivity.description).toBe(
      `Walked with @[Alicia Chen](contact:${retainedId}).`,
    );
    expect(migratedActivity.participants).toEqual([
      expect.objectContaining({ id: retainedId }),
    ]);
    expect(migratedActivity.mentioned_contacts).toEqual([
      expect.objectContaining({ id: retainedId }),
    ]);
    await page.goto(`/vaults/${vaultId}/activities/${activityId}`);
    const activityLink = page
      .locator("p")
      .filter({ hasText: "Walked with" })
      .getByRole("link");
    await expect(activityLink).toHaveAttribute(
      "href",
      `/vaults/${vaultId}/contacts/${retainedId}`,
    );
    if (evidenceDirectory)
      await page.screenshot({
        path: join(
          evidenceDirectory,
          `contact-merge-${viewport.name}-activity.png`,
        ),
        animations: "disabled",
      });
    await activityLink.click();
    await expect(page).toHaveURL(new RegExp(`/contacts/${retainedId}$`));
    await expect(
      page.getByText("Alice Chen", { exact: true }).first(),
    ).toBeVisible();
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
    // A refreshed article must remain editable: its body is authoritative for
    // contact associations, so moving only contact_post is insufficient.
    const mergedPost = await request.get(apiUrl(`${postPath}/${postId}`), {
      headers,
    });
    expect(mergedPost.status()).toBe(200);
    const mergedArticle = (await mergedPost.json()).data;
    expect(mergedArticle.sections).toHaveLength(1);
    expect(mergedArticle.sections[0].content).toBe(
      `Walked with @[Alice Work](contact:${retainedId}).`,
    );
    expect(mergedArticle.sections[0].content_format).toBe(postContentFormat);
    await page.goto(`${postPath}/${postId}`);
    await expect(
      page.getByText("Garden walk before merging", { exact: true }).first(),
    ).toBeVisible();
    await page.getByRole("button", { name: "edit Edit", exact: true }).click();
    await page.getByPlaceholder("Post title").fill("Garden walk after merging");
    const saved = page.waitForResponse(
      (response) =>
        response.url().endsWith(`${postPath}/${postId}`) &&
        response.request().method() === "PUT",
    );
    await page.getByRole("button", { name: "Save", exact: true }).click();
    expect((await saved).status()).toBe(200);
    await expect(
      page.getByText("Garden walk after merging", { exact: true }).first(),
    ).toBeVisible();
    const refreshedPost = await request.get(apiUrl(`${postPath}/${postId}`), {
      headers,
    });
    expect(refreshedPost.status()).toBe(200);
    const article = (await refreshedPost.json()).data;
    expect(article.contacts).toEqual([
      expect.objectContaining({ id: retainedId }),
    ]);
    expect(article.sections).toHaveLength(1);
    // The existing editor upgrades plain text to Markdown and escapes prose
    // punctuation. The merge itself preserves the original format above.
    expect(article.sections[0].content).toBe(
      `Walked with @[Alice Work](contact:${retainedId})${postContentFormat === "plain" ? "\\." : "."}`,
    );
    expect(article.sections[0].content_format).toBe("markdown");
    expect(article.title).toBe("Garden walk after merging");
    await expect(
      page.locator("p").filter({ hasText: "Walked with" }),
    ).toHaveText("Walked with Alice Chen.");
    expect(pageErrors).toEqual([]);
    if (evidenceDirectory)
      await page.screenshot({
        path: join(
          evidenceDirectory,
          `contact-merge-${viewport.name}-journal.png`,
        ),
        fullPage: false,
        animations: "disabled",
      });
  });
}
