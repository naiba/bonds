import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  mockDateTypesReturn,
  mockDates,
  mockDatesReturn,
  mockPrefsReturn,
  renderImportantDatesModule,
} from "./importantDatesModuleTestHarness";

describe("ImportantDatesModule display", () => {
  it("renders title and add button", () => {
    renderImportantDatesModule();
    expect(screen.getByText("Important Dates")).toBeInTheDocument();
    expect(screen.getByText("Add")).toBeInTheDocument();
  });

  it("renders empty state", () => {
    renderImportantDatesModule();
    expect(screen.getByText("No important dates")).toBeInTheDocument();
  });

  it("renders important dates list", () => {
    Object.assign(mockDatesReturn as object, { data: mockDates, isLoading: false });
    renderImportantDatesModule();
    expect(screen.getByText("Birthday")).toBeInTheDocument();
    expect(screen.getByText("Lunar NY")).toBeInTheDocument();
  });

  it("renders lunar date with tag when alternative calendar enabled", () => {
    Object.assign(mockDatesReturn as object, { data: mockDates, isLoading: false });
    Object.assign(mockPrefsReturn as object, { data: { enable_alternative_calendar: true } });
    renderImportantDatesModule();
    expect(screen.getByText("lunar")).toBeInTheDocument();
  });

  it("renders gregorian date without tag", () => {
    Object.assign(mockDatesReturn as object, { data: [mockDates[0]], isLoading: false });
    renderImportantDatesModule();
    expect(screen.getByText("Birthday")).toBeInTheDocument();
    expect(screen.queryByText("gregorian")).not.toBeInTheDocument();
  });

  it("shows age next to birthdate when contact is alive", () => {
    Object.assign(mockDateTypesReturn as object, {
      data: [
        { id: 10, label: "Birthdate", internal_type: "birthdate", can_be_deleted: false },
        { id: 11, label: "Deceased date", internal_type: "deceased_date", can_be_deleted: false },
      ],
    });
    Object.assign(mockDatesReturn as object, {
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
    });
    renderImportantDatesModule();
    expect(screen.getByText(/years old|year old/)).toBeInTheDocument();
  });

  it("shows age-at-death next to deceased date, not birthdate", () => {
    Object.assign(mockDateTypesReturn as object, {
      data: [
        { id: 10, label: "Birthdate", internal_type: "birthdate", can_be_deleted: false },
        { id: 11, label: "Deceased date", internal_type: "deceased_date", can_be_deleted: false },
      ],
    });
    Object.assign(mockDatesReturn as object, {
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
    });
    renderImportantDatesModule();
    expect(screen.getByText(/69 years old/)).toBeInTheDocument();
    expect(screen.getAllByText(/years old|year old/).length).toBe(1);
  });

  it("displays date without year using short format", () => {
    Object.assign(mockDatesReturn as object, {
      data: [{
        id: 3,
        contact_id: "c1",
        label: "Nameday",
        day: 15,
        month: 6,
        year: null,
        date_precision: "month_day",
        calendar_type: "gregorian",
        original_day: null,
        original_month: null,
        original_year: null,
        contact_important_date_type_id: null,
        created_at: "2025-01-01",
        updated_at: "2025-01-01",
      }],
      isLoading: false,
    });
    renderImportantDatesModule();
    expect(screen.getByText("Nameday")).toBeInTheDocument();
    expect(screen.getByText(/Jun 15/)).toBeInTheDocument();
  });

  it("displays month-year dates", () => {
    Object.assign(mockDatesReturn as object, {
      data: [{
        id: 4,
        contact_id: "c1",
        label: "Graduation",
        day: null,
        month: 8,
        year: 2025,
        date_precision: "month",
        calendar_type: "gregorian",
        original_day: null,
        original_month: null,
        original_year: null,
        contact_important_date_type_id: null,
        created_at: "2025-01-01",
        updated_at: "2025-01-01",
      }],
      isLoading: false,
    });
    renderImportantDatesModule();
    expect(screen.getByText("Graduation")).toBeInTheDocument();
    expect(screen.getByText(/Aug 2025/)).toBeInTheDocument();
  });

  it("displays year-only dates", () => {
    Object.assign(mockDatesReturn as object, {
      data: [{
        id: 5,
        contact_id: "c1",
        label: "Founded",
        day: null,
        month: null,
        year: 2025,
        date_precision: "year",
        calendar_type: "gregorian",
        original_day: null,
        original_month: null,
        original_year: null,
        contact_important_date_type_id: null,
        created_at: "2025-01-01",
        updated_at: "2025-01-01",
      }],
      isLoading: false,
    });
    renderImportantDatesModule();
    expect(screen.getByText("Founded")).toBeInTheDocument();
    expect(screen.getByText("2025")).toBeInTheDocument();
  });

  it("restores lunar full-date edits using original calendar fields", async () => {
    const user = userEvent.setup();
    Object.assign(mockDatesReturn as object, {
      data: [mockDates[1]],
      isLoading: false,
    });
    Object.assign(mockPrefsReturn as object, {
      data: { enable_alternative_calendar: true },
    });
    renderImportantDatesModule();

    await user.click(screen.getByRole("button", { name: "edit" }));

    expect(screen.getByTestId("calendar-picker-value")).toHaveTextContent('"calendarType":"lunar"');
    expect(screen.getByTestId("calendar-picker-value")).toHaveTextContent('"day":15');
    expect(screen.getByTestId("calendar-picker-value")).toHaveTextContent('"month":1');
    expect(screen.getByTestId("calendar-picker-value")).toHaveTextContent('"year":2025');
  });
});
