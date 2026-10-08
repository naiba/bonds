import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import ContactMentionEditor from "@/components/journal/ContactMentionEditor";
import {
  appendMissingContactMentions,
  contactIdsFromMentions,
  parseContactMentions,
  serializeContactMention,
} from "@/components/journal/contactMentionSerialization";
import type { JournalContactReference } from "@/components/journal/contactMentionTypes";

const CONTACT_ID = "550e8400-e29b-41d4-a716-446655440000";
const mockSelectableContacts = vi.fn();

vi.mock("@/api", () => ({
  api: {
    contacts: {
      contactsSelectableList: (...args: unknown[]) =>
        mockSelectableContacts(...args),
    },
  },
}));

function ContactMentionEditorFixture({
  onMentionSelect,
}: {
  readonly onMentionSelect: (contact: JournalContactReference) => void;
}) {
  const [value, setValue] = useState("");
  return (
    <ContactMentionEditor
      vaultId="v1"
      value={value}
      onChange={setValue}
      onMentionSelect={onMentionSelect}
      ariaLabel="Mention contact"
      placeholder="Write a mention"
    />
  );
}

describe("journal contact mention serialization", () => {
  it("loads contact choices as soon as a bare @ is typed", async () => {
    const contact = { id: CONTACT_ID, name: "Alice Example" };
    mockSelectableContacts.mockResolvedValue({ data: [contact] });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <ConfigProvider>
          <AntApp>
            <ContactMentionEditorFixture onMentionSelect={vi.fn()} />
          </AntApp>
        </ConfigProvider>
      </QueryClientProvider>,
    );
    const user = userEvent.setup();

    await user.type(
      screen.getByRole("textbox", { name: "Mention contact" }),
      "@",
    );

    expect(await screen.findByText(contact.name)).toBeInTheDocument();
    expect(mockSelectableContacts).toHaveBeenCalledWith("v1", { search: "" });
  });

  it("serializes a closing bracket into the marker inserted by the editor", async () => {
    const contact = { id: CONTACT_ID, name: "Research ] Team" };
    mockSelectableContacts.mockResolvedValue({ data: [contact] });
    const onMentionSelect = vi.fn();
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <ConfigProvider>
          <AntApp>
            <ContactMentionEditorFixture onMentionSelect={onMentionSelect} />
          </AntApp>
        </ConfigProvider>
      </QueryClientProvider>,
    );
    const user = userEvent.setup();
    const editor = screen.getByRole("textbox", { name: "Mention contact" });

    await user.type(editor, "@research");
    await user.click(await screen.findByText(contact.name));

    expect(editor).toHaveValue(`@[Research \\] Team](contact:${CONTACT_ID}) `);
    expect(onMentionSelect).toHaveBeenCalledWith(contact);
  });

  it("round-trips a closing bracket in a contact display name", () => {
    const serializedMention = serializeContactMention({
      id: CONTACT_ID,
      name: "Research ] Team",
    });

    expect(serializedMention.marker).toBe(
      `@[Research \\] Team](contact:${CONTACT_ID})`,
    );
    expect(parseContactMentions(serializedMention.marker)).toEqual([
      {
        marker: serializedMention.marker,
        displayName: "Research ] Team",
        contactId: CONTACT_ID,
        index: 0,
      },
    ]);
  });

  it("round-trips backslashes in a contact display name", () => {
    const serializedMention = serializeContactMention({
      id: CONTACT_ID,
      name: "Research \\ Team",
    });

    expect(serializedMention.marker).toBe(
      `@[Research \\\\ Team](contact:${CONTACT_ID})`,
    );
    expect(parseContactMentions(serializedMention.marker)[0]?.displayName).toBe(
      "Research \\ Team",
    );
  });

  it("normalizes carriage returns and newlines to spaces", () => {
    const serializedMention = serializeContactMention({
      id: CONTACT_ID,
      name: "Research\r\nLine\nBreak\rTeam",
    });

    expect(serializedMention.marker).toBe(
      `@[Research Line Break Team](contact:${CONTACT_ID})`,
    );
    expect(serializedMention.marker).not.toMatch(/[\r\n]/);
    expect(parseContactMentions(serializedMention.marker)[0]?.displayName).toBe(
      "Research Line Break Team",
    );
  });

  it("normalizes legacy associations into stable inline markers without duplicates", () => {
    const alice = { id: CONTACT_ID, name: "Alice" };
    const normalized = appendMissingContactMentions("Dinner together", [alice]);

    expect(normalized).toBe(`Dinner together @[Alice](contact:${CONTACT_ID})`);
    expect(appendMissingContactMentions(normalized, [alice])).toBe(normalized);
    expect(contactIdsFromMentions(normalized)).toEqual([CONTACT_ID]);
  });
});

// Vditor consumes the @ trigger; old saved links must remain discoverable too.
it("recognizes Vditor links and legacy mentions without adding duplicate associations", () => {
  const contact = { id: CONTACT_ID, name: "Research ] Team" };
  const { optionValue, marker } = serializeContactMention(contact);
  const body = `Met ${optionValue} and ${marker}.`;
  expect(parseContactMentions(body)).toEqual([
    {
      marker: optionValue,
      displayName: contact.name,
      contactId: CONTACT_ID,
      index: 4,
    },
    {
      marker,
      displayName: contact.name,
      contactId: CONTACT_ID,
      index: 4 + optionValue.length + 5,
    },
  ]);
  expect(contactIdsFromMentions(body)).toEqual([CONTACT_ID]);
  expect(appendMissingContactMentions(`Met ${optionValue}`, [contact])).toBe(
    `Met ${optionValue}`,
  );
  expect(
    parseContactMentions(
      `[ordinary](https://example.test/${CONTACT_ID}) [bad](contact:missing)`,
    ),
  ).toEqual([]);
});
