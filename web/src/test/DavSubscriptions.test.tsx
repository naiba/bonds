import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import DavSubscriptions from "@/pages/vault/DavSubscriptions";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    davSubscriptions: {
      davSubscriptionsList: vi.fn(),
      davSubscriptionsCreate: vi.fn(),
      davSubscriptionsUpdate: vi.fn(),
      davSubscriptionsDelete: vi.fn(),
      davSubscriptionsTestCreate: vi.fn(),
      davSubscriptionsSyncCreate: vi.fn(),
      davSubscriptionsLogsList: vi.fn(),
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
    useParams: () => ({ id: "vault-1" }),
    useNavigate: () => vi.fn(),
  };
});

vi.mock("@/stores/auth", () => ({
  useAuth: () => ({
    user: { email: "test@example.com" },
    token: "test-token",
    login: vi.fn(),
    logout: vi.fn(),
  }),
}));

function renderPage() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <DavSubscriptions />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("DavSubscriptions", () => {
  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderPage();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  }, 15000);

  it("renders empty state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderPage();
    expect(screen.getByText("No CardDAV subscriptions")).toBeInTheDocument();
  });

  it("renders page title", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderPage();
    expect(screen.getByText("CardDAV Subscriptions")).toBeInTheDocument();
  });

  it("renders add subscription button", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderPage();
    expect(screen.getByText("Add Subscription")).toBeInTheDocument();
  });
});
