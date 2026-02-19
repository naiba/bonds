import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import ImportantDatesModule from "@/pages/contact/modules/ImportantDatesModule";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/importantDates", () => ({
  importantDatesApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock("@/components/CalendarDatePicker", () => ({
  default: () => <div data-testid="calendar-date-picker" />,
}));

let mockDatesReturn: unknown = { data: [], isLoading: false };
let mockPrefsReturn: unknown = { data: undefined };
vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: { queryKey: unknown[] }) => {
    const key = JSON.stringify(opts.queryKey);
    if (key.includes("preferences")) return mockPrefsReturn;
    return mockDatesReturn;
  },
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

function renderModule() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <ImportantDatesModule vaultId="v1" contactId="c1" />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

const mockDates = [
  {
    id: 1,
    contact_id: "c1",
    label: "Birthday",
    day: 15,
    month: 3,
    year: 2025,
    calendar_type: "gregorian",
    original_day: null,
    original_month: null,
    original_year: null,
    contact_important_date_type_id: null,
    created_at: "2025-01-01",
    updated_at: "2025-01-01",
  },
  {
    id: 2,
    contact_id: "c1",
    label: "Lunar NY",
    day: 12,
    month: 2,
    year: 2025,
    calendar_type: "lunar",
    original_day: 15,
    original_month: 1,
    original_year: 2025,
    created_at: "2025-01-01",
    updated_at: "2025-01-01",
    contact_important_date_type_id: null,
  },
];

describe("ImportantDatesModule", () => {
  it("renders title and add button", () => {
    mockDatesReturn = { data: [], isLoading: false };
    renderModule();
    expect(screen.getByText("Important Dates")).toBeInTheDocument();
    expect(screen.getByText("Add")).toBeInTheDocument();
  });

  it("renders empty state", () => {
    mockDatesReturn = { data: [], isLoading: false };
    renderModule();
    expect(screen.getByText("No important dates")).toBeInTheDocument();
  });

  it("renders important dates list", () => {
    mockDatesReturn = { data: mockDates, isLoading: false };
    renderModule();
    expect(screen.getByText("Birthday")).toBeInTheDocument();
    expect(screen.getByText("Lunar NY")).toBeInTheDocument();
  });

  it("renders lunar date with tag when alternative calendar enabled", () => {
    mockDatesReturn = { data: mockDates, isLoading: false };
    mockPrefsReturn = { data: { enable_alternative_calendar: true } };
    renderModule();
    expect(screen.getByText("lunar")).toBeInTheDocument();
  });

  it("renders gregorian date without tag", () => {
    mockDatesReturn = { data: [mockDates[0]], isLoading: false };
    mockPrefsReturn = { data: undefined };
    renderModule();
    expect(screen.getByText("Birthday")).toBeInTheDocument();
    expect(screen.queryByText("gregorian")).not.toBeInTheDocument();
  });
});
