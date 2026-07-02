import { getCalendarSystem } from "@/utils/calendar";
import type { CalendarType } from "@/utils/calendar";
import type {
  CalendarDatePickerValue,
  ImportantDatePrecision,
} from "./calendarDatePickerValue";

const MONTH_DAY_REFERENCE_YEAR = 2000;

type EmitCalendarDatePickerValueArgs = {
  readonly nextCalendarType: CalendarType;
  readonly nextYear: number | null;
  readonly nextMonth: number | null;
  readonly nextDay: number | null;
  readonly precision: ImportantDatePrecision;
  readonly selectedYear: number;
  readonly selectedMonth: number;
  readonly selectedDay: number;
  readonly onChange?: (value: CalendarDatePickerValue) => void;
};

export function emitCalendarDatePickerValue({
  nextCalendarType,
  nextYear,
  nextMonth,
  nextDay,
  precision,
  selectedYear,
  selectedMonth,
  selectedDay,
  onChange,
}: EmitCalendarDatePickerValueArgs) {
  if (precision === "year") {
    onChange?.({
      calendarType: nextCalendarType,
      year: nextYear ?? selectedYear,
      month: null,
      day: null,
      datePrecision: precision,
    });
    return;
  }

  if (precision === "month") {
    onChange?.({
      calendarType: nextCalendarType,
      year: nextYear ?? selectedYear,
      month: nextMonth ?? selectedMonth,
      day: null,
      datePrecision: precision,
    });
    return;
  }

  if (precision === "month_day") {
    const safeMonth = nextMonth ?? selectedMonth;
    const referenceYear = nextCalendarType === "gregorian"
      ? MONTH_DAY_REFERENCE_YEAR
      : selectedYear;
    const maxDay = getCalendarSystem(nextCalendarType).getDaysInMonth(
      referenceYear,
      safeMonth,
    );
    onChange?.({
      calendarType: nextCalendarType,
      year: null,
      month: safeMonth,
      day: Math.min(nextDay ?? selectedDay, maxDay),
      datePrecision: precision,
    });
    return;
  }

  const safeYear = nextYear ?? selectedYear;
  const safeMonth = nextMonth ?? selectedMonth;
  const maxDay = getCalendarSystem(nextCalendarType).getDaysInMonth(
    safeYear,
    safeMonth,
  );
  onChange?.({
    calendarType: nextCalendarType,
    year: safeYear,
    month: safeMonth,
    day: Math.min(nextDay ?? selectedDay, maxDay),
    datePrecision: "full",
  });
}
