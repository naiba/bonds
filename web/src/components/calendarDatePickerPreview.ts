import { getCalendarSystem } from "@/utils/calendar";
import type { CalendarType } from "@/utils/calendar";
import type { ImportantDatePrecision } from "./calendarDatePickerValue";

type CalendarDatePickerPreviewArgs = {
  readonly calendarType: CalendarType;
  readonly datePrecision: ImportantDatePrecision;
  readonly selectedDay: number;
  readonly selectedMonth: number;
  readonly selectedYear: number;
  readonly gregorianLabel: string;
  readonly lunarLabel: string;
};

export function formatCalendarDatePickerPreview({
  calendarType,
  datePrecision,
  selectedDay,
  selectedMonth,
  selectedYear,
  gregorianLabel,
  lunarLabel,
}: CalendarDatePickerPreviewArgs): string {
  if (datePrecision !== "full") {
    return "";
  }

  if (calendarType === "gregorian") {
    const lunarSystem = getCalendarSystem("lunar");
    const lunarDate = lunarSystem.fromGregorian({
      day: selectedDay,
      month: selectedMonth,
      year: selectedYear,
    });
    return `${lunarLabel}: ${lunarSystem.formatDate(lunarDate)}`;
  }

  const calendarSystem = getCalendarSystem(calendarType);
  const gregorianDate = calendarSystem.toGregorian({
    day: selectedDay,
    month: selectedMonth,
    year: selectedYear,
  });
  return `${gregorianLabel}: ${gregorianDate.year}-${String(gregorianDate.month).padStart(2, "0")}-${String(gregorianDate.day).padStart(2, "0")}`;
}
