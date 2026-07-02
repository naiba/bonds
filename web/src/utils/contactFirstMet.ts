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
  const parsed = dayjs(timestamp);
  if (!parsed.isValid()) {
    return undefined;
  }

  return {
    calendarType: "gregorian",
    year: parsed.year(),
    month: parsed.month() + 1,
    day: parsed.date(),
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
