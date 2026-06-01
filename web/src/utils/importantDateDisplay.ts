import dayjs from "dayjs";
import type { Dayjs } from "dayjs";
import type { ImportantDate } from "@/api";
import { getCalendarSystem } from "@/utils/calendar";
import type { CalendarType } from "@/utils/calendar";
import type { DateFormatVariants } from "@/utils/dateFormat";

export function computeImportantDateAge(date: ImportantDate, reference: Dayjs = dayjs()): number | null {
  if (!date.year || !date.month || !date.day) return null;

  const birth = dayjs(new Date(date.year, date.month - 1, date.day));
  if (!birth.isValid() || birth.isAfter(reference)) return null;

  let age = reference.year() - birth.year();
  if (reference.month() < birth.month() || (reference.month() === birth.month() && reference.date() < birth.date())) {
    age -= 1;
  }

  return age >= 0 ? age : null;
}

export function formatImportantDateDisplay(date: ImportantDate, formats: DateFormatVariants): string {
  if (date.calendar_type && date.calendar_type !== "gregorian" && date.original_month != null && date.original_day != null) {
    const calendar = getCalendarSystem(date.calendar_type as CalendarType);
    const originalDate = calendar.formatDate({
      day: date.original_day,
      month: date.original_month,
      year: date.original_year ?? 0,
    });
    const gregorianDate = date.year && date.month && date.day
      ? dayjs(new Date(date.year, date.month - 1, date.day)).format(formats.full)
      : "";
    return gregorianDate ? `${originalDate} (${gregorianDate})` : originalDate;
  }

  if (date.year && date.month && date.day) {
    return dayjs(new Date(date.year, date.month - 1, date.day)).format(formats.full);
  }

  if (!date.year && date.month && date.day) {
    return dayjs(new Date(2000, date.month - 1, date.day)).format(formats.short);
  }

  return "";
}

export function computeAgeAtImportantDate(subjectDate: ImportantDate | undefined, referenceDate: ImportantDate | undefined): number | null {
  if (!subjectDate || !referenceDate?.year || !referenceDate.month || !referenceDate.day) return null;

  const reference = dayjs(new Date(referenceDate.year, referenceDate.month - 1, referenceDate.day));
  return computeImportantDateAge(subjectDate, reference);
}
