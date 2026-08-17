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
  it("renders contact name with label for calendar dates", () => {
    // Use current month so the date appears in the default panel view
    // (panelDate defaults to dayjs(), so only current month dates are rendered)
    const now = new Date();
    const currentMonth = now.getMonth() + 1;
    const currentYear = now.getFullYear();
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      const key = opts.queryKey;
      if (Array.isArray(key) && key.includes("month")) {
        return {
          data: {
            important_dates: [
              {
                id: 1,
                contact_id: "c1",
                contact_name: "John Doe",
                label: "Lunar Birthday",
                day: 15,
                month: currentMonth,
                year: currentYear,
                calendar_type: "lunar",
                original_day: 15,
                original_month: 1,
                original_year: currentYear,
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
    expect(document.body.textContent).toContain("John Doe - Lunar Birthday");
  });

  it("renders yearly recurring reminders on every year, not only the stored year", () => {
    const now = new Date();
    const currentMonth = now.getMonth() + 1;
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      const key = opts.queryKey;
      if (Array.isArray(key) && key.includes("month")) {
        return {
          data: {
            important_dates: [],
            reminders: [
              {
                id: 2,
                contact_id: "c1",
                contact_name: "Jane Doe",
                label: "Anniversary Reminder",
                day: 15,
                month: currentMonth,
                year: 2000,
                type: "recurring_year",
              },
            ],
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
    expect(document.body.textContent).toContain("Jane Doe - Anniversary Reminder");
  });

  it("does not render one-time reminders from other years on the current panel", () => {
    const now = new Date();
    const currentMonth = now.getMonth() + 1;
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      const key = opts.queryKey;
      if (Array.isArray(key) && key.includes("month")) {
        return {
          data: {
            important_dates: [],
            reminders: [
              {
                id: 3,
                contact_id: "c1",
                contact_name: "John Doe",
                label: "One Time Old Reminder",
                day: 15,
                month: currentMonth,
                year: 2000,
                type: "one_time",
              },
            ],
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
    expect(document.body.textContent).not.toContain("One Time Old Reminder");
  });
});
