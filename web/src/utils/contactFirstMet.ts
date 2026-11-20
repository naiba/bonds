import type { CreateContactRequest, Contact, UpdateContactRequest } from "@/api";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";
import type { DateFormatVariants } from "@/utils/dateFormat";
import dayjs from "dayjs";

type ContactFirstMetPrecision = "full" | "month" | "year";

export type ContactFirstMetFormValue = CalendarDatePickerValue;

type ContactRequestWithFirstMet = CreateContactRequest | UpdateContactRequest;

function inferContactFirstMetPrecision(contact: Pick<Contact, "first_met_at" | "first_met_date_precision" | "first_met_year" | "first_met_month" | "first_met_day">): ContactFirstMetPrecision | null {
  if (contact.first_met_date_precision === "full" || contact.first_met_date_precision === "month" || contact.first_met_date_precision === "year") {
    return contact.first_met_date_precision;
  }
  if (contact.first_met_at) {
    return "full";
  }
  if (contact.first_met_year != null && contact.first_met_month != null) {
    return "month";
  }
  if (contact.first_met_year != null) {
    return "year";
  }
  return null;
}

export function buildContactFirstMetRequest(calendarDate: ContactFirstMetFormValue | undefined): Pick<ContactRequestWithFirstMet, "first_met_at" | "first_met_date_precision" | "first_met_year" | "first_met_month" | "first_met_day"> {
  if (!calendarDate) {
    return {};
  }

  const precision = calendarDate.datePrecision ?? "full";
  if (precision === "year") {
    return {
      first_met_date_precision: "year",
      first_met_year: calendarDate.year ?? undefined,
    };
  }
  if (precision === "month") {
    return {
      first_met_date_precision: "month",
      first_met_year: calendarDate.year ?? undefined,
      first_met_month: calendarDate.month ?? undefined,
    };
  }

  if (calendarDate.year == null || calendarDate.month == null || calendarDate.day == null) {
    return {};
  }

  const fullDate = `${String(calendarDate.year).padStart(4, "0")}-${String(calendarDate.month).padStart(2, "0")}-${String(calendarDate.day).padStart(2, "0")}T00:00:00Z`;
  return {
    first_met_at: fullDate,
  };
}

export function contactFirstMetToCalendarDate(contact: Pick<Contact, "first_met_at" | "first_met_date_precision" | "first_met_year" | "first_met_month" | "first_met_day">): ContactFirstMetFormValue | undefined {
  const precision = inferContactFirstMetPrecision(contact);
  if (!precision) {
    return undefined;
  }

  if (precision === "year") {
    return {
      calendarType: "gregorian",
      year: contact.first_met_year ?? null,
      month: null,
      day: null,
      datePrecision: "year",
    };
  }

  if (precision === "month") {
    return {
      calendarType: "gregorian",
      year: contact.first_met_year ?? null,
      month: contact.first_met_month ?? null,
      day: null,
      datePrecision: "month",
    };
  }

  const timestamp = contact.first_met_at;
  if (!timestamp) {
    return undefined;
  }
  // 这是一个“日历日期”，后端存成当天的 UTC 零点。只能取字符串里的年月日，
  // 不能当成时刻交给 dayjs 按本地时区解析：在 UTC 以西的时区会读成前一天，
  // 而编辑表单会把读到的值原样存回去，于是每保存一次就再往前挪一天。
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(timestamp);
  if (!match) {
    return undefined;
  }

  return {
    calendarType: "gregorian",
    year: Number(match[1]),
    month: Number(match[2]),
    day: Number(match[3]),
    datePrecision: "full",
  };
}

export function formatContactFirstMetDisplay(contact: Pick<Contact, "first_met_at" | "first_met_date_precision" | "first_met_year" | "first_met_month" | "first_met_day">, formats: DateFormatVariants): string {
  const precision = inferContactFirstMetPrecision(contact);
  if (!precision) {
    return "";
  }

  if (precision === "year" && contact.first_met_year != null) {
    return String(contact.first_met_year);
  }

  if (precision === "month" && contact.first_met_year != null && contact.first_met_month != null) {
    return dayjs(new Date(contact.first_met_year, contact.first_met_month - 1, 1)).format(formats.monthYear);
  }

  if (precision === "full" && contact.first_met_at) {
    const dateInput = contact.first_met_at.slice(0, 10);
    return dayjs(dateInput).format(formats.full);
  }

  return "";
}

export function hasContactFirstMetValue(contact: Pick<Contact, "first_met_at" | "first_met_date_precision" | "first_met_year" | "first_met_month" | "first_met_day">): boolean {
  return inferContactFirstMetPrecision(contact) !== null;
}
