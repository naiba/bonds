import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  apiMock,
  mockDateTypesReturn,
  mockDatesReturn,
  mockPrefsReturn,
  renderImportantDatesModule,
} from "./importantDatesModuleTestHarness";

describe("ImportantDatesModule payloads", () => {
  it("submits month-year payload without faking day", async () => {
    const user = userEvent.setup();
    renderImportantDatesModule();

    await user.click(screen.getByText("Add"));
    await user.click(screen.getByTestId("mock-calendar-change-month"));
    await user.type(screen.getByRole("textbox", { name: /Label|label/i }), "Partial Test");
    await user.click(screen.getByRole("button", { name: /Save|OK/i }));

    await waitFor(() => {
      expect(apiMock.contactsDatesCreate).toHaveBeenCalled();
    });

    const payload = apiMock.contactsDatesCreate.mock.calls[0][2];
    expect(payload).toMatchObject({
      label: "Partial Test",
      date_precision: "month",
      month: 8,
      year: 2025,
      calendar_type: "gregorian",
      remind_me: false,
    });
    expect(payload.day).toBeUndefined();
  });

  it("submits year-only payload without faking month or day", async () => {
    const user = userEvent.setup();
    renderImportantDatesModule();

    await user.click(screen.getByText("Add"));
    await user.click(screen.getByTestId("mock-calendar-change-year"));
    await user.type(screen.getByRole("textbox", { name: /Label|label/i }), "Founding Year");
    await user.click(screen.getByRole("button", { name: /Save|OK/i }));

    await waitFor(() => {
      expect(apiMock.contactsDatesCreate).toHaveBeenCalled();
    });

    const payload = apiMock.contactsDatesCreate.mock.calls[0][2];
    expect(payload).toMatchObject({
      label: "Founding Year",
      date_precision: "year",
      year: 2025,
      calendar_type: "gregorian",
      remind_me: false,
    });
    expect(payload.month).toBeUndefined();
    expect(payload.day).toBeUndefined();
  });

  it("submits month-day payload without faking year", async () => {
    const user = userEvent.setup();
    renderImportantDatesModule();

    await user.click(screen.getByText("Add"));
    await user.click(screen.getByTestId("mock-calendar-change-month-day"));
    await user.type(screen.getByRole("textbox", { name: /Label|label/i }), "Nameday");
    await user.click(screen.getByRole("button", { name: /Save|OK/i }));

    await waitFor(() => {
      expect(apiMock.contactsDatesCreate).toHaveBeenCalled();
    });

    const payload = apiMock.contactsDatesCreate.mock.calls[0][2];
    expect(payload).toMatchObject({
      label: "Nameday",
      date_precision: "month_day",
      month: 8,
      day: 15,
      calendar_type: "gregorian",
    });
    expect(payload.year).toBeUndefined();
  });

  it("submits a new date with the default calendar date when fields are unchanged", async () => {
    const user = userEvent.setup();

    renderImportantDatesModule();

    await user.click(screen.getByRole("button", { name: /add/i }));
    await user.type(screen.getByRole("textbox", { name: /label/i }), "Graduation Day");
    await user.click(screen.getByRole("button", { name: /ok/i }));

    await waitFor(() => {
      expect(apiMock.contactsDatesCreate).toHaveBeenCalled();
    });

    const payload = apiMock.contactsDatesCreate.mock.calls[0][2];
    expect(payload).toMatchObject({
      label: "Graduation Day",
      date_precision: "full",
      calendar_type: "gregorian",
    });
    expect(payload.day).toEqual(expect.any(Number));
    expect(payload.month).toEqual(expect.any(Number));
    expect(payload.year).toEqual(expect.any(Number));
  });

  it("submits full lunar dates with converted gregorian fields and original fields", async () => {
    const user = userEvent.setup();
    Object.assign(mockPrefsReturn as object, {
      data: { enable_alternative_calendar: true },
    });
    renderImportantDatesModule();

    await user.click(screen.getByText("Add"));
    await user.click(screen.getByTestId("mock-calendar-change-lunar-full"));
    await user.type(screen.getByRole("textbox", { name: /Label|label/i }), "Lunar Birthday");
    await user.click(screen.getByRole("button", { name: /Save|OK/i }));

    await waitFor(() => {
      expect(apiMock.contactsDatesCreate).toHaveBeenCalled();
    });

    const payload = apiMock.contactsDatesCreate.mock.calls.at(-1)?.[2];
    expect(payload).toMatchObject({
      label: "Lunar Birthday",
      date_precision: "full",
      calendar_type: "lunar",
      original_day: 15,
      original_month: 1,
      original_year: 2025,
      day: 12,
      month: 2,
      year: 2025,
    });
  });

  it("clears remind_me when switching to year-only precision", async () => {
    const user = userEvent.setup();
    Object.assign(mockDateTypesReturn as object, {
      data: [
        { id: 10, label: "Birthdate", internal_type: "birthdate", can_be_deleted: false },
      ],
    });
    renderImportantDatesModule();

    await user.click(screen.getByText("Add"));
    await user.click(screen.getByTestId("mock-calendar-change-full"));

    const typeSelect = screen.getByRole("combobox");
    await user.click(typeSelect);
    await user.click(await screen.findByTitle("Birthdate"));

    const remindToggle = screen.getByRole("switch");
    expect(remindToggle).toBeEnabled();

    await user.click(screen.getByTestId("mock-calendar-change-year"));
    expect(remindToggle).toBeDisabled();
    expect(remindToggle).toHaveAttribute("aria-checked", "false");

    await user.click(screen.getByRole("button", { name: /Save|OK/i }));
    await waitFor(() => {
      expect(apiMock.contactsDatesCreate).toHaveBeenCalled();
    });

    const payload = apiMock.contactsDatesCreate.mock.calls.at(-1)?.[2];
    expect(payload.remind_me).toBe(false);
  });

  it("restores existing year-only dates and keeps sparse update payload", async () => {
    const user = userEvent.setup();
    Object.assign(mockDatesReturn as object, {
      data: [{
        id: 77,
        contact_id: "c1",
        label: "Founding Year",
        day: null,
        month: null,
        year: 2025,
        date_precision: "year",
        calendar_type: "gregorian",
        original_day: null,
        original_month: null,
        original_year: null,
        contact_important_date_type_id: null,
      }],
      isLoading: false,
    });
    renderImportantDatesModule();

    await user.click(screen.getByRole("button", { name: "edit" }));
    expect(screen.getByTestId("calendar-picker-value")).toHaveTextContent('"datePrecision":"year"');

    await user.click(screen.getByRole("button", { name: /Save|OK/i }));
    await waitFor(() => {
      expect(apiMock.contactsDatesUpdate).toHaveBeenCalled();
    });

    const payload = apiMock.contactsDatesUpdate.mock.calls.at(-1)?.[3];
    expect(payload).toMatchObject({
      label: "Founding Year",
      date_precision: "year",
      year: 2025,
      calendar_type: "gregorian",
      remind_me: false,
    });
    expect(payload.month).toBeUndefined();
    expect(payload.day).toBeUndefined();
  });
});
