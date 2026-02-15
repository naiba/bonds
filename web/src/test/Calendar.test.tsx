import { describe, it, expect } from "vitest";
import {
  getCalendarSystem,
  supportedCalendarTypes,
} from "@/utils/calendar";
import type { CalendarType, CalendarDate } from "@/utils/calendar";

describe("calendar utils", () => {
  it("has at least gregorian and lunar types", () => {
    expect(supportedCalendarTypes).toContain("gregorian");
    expect(supportedCalendarTypes).toContain("lunar");
    expect(supportedCalendarTypes.length).toBeGreaterThanOrEqual(2);
  });

  describe("getCalendarSystem", () => {
    it("returns a system for gregorian", () => {
      const sys = getCalendarSystem("gregorian");
      expect(sys.type).toBe("gregorian");
      expect(sys.labelKey).toBe("calendar.gregorian");
    });

    it("returns a system for lunar", () => {
      const sys = getCalendarSystem("lunar");
      expect(sys.type).toBe("lunar");
      expect(sys.labelKey).toBe("calendar.lunar");
    });

    it("falls back to gregorian for unknown type", () => {
      const sys = getCalendarSystem("unknown" as CalendarType);
      expect(sys.type).toBe("gregorian");
    });
  });

  describe("gregorian system", () => {
    const sys = getCalendarSystem("gregorian");

    it("toGregorian is identity", () => {
      const date: CalendarDate = { day: 15, month: 6, year: 2025 };
      expect(sys.toGregorian(date)).toEqual(date);
    });

    it("fromGregorian is identity", () => {
      const date: CalendarDate = { day: 15, month: 6, year: 2025 };
      expect(sys.fromGregorian(date)).toEqual(date);
    });

    it("formatDate returns YYYY-MM-DD", () => {
      expect(sys.formatDate({ day: 5, month: 3, year: 2025 })).toBe("2025-03-05");
    });

    it("getMonths returns 12 months", () => {
      const months = sys.getMonths(2025);
      expect(months).toHaveLength(12);
      expect(months[0].value).toBe(1);
      expect(months[11].value).toBe(12);
    });

    it("getDaysInMonth returns correct days", () => {
      expect(sys.getDaysInMonth(2025, 2)).toBe(28);
      expect(sys.getDaysInMonth(2024, 2)).toBe(29);
      expect(sys.getDaysInMonth(2025, 1)).toBe(31);
    });

    it("getYearRange returns reasonable range", () => {
      const [min, max] = sys.getYearRange();
      expect(min).toBeLessThanOrEqual(1900);
      expect(max).toBeGreaterThanOrEqual(2100);
    });
  });

  describe("lunar system", () => {
    const sys = getCalendarSystem("lunar");

    it("converts lunar to gregorian", () => {
      const gd = sys.toGregorian({ day: 1, month: 1, year: 2025 });
      expect(gd.year).toBe(2025);
      expect(gd.month).toBeGreaterThanOrEqual(1);
      expect(gd.month).toBeLessThanOrEqual(2);
    });

    it("converts gregorian to lunar", () => {
      const ld = sys.fromGregorian({ day: 1, month: 2, year: 2025 });
      expect(ld.year).toBe(2025);
      expect(ld.month).toBe(1);
    });

    it("round-trips correctly", () => {
      const original: CalendarDate = { day: 15, month: 1, year: 2025 };
      const gd = sys.toGregorian(original);
      const back = sys.fromGregorian(gd);
      expect(back.day).toBe(original.day);
      expect(back.month).toBe(original.month);
      expect(back.year).toBe(original.year);
    });

    it("formatDate returns Chinese characters", () => {
      const formatted = sys.formatDate({ day: 15, month: 1, year: 2025 });
      expect(formatted).toContain("月");
    });

    it("getMonths returns at least 12 months", () => {
      const months = sys.getMonths(2025);
      expect(months.length).toBeGreaterThanOrEqual(12);
    });

    it("getMonths includes leap month when applicable", () => {
      const months2025 = sys.getMonths(2025);
      const hasLeap = months2025.some((m) => m.value < 0);
      if (hasLeap) {
        expect(months2025.length).toBeGreaterThan(12);
      }
    });

    it("getDaysInMonth returns 29 or 30", () => {
      const days = sys.getDaysInMonth(2025, 1);
      expect(days).toBeGreaterThanOrEqual(29);
      expect(days).toBeLessThanOrEqual(30);
    });
  });
});
