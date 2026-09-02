import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement } from "react";
import ContactInfoModule from "@/pages/contact/modules/ContactInfoModule";
import NotesModule from "@/pages/contact/modules/NotesModule";
import PetsModule from "@/pages/contact/modules/PetsModule";
import { api } from "@/api";

vi.mock("@/api", () => ({
  api: {
    notes: {
      contactsNotesList: vi.fn(),
      contactsNotesCreate: vi.fn(),
      contactsNotesUpdate: vi.fn(),
      contactsNotesDelete: vi.fn(),
    },
    contactInformation: {
      contactsContactInformationList: vi.fn(),
      contactsContactInformationCreate: vi.fn(),
      contactsContactInformationUpdate: vi.fn(),
      contactsContactInformationDelete: vi.fn(),
    },
    personalize: { personalizeDetail: vi.fn() },
    pets: {
      contactsPetsList: vi.fn(),
      contactsPetsCreate: vi.fn(),
      contactsPetsUpdate: vi.fn(),
      contactsPetsDelete: vi.fn(),
      petCategoriesList: vi.fn(),
    },
    preferences: { preferencesList: vi.fn() },
  },
}));

vi.mock("@/components/markdown/MarkdownEditor", () => ({
  default: (props: {
    value: string;
    onChange: (value: string) => void;
    ariaLabel: string;
  }) => (
    <textarea
      aria-label={props.ariaLabel}
      value={props.value}
      onChange={(event) => props.onChange(event.target.value)}
    />
  ),
}));

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(api.notes.contactsNotesList).mockResolvedValue({
    data: [
      {
        id: 1,
        title: "Trip notes",
        body: "Packing list",
        body_format: "markdown",
        rendered_body: "<p>Packing list</p>",
        created_at: "2026-09-01T08:30:00Z",
      },
    ],
    meta: { page: 1, per_page: 15, total: 1, total_pages: 1 },
  });
  vi.mocked(
    api.contactInformation.contactsContactInformationList,
  ).mockResolvedValue({
    data: [{ id: 2, type_id: 4, kind: "Personal", data: "alice@example.com" }],
  });
  vi.mocked(api.personalize.personalizeDetail).mockResolvedValue({
    data: [{ id: 4, name: "Email" }],
  });
  vi.mocked(api.pets.contactsPetsList).mockResolvedValue({
    data: [
      { id: 3, name: "Milo", pet_category_id: 6, pet_category_name: "Cat" },
    ],
  });
  vi.mocked(api.pets.petCategoriesList).mockResolvedValue({
    data: [{ id: 6, name: "Cat" }],
  });
  vi.mocked(api.preferences.preferencesList).mockResolvedValue({
    data: { date_format: "YYYY-MM-DD", timezone: "UTC" },
  });
});

function renderModule(component: ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <ConfigProvider>
        <AntApp>{component}</AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

async function clickEdit(container: HTMLElement) {
  const button = container
    .querySelector(".anticon-edit")
    ?.closest<HTMLButtonElement>("button");
  expect(button).toBeInTheDocument();
  if (!button) throw new Error("Edit button was not rendered");
  await userEvent.setup().click(button);
}

describe("contact module editor modals", () => {
  it("edits notes in a modal instead of a form above the list", async () => {
    const view = renderModule(<NotesModule vaultId="v1" contactId="c1" />);
    await screen.findByText("Trip notes");
    await clickEdit(view.container);

    expect(await screen.findByRole("dialog")).toHaveTextContent("Edit Note");
    expect(screen.getByDisplayValue("Trip notes")).toBeInTheDocument();
  });

  it("edits contact information in a modal", async () => {
    const view = renderModule(
      <ContactInfoModule vaultId="v1" contactId="c1" />,
    );
    await screen.findByText("alice@example.com");
    await clickEdit(view.container);

    expect(await screen.findByRole("dialog")).toHaveTextContent(
      "Edit Contact Information",
    );
    expect(screen.getByDisplayValue("alice@example.com")).toBeInTheDocument();
  });

  it("edits pets in a modal", async () => {
    const view = renderModule(<PetsModule vaultId="v1" contactId="c1" />);
    await screen.findByText("Milo");
    await clickEdit(view.container);

    expect(await screen.findByRole("dialog")).toHaveTextContent("Edit Pet");
    expect(screen.getByDisplayValue("Milo")).toBeInTheDocument();
  });
});
