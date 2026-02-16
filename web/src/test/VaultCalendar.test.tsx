import { describe, it, expect, vi, beforeAll } from "vitest";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import VaultCalendar from "@/pages/vault/VaultCalendar";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/client", () => ({
  default: { get: vi.fn() },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "v1" }),
    useNavigate: () => vi.fn(),
  };
});

function renderCalendar() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultCalendar />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("VaultCalendar", () => {
  it("renders lunar moon emoji for non-gregorian dates", () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      const key = opts.queryKey;
      if (Array.isArray(key) && key.includes("month")) {
        return {
          data: {
            important_dates: [
              {
                id: 1,
                contact_id: "c1",
                label: "Lunar Birthday",
                day: 15,
                month: 2,
                year: 2026,
                calendar_type: "lunar",
                original_day: 15,
                original_month: 1,
                original_year: 2026,
                contact_important_date_type_id: null,
                created_at: "2025-01-01",
                updated_at: "2025-01-01",
              },
            ],
            reminders: [],
          },
          isLoading: false,
        };
      }
      if (Array.isArray(key) && key.includes("day")) {
        return { data: undefined, isLoading: false };
      }
      return { data: undefined, isLoading: false };
    });
    renderCalendar();
    expect(document.body.textContent).toContain("🌙");
  });
});
