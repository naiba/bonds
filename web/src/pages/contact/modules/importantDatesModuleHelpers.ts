import type { Dayjs } from "dayjs";
import type { ImportantDate } from "@/api";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";
import { inferImportantDatePrecision } from "@/utils/importantDatePrecision";

export function buildImportantDatePickerValue(
  date: ImportantDate,
): CalendarDatePickerValue {
  const datePrecision = inferImportantDatePrecision(date);
  const calendarType = date.calendar_type === "lunar" ? "lunar" : "gregorian";

  if (
    datePrecision === "full"
    && calendarType !== "gregorian"
    && date.original_day != null
    && date.original_month != null
    && date.original_year != null
  ) {
    return {
      calendarType,
      day: date.original_day,
      month: date.original_month,
      year: date.original_year,
      datePrecision,
    };
  }

  return {
    calendarType: "gregorian",
    day: date.day ?? null,
    month: date.month ?? null,
    year: date.year ?? null,
    datePrecision,
  };
}

export function buildDefaultImportantDatePickerValue(
  referenceDate: Dayjs,
): CalendarDatePickerValue {
  return {
    calendarType: "gregorian",
    day: referenceDate.date(),
    month: referenceDate.month() + 1,
    year: referenceDate.year(),
    datePrecision: "full",
  };
}
