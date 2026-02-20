import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import AdminSettings from "@/pages/admin/Settings";

vi.mock("@/api", () => ({
  api: {
    admin: {
      settingsList: vi.fn(),
      settingsUpdate: vi.fn(),
    },
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

function renderAdminSettings() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <AdminSettings />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("AdminSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderAdminSettings();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders settings form when loaded", () => {
    mockUseQuery.mockReturnValue({
      data: [],
      isLoading: false,
    });
    renderAdminSettings();
    expect(screen.getByText("System Settings")).toBeInTheDocument();
    expect(screen.getByText("Save Settings")).toBeInTheDocument();
  });

  it("renders known setting labels", () => {
    mockUseQuery.mockReturnValue({
      data: [],
      isLoading: false,
    });
    renderAdminSettings();
    // Collapse panels: app and auth are expanded by default
    expect(screen.getByText("Application Name")).toBeInTheDocument();
    expect(screen.getByText("Application URL")).toBeInTheDocument();
    expect(screen.getByText("Password Authentication")).toBeInTheDocument();
    expect(screen.getByText("User Registration")).toBeInTheDocument();
    // Collapsed section headers should also be visible
    expect(screen.getByText("SMTP Email")).toBeInTheDocument();
    expect(screen.getByText("OAuth / OIDC")).toBeInTheDocument();
    expect(screen.getByText("WebAuthn")).toBeInTheDocument();
  });
});
