import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import Notifications from "@/pages/settings/Notifications";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/settings", () => ({
  settingsApi: {
    listNotificationChannels: vi.fn(),
    createNotificationChannel: vi.fn(),
    toggleNotificationChannel: vi.fn(),
    deleteNotificationChannel: vi.fn(),
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

function renderNotifications() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <Notifications />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("Notifications", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderNotifications();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders empty state when no channels", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderNotifications();
    expect(screen.getByText("No notification channels")).toBeInTheDocument();
  });

  it("renders add button", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderNotifications();
    expect(
      screen.getByRole("button", { name: /add channel/i }),
    ).toBeInTheDocument();
  });

  it("renders edit button for each channel", () => {
    mockUseQuery.mockImplementation(({ queryKey }: { queryKey: string[] }) => {
      if (queryKey[0] === "settings" && queryKey[1] === "notifications") {
        return {
          data: [
            { id: 1, type: "email", label: "My Email", content: "test@example.com", active: true, verified_at: "2026-01-01" },
          ],
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });
    renderNotifications();
    const editButton = document.querySelector('[title="Edit Channel"]');
    expect(editButton).toBeInTheDocument();
  });
});
