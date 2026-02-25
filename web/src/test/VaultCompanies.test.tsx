import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import VaultCompanies from "@/pages/vault/VaultCompanies";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/companies", () => ({
  companiesApi: {
    list: vi.fn(),
    get: vi.fn(),
    listForContact: vi.fn(),
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

function renderVaultCompanies() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultCompanies />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("VaultCompanies", () => {
  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderVaultCompanies();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  }, 15000);

  it("renders empty table", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultCompanies();
    // Empty state uses bonds-empty-hero instead of ant-empty
    expect(document.querySelector(".bonds-empty-hero, .ant-empty")).toBeInTheDocument();
  });

  it("renders company title", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultCompanies();
    // Page heading and empty hero title both say "Companies"
    expect(screen.getAllByText("Companies").length).toBeGreaterThanOrEqual(1);
  });
});
