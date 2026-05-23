import dayjs from "dayjs";
import type { Dayjs } from "dayjs";
import type { CalendarType } from "@/utils/calendar";

// The shape CalendarAwareDatePicker round-trips through AntD Form: a
// Gregorian Dayjs (so existing onSubmit code can keep calling
// .toISOString()) plus the alternate-calendar metadata the backend persists
// alongside it.
//
// When the user is in pure-Gregorian mode the metadata stays empty — the API
// DTO defaults calendar_type to "gregorian" and clears Original*, so
// untouched code paths keep behaving the way they did before D16-3.
//
// Lives next to (rather than inside) the component file so the picker
// component can keep `react-refresh/only-export-components` clean while
// callers still import a single, stable type + helper pair.
export interface CalendarAwareDateValue {
  date: Dayjs;
  calendarType: CalendarType;
  originalDay: number | null;
  originalMonth: number | null;
  originalYear: number | null;
}

// Convenience helper for forms that need to seed the picker from an existing
// record's `{ date, calendar_type, original_* }` payload.
export function buildCalendarAwareValue(
  isoOrDayjs: string | Dayjs | null | undefined,
  calendarType: string | undefined,
  originalDay: number | null | undefined,
  originalMonth: number | null | undefined,
  originalYear: number | null | undefined,
): CalendarAwareDateValue {
  const date = isoOrDayjs ? dayjs(isoOrDayjs) : dayjs();
  const ct = (calendarType || "gregorian") as CalendarType;
  return {
    date,
    calendarType: ct,
    originalDay: ct !== "gregorian" ? originalDay ?? null : null,
    originalMonth: ct !== "gregorian" ? originalMonth ?? null : null,
    originalYear: ct !== "gregorian" ? originalYear ?? null : null,
  };
}
