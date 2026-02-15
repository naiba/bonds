import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import Users from "@/pages/settings/Users";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/settings", () => ({
  settingsApi: {
    listUsers: vi.fn(),
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
}));

function renderUsers() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <Users />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("Users", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderUsers();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders empty state when no users", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderUsers();
    expect(document.querySelector(".ant-empty")).toBeInTheDocument();
  });

  it("renders user list when data present", () => {
    mockUseQuery.mockReturnValue({
      data: [
        {
          id: "u1",
          first_name: "Alice",
          last_name: "Smith",
          email: "alice@example.com",
          is_admin: true,
          created_at: "2024-01-15T00:00:00Z",
        },
      ],
      isLoading: false,
    });
    renderUsers();
    expect(screen.getByText("Alice Smith")).toBeInTheDocument();
    expect(screen.getByText("alice@example.com")).toBeInTheDocument();
    expect(screen.getByText("Admin")).toBeInTheDocument();
  });
});
