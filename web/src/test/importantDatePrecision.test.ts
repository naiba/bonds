import { describe, expect, it } from "vitest";
import { buildImportantDateRequest, inferImportantDatePrecision } from "@/utils/importantDatePrecision";
import type { ImportantDate } from "@/api";
import type { ImportantDateFormValues } from "@/utils/importantDatePrecision";

function buildValues(calendarDate: ImportantDateFormValues["calendarDate"]): ImportantDateFormValues {
  return {
    label: "",
    calendarDate,
    contact_important_date_type_id: 7,
    remind_me: true,
  };
}

describe("important date precision helpers", () => {
  it("builds a full date request with converted gregorian fields", () => {
    const request = buildImportantDateRequest(
      buildValues({ calendarType: "gregorian", day: 15, month: 8, year: 2025, datePrecision: "full" }),
      "Fallback",
    );

    expect(request).toMatchObject({
      label: "Fallback",
      date_precision: "full",
      calendar_type: "gregorian",
      day: 15,
      month: 8,
      year: 2025,
      contact_important_date_type_id: 7,
      remind_me: true,
    });
  });

  it("builds a full lunar date request with original and converted fields", () => {
    const request = buildImportantDateRequest(
      buildValues({
        calendarType: "lunar",
        day: 15,
        month: 1,
        year: 2025,
        datePrecision: "full",
      }),
      "Fallback",
    );

    expect(request).toMatchObject({
      label: "Fallback",
      date_precision: "full",
      calendar_type: "lunar",
      original_day: 15,
      original_month: 1,
      original_year: 2025,
      day: 12,
      month: 2,
      year: 2025,
      remind_me: true,
    });
  });

  it("builds partial requests without faking omitted date fields", () => {
    expect(
      buildImportantDateRequest(
        buildValues({ calendarType: "gregorian", day: null, month: 8, year: 2025, datePrecision: "month" }),
        "Fallback",
      ),
    ).toMatchObject({ date_precision: "month", month: 8, year: 2025 });

    expect(
      buildImportantDateRequest(
        buildValues({ calendarType: "gregorian", day: null, month: null, year: 2025, datePrecision: "year" }),
        "Fallback",
      ),
    ).toMatchObject({ date_precision: "year", year: 2025 });

    expect(
      buildImportantDateRequest(
        buildValues({ calendarType: "gregorian", day: 15, month: 8, year: null, datePrecision: "month_day" }),
        "Fallback",
      ),
    ).toMatchObject({ date_precision: "month_day", day: 15, month: 8 });

    expect(
      buildImportantDateRequest(
        buildValues({ calendarType: "lunar", day: null, month: 8, year: 2025, datePrecision: "month" }),
        "Fallback",
      ),
    ).toMatchObject({ date_precision: "month", calendar_type: "gregorian", month: 8, year: 2025 });
  });

  it("forces remind_me false when precision cannot schedule reminders", () => {
    const request = buildImportantDateRequest(
      buildValues({ calendarType: "gregorian", day: null, month: null, year: 2025, datePrecision: "year" }),
      "Fallback",
    );

    expect(request.remind_me).toBe(false);
  });

  it("infers backend precision when older responses omit date_precision", () => {
    expect(inferImportantDatePrecision({ day: 15, month: 8, year: 2025 } as ImportantDate)).toBe("full");
    expect(inferImportantDatePrecision({ day: undefined, month: 8, year: 2025 } as ImportantDate)).toBe("month");
    expect(inferImportantDatePrecision({ day: undefined, month: undefined, year: 2025 } as ImportantDate)).toBe("year");
    expect(inferImportantDatePrecision({ day: 15, month: 8, year: undefined } as ImportantDate)).toBe("month_day");
  });
});
