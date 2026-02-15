declare module "lunar-javascript" {
  export class Solar {
    static fromYmd(year: number, month: number, day: number): Solar;
    static fromDate(date: Date): Solar;
    getYear(): number;
    getMonth(): number;
    getDay(): number;
    getLunar(): Lunar;
    toYmd(): string;
  }

  export class Lunar {
    static fromYmd(year: number, month: number, day: number): Lunar;
    static fromDate(date: Date): Lunar;
    getYear(): number;
    getMonth(): number;
    getDay(): number;
    getMonthInChinese(): string;
    getDayInChinese(): string;
    getSolar(): Solar;
  }

  export class LunarYear {
    static fromYear(year: number): LunarYear;
    getYear(): number;
    getLeapMonth(): number;
  }

  export class LunarMonth {
    static fromYm(year: number, month: number): LunarMonth | null;
    getYear(): number;
    getMonth(): number;
    getDayCount(): number;
    isLeap(): boolean;
  }
}
