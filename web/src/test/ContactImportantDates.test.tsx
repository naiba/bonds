import { describe, expect, it } from "vitest";
import type { ImportantDate } from "@/api";
import {
  areImportantDateDraftsValid,
  buildImportantDateChanges,
  buildImportantDateCreates,
  createEmptyImportantDateDraft,
  importantDatesToDrafts,
} from "@/pages/contact/contactImportantDates";

const birthdate: ImportantDate = {
  id: 10,
  contact_id: "contact-1",
  label: "Birthdate",
  date_precision: "full",
  year: 1990,
  month: 6,
  day: 15,
  calendar_type: "gregorian",
  contact_important_date_type_id: 1,
  remind_me: true,
};

const anniversary: ImportantDate = {
  id: 11,
  contact_id: "contact-1",
  label: "Wedding anniversary",
  date_precision: "month_day",
  month: 9,
  day: 6,
  calendar_type: "gregorian",
  contact_important_date_type_id: 3,
  remind_me: false,
};

describe("contact important-date aggregate helpers", () => {
  it("does not emit updates for unchanged dates", () => {
    const drafts = importantDatesToDrafts([birthdate, anniversary]);

    expect(buildImportantDateChanges([birthdate, anniversary], drafts)).toEqual(
      {
        create: [],
        update: [],
        delete: [],
      },
    );
  });

  it("builds create, update, and delete mutations from the edited collection", () => {
    const drafts = importantDatesToDrafts([birthdate, anniversary]);
    const changedBirthdate = {
      ...drafts[0]!,
      calendarDate: {
        ...drafts[0]!.calendarDate!,
        year: 1991,
      },
    };
    const newDate = {
      ...createEmptyImportantDateDraft(),
      label: "Graduation",
      calendarDate: {
        calendarType: "gregorian" as const,
        datePrecision: "year" as const,
        year: 2012,
        month: null,
        day: null,
      },
    };

    const changes = buildImportantDateChanges(
      [birthdate, anniversary],
      [changedBirthdate, newDate],
    );

    expect(changes.create).toEqual([
      expect.objectContaining({
        label: "Graduation",
        date_precision: "year",
        year: 2012,
      }),
    ]);
    expect(changes.update).toEqual([
      {
        id: 10,
        important_date: expect.objectContaining({
          label: "Birthdate",
          year: 1991,
        }),
      },
    ]);
    expect(changes.delete).toEqual([11]);
  });

  it("requires every added row to have a label and a complete precision", () => {
    expect(areImportantDateDraftsValid([])).toBe(true);
    expect(areImportantDateDraftsValid([createEmptyImportantDateDraft()])).toBe(
      false,
    );
    expect(
      areImportantDateDraftsValid([
        {
          ...createEmptyImportantDateDraft(),
          label: "Known year",
          calendarDate: {
            calendarType: "gregorian",
            datePrecision: "year",
            year: 2000,
            month: null,
            day: null,
          },
        },
      ]),
    ).toBe(true);
  });

  it("serializes all valid dates during contact creation", () => {
    const creates = buildImportantDateCreates(
      importantDatesToDrafts([birthdate, anniversary]).map((draft) => ({
        ...draft,
        id: undefined,
      })),
    );

    expect(creates).toHaveLength(2);
    expect(creates[0]).toEqual(
      expect.objectContaining({
        contact_important_date_type_id: 1,
        date_precision: "full",
        year: 1990,
      }),
    );
    expect(creates[1]).toEqual(
      expect.objectContaining({
        contact_important_date_type_id: 3,
        date_precision: "month_day",
      }),
    );
    expect(creates[1]?.year).toBeUndefined();
  });
});
