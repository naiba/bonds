import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import AdminUsers from "@/pages/admin/Users";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    admin: {
      usersList: vi.fn(),
      usersToggleUpdate: vi.fn(),
      usersAdminUpdate: vi.fn(),
      usersDelete: vi.fn(),
    },
  },
}));

vi.mock("filesize", () => ({
  filesize: (bytes: number) => bytes + " B",
}));

vi.mock("@/stores/auth", () => ({
  useAuth: () => ({
    user: { id: "admin-1", is_instance_administrator: true },
  }),
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

function renderAdminUsers() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <AdminUsers />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("AdminUsers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderAdminUsers();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders empty table when no users", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderAdminUsers();
    expect(screen.getByText("User Management")).toBeInTheDocument();
    expect(document.querySelector(".ant-empty")).toBeInTheDocument();
  });

  it("renders user list with admin and regular user", () => {
    mockUseQuery.mockReturnValue({
      data: [
        {
          id: "admin-1",
          first_name: "Admin",
          last_name: "User",
          email: "admin@example.com",
          is_instance_administrator: true,
          disabled: false,
          contact_count: 10,
          vault_count: 2,
          storage_used: 5242880,
          created_at: "2024-01-15T00:00:00Z",
        },
        {
          id: "user-2",
          first_name: "Regular",
          last_name: "User",
          email: "regular@example.com",
          is_instance_administrator: false,
          disabled: false,
          contact_count: 5,
          vault_count: 1,
          storage_used: 0,
          created_at: "2024-06-01T00:00:00Z",
        },
      ],
      isLoading: false,
    });
    renderAdminUsers();
    expect(screen.getByText("admin@example.com")).toBeInTheDocument();
    expect(screen.getByText("regular@example.com")).toBeInTheDocument();
    expect(screen.getByText("Admin")).toBeInTheDocument();
  });

  it("renders disabled user with correct status tag", () => {
    mockUseQuery.mockReturnValue({
      data: [
        {
          id: "user-3",
          first_name: "Disabled",
          last_name: "User",
          email: "disabled@example.com",
          is_instance_administrator: false,
          disabled: true,
          contact_count: 0,
          vault_count: 0,
          storage_used: 0,
          created_at: "2024-01-01T00:00:00Z",
        },
      ],
      isLoading: false,
    });
    renderAdminUsers();
    expect(screen.getByText("Disabled")).toBeInTheDocument();
  });

  it("does not show action buttons for current user", () => {
    mockUseQuery.mockReturnValue({
      data: [
        {
          id: "admin-1",
          first_name: "Self",
          last_name: "Admin",
          email: "self@example.com",
          is_instance_administrator: true,
          disabled: false,
          contact_count: 0,
          vault_count: 0,
          storage_used: 0,
          created_at: "2024-01-01T00:00:00Z",
        },
      ],
      isLoading: false,
    });
    renderAdminUsers();
    expect(screen.queryByText("Disable")).not.toBeInTheDocument();
    expect(screen.queryByText("Remove Admin")).not.toBeInTheDocument();
  });
});
