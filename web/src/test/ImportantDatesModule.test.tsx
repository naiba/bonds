import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

const mutationMock = vi.hoisted(() => ({
  mutate: vi.fn(),
}));

let mockDatesReturn: unknown = { data: [], isLoading: false };
let mockPrefsReturn: unknown = { data: undefined };
let mockDateTypesReturn: unknown = { data: [], isLoading: false };
vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: { queryKey: unknown[] }) => {
    const key = JSON.stringify(opts.queryKey);
    if (key.includes("preferences")) return mockPrefsReturn;
    if (key.includes("date-types")) return mockDateTypesReturn;
    return mockDatesReturn;
  },
  useMutation: () => ({ mutate: mutationMock.mutate, isPending: false }),
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
  beforeEach(() => {
    mockDatesReturn = { data: [], isLoading: false };
    mockPrefsReturn = { data: undefined };
    mockDateTypesReturn = { data: [], isLoading: false };
    mutationMock.mutate.mockClear();
  });

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

  it("shows age next to birthdate when contact is alive (Issue #132)", () => {
    mockDateTypesReturn = {
      data: [
        { id: 10, label: "Birthdate", internal_type: "birthdate", can_be_deleted: false },
        { id: 11, label: "Deceased date", internal_type: "deceased_date", can_be_deleted: false },
      ],
    };
    mockDatesReturn = {
      data: [
        {
          id: 1,
          contact_id: "c1",
          label: "Birthdate",
          day: 15,
          month: 3,
          year: 1990,
          calendar_type: "gregorian",
          contact_important_date_type_id: 10,
        },
      ],
      isLoading: false,
    };
    mockPrefsReturn = { data: undefined };
    renderModule();
    expect(screen.getByText(/years old|year old/)).toBeInTheDocument();
  });

  it("shows age-at-death next to deceased date, not birthdate (Issue #132)", () => {
    mockDateTypesReturn = {
      data: [
        { id: 10, label: "Birthdate", internal_type: "birthdate", can_be_deleted: false },
        { id: 11, label: "Deceased date", internal_type: "deceased_date", can_be_deleted: false },
      ],
    };
    mockDatesReturn = {
      data: [
        {
          id: 1,
          contact_id: "c1",
          label: "Birthdate",
          day: 15,
          month: 3,
          year: 1950,
          calendar_type: "gregorian",
          contact_important_date_type_id: 10,
        },
        {
          id: 2,
          contact_id: "c1",
          label: "Deceased date",
          day: 14,
          month: 3,
          year: 2020,
          calendar_type: "gregorian",
          contact_important_date_type_id: 11,
        },
      ],
      isLoading: false,
    };
    mockPrefsReturn = { data: undefined };
    renderModule();
    expect(screen.getByText(/69 years old/)).toBeInTheDocument();
    expect(screen.getAllByText(/years old|year old/).length).toBe(1);
  });

  it("displays date without year using short format (Issue #76)", () => {
    const noYearDate = {
      id: 3,
      contact_id: "c1",
      label: "Nameday",
      day: 15,
      month: 6,
      year: null,
      calendar_type: "gregorian",
      original_day: null,
      original_month: null,
      original_year: null,
      contact_important_date_type_id: null,
      created_at: "2025-01-01",
      updated_at: "2025-01-01",
    };
    mockDatesReturn = { data: [noYearDate], isLoading: false };
    mockPrefsReturn = { data: undefined };
    renderModule();
    expect(screen.getByText("Nameday")).toBeInTheDocument();
    expect(screen.getByText(/Jun 15/)).toBeInTheDocument();
  });

  it("submits a new date with the default calendar date when date fields are unchanged", async () => {
    const user = userEvent.setup();

    renderModule();

    await user.click(screen.getByRole("button", { name: /add/i }));
    await user.type(screen.getByRole("textbox", { name: /label/i }), "Graduation Day");
    await user.click(screen.getByRole("button", { name: /ok/i }));

    await waitFor(() => {
      expect(mutationMock.mutate).toHaveBeenCalledWith(
        expect.objectContaining({
          label: "Graduation Day",
          calendarDate: expect.objectContaining({
            calendarType: "gregorian",
            day: expect.any(Number),
            month: expect.any(Number),
            year: expect.any(Number),
          }),
        }),
      );
    });
  });
});
