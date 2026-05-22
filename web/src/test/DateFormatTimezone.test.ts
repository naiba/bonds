import { describe, it, expect } from "vitest";
import {
  formatDateTime,
  formatDate,
  type DateFormatVariants,
} from "@/utils/dateFormat";

const baseFmt: DateFormatVariants = {
  full: "YYYY-MM-DD",
  monthYear: "YYYY-MM",
  short: "MM-DD",
  dateTime: "YYYY-MM-DD HH:mm",
  dateTimeFull: "YYYY-MM-DD HH:mm:ss",
};

// Guards a real bug: useDateFormat used to ignore the user's saved
// timezone, so a stored UTC timestamp like "2026-03-12T00:00:00Z"
// rendered in the browser's local tz instead of the user's preference.
// A Tokyo user reading what should be "March 12 09:00" saw the browser
// time (UTC or whatever the user's laptop was set to).
describe("formatDateTime respects user timezone", () => {
  it("renders the same UTC instant differently in different zones", () => {
    const instant = "2026-03-12T00:00:00Z";
    const tokyo = formatDateTime(instant, { ...baseFmt, tz: "Asia/Tokyo" });
    const newYork = formatDateTime(instant, { ...baseFmt, tz: "America/New_York" });
    expect(tokyo).toBe("2026-03-12 09:00"); // UTC midnight = 9 AM Tokyo (no DST)
    // 2026-03-12 is past the second Sunday of March, so NYC is on EDT (UTC-4),
    // which puts UTC midnight at 20:00 the previous day rather than 19:00.
    expect(newYork).toBe("2026-03-11 20:00");
  });

  it("falls back gracefully when tz is omitted (legacy callers)", () => {
    const instant = "2026-03-12T00:00:00Z";
    const result = formatDateTime(instant, baseFmt);
    // Result depends on the JS host TZ; just assert the call doesn't throw
    // and produces a recognizable shape.
    expect(result).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/);
  });

  it("falls back gracefully when tz is an invalid IANA string", () => {
    const instant = "2026-03-12T00:00:00Z";
    expect(() =>
      formatDateTime(instant, { ...baseFmt, tz: "Mars/Olympus_Mons" }),
    ).not.toThrow();
  });

  it("applies tz to date-only format too", () => {
    // 00:30 UTC on Mar 12 is still Mar 11 in Honolulu (UTC-10).
    const instant = "2026-03-12T00:30:00Z";
    const honolulu = formatDate(instant, { ...baseFmt, tz: "Pacific/Honolulu" });
    expect(honolulu).toBe("2026-03-11");
  });
});
