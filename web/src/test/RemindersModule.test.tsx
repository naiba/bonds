import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import RemindersModule from "@/pages/contact/modules/RemindersModule";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/reminders", () => ({
  remindersApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock("@/components/CalendarDatePicker", () => ({
  default: () => <div data-testid="calendar-date-picker" />,
}));

let mockRemindersReturn: unknown = { data: [], isLoading: false };
let mockPrefsReturn: unknown = { data: undefined };
vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: { queryKey: unknown[] }) => {
    const key = JSON.stringify(opts.queryKey);
    if (key.includes("preferences")) return mockPrefsReturn;
    return mockRemindersReturn;
  },
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

function renderModule() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <RemindersModule vaultId="v1" contactId="c1" />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

const mockReminders = [
  {
    id: 1,
    contact_id: "c1",
    label: "Call Mom",
    day: 15,
    month: 3,
    year: 2025,
    type: "recurring_year",
    frequency_number: null,
    calendar_type: "gregorian",
    original_day: null,
    original_month: null,
    original_year: null,
    last_triggered_at: null,
    number_times_triggered: 0,
    created_at: "2025-01-01",
    updated_at: "2025-01-01",
  },
  {
    id: 2,
    contact_id: "c1",
    label: "Lunar Bday",
    day: 12,
    month: 2,
    year: 2025,
    type: "recurring_year",
    frequency_number: null,
    calendar_type: "lunar",
    original_day: 15,
    original_month: 1,
    original_year: 2025,
    last_triggered_at: null,
    number_times_triggered: 0,
    created_at: "2025-01-01",
    updated_at: "2025-01-01",
  },
];

describe("RemindersModule", () => {
  it("renders title and add button", () => {
    mockRemindersReturn = { data: [], isLoading: false };
    renderModule();
    expect(screen.getByText("Reminders")).toBeInTheDocument();
    expect(screen.getByText("Add")).toBeInTheDocument();
  });

  it("renders empty state", () => {
    mockRemindersReturn = { data: [], isLoading: false };
    renderModule();
    expect(screen.getByText("No reminders")).toBeInTheDocument();
  });

  it("renders reminders list", () => {
    mockRemindersReturn = { data: mockReminders, isLoading: false };
    renderModule();
    expect(screen.getByText("Call Mom")).toBeInTheDocument();
    expect(screen.getByText("Lunar Bday")).toBeInTheDocument();
  });

  it("renders lunar reminder with tag when alternative calendar enabled", () => {
    mockRemindersReturn = { data: mockReminders, isLoading: false };
    mockPrefsReturn = { data: { enable_alternative_calendar: true } };
    renderModule();
    expect(screen.getByText("lunar")).toBeInTheDocument();
  });

  it("renders frequency tag", () => {
    mockRemindersReturn = { data: mockReminders, isLoading: false };
    mockPrefsReturn = { data: undefined };
    renderModule();
    const recurringTags = screen.getAllByText("recurring_year");
    expect(recurringTags.length).toBeGreaterThanOrEqual(1);
  });
});
