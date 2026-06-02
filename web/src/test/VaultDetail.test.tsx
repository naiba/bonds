import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import VaultDetail from "@/pages/vault/VaultDetail";

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
    reminders: { remindersList: vi.fn() },
    search: { searchMostConsultedList: vi.fn() },
    vaultTasks: { tasksList: vi.fn() },
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

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
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

describe("VaultDetail", () => {
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
    expect(screen.getByRole("button", { name: /Jane Doe/i })).toBeInTheDocument();
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
});
