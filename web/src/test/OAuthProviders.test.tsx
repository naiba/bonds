import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import OAuthProviders from "@/pages/settings/OAuthProviders";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/oauth", () => ({
  oauthApi: {
    listProviders: vi.fn(),
    unlinkProvider: vi.fn(),
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

function renderOAuthProviders() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <OAuthProviders />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("OAuthProviders", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderOAuthProviders();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders empty state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderOAuthProviders();
    expect(document.querySelector(".ant-empty")).toBeInTheDocument();
  });

  it("renders providers when data present", () => {
    mockUseQuery.mockReturnValue({
      data: [
        {
          driver: "github",
          id: "12345",
          name: "octocat",
          avatar_url: "",
          created_at: "2024-01-01T00:00:00Z",
        },
      ],
      isLoading: false,
    });
    renderOAuthProviders();
    expect(screen.getByText("github")).toBeInTheDocument();
    expect(screen.getByText("octocat")).toBeInTheDocument();
  });
});
