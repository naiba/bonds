import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { beforeAll, beforeEach, vi } from "vitest";
import { render } from "@testing-library/react";
import ImportantDatesModule from "@/pages/contact/modules/ImportantDatesModule";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";

type MockCalendarDatePickerProps = {
  readonly value?: CalendarDatePickerValue;
  readonly onChange?: (value: CalendarDatePickerValue) => void;
};

type MutationOptions<TVariables> = {
  readonly mutationFn: (values: TVariables) => unknown;
};

const hoistedMocks = vi.hoisted(() => ({
  apiMock: {
    contactsDatesCreate: vi.fn(),
    contactsDatesUpdate: vi.fn(),
    contactsDatesDelete: vi.fn(),
  },
  mutationMock: {
    mutate: vi.fn(),
  },
}));

export const apiMock = hoistedMocks.apiMock;
export const mutationMock = hoistedMocks.mutationMock;

export let mockDatesReturn: unknown = { data: [], isLoading: false };
export let mockPrefsReturn: unknown = { data: undefined };
export let mockDateTypesReturn: unknown = { data: [], isLoading: false };

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    importantDates: {
      contactsDatesList: vi.fn(),
      contactsDatesCreate: hoistedMocks.apiMock.contactsDatesCreate,
      contactsDatesUpdate: hoistedMocks.apiMock.contactsDatesUpdate,
      contactsDatesDelete: hoistedMocks.apiMock.contactsDatesDelete,
    },
    preferences: {
      preferencesList: vi.fn(),
    },
    vaultSettings: {
      settingsDateTypesList: vi.fn(),
    },
  },
}));

vi.mock("@/components/CalendarDatePicker", () => ({
  default: ({ value, onChange }: MockCalendarDatePickerProps) => (
    <div data-testid="calendar-date-picker">
      <output data-testid="calendar-picker-value">
        {JSON.stringify(value ?? null)}
      </output>
      <button
        data-testid="mock-calendar-change-full"
        onClick={() => onChange?.({ calendarType: "gregorian", day: 15, month: 8, year: 2025, datePrecision: "full" })}
      >
        Set Full Date
      </button>
      <button
        data-testid="mock-calendar-change-month"
        onClick={() => onChange?.({ calendarType: "gregorian", day: null, month: 8, year: 2025, datePrecision: "month" })}
      >
        Set Month And Year
      </button>
      <button
        data-testid="mock-calendar-change-year"
        onClick={() => onChange?.({ calendarType: "gregorian", day: null, month: null, year: 2025, datePrecision: "year" })}
      >
        Set Year Only
      </button>
      <button
        data-testid="mock-calendar-change-month-day"
        onClick={() => onChange?.({ calendarType: "gregorian", day: 15, month: 8, year: null, datePrecision: "month_day" })}
      >
        Set Month And Day
      </button>
      <button
        data-testid="mock-calendar-change-lunar-full"
        onClick={() => onChange?.({ calendarType: "lunar", day: 15, month: 1, year: 2025, datePrecision: "full" })}
      >
        Set Lunar Full Date
      </button>
    </div>
  ),
}));

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQuery: (opts: { queryKey: readonly unknown[] }) => {
      const key = JSON.stringify(opts.queryKey);
      if (key.includes("preferences")) return mockPrefsReturn;
      if (key.includes("date-types")) return mockDateTypesReturn;
      return mockDatesReturn;
    },
    useMutation: <TVariables,>(options: MutationOptions<TVariables>) => ({
      mutate: (values: TVariables) => {
        hoistedMocks.mutationMock.mutate(values);
        void options.mutationFn(values);
      },
      isPending: false,
    }),
    useQueryClient: () => ({ invalidateQueries: vi.fn() }),
  };
});

export const mockDates = [
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

beforeEach(() => {
  mockDatesReturn = { data: [], isLoading: false };
  mockPrefsReturn = { data: undefined };
  mockDateTypesReturn = { data: [], isLoading: false };
  mutationMock.mutate.mockClear();
  apiMock.contactsDatesCreate.mockClear();
  apiMock.contactsDatesUpdate.mockClear();
  apiMock.contactsDatesDelete.mockClear();
});

export function renderImportantDatesModule() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <ImportantDatesModule vaultId="v1" contactId="c1" />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}
