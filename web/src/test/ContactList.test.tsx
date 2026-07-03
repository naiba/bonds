import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import ContactList from "@/pages/contact/ContactList";
import { api } from "@/api";
import type { Contact, PaginationMeta } from "@/api";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: (query: string) => ({
      matches: query.includes("min-width"),
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }),
  });
});

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location-probe">{location.pathname}{location.search}</div>;
}

vi.mock("@/api", () => ({
  api: {
    contacts: {
      contactsList: vi.fn(),
      contactsLabelsDetail: vi.fn(),
      contactsSortUpdate: vi.fn(),
      contactsBulkMoveCreate: vi.fn(),
    },
    contactLabels: { contactLabelsList: vi.fn() },
    groups: { groupsList: vi.fn() },
    vaults: { vaultsList: vi.fn() },
    vaultSettings: { settingsLabelsList: vi.fn() },
    vcard: { contactsExportList: vi.fn(), contactsImportCreate: vi.fn() },
  },
  httpClient: {
    instance: {
      get: vi.fn().mockRejectedValue(new Error("mocked")),
    },
  },
}));

vi.mock("@/components/ContactAvatar", () => ({
  default: () => <div data-testid="contact-avatar" />,
}));

const mockUseQuery = vi.fn();
const mockInvalidateQueries = vi.fn();

type MutationOptions<TVariables> = {
  mutationFn?: (variables: TVariables) => Promise<unknown> | unknown;
  onSuccess?: (data: unknown, variables: TVariables, context: unknown) => void;
  onError?: (error: Error, variables: TVariables, context: unknown) => void;
};

vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: <TVariables,>(options?: MutationOptions<TVariables>) => ({
    mutate: vi.fn(async (variables: TVariables) => {
      try {
        const data = await options?.mutationFn?.(variables);
        options?.onSuccess?.(data, variables, undefined);
        return data;
      } catch (error) {
        options?.onError?.(error instanceof Error ? error : new Error(String(error)), variables, undefined);
        return undefined;
      }
    }),
    isPending: false,
  }),
  useQueryClient: () => ({ invalidateQueries: mockInvalidateQueries }),
}));

let mockLabels: { id: number; name: string }[] = [];

function mockContactListQuery(
  contacts: Contact[] = [],
  meta: PaginationMeta = { total: contacts.length },
  vaults: { id: string; name: string }[] = [],
) {
  mockUseQuery.mockImplementation((opts) => {
    const key = Array.isArray(opts?.queryKey) ? opts.queryKey : [];
    if (key.includes("labels")) {
      return { data: mockLabels, isLoading: false };
    }
    if (key.includes("groups")) {
      return { data: [], isLoading: false };
    }
    if (key[0] === "vaults" && key[1] === "bulkMoveTargets") {
      return { data: vaults, isLoading: false };
    }
    if (key[0] === "vaults" && key[2] === "contacts") {
      return { data: { contacts, meta }, isLoading: false };
    }
    return { data: undefined, isLoading: false };
  });
}

function getContactsQueryKey() {
  const call = mockUseQuery.mock.calls.find(([opts]) => {
    const key = Array.isArray(opts?.queryKey) ? opts.queryKey : [];
    return key[0] === "vaults" && key[2] === "contacts";
  });
  return call?.[0]?.queryKey as unknown[] | undefined;
}

type QueryOptions = { queryKey?: unknown[]; queryFn?: () => Promise<unknown> };

function getLatestContactsQueryOptions() {
  const calls = mockUseQuery.mock.calls.filter(([opts]) => {
    const key = Array.isArray(opts?.queryKey) ? opts.queryKey : [];
    return key[0] === "vaults" && key[2] === "contacts";
  });
  return calls.at(-1)?.[0] as QueryOptions | undefined;
}

