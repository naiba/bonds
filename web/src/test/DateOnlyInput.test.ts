import { describe, expect, it } from "vitest";
import { dateInputToTimestamp, formatDateOnly, formatShortDateOnly, timestampToDateInput } from "@/utils/dateOnlyInput";
import type { DateFormatVariants } from "@/utils/dateFormat";

const isoDateFormat: DateFormatVariants = {
  full: "YYYY-MM-DD",
  monthYear: "YYYY-MM",
  short: "MM-DD",
  dateTime: "YYYY-MM-DD HH:mm",
  dateTimeFull: "YYYY-MM-DD HH:mm:ss",
};

describe("date-only input helpers", () => {
  it("keeps UTC-midnight API timestamps on the same calendar date", () => {
    expect(timestampToDateInput("2026-01-02T00:00:00Z")).toBe("2026-01-02");
    expect(formatDateOnly("2026-01-02T00:00:00Z", isoDateFormat)).toBe("2026-01-02");
    expect(formatShortDateOnly("2026-01-02T00:00:00Z", isoDateFormat)).toBe("01-02");
  });

  it("round-trips valid date input values through API timestamps", () => {
    const timestamp = dateInputToTimestamp("2026-01-02");
    expect(timestamp).toBe("2026-01-02T00:00:00Z");
    expect(timestampToDateInput(timestamp)).toBe("2026-01-02");
  });

  it("omits invalid date input values", () => {
    expect(dateInputToTimestamp("2026-1-2")).toBeUndefined();
    expect(timestampToDateInput("not-a-date")).toBeUndefined();
  });
});
