import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import AdminOAuthProviders from "@/pages/admin/OAuthProviders";

vi.mock("@/api", () => ({
  httpClient: {
    instance: {
      get: vi.fn().mockResolvedValue({ data: { success: true, data: [] } }),
      post: vi.fn().mockResolvedValue({ data: { success: true, data: {} } }),
      put: vi.fn().mockResolvedValue({ data: { success: true, data: {} } }),
      delete: vi.fn().mockResolvedValue({ data: { success: true } }),
    },
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("react-router-dom", async () => {
  const actual =
    await vi.importActual<typeof import("react-router-dom")>(
      "react-router-dom",
    );
  return { ...actual, useNavigate: () => vi.fn() };
});

function renderPage() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <AdminOAuthProviders />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("AdminOAuthProviders", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading spinner when data is loading", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderPage();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders page title", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderPage();
    expect(screen.getByText("OAuth Providers")).toBeInTheDocument();
  });

  it("renders the 4-tab segmented navigation", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderPage();
    expect(screen.getByText("Users")).toBeInTheDocument();
    expect(screen.getByText("Settings")).toBeInTheDocument();
    expect(screen.getByText("Backups")).toBeInTheDocument();
    expect(screen.getByText("OAuth")).toBeInTheDocument();
  });

  it("renders add provider button", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderPage();
    expect(screen.getByText("Add Provider")).toBeInTheDocument();
  });

  it("renders provider table with data", () => {
    mockUseQuery.mockReturnValue({
      data: [
        {
          id: 1,
          type: "github",
          name: "github",
          client_id: "abc123",
          has_secret: true,
          enabled: true,
          display_name: "GitHub Login",
          discovery_url: "",
          scopes: "",
        },
      ],
      isLoading: false,
    });
    renderPage();
    expect(screen.getByText("abc123")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
  });
});
