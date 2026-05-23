import { useMemo } from "react";
import { DatePicker } from "antd";
import dayjs from "dayjs";
import CalendarDatePicker from "./CalendarDatePicker";
import type { CalendarDatePickerValue } from "./CalendarDatePicker";
import { getCalendarSystem } from "@/utils/calendar";
import type { CalendarAwareDateValue } from "./calendarAwareDateValue";

interface CalendarAwareDatePickerProps {
  value?: CalendarAwareDateValue | null;
  // When allowClear=true callers receive null on clear. Without allowClear the
  // picker only emits non-null values, so the union keeps the callsite honest
  // about which one it actually handles.
  onChange?: (next: CalendarAwareDateValue | null) => void;
  enableAlternativeCalendar: boolean;
  // showTime is only honored in plain-gregorian mode. The lunar/CalendarDatePicker
  // path is day-precision (no time component) because lunar dates are
  // calendar-system anchors (festivals, anniversaries) not time-of-day events.
  showTime?: boolean;
  format?: string;
  allowClear?: boolean;
  style?: React.CSSProperties;
}

export default function CalendarAwareDatePicker({
  value,
  onChange,
  enableAlternativeCalendar,
  showTime,
  format,
  allowClear,
  style,
}: CalendarAwareDatePickerProps) {
  const pickerValue = useMemo<CalendarDatePickerValue>(() => {
    if (!value) {
      const now = dayjs();
      return { calendarType: "gregorian", day: now.date(), month: now.month() + 1, year: now.year() };
    }
    if (value.calendarType !== "gregorian" && value.originalDay != null && value.originalMonth != null) {
      return {
        calendarType: value.calendarType,
        day: value.originalDay,
        month: value.originalMonth,
        year: value.originalYear ?? value.date.year(),
      };
    }
    return {
      calendarType: "gregorian",
      day: value.date.date(),
      month: value.date.month() + 1,
      year: value.date.year(),
    };
  }, [value]);

  if (!enableAlternativeCalendar) {
    return (
      <DatePicker
        value={value?.date ?? null}
        showTime={showTime ? { format: "HH:mm" } : undefined}
        format={format}
        allowClear={allowClear}
        style={style ?? { width: "100%" }}
        onChange={(d) => {
          if (!d) {
            // Propagate the clear verbatim so Form fields can store `null`
            // (which the API treats as "no due date"). Swallowing the clear
            // and re-emitting a fresh `dayjs()` would silently reset cleared
            // tasks back to "due now" on save.
            onChange?.(null);
            return;
          }
          onChange?.({
            date: d,
            calendarType: "gregorian",
            originalDay: null,
            originalMonth: null,
            originalYear: null,
          });
        }}
      />
    );
  }

  return (
    <CalendarDatePicker
      enableAlternativeCalendar
      value={pickerValue}
      onChange={(next) => {
        // year is non-null on this code path because we never expose enableNoYear
        // here — but be defensive: fall back to current year if it ever is.
        const year = next.year ?? dayjs().year();
        const sys = getCalendarSystem(next.calendarType);
        const gd = sys.toGregorian({ day: next.day, month: next.month, year });
        // Preserve the time-of-day from the previous value so editing the
        // date doesn't silently reset the hour/minute (relevant for tasks
        // sync'd via CalDAV that carry a real DUE time).
        const prev = value?.date ?? dayjs();
        const merged = dayjs()
          .year(gd.year)
          .month(gd.month - 1)
          .date(gd.day)
          .hour(prev.hour())
          .minute(prev.minute())
          .second(prev.second())
          .millisecond(prev.millisecond());
        onChange?.({
          date: merged,
          calendarType: next.calendarType,
          originalDay: next.calendarType === "gregorian" ? null : next.day,
          originalMonth: next.calendarType === "gregorian" ? null : next.month,
          originalYear: next.calendarType === "gregorian" ? null : year,
        });
      }}
    />
  );
}

