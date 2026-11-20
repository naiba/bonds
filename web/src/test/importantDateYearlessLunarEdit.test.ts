import { describe, expect, it } from "vitest";
import type { ImportantDate } from "@/api";
import { buildImportantDatePickerValue } from "@/pages/contact/modules/importantDatesModuleHelpers";
import { buildImportantDateRequest } from "@/utils/importantDatePrecision";

describe("editing a yearless lunar important date", () => {
  // What the server returns for "lunar 8th month, 15th day, no year": month/day
  // hold a Gregorian storage projection, original_* hold what the user chose.
  const yearlessLunarDate: ImportantDate = {
    id: 7,
    label: "Mid-Autumn birthday",
    date_precision: "month_day",
    calendar_type: "lunar",
    day: 6,
    month: 10,
    original_day: 15,
    original_month: 8,
  };

  it("opens the edit form on the lunar month and day the user entered", () => {
    expect(buildImportantDatePickerValue(yearlessLunarDate)).toEqual({
      calendarType: "lunar",
      day: 15,
      month: 8,
      year: null,
      datePrecision: "month_day",
    });
  });

  it("stays a lunar date when saved again without changes", () => {
    const request = buildImportantDateRequest(
      {
        label: "Mid-Autumn birthday",
        calendarDate: buildImportantDatePickerValue(yearlessLunarDate),
      },
      "Mid-Autumn birthday",
    );

    expect(request.calendar_type).toBe("lunar");
    expect(request.original_month).toBe(8);
    expect(request.original_day).toBe(15);
    expect(request.date_precision).toBe("month_day");
  });

  it("keeps yearless Gregorian dates Gregorian", () => {
    expect(
      buildImportantDatePickerValue({
        id: 8,
        date_precision: "month_day",
        calendar_type: "gregorian",
        day: 6,
        month: 10,
      }),
    ).toEqual({
      calendarType: "gregorian",
      day: 6,
      month: 10,
      year: null,
      datePrecision: "month_day",
    });
  });
});
