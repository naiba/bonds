import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider, Modal } from "antd";
import VaultDetail from "@/pages/vault/VaultDetail";
import { api } from "@/api";
import type { Contact, TimelineEvent, LifeEventCategoryResponse } from "@/api";

const mockAppMessage = {
  info: vi.fn(),
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  loading: vi.fn(),
  open: vi.fn(),
  destroy: vi.fn(),
};

const mockAppNotification = {
  success: vi.fn(),
  error: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  open: vi.fn(),
  destroy: vi.fn(),
};

const mockAppModal = {
  info: vi.fn(),
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  confirm: vi.fn(),
};

vi.mock("antd", async () => {
  const actual = await vi.importActual<typeof import("antd")>("antd");
  return {
    ...actual,
    App: Object.assign(actual.App, {
      useApp: () => ({
        message: mockAppMessage,
        notification: mockAppNotification,
        modal: mockAppModal,
      }),
    }),
  };
});

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    vaults: {
      vaultsDetail: vi.fn(),
      vaultsUpdate: vi.fn(),
      vaultsDelete: vi.fn(),
    },
    contacts: { contactsList: vi.fn(), contactsCatchUpCreate: vi.fn() },
    dashboard: { dashboardCatchUpList: vi.fn() },
    lifeEvents: {
      dashboardLifeEventsList: vi.fn(),
      dashboardLifeEventsCreate: vi.fn(),
      dashboardLifeEventsUpdate: vi.fn(),
      dashboardLifeEventsDelete: vi.fn(),
      contactsTimelineEventsCreate: vi.fn(),
      contactsTimelineEventsLifeEventsCreate: vi.fn(),
      contactsTimelineEventsLifeEventsUpdate: vi.fn(),
    },
    preferences: { preferencesList: vi.fn() },
    reminders: { remindersList: vi.fn() },
    search: { searchMostConsultedList: vi.fn() },
    vaultTasks: { tasksList: vi.fn() },
    vaultSettings: { settingsLifeEventCategoriesList: vi.fn() },
  },
  httpClient: {
    instance: {
      get: vi.fn().mockRejectedValue(new Error("mocked")),
      put: vi.fn().mockResolvedValue({}),
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() },
      },
    },
  },
}));

vi.mock("@/components/ContactAvatar", () => ({
  default: () => <div data-testid="contact-avatar" />,
}));

const mockUseQuery = vi.fn();
const mockInvalidateQueries = vi.fn();
const mockGetQueryData = vi.fn();

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
  useQueryClient: () => ({ invalidateQueries: mockInvalidateQueries, getQueryData: mockGetQueryData }),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "1" }),
    useNavigate: () => vi.fn(),
  };
});

function renderVaultDetail() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultDetail />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

const baseVault = {
  id: 1,
  name: "My Vault",
  description: null,
  user_contact_id: "contact-self",
  created_at: "2024-06-01T00:00:00Z",
  updated_at: "2024-06-02T00:00:00Z",
};

const lifeEventCategories: LifeEventCategoryResponse[] = [
  {
    id: 5,
    label: "Personal",
    types: [{ id: 9, label: "Milestone" }],
  },
];

const dashboardContacts: Contact[] = [
  {
    id: "contact-2",
    first_name: "Ada",
    last_name: "Lovelace",
    updated_at: "2024-06-01T00:00:00Z",
  },
];

const dashboardTimelines: TimelineEvent[] = [
  {
    id: 11,
    label: "January 2026",
    started_at: "2026-01-01T00:00:00Z",
    life_events: [
      {
        id: 77,
        life_event_type_id: 9,
        happened_at: "2026-01-15T00:00:00Z",
        summary: "Existing milestone",
        description: "Existing details",
        calendar_type: "gregorian",
        original_day: undefined,
        original_month: undefined,
        original_year: undefined,
        participants: [{ id: "contact-2", name: "Ada Lovelace" }],
      },
    ],
  },
];

