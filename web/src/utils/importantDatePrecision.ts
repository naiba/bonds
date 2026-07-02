import type { CreateImportantDateRequest, ImportantDate } from "@/api";
import type { CalendarDatePickerValue, ImportantDatePrecision } from "@/components/CalendarDatePicker";
import { getCalendarSystem } from "@/utils/calendar";

export type ImportantDateFormValues = {
  label: string;
  calendarDate: CalendarDatePickerValue;
  contact_important_date_type_id?: number;
  remind_me?: boolean;
};

export function canScheduleImportantDateReminder(
  calendarDate: CalendarDatePickerValue | undefined,
): boolean {
  if (!calendarDate) {
    return false;
  }

  return calendarDate.day != null && calendarDate.month != null;
}

export function inferImportantDatePrecision(date: ImportantDate): ImportantDatePrecision {
  switch (date.date_precision) {
    case "full":
    case "month":
    case "year":
    case "month_day":
      return date.date_precision;
    default:
      break;
  }
  if (date.year != null && date.month != null && date.day != null) return "full";
  if (date.year != null && date.month != null) return "month";
  if (date.year != null) return "year";
  return "month_day";
}

export function buildImportantDateRequest(values: ImportantDateFormValues, fallbackLabel: string): CreateImportantDateRequest {
  const calendarDate = values.calendarDate;
  const precision = calendarDate.datePrecision ?? "full";
  const data: CreateImportantDateRequest = {
    label: values.label || fallbackLabel,
    date_precision: precision,
    calendar_type: "gregorian",
    contact_important_date_type_id: values.contact_important_date_type_id,
    remind_me: canScheduleImportantDateReminder(calendarDate)
      ? values.remind_me ?? false
      : false,
  };

  if (precision === "year") {
    data.year = calendarDate.year ?? undefined;
    return data;
  }
  if (precision === "month") {
    data.year = calendarDate.year ?? undefined;
    data.month = calendarDate.month ?? undefined;
    return data;
  }
  if (precision === "month_day") {
    data.month = calendarDate.month ?? undefined;
    data.day = calendarDate.day ?? undefined;
    return data;
  }
  if (precision !== "full") {
    const exhaustive: never = precision;
    return exhaustive;
  }

  if (calendarDate.year == null || calendarDate.month == null || calendarDate.day == null) return data;

  const sys = getCalendarSystem(calendarDate.calendarType);
  const gd = sys.toGregorian({ day: calendarDate.day, month: calendarDate.month, year: calendarDate.year });
  data.calendar_type = calendarDate.calendarType;
  data.year = gd.year;
  data.month = gd.month;
  data.day = gd.day;

  if (calendarDate.calendarType !== "gregorian") {
    data.original_day = calendarDate.day;
    data.original_month = calendarDate.month;
    data.original_year = calendarDate.year;
  }

  return data;
}
