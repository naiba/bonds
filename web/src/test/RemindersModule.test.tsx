import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import RemindersModule from "@/pages/contact/modules/RemindersModule";
import { api } from "@/api";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/components/CalendarDatePicker", () => ({
  default: ({ onChange, allowedDatePrecisions }: { onChange?: (value: { calendarType: string; day: number | null; month: number | null; year: number | null; datePrecision?: string }) => void; allowedDatePrecisions?: readonly string[] }) => (
    <div data-testid="calendar-date-picker">
      <output data-testid="allowed-date-precisions">{JSON.stringify(allowedDatePrecisions ?? [])}</output>
      <button
        data-testid="reminder-full-date"
        onClick={() => onChange?.({ calendarType: "gregorian", day: 15, month: 3, year: 2026, datePrecision: "full" })}
      >
        Reminder full date
      </button>
      <button
        data-testid="reminder-month-day"
        onClick={() => onChange?.({ calendarType: "gregorian", day: 15, month: 3, year: null, datePrecision: "month_day" })}
      >
        Reminder month day
      </button>
    </div>
  ),
}));

vi.mock("@/api", () => ({
  api: {
    reminders: {
      contactsRemindersList: vi.fn(),
      contactsRemindersCreate: vi.fn(),
      contactsRemindersUpdate: vi.fn(),
      contactsRemindersDelete: vi.fn(),
    },
    preferences: {
      preferencesList: vi.fn(),
    },
  },
}));

let mockRemindersReturn: unknown = { data: [], isLoading: false };
let mockPrefsReturn: unknown = { data: undefined };
const mutationMock = vi.hoisted(() => ({
  mutate: vi.fn(),
}));
vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: { queryKey: unknown[] }) => {
    const key = JSON.stringify(opts.queryKey);
    if (key.includes("preferences")) return mockPrefsReturn;
    return mockRemindersReturn;
  },
  useMutation: (options?: { mutationFn?: (variables: unknown) => unknown }) => ({
    mutate: vi.fn(async (variables: unknown) => {
      mutationMock.mutate(variables);
      return options?.mutationFn?.(variables);
    }),
    isPending: false,
  }),
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
  beforeEach(() => {
    vi.clearAllMocks();
    mutationMock.mutate.mockReset();
    mockRemindersReturn = { data: [], isLoading: false };
    mockPrefsReturn = { data: undefined };
    vi.mocked(api.reminders.contactsRemindersList).mockResolvedValue({ data: [] });
    vi.mocked(api.reminders.contactsRemindersCreate).mockResolvedValue({ data: {} });
    vi.mocked(api.reminders.contactsRemindersUpdate).mockResolvedValue({ data: {} });
    vi.mocked(api.reminders.contactsRemindersDelete).mockResolvedValue({ data: {} });
    vi.mocked(api.preferences.preferencesList).mockResolvedValue({ data: undefined });
  });

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
    const recurringTags = screen.getAllByText("Yearly");
    expect(recurringTags.length).toBeGreaterThanOrEqual(1);
  });

  it("renders year-null yearly reminders without a year in the display", () => {
    mockRemindersReturn = {
      data: [{
        ...mockReminders[0],
        year: null,
      }],
      isLoading: false,
    };
    renderModule();
    expect(screen.getByText(/Mar 15/)).toBeInTheDocument();
  });

  it("passes only full and month_day precision options to reminder date picker", async () => {
    const user = userEvent.setup();
    renderModule();

    await user.click(screen.getByText("Add"));

    expect(await screen.findByTestId("allowed-date-precisions")).toHaveTextContent('["full","month_day"]');
  });

  it("submits a yearless month_day reminder without fabricating a year", async () => {
    const user = userEvent.setup();
    renderModule();

    await user.click(screen.getByText("Add"));
    await user.type(await screen.findByRole("textbox", { name: /label/i }), "Yearless Reminder");
    await user.click(screen.getByTestId("reminder-month-day"));
    await user.click(screen.getByRole("combobox", { name: /frequency/i }));
    await user.click(await screen.findByTitle("Yearly"));
    await user.click(screen.getByRole("button", { name: /ok|save/i }));

    await waitFor(() => {
      expect(api.reminders.contactsRemindersCreate).toHaveBeenCalledWith(
        "v1",
        "c1",
        expect.objectContaining({
          label: "Yearless Reminder",
          day: 15,
          month: 3,
          type: "recurring_year",
        }),
      );
    });

    const payload = vi.mocked(api.reminders.contactsRemindersCreate).mock.calls.at(-1)?.[2];
    expect(payload?.year).toBeUndefined();
  });

  it("keeps the calendar picker mounted for full-date reminder consumers", () => {
    renderModule();
    expect(screen.queryByTestId("calendar-date-picker")).not.toBeInTheDocument();
    expect(screen.getByText("Add")).toBeInTheDocument();
  });
});
