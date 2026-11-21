import { describe, expect, it, vi } from "vitest";
import {
  dateInputToTimestamp,
  formatDateOnly,
  formatShortDateOnly,
  isDateOnlyTimestamp,
  parseDateOnly,
} from "@/utils/dateOnlyInput";
import { formatDate, type DateFormatVariants } from "@/utils/dateFormat";

// Node reads TZ when a date is computed, so setting it here is enough.
vi.stubEnv("TZ", "Asia/Shanghai");

// The process runs in Asia/Shanghai (UTC+8) for this file: local midnight is
// 16:00 of the previous day in UTC, which is how calendar dates lost a day.
describe("calendar dates east of UTC", () => {
  const formats: DateFormatVariants = {
    full: "YYYY-MM-DD",
    dateTime: "YYYY-MM-DD HH:mm",
    dateTimeFull: "YYYY-MM-DD HH:mm:ss",
    monthYear: "YYYY-MM",
    short: "MM-DD",
    tz: "UTC",
  };

  it("runs in a timezone east of UTC", () => {
    expect(new Date("2026-11-21T00:00:00").getTimezoneOffset()).toBe(-480);
  });

  it("shows a quick fact's date as the day that was entered", () => {
    expect(formatDateOnly("2026-11-21", formats)).toBe("2026-11-21");
    // What the module did before: local midnight, converted to the UTC day.
    expect(formatDate("2026-11-21T00:00:00", { ...formats, tz: undefined })).toBe("2026-11-20");
  });

  it("shows a reminder's month and day unchanged", () => {
    expect(formatShortDateOnly("2000-11-21", formats)).toBe("11-21");
    expect(formatShortDateOnly("2000-01-01", formats)).toBe("01-01");
  });

  it("saves a picked day as that calendar date and reads it back", () => {
    const saved = dateInputToTimestamp("2026-11-21");
    expect(saved).toBe("2026-11-21T00:00:00Z");
    expect(isDateOnlyTimestamp(saved)).toBe(true);
    expect(formatDateOnly(saved, formats)).toBe("2026-11-21");
    expect(parseDateOnly(saved)?.format("YYYY-MM-DD")).toBe("2026-11-21");
  });

  it("tells earlier rows saved with a time of day from calendar dates", () => {
    expect(isDateOnlyTimestamp("2026-11-20T23:30:00Z")).toBe(false);
    expect(isDateOnlyTimestamp("2026-11-21T08:00:00+08:00")).toBe(true);
    expect(isDateOnlyTimestamp("")).toBe(false);
    expect(isDateOnlyTimestamp(undefined)).toBe(false);
    expect(isDateOnlyTimestamp("not a date")).toBe(false);
  });
});
