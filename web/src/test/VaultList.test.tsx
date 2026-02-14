import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import VaultList from "@/pages/vault/VaultList";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/vaults", () => ({
  vaultsApi: {
    list: vi.fn(),
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
}));

function renderVaultList() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultList />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("VaultList", () => {
  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderVaultList();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders empty state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultList();
    expect(screen.getByText("No vaults yet")).toBeInTheDocument();
    expect(screen.getByText("Create vault")).toBeInTheDocument();
  });

  it("renders vault cards", () => {
    mockUseQuery.mockReturnValue({
      data: [
        {
          id: 1,
          name: "Personal",
          description: "My personal vault",
          created_at: "2024-06-01T00:00:00Z",
        },
        {
          id: 2,
          name: "Work",
          description: null,
          created_at: "2024-07-15T00:00:00Z",
        },
      ],
      isLoading: false,
    });
    renderVaultList();
    expect(screen.getByText("Personal")).toBeInTheDocument();
    expect(screen.getByText("Work")).toBeInTheDocument();
    expect(screen.getByText("My personal vault")).toBeInTheDocument();
  });

  it("renders title and new vault button", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultList();
    expect(screen.getByText("Vaults")).toBeInTheDocument();
    expect(screen.getByText("New vault")).toBeInTheDocument();
  });
});
