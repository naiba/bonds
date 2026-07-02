import { supportedCalendarTypes } from "@/utils/calendar";
import type { CalendarType } from "@/utils/calendar";

export type ImportantDatePrecision = "full" | "month" | "year" | "month_day";

const importantDatePrecisionValues = [
  "full",
  "month",
  "year",
  "month_day",
] as const;

export interface CalendarDatePickerValue {
  calendarType: CalendarType;
  day: number | null;
  month: number | null;
  year: number | null;
  datePrecision?: ImportantDatePrecision;
}

export function isCalendarType(value: string | number): value is CalendarType {
  return typeof value === "string"
    && supportedCalendarTypes.some((supportedType) => supportedType === value);
}

export function isImportantDatePrecision(
  value: string | number,
): value is ImportantDatePrecision {
  return typeof value === "string"
    && importantDatePrecisionValues.some((precision) => precision === value);
}

export function inferPrecisionFromValue(
  value: CalendarDatePickerValue | undefined,
): ImportantDatePrecision {
  if (value?.datePrecision) {
    return value.datePrecision;
  }

  if (value?.day != null && value.month != null && value.year == null) {
    return "month_day";
  }

  if (value?.day == null && value?.month != null && value.year != null) {
    return "month";
  }

  if (value?.day == null && value?.month == null && value?.year != null) {
    return "year";
  }

  return "full";
}
