import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { App as AntApp, ConfigProvider } from "antd";
import QuickFactsModule from "@/pages/contact/modules/QuickFactsModule";
import { api } from "@/api";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    quickFacts: {
      contactsQuickFactsList: vi.fn(),
      contactsQuickFactsCreate: vi.fn(),
      contactsQuickFactsUpdate: vi.fn(),
      contactsQuickFactsDelete: vi.fn(),
      contactsQuickFactsToggleUpdate: vi.fn(),
      contactsQuickFactsFileCreate: vi.fn(),
      contactsQuickFactsFileUpdate: vi.fn(),
    },
    preferences: {
      preferencesList: vi.fn(),
    },
  },
}));

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function renderQuickFactsModule() {
  return render(
    <QueryClientProvider client={createTestQueryClient()}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <QuickFactsModule vaultId="v1" contactId="c1" />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

const typedGroups = [
  {
    template_id: 1,
    template_label: "Favorite food",
    field_type: "text",
    facts: [{ id: 11, vault_quick_facts_template_id: 1, value_text: "Pizza", content: "Pizza" }],
  },
  {
    template_id: 2,
    template_label: "Score",
    field_type: "number",
    facts: [{ id: 12, vault_quick_facts_template_id: 2, value_number: 42, content: "42" }],
  },
  {
    template_id: 3,
    template_label: "Anniversary",
    field_type: "date",
    facts: [{ id: 13, vault_quick_facts_template_id: 3, value_date: "2026-01-15", content: "2026-01-15" }],
  },
  {
    template_id: 4,
    template_label: "Vegetarian",
    field_type: "select",
    select_options: ["Yes", "No"],
    facts: [{ id: 14, vault_quick_facts_template_id: 4, value_option: "Yes", content: "Yes" }],
  },
  {
    template_id: 5,
    template_label: "Portrait",
    field_type: "photo",
    facts: [
      {
        id: 15,
        vault_quick_facts_template_id: 5,
        file_id: 51,
        file: { id: 51, name: "portrait.jpg", mime_type: "image/jpeg", size: 4096, type: "photo" },
      },
    ],
  },
  {
    template_id: 6,
    template_label: "Passport",
    field_type: "document",
    facts: [
      {
        id: 16,
        vault_quick_facts_template_id: 6,
        file_id: 61,
        file: { id: 61, name: "passport.pdf", mime_type: "application/pdf", size: 8192, type: "document" },
      },
    ],
  },
];

describe("QuickFactsModule", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.setItem("token", "test-token");
    vi.mocked(api.preferences.preferencesList).mockResolvedValue({ data: { date_format: "MMM D, YYYY" } });
    vi.mocked(api.quickFacts.contactsQuickFactsCreate).mockResolvedValue({ data: { id: 100 } });
    vi.mocked(api.quickFacts.contactsQuickFactsUpdate).mockResolvedValue({ data: { id: 100 } });
    vi.mocked(api.quickFacts.contactsQuickFactsFileCreate).mockResolvedValue({ data: { id: 200 } });
    vi.mocked(api.quickFacts.contactsQuickFactsFileUpdate).mockResolvedValue({ data: { id: 200 } });
    vi.mocked(api.quickFacts.contactsQuickFactsDelete).mockResolvedValue(undefined);
    vi.mocked(api.quickFacts.contactsQuickFactsToggleUpdate).mockResolvedValue({ data: undefined });
  });

  it("renders text, number, date, select, photo, and document quick facts", async () => {
    vi.mocked(api.quickFacts.contactsQuickFactsList).mockResolvedValue({ data: typedGroups });

    renderQuickFactsModule();

    expect(await screen.findByText("Favorite food")).toBeInTheDocument();
    expect(screen.getByText("Pizza")).toBeInTheDocument();
    expect(screen.getByText("Score")).toBeInTheDocument();
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("Vegetarian")).toBeInTheDocument();
    expect(screen.getByText("Yes")).toBeInTheDocument();
    expect(await screen.findByText("Jan 15, 2026")).toBeInTheDocument();
    expect(screen.getByText("Portrait")).toBeInTheDocument();
    expect(screen.getByText("passport.pdf")).toBeInTheDocument();
  });

  it("creates a typed text quick fact", async () => {
    const user = userEvent.setup();
    vi.mocked(api.quickFacts.contactsQuickFactsList).mockResolvedValue({
      data: [{ template_id: 1, template_label: "Favorite food", field_type: "text", facts: [] }],
    });

    renderQuickFactsModule();

    await user.click(await screen.findByRole("button", { name: /Add/ }));
    await user.type(screen.getByPlaceholderText("Add a quick fact"), "Loves hiking");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(api.quickFacts.contactsQuickFactsCreate).toHaveBeenCalledWith("v1", "c1", 1, { value_text: "Loves hiking" });
    });
  });

  it("uploads a photo-backed quick fact", async () => {
    const user = userEvent.setup();
    vi.mocked(api.quickFacts.contactsQuickFactsList).mockResolvedValue({
      data: [{ template_id: 5, template_label: "Portrait", field_type: "photo", facts: [] }],
    });

    renderQuickFactsModule();

    await user.click(await screen.findByRole("button", { name: /Add/ }));
    expect(screen.getByRole("button", { name: /Upload photo/ })).toBeInTheDocument();

    const input = document.querySelector<HTMLInputElement>('input[type="file"]');
    if (!input) throw new Error("Upload input was not rendered");
    const file = new File(["image"], "portrait.jpg", { type: "image/jpeg" });
    await user.upload(input, file);

    await waitFor(() => {
      expect(api.quickFacts.contactsQuickFactsFileCreate).toHaveBeenCalledWith("v1", "c1", 5, { file });
    });
  });
});
