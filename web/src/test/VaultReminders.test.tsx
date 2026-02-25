import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import VaultReminders from "@/pages/vault/VaultReminders";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/calendar", () => ({
  calendarApi: {
    getReminders: vi.fn(),
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "1" }),
    useNavigate: () => vi.fn(),
  };
});

function renderVaultReminders() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultReminders />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("VaultReminders", () => {
  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderVaultReminders();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  }, 15000);

  it("renders back button", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultReminders();
    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  it("renders page title", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultReminders();
    // Page heading and empty hero title may both contain the text
    expect(screen.getAllByText("All Reminders").length).toBeGreaterThanOrEqual(1);
  });
});
