import dayjs from "dayjs";
import type { Dayjs } from "dayjs";
import type { ImportantDate } from "@/api";
import { getCalendarSystem } from "@/utils/calendar";
import type { CalendarType } from "@/utils/calendar";
import type { DateFormatVariants } from "@/utils/dateFormat";

function inferDisplayPrecision(date: ImportantDate): string {
  if (date.date_precision) {
    return date.date_precision;
  }

  if (date.day != null && date.month != null && date.year != null) {
    return "full";
  }

  if (date.day == null && date.month != null && date.year != null) {
    return "month";
  }

  if (date.day == null && date.month == null && date.year != null) {
    return "year";
  }

  if (date.day != null && date.month != null && date.year == null) {
    return "month_day";
  }

  return "";
}

export function computeImportantDateAge(date: ImportantDate, reference: Dayjs = dayjs()): number | null {
  if (date.year == null || date.month == null || date.day == null) return null;

  const birth = dayjs(new Date(date.year, date.month - 1, date.day));
  if (!birth.isValid() || birth.isAfter(reference)) return null;

  let age = reference.year() - birth.year();
  if (reference.month() < birth.month() || (reference.month() === birth.month() && reference.date() < birth.date())) {
    age -= 1;
  }

  return age >= 0 ? age : null;
}

export function formatImportantDateDisplay(date: ImportantDate, formats: DateFormatVariants): string {
  const precision = inferDisplayPrecision(date);
  if (
    precision === "full"
    && date.calendar_type
    && date.calendar_type !== "gregorian"
    && date.original_month != null
    && date.original_day != null
    && date.original_year != null
  ) {
    const calendar = getCalendarSystem(date.calendar_type as CalendarType);
    const originalDate = calendar.formatDate({
      day: date.original_day,
      month: date.original_month,
      year: date.original_year,
    });
    const gregorianDate = date.year != null && date.month != null && date.day != null
      ? dayjs(new Date(date.year, date.month - 1, date.day)).format(formats.full)
      : "";
    return gregorianDate ? `${originalDate} (${gregorianDate})` : originalDate;
  }

  if (precision === "full" && date.year != null && date.month != null && date.day != null) {
    return dayjs(new Date(date.year, date.month - 1, date.day)).format(formats.full);
  }

  if (precision === "month" && date.year != null && date.month != null) {
    return dayjs(new Date(date.year, date.month - 1, 1)).format(formats.monthYear);
  }

  if (precision === "year" && date.year != null) {
    return String(date.year);
  }

  if (precision === "month_day" && date.month != null && date.day != null) {
    return dayjs(new Date(2000, date.month - 1, date.day)).format(formats.short);
  }

  return "";
}

export function computeAgeAtImportantDate(subjectDate: ImportantDate | undefined, referenceDate: ImportantDate | undefined): number | null {
  if (
    !subjectDate
    || referenceDate?.year == null
    || referenceDate.month == null
    || referenceDate.day == null
  ) return null;

  const reference = dayjs(new Date(referenceDate.year, referenceDate.month - 1, referenceDate.day));
  return computeImportantDateAge(subjectDate, reference);
}
