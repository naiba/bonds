import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import VaultLifeMetrics from "@/pages/vault/VaultLifeMetrics";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/lifeMetrics", () => ({
  lifeMetricsApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    addContact: vi.fn(),
    removeContact: vi.fn(),
  },
}));

vi.mock("@/api/search", () => ({
  searchApi: {
    search: vi.fn(),
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

function renderVaultLifeMetrics() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultLifeMetrics />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("VaultLifeMetrics", () => {
  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderVaultLifeMetrics();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  }, 15000);

  it("renders add button", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultLifeMetrics();
    expect(screen.getByText("Add Metric")).toBeInTheDocument();
  });

  it("renders page title", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultLifeMetrics();
    expect(screen.getByText("Life Metrics")).toBeInTheDocument();
  });
});
