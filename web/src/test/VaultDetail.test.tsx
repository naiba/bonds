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

vi.mock("@/api/vaults", () => ({
  vaultsApi: {
    get: vi.fn(),
  },
}));

vi.mock("@/api/contacts", () => ({
  contactsApi: {
    list: vi.fn(),
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
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
