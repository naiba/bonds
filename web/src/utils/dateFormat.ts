import dayjs from "dayjs";
import type { Dayjs } from "dayjs";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api";

type DateInput = string | number | Date | Dayjs | null | undefined;

const DEFAULT_DATE_FORMAT = "MMM D, YYYY";

/**
 * Derive variant format strings from the user's base date format preference.
 *
 * User stores a "full date" format like "YYYY-MM-DD" or "MMM D, YYYY".
 * Many UI locations need shorter variants (month+year, short date, date+time).
 * This mapping keeps all variants consistent with the user's choice.
 *
 * Variant table:
 * | User preference | full          | monthYear | short  | dateTime              |
 * |-----------------|---------------|-----------|--------|-----------------------|
 * | YYYY-MM-DD      | YYYY-MM-DD    | YYYY-MM   | MM-DD  | YYYY-MM-DD HH:mm     |
 * | MM/DD/YYYY      | MM/DD/YYYY    | MM/YYYY   | MM/DD  | MM/DD/YYYY HH:mm     |
 * | DD/MM/YYYY      | DD/MM/YYYY    | MM/YYYY   | DD/MM  | DD/MM/YYYY HH:mm     |
 * | MMM D, YYYY     | MMM D, YYYY   | MMM YYYY  | MMM D  | MMM D, YYYY HH:mm    |
 * | MMM DD, YYYY    | MMM DD, YYYY  | MMM YYYY  | MMM DD | MMM DD, YYYY HH:mm   |
 */
export interface DateFormatVariants {
  /** Full date, e.g. "2026-03-12" or "Mar 12, 2026" */
  full: string;
  /** Month + year only, e.g. "2026-03" or "Mar 2026" */
  monthYear: string;
  /** Short date (no year), e.g. "03-12" or "Mar 12" */
  short: string;
  /** Full date + time, e.g. "2026-03-12 14:30" or "Mar 12, 2026 14:30" */
  dateTime: string;
  /** Full date + time with seconds, e.g. "2026-03-12 14:30:00" */
  dateTimeFull: string;
}

function buildVariants(fmt: string): DateFormatVariants {
  switch (fmt) {
    case "YYYY-MM-DD":
      return {
        full: "YYYY-MM-DD",
        monthYear: "YYYY-MM",
        short: "MM-DD",
        dateTime: "YYYY-MM-DD HH:mm",
        dateTimeFull: "YYYY-MM-DD HH:mm:ss",
      };
    case "MM/DD/YYYY":
      return {
        full: "MM/DD/YYYY",
        monthYear: "MM/YYYY",
        short: "MM/DD",
        dateTime: "MM/DD/YYYY HH:mm",
        dateTimeFull: "MM/DD/YYYY HH:mm:ss",
      };
    case "DD/MM/YYYY":
      return {
        full: "DD/MM/YYYY",
        monthYear: "MM/YYYY",
        short: "DD/MM",
        dateTime: "DD/MM/YYYY HH:mm",
        dateTimeFull: "DD/MM/YYYY HH:mm:ss",
      };
    // "MMM D, YYYY" and legacy "MMM DD, YYYY" both map to the same variants
    default:
      return {
        full: "MMM D, YYYY",
        monthYear: "MMM YYYY",
        short: "MMM D",
        dateTime: "MMM D, YYYY HH:mm",
        dateTimeFull: "MMM D, YYYY HH:mm:ss",
      };
  }
}

/**
 * Hook that returns the user's date format preference and pre-built variants.
 * Shares the same query cache as Preferences page (queryKey: ["settings", "preferences"]).
 */
export function useDateFormat(): DateFormatVariants {
  const { data } = useQuery({
    queryKey: ["settings", "preferences"],
    queryFn: async () => {
      const res = await api.preferences.preferencesList();
      return res.data!;
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
  });

  return buildVariants(data?.date_format || DEFAULT_DATE_FORMAT);
}

/** Format a date using the full date format (e.g. "Mar 12, 2026" or "2026-03-12"). */
export function formatDate(date: DateInput, variants: DateFormatVariants): string {
  if (!date) return "";
  return dayjs(date).format(variants.full);
}

/** Format a date with time (e.g. "Mar 12, 2026 14:30" or "2026-03-12 14:30"). */
export function formatDateTime(date: DateInput, variants: DateFormatVariants): string {
  if (!date) return "";
  return dayjs(date).format(variants.dateTime);
}

/** Format a date with full time including seconds. */
export function formatDateTimeFull(date: DateInput, variants: DateFormatVariants): string {
  if (!date) return "";
  return dayjs(date).format(variants.dateTimeFull);
}

/** Format month + year only (e.g. "Mar 2026" or "2026-03"). */
export function formatMonthYear(date: DateInput, variants: DateFormatVariants): string {
  if (!date) return "";
  return dayjs(date).format(variants.monthYear);
}

/** Format short date without year (e.g. "Mar 12" or "03-12"). Used for task due dates etc. */
export function formatShortDate(date: DateInput, variants: DateFormatVariants): string {
  if (!date) return "";
  return dayjs(date).format(variants.short);
}
