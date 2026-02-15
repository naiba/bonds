import { Solar, Lunar, LunarYear, LunarMonth } from "lunar-javascript";

export type CalendarType = "gregorian" | "lunar";

export interface CalendarDate {
  day: number;
  month: number;
  year: number;
}

export interface MonthOption {
  value: number;
  label: string;
}

export interface CalendarSystem {
  type: CalendarType;
  labelKey: string;
  toGregorian: (date: CalendarDate) => CalendarDate;
  fromGregorian: (date: CalendarDate) => CalendarDate;
  formatDate: (date: CalendarDate) => string;
  getMonths: (year: number) => MonthOption[];
  getDaysInMonth: (year: number, month: number) => number;
  getYearRange: () => [number, number];
}

const LUNAR_MONTH_NAMES = [
  "\u6b63", "\u4e8c", "\u4e09", "\u56db", "\u4e94", "\u516d",
  "\u4e03", "\u516b", "\u4e5d", "\u5341", "\u5341\u4e00", "\u814a",
];

const gregorianSystem: CalendarSystem = {
  type: "gregorian",
  labelKey: "calendar.gregorian",
  toGregorian: (date) => date,
  fromGregorian: (date) => date,
  formatDate: (date) => `${date.year}-${String(date.month).padStart(2, "0")}-${String(date.day).padStart(2, "0")}`,
  getMonths: () =>
    Array.from({ length: 12 }, (_, i) => ({
      value: i + 1,
      label: String(i + 1),
    })),
  getDaysInMonth: (year, month) => new Date(year, month, 0).getDate(),
  getYearRange: () => [1900, 2100],
};

const lunarSystem: CalendarSystem = {
  type: "lunar",
  labelKey: "calendar.lunar",

  toGregorian: (date) => {
    const lunar = Lunar.fromYmd(date.year, date.month, date.day);
    const solar = lunar.getSolar();
    return { day: solar.getDay(), month: solar.getMonth(), year: solar.getYear() };
  },

  fromGregorian: (date) => {
    const solar = Solar.fromYmd(date.year, date.month, date.day);
    const lunar = solar.getLunar();
    return { day: lunar.getDay(), month: lunar.getMonth(), year: lunar.getYear() };
  },

  formatDate: (date) => {
    const absMonth = date.month < 0 ? -date.month : date.month;
    const isLeap = date.month < 0;
    const lunar = Lunar.fromYmd(date.year, date.month, date.day);
    const prefix = isLeap ? "\u95f0" : "";
    const monthName = LUNAR_MONTH_NAMES[absMonth - 1] ?? String(absMonth);
    return `${prefix}${monthName}\u6708${lunar.getDayInChinese()}`;
  },

  getMonths: (year) => {
    const months: MonthOption[] = [];
    for (let m = 1; m <= 12; m++) {
      months.push({ value: m, label: `${LUNAR_MONTH_NAMES[m - 1]}\u6708` });
    }
    const leapMonth = LunarYear.fromYear(year).getLeapMonth();
    if (leapMonth > 0) {
      const idx = months.findIndex((mo) => mo.value === leapMonth);
      if (idx >= 0) {
        months.splice(idx + 1, 0, {
          value: -leapMonth,
          label: `\u95f0${LUNAR_MONTH_NAMES[leapMonth - 1]}\u6708`,
        });
      }
    }
    return months;
  },

  getDaysInMonth: (year, month) => {
    const lm = LunarMonth.fromYm(year, month);
    return lm ? lm.getDayCount() : 30;
  },

  getYearRange: () => [1900, 2100],
};

const systems: Record<CalendarType, CalendarSystem> = {
  gregorian: gregorianSystem,
  lunar: lunarSystem,
};

export const supportedCalendarTypes: CalendarType[] = Object.keys(systems) as CalendarType[];

export function getCalendarSystem(type: CalendarType): CalendarSystem {
  return systems[type] ?? gregorianSystem;
}