function renderContactList(initialUrl = "/vaults/1/contacts") {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={[initialUrl]}>
          <Routes>
            <Route path="/vaults/:id/contacts" element={
              <>
                <ContactList />
                <LocationProbe />
              </>
            } />
            <Route path="/vaults/:id/contacts/:contactId" element={
              <>
                <LocationProbe />
              </>
            } />
          </Routes>
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

async function chooseSelectOption(selectTestId: string, optionText: string) {
  const select = screen.getByTestId(selectTestId);
  const control = select.querySelector<HTMLElement>("input") ?? select;
  fireEvent.mouseDown(control);
  fireEvent.click(control);

  const optionByTitle = await screen.findByTitle(optionText);
  fireEvent.click(optionByTitle);
}

describe("ContactList", () => {
  beforeEach(() => {
    localStorage.removeItem("bonds_contact_list_columns");
    mockLabels = [];
    mockUseQuery.mockReset();
    vi.mocked(api.contacts.contactsList).mockReset();
    vi.mocked(api.contacts.contactsLabelsDetail).mockReset();
    vi.mocked(api.contacts.contactsBulkMoveCreate).mockReset();
    mockInvalidateQueries.mockReset();
    mockContactListQuery();
  });

  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderContactList();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  }, 15000);

  it("renders empty state", () => {
    mockContactListQuery();
    renderContactList();
    expect(screen.getByText("No contacts yet")).toBeInTheDocument();
  });

  it("renders search input", () => {
    mockContactListQuery();
    renderContactList();
    expect(
      screen.getByPlaceholderText("Quick search"),
    ).toBeInTheDocument();
  });

  it("reads page and per_page from URL query parameters", () => {
    renderContactList("/vaults/1/contacts?page=3&per_page=50");

    expect(getContactsQueryKey()).toEqual([
      "vaults",
      "1",
      "contacts",
      null,
      null,
      3,
      50,
      "name",
      "",
      "active",
    ]);
  });

  it("falls back to default pagination when URL query values are invalid", () => {
    renderContactList("/vaults/1/contacts?page=abc&per_page=0");

    expect(getContactsQueryKey()).toEqual([
      "vaults",
      "1",
      "contacts",
      null,
      null,
      1,
      20,
      "name",
      "",
      "active",
    ]);
  });

  it("updates URL when pagination changes", async () => {
    const user = userEvent.setup();
    mockContactListQuery(
      Array.from({ length: 20 }).map((_, i) => ({
        id: String(i + 1),
        first_name: `User ${i + 1}`,
        last_name: "Example",
        updated_at: "2024-06-01T00:00:00Z",
      })),
      { total: 60 },
    );

    renderContactList("/vaults/1/contacts");

    const page2Button = document.querySelector<HTMLElement>(".ant-pagination-item-2 a");
    expect(page2Button).toBeInTheDocument();
    if (!page2Button) throw new Error("Page 2 pagination link was not rendered");
    await user.click(page2Button);

    await waitFor(() => {
      expect(screen.getByTestId("location-probe")).toHaveTextContent("/vaults/1/contacts?page=2&per_page=20");
    });
  });

  it("preserves pagination query parameters when navigating to a contact", async () => {
    const user = userEvent.setup();
    mockContactListQuery(
      [{
        id: "42",
        first_name: "Test",
        last_name: "User",
        updated_at: "2024-06-01T00:00:00Z",
      }],
      { total: 100 },
    );

    renderContactList("/vaults/1/contacts?page=3&per_page=50");

    const contactRow = await screen.findByText("Test User");
    await user.click(contactRow);

    await waitFor(() => {
      expect(screen.getByTestId("location-probe")).toHaveTextContent("/vaults/1/contacts/42?page=3&per_page=50");
    });
  });

  it("renders first-met dates in the default visible columns", () => {
    mockContactListQuery(
      [{
        id: "42",
        first_name: "Ada",
        last_name: "Lovelace",
        first_met_at: "2026-01-15T00:00:00Z",
        updated_at: "2026-01-20T00:00:00Z",
      }],
      { total: 1 },
    );

    renderContactList();

    expect(screen.getByText("First met")).toBeInTheDocument();
    expect(screen.getByText("Jan 15, 2026")).toBeInTheDocument();
  });

  it("renders imprecise first-met dates in the default visible columns", () => {
    mockContactListQuery(
      [{
        id: "42",
        first_name: "Ada",
        last_name: "Lovelace",
        first_met_date_precision: "month",
        first_met_year: 2026,
        first_met_month: 5,
        updated_at: "2026-01-20T00:00:00Z",
      } as Contact],
      { total: 1 },
    );

    renderContactList();

    expect(screen.getByText("First met")).toBeInTheDocument();
    expect(screen.getByText("May 2026")).toBeInTheDocument();
  });

  it("uses first_met_at when the first-met sort option is selected", async () => {
    mockContactListQuery();

    renderContactList();

    await chooseSelectOption("contact-sort-select", "First met");

    await waitFor(() => {
      expect(getLatestContactsQueryOptions()?.queryKey).toEqual([
        "vaults",
        "1",
        "contacts",
        null,
        null,
        1,
        20,
        "first_met_at",
        "",
        "active",
      ]);
    });
  });

  it("passes the selected first-met sort through label-filtered contact queries", async () => {
    mockLabels = [{ id: 7, name: "Friends" }];
    mockContactListQuery();
    vi.mocked(api.contacts.contactsLabelsDetail).mockResolvedValue({
      data: [],
      meta: { total: 0 },
    });

    renderContactList();

    await chooseSelectOption("contact-sort-select", "First met");
    await chooseSelectOption("contact-label-filter", "Friends");

    const queryOptions = getLatestContactsQueryOptions();
    await queryOptions?.queryFn?.();

    expect(api.contacts.contactsLabelsDetail).toHaveBeenCalledWith("1", 7, {
      page: 1,
      per_page: 20,
      sort: "first_met_at",
      filter: "active",
    });
  });

  it("moves selected contacts with the bulk move API", async () => {
    const user = userEvent.setup();
    mockContactListQuery(
      [{
        id: "42",
        first_name: "Ada",
        last_name: "Lovelace",
        updated_at: "2024-06-01T00:00:00Z",
      }],
      { total: 1 },
      [
        { id: "1", name: "Current Vault" },
        { id: "2", name: "Family Vault" },
      ],
    );
    vi.mocked(api.contacts.contactsBulkMoveCreate).mockResolvedValue({ data: { moved_count: 1 } });

    renderContactList();

    const rowCheckbox = screen.getAllByRole("checkbox")[1];
    if (!rowCheckbox) throw new Error("Contact row checkbox was not rendered");
    await user.click(rowCheckbox);

    await user.click(screen.getByRole("button", { name: /move 1 selected/i }));
    expect(screen.getByText("Move selected contacts")).toBeInTheDocument();
    expect(screen.queryByText("Current Vault")).not.toBeInTheDocument();

    await chooseSelectOption("bulk-move-vault-select", "Family Vault");
    await user.click(screen.getByRole("button", { name: "Move contacts" }));

    await waitFor(() => {
      expect(api.contacts.contactsBulkMoveCreate).toHaveBeenCalledWith("1", {
        contact_ids: ["42"],
        target_vault_id: "2",
      });
    });
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ["vaults", "1", "contacts"] });
  }, 15000);
});
