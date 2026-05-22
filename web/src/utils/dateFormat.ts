import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone";
import type { Dayjs } from "dayjs";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api";

// Plug timezone + utc support into the global dayjs. Without these,
// dayjs.tz() throws and the format functions below silently return browser-
// local times — which means a user in Asia/Tokyo viewing the app from a
// server logging UTC sees timestamps off by 9 hours.
dayjs.extend(utc);
dayjs.extend(timezone);

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
  /**
   * IANA timezone the user has saved in Preferences (e.g. "Asia/Tokyo").
   * The format helpers below pin date arithmetic to this zone — without it
   * timestamps render in the browser's tz, so a UTC server log of "09:00Z"
   * appears as "18:00" to someone in Tokyo. Optional because the hook
   * gracefully falls back to local time when the preference isn't loaded.
   */
  tz?: string;
}

function buildVariants(fmt: string, tz?: string): DateFormatVariants {
  switch (fmt) {
    case "YYYY-MM-DD":
      return {
        full: "YYYY-MM-DD",
        monthYear: "YYYY-MM",
        short: "MM-DD",
        dateTime: "YYYY-MM-DD HH:mm",
        dateTimeFull: "YYYY-MM-DD HH:mm:ss",
        tz,
      };
    case "MM/DD/YYYY":
      return {
        full: "MM/DD/YYYY",
        monthYear: "MM/YYYY",
        short: "MM/DD",
        dateTime: "MM/DD/YYYY HH:mm",
        dateTimeFull: "MM/DD/YYYY HH:mm:ss",
        tz,
      };
    case "DD/MM/YYYY":
      return {
        full: "DD/MM/YYYY",
        monthYear: "MM/YYYY",
        short: "DD/MM",
        dateTime: "DD/MM/YYYY HH:mm",
        dateTimeFull: "DD/MM/YYYY HH:mm:ss",
        tz,
      };
    // "MMM D, YYYY" and legacy "MMM DD, YYYY" both map to the same variants
    default:
      return {
        full: "MMM D, YYYY",
        monthYear: "MMM YYYY",
        short: "MMM D",
        dateTime: "MMM D, YYYY HH:mm",
        dateTimeFull: "MMM D, YYYY HH:mm:ss",
        tz,
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

  return buildVariants(data?.date_format || DEFAULT_DATE_FORMAT, data?.timezone);
}

function applyTz(d: dayjs.Dayjs, tz?: string): dayjs.Dayjs {
  if (!tz) return d;
  try {
    return d.tz(tz);
  } catch {
    // An invalid timezone string (e.g. a legacy preference we no longer
    // support) shouldn't blow up the entire formatter. Fall back to
    // browser-local rather than throwing.
    return d;
  }
}

/** Format a date using the full date format (e.g. "Mar 12, 2026" or "2026-03-12"). */
export function formatDate(date: DateInput, variants: DateFormatVariants): string {
  if (!date) return "";
  return applyTz(dayjs(date), variants.tz).format(variants.full);
}

/** Format a date with time (e.g. "Mar 12, 2026 14:30" or "2026-03-12 14:30"). */
export function formatDateTime(date: DateInput, variants: DateFormatVariants): string {
  if (!date) return "";
  return applyTz(dayjs(date), variants.tz).format(variants.dateTime);
}

/** Format a date with full time including seconds. */
export function formatDateTimeFull(date: DateInput, variants: DateFormatVariants): string {
  if (!date) return "";
  return applyTz(dayjs(date), variants.tz).format(variants.dateTimeFull);
}

/** Format month + year only (e.g. "Mar 2026" or "2026-03"). */
export function formatMonthYear(date: DateInput, variants: DateFormatVariants): string {
  if (!date) return "";
  return applyTz(dayjs(date), variants.tz).format(variants.monthYear);
}

/** Format short date without year (e.g. "Mar 12" or "03-12"). Used for task due dates etc. */
export function formatShortDate(date: DateInput, variants: DateFormatVariants): string {
  if (!date) return "";
  return applyTz(dayjs(date), variants.tz).format(variants.short);
}
