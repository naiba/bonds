import dayjs from "dayjs";
import type { Dayjs } from "dayjs";
import type { DateFormatVariants } from "@/utils/dateFormat";

const dateInputPattern = /^\d{4}-\d{2}-\d{2}$/;

export function dateInputToTimestamp(value?: string): string | undefined {
  if (!value) return undefined;
  if (!dateInputPattern.test(value)) return undefined;
  return `${value}T00:00:00Z`;
}

export function timestampToDateInput(value?: string): string | undefined {
  if (!value) return undefined;
  const datePrefix = value.slice(0, 10);
  return dateInputPattern.test(datePrefix) ? datePrefix : undefined;
}

export function parseDateOnly(value?: string): Dayjs | undefined {
  const dateInput = timestampToDateInput(value);
  return dateInput ? dayjs(dateInput) : undefined;
}

export function formatDateOnly(
  value: string | undefined,
  variants: DateFormatVariants,
): string {
  return parseDateOnly(value)?.format(variants.full) ?? "";
}

export function formatMonthYearOnly(
  value: string | undefined,
  variants: DateFormatVariants,
): string {
  return parseDateOnly(value)?.format(variants.monthYear) ?? "";
}

export function formatYearOnly(value?: string): string {
  return parseDateOnly(value)?.format("YYYY") ?? "";
}

export function formatShortDateOnly(
  value: string | undefined,
  variants: DateFormatVariants,
): string {
  return parseDateOnly(value)?.format(variants.short) ?? "";
}

/**
 * Whether a stored timestamp is a calendar date (exactly midnight UTC) rather
 * than an instant. Fields that used to save the picker's value as an instant
 * keep rows with a time of day; those must go on being shown as instants.
 */
export function isDateOnlyTimestamp(value?: string): boolean {
  if (!value) return false;
  const milliseconds = Date.parse(value);
  return !Number.isNaN(milliseconds) && milliseconds % 86_400_000 === 0;
}