function mockVaultQueries({
  defaultTab = "activity",
  timelines = [],
  contacts = [],
  categories = [],
}: {
  defaultTab?: string;
  timelines?: TimelineEvent[];
  contacts?: Contact[];
  categories?: LifeEventCategoryResponse[];
} = {}) {
  mockUseQuery.mockImplementation((opts: { queryKey?: unknown[] }) => {
    const key = Array.isArray(opts.queryKey) ? opts.queryKey : [];
    if (key.length === 2 && key[0] === "vaults") {
      return { data: { ...baseVault, default_activity_tab: defaultTab }, isLoading: false };
    }
    if (key[0] === "settings" && key[1] === "preferences") {
      return { data: { enable_alternative_calendar: false }, isLoading: false };
    }
    if (key[0] === "vaults" && key[2] === "dashboardLifeEvents") {
      return {
        data: { items: timelines, meta: { page: 1, total_pages: 1 }, page: 1 },
        isLoading: false,
        isFetching: false,
      };
    }
    if (key[0] === "vaults" && key.includes("lifeEventCategories")) {
      return { data: categories, isLoading: false };
    }
    if (key[0] === "vaults" && key[2] === "contacts") {
      return { data: contacts, isLoading: false };
    }
    return { data: [], isLoading: false, isFetching: false };
  });
}

async function chooseSelectOption(selectTestId: string, optionText: string) {
  const select = screen.getByTestId(selectTestId);
  const control = select.querySelector<HTMLElement>("input") ?? select;
  fireEvent.mouseDown(control);
  fireEvent.click(control);

  const optionByTitle = await screen.findByTitle(optionText);
  fireEvent.click(optionByTitle);
}

