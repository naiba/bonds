import type {
  CreateImportantDateRequest,
  ImportantDate,
  UpdateContactRequest,
} from "@/api";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";
import { buildImportantDateRequest } from "@/utils/importantDatePrecision";
import { buildImportantDatePickerValue } from "./modules/importantDatesModuleHelpers";

export type ContactImportantDateDraft = {
  readonly key: string;
  readonly id?: number;
  readonly contact_important_date_type_id?: number;
  readonly label: string;
  readonly calendarDate?: CalendarDatePickerValue;
  readonly remind_me: boolean;
};

type ImportantDateChanges = NonNullable<
  UpdateContactRequest["important_date_changes"]
>;

let nextDraftNumber = 1;

export function createEmptyImportantDateDraft(): ContactImportantDateDraft {
  const key = `new-important-date-${nextDraftNumber}`;
  nextDraftNumber += 1;
  return {
    key,
    label: "",
    remind_me: false,
  };
}

export function importantDatesToDrafts(
  dates: readonly ImportantDate[] | undefined,
): ContactImportantDateDraft[] {
  return (dates ?? []).map((date, index) => ({
    key: `important-date-${date.id ?? index}`,
    id: date.id,
    contact_important_date_type_id: date.contact_important_date_type_id,
    label: date.label ?? "",
    calendarDate: buildImportantDatePickerValue(date),
    remind_me: date.remind_me ?? false,
  }));
}

function hasValidCalendarDate(
  value: CalendarDatePickerValue | undefined,
): value is CalendarDatePickerValue {
  if (!value) return false;
  switch (value.datePrecision ?? "full") {
    case "full":
      return value.year != null && value.month != null && value.day != null;
    case "month":
      return value.year != null && value.month != null;
    case "year":
      return value.year != null;
    case "month_day":
      return value.month != null && value.day != null;
  }
}

export function areImportantDateDraftsValid(
  drafts: readonly ContactImportantDateDraft[] | undefined,
): boolean {
  return (drafts ?? []).every(
    (draft) =>
      draft.label.trim() !== "" && hasValidCalendarDate(draft.calendarDate),
  );
}

function draftToRequest(
  draft: ContactImportantDateDraft,
): CreateImportantDateRequest {
  if (!draft.calendarDate) {
    throw new Error("important date draft is missing its date");
  }
  return buildImportantDateRequest(
    {
      label: draft.label.trim(),
      calendarDate: draft.calendarDate,
      contact_important_date_type_id: draft.contact_important_date_type_id,
      remind_me: draft.remind_me,
    },
    draft.label.trim(),
  );
}

export function buildImportantDateCreates(
  drafts: readonly ContactImportantDateDraft[] | undefined,
): CreateImportantDateRequest[] {
  return (drafts ?? []).map(draftToRequest);
}

function sameOptionalNumber(left?: number, right?: number): boolean {
  return left === right || (left == null && right == null);
}

function importantDateMatchesRequest(
  date: ImportantDate,
  request: CreateImportantDateRequest,
): boolean {
  return (
    (date.label ?? "") === (request.label ?? "") &&
    (date.date_precision ?? "") === (request.date_precision ?? "") &&
    sameOptionalNumber(date.day, request.day) &&
    sameOptionalNumber(date.month, request.month) &&
    sameOptionalNumber(date.year, request.year) &&
    (date.calendar_type || "gregorian") ===
      (request.calendar_type || "gregorian") &&
    sameOptionalNumber(date.original_day, request.original_day) &&
    sameOptionalNumber(date.original_month, request.original_month) &&
    sameOptionalNumber(date.original_year, request.original_year) &&
    sameOptionalNumber(
      date.contact_important_date_type_id,
      request.contact_important_date_type_id,
    ) &&
    (date.remind_me ?? false) === (request.remind_me ?? false)
  );
}

export function buildImportantDateChanges(
  originalDates: readonly ImportantDate[] | undefined,
  drafts: readonly ContactImportantDateDraft[] | undefined,
): ImportantDateChanges {
  const originals = originalDates ?? [];
  const currentDrafts = drafts ?? [];
  const originalByID = new Map(
    originals.flatMap((date) =>
      date.id === undefined ? [] : ([[date.id, date]] as const),
    ),
  );
  const retainedIDs = new Set(
    currentDrafts.flatMap((draft) =>
      draft.id === undefined ? [] : [draft.id],
    ),
  );

  const create: CreateImportantDateRequest[] = [];
  const update: ImportantDateChanges["update"] = [];
  for (const draft of currentDrafts) {
    const request = draftToRequest(draft);
    if (draft.id === undefined) {
      create.push(request);
      continue;
    }
    const original = originalByID.get(draft.id);
    if (!original || !importantDateMatchesRequest(original, request)) {
      update.push({ id: draft.id, important_date: request });
    }
  }

  return {
    create,
    update,
    delete: originals.flatMap((date) =>
      date.id !== undefined && !retainedIDs.has(date.id) ? [date.id] : [],
    ),
  };
}
