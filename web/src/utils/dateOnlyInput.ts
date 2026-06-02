import dayjs from "dayjs";
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

export function formatDateOnly(value: string | undefined, variants: DateFormatVariants): string {
  const dateInput = timestampToDateInput(value);
  return dateInput ? dayjs(dateInput).format(variants.full) : "";
}

export function formatShortDateOnly(value: string | undefined, variants: DateFormatVariants): string {
  const dateInput = timestampToDateInput(value);
  return dateInput ? dayjs(dateInput).format(variants.short) : "";
}