describe("VaultDetail", () => {
  beforeEach(() => {
    mockUseQuery.mockReset();
    mockInvalidateQueries.mockReset();
    mockGetQueryData.mockReset();
    Object.values(mockAppMessage).forEach((mockFn) => mockFn.mockReset());
    Object.values(mockAppNotification).forEach((mockFn) => mockFn.mockReset());
    Object.values(mockAppModal).forEach((mockFn) => mockFn.mockReset());
    vi.mocked(api.lifeEvents.dashboardLifeEventsCreate).mockReset();
    vi.mocked(api.lifeEvents.dashboardLifeEventsUpdate).mockReset();
    vi.mocked(api.lifeEvents.dashboardLifeEventsDelete).mockReset();
    vi.mocked(api.lifeEvents.contactsTimelineEventsCreate).mockReset();
    vi.mocked(api.lifeEvents.contactsTimelineEventsLifeEventsCreate).mockReset();
    vi.mocked(api.lifeEvents.contactsTimelineEventsLifeEventsUpdate).mockReset();
  });

  afterEach(() => {
    Modal.destroyAll();
  });

  it("renders loading spinner when loading", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderVaultDetail();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders vault name when loaded", () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (
        Array.isArray(opts.queryKey) &&
        opts.queryKey.length === 2 &&
        opts.queryKey[0] === "vaults"
      ) {
        return {
          data: {
            id: 1,
            name: "My Vault",
            description: "Test description",
            created_at: "2024-06-01T00:00:00Z",
            updated_at: "2024-06-02T00:00:00Z",
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });
    renderVaultDetail();
    expect(screen.getByText("My Vault")).toBeInTheDocument();
  });

  it("renders Add contact button", () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (
        Array.isArray(opts.queryKey) &&
        opts.queryKey.length === 2 &&
        opts.queryKey[0] === "vaults"
      ) {
        return {
          data: {
            id: 1,
            name: "My Vault",
            description: null,
            created_at: "2024-06-01T00:00:00Z",
            updated_at: "2024-06-02T00:00:00Z",
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });
    renderVaultDetail();
    expect(
      screen.getByRole("button", { name: /add contact/i }),
    ).toBeInTheDocument();
  });

  it("shows contact name alongside reminder label in upcoming reminders widget (#82)", () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (
        Array.isArray(opts.queryKey) &&
        opts.queryKey.length === 2 &&
        opts.queryKey[0] === "vaults"
      ) {
        return {
          data: {
            id: 1,
            name: "My Vault",
            description: null,
            created_at: "2024-06-01T00:00:00Z",
            updated_at: "2024-06-02T00:00:00Z",
          },
          isLoading: false,
        };
      }
      if (
        Array.isArray(opts.queryKey) &&
        opts.queryKey.includes("reminders")
      ) {
        return {
          data: [
            {
              id: 7,
              label: "Birthday",
              day: 31,
              month: 12,
              contact_first_name: "John",
              contact_last_name: "Doe",
            },
          ],
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });
    renderVaultDetail();
    expect(screen.getByText(/John Doe/)).toBeInTheDocument();
    expect(screen.getByText(/Birthday/)).toBeInTheDocument();
  });

  it("renders catch-up prompts on the dashboard", () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (
        Array.isArray(opts.queryKey) &&
        opts.queryKey.length === 2 &&
        opts.queryKey[0] === "vaults"
      ) {
        return {
          data: {
            id: 1,
            name: "My Vault",
            description: null,
            created_at: "2024-06-01T00:00:00Z",
            updated_at: "2024-06-02T00:00:00Z",
          },
          isLoading: false,
        };
      }
      if (
        Array.isArray(opts.queryKey) &&
        opts.queryKey.includes("catchUp")
      ) {
        return {
          data: [
            {
              contact_id: "contact-1",
              name: "Zephyr, Alice (Ace)",
              first_name: "Jane",
              last_name: "Doe",
              days_overdue: 12,
              last_talked_to: "2026-01-02T00:00:00Z",
            },
          ],
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });

    renderVaultDetail();

    expect(screen.getByText("Catch-Up")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Zephyr, Alice \(Ace\)/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Jane Doe/i })).not.toBeInTheDocument();
    expect(screen.getByText("12 days overdue")).toBeInTheDocument();
  });

  it("renders catch-up empty state", () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (
        Array.isArray(opts.queryKey) &&
        opts.queryKey.length === 2 &&
        opts.queryKey[0] === "vaults"
      ) {
        return {
          data: {
            id: 1,
            name: "My Vault",
            description: null,
            created_at: "2024-06-01T00:00:00Z",
            updated_at: "2024-06-02T00:00:00Z",
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });

    renderVaultDetail();

    expect(screen.getByText("No one is due for a catch-up")).toBeInTheDocument();
  });

  it("renders No contacts yet when empty contacts", () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (
        Array.isArray(opts.queryKey) &&
        opts.queryKey.length === 2 &&
        opts.queryKey[0] === "vaults"
      ) {
        return {
          data: {
            id: 1,
            name: "My Vault",
            description: null,
            created_at: "2024-06-01T00:00:00Z",
            updated_at: "2024-06-02T00:00:00Z",
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });
    renderVaultDetail();
    expect(screen.getByText("No contacts yet")).toBeInTheDocument();
  });

  it("creates dashboard life events with the vault-scoped API", async () => {
    const user = userEvent.setup();
    mockVaultQueries({ defaultTab: "life_events", contacts: dashboardContacts, categories: lifeEventCategories });
    vi.mocked(api.lifeEvents.dashboardLifeEventsCreate).mockResolvedValue({});

    renderVaultDetail();

    await user.click(screen.getByRole("button", { name: /add a life event/i }));
    await chooseSelectOption("dashboard-life-event-category-select", "Personal");
    await chooseSelectOption("dashboard-life-event-type-select", "Milestone");
    await user.type(screen.getByLabelText("Summary"), "New milestone");
    await user.type(screen.getByLabelText("Description"), "New details");
    await chooseSelectOption("dashboard-life-event-participants-select", "Ada Lovelace");
    await user.click(screen.getByRole("button", { name: "OK" }));

    await waitFor(() => {
      expect(api.lifeEvents.dashboardLifeEventsCreate).toHaveBeenCalledWith(
        "1",
        expect.objectContaining({
          life_event_type_id: 9,
          summary: "New milestone",
          description: "New details",
          calendar_type: "gregorian",
          original_day: undefined,
          original_month: undefined,
          original_year: undefined,
          participants: ["contact-2"],
        }),
      );
    });
    await waitFor(() => {
      expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ["vaults", "1", "dashboardLifeEvents"] });
    });
    expect(api.lifeEvents.contactsTimelineEventsCreate).not.toHaveBeenCalled();
    expect(api.lifeEvents.contactsTimelineEventsLifeEventsCreate).not.toHaveBeenCalled();
  }, 15000);

  it("offers the vault user contact as a dashboard life event participant", async () => {
    const user = userEvent.setup();
    mockVaultQueries({ defaultTab: "life_events", contacts: dashboardContacts, categories: lifeEventCategories });
    vi.mocked(api.lifeEvents.dashboardLifeEventsCreate).mockResolvedValue({});

    renderVaultDetail();

    await user.click(screen.getByRole("button", { name: /add a life event/i }));
    await chooseSelectOption("dashboard-life-event-category-select", "Personal");
    await chooseSelectOption("dashboard-life-event-type-select", "Milestone");
    await user.type(screen.getByLabelText("Summary"), "Self milestone");
    await chooseSelectOption("dashboard-life-event-participants-select", "You");
    await user.click(screen.getByRole("button", { name: "OK" }));

    await waitFor(() => {
      expect(api.lifeEvents.dashboardLifeEventsCreate).toHaveBeenCalledWith(
        "1",
        expect.objectContaining({
          summary: "Self milestone",
          participants: ["contact-self"],
        }),
      );
    });
    await waitFor(() => {
      expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ["vaults", "1", "dashboardLifeEvents"] });
    });
  }, 15000);

  it("updates dashboard life events with the vault-scoped API", async () => {
    const user = userEvent.setup();
    mockVaultQueries({
      defaultTab: "life_events",
      timelines: dashboardTimelines,
      contacts: dashboardContacts,
      categories: lifeEventCategories,
    });
    vi.mocked(api.lifeEvents.dashboardLifeEventsUpdate).mockResolvedValue({});

    renderVaultDetail();

    await screen.findByText("Existing milestone");
    await user.click(screen.getByRole("button", { name: "Actions" }));
    await user.click(await screen.findByText("Edit"));
    const summaryInput = await screen.findByLabelText("Summary");
    await user.clear(summaryInput);
    await user.type(summaryInput, "Updated milestone");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(api.lifeEvents.dashboardLifeEventsUpdate).toHaveBeenCalledWith(
        "1",
        77,
        expect.objectContaining({
          life_event_type_id: 9,
          summary: "Updated milestone",
          description: "Existing details",
          calendar_type: "gregorian",
          participants: ["contact-2"],
        }),
      );
    });
    await waitFor(() => {
      expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ["vaults", "1", "dashboardLifeEvents"] });
    });
    expect(api.lifeEvents.contactsTimelineEventsLifeEventsUpdate).not.toHaveBeenCalled();
  }, 15000);

  it("deletes dashboard life events with the vault-scoped API", async () => {
    const user = userEvent.setup();
    mockVaultQueries({ defaultTab: "life_events", timelines: dashboardTimelines, categories: lifeEventCategories });
    vi.mocked(api.lifeEvents.dashboardLifeEventsDelete).mockResolvedValue({});

    renderVaultDetail();

    await screen.findByText("Existing milestone");
    await user.click(screen.getByRole("button", { name: "Actions" }));
    await user.click(await screen.findByText("Delete"));
    await waitFor(() => {
      expect(screen.getAllByText("Are you sure you want to delete this?").length).toBeGreaterThan(0);
    });
    const deleteButtons = screen.getAllByRole("button", { name: "Delete" });
    await user.click(deleteButtons[deleteButtons.length - 1]);

    await waitFor(() => {
      expect(api.lifeEvents.dashboardLifeEventsDelete).toHaveBeenCalledWith("1", 77);
    });
    await waitFor(() => {
      expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ["vaults", "1", "dashboardLifeEvents"] });
    });
  }, 15000);
});
