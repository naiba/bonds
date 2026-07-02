import { getCalendarSystem } from "@/utils/calendar";
import type { CalendarType } from "@/utils/calendar";
import { emitCalendarDatePickerValue } from "./calendarDatePickerEmit";
import {
  isCalendarType,
  isImportantDatePrecision,
} from "./calendarDatePickerValue";
import type {
  CalendarDatePickerValue,
  ImportantDatePrecision,
} from "./calendarDatePickerValue";

type CalendarDatePickerHandlersArgs = {
  readonly calendarType: CalendarType;
  readonly datePrecision: ImportantDatePrecision;
  readonly displayYear: number | null;
  readonly selectedYear: number;
  readonly selectedMonth: number;
  readonly selectedDay: number;
  readonly noYearValue: number;
  readonly onChange?: (value: CalendarDatePickerValue) => void;
};

export function createCalendarDatePickerHandlers({
  calendarType,
  datePrecision,
  displayYear,
  selectedYear,
  selectedMonth,
  selectedDay,
  noYearValue,
  onChange,
}: CalendarDatePickerHandlersArgs) {
  function emit(
    nextCalendarType: CalendarType,
    nextYear: number | null,
    nextMonth: number | null,
    nextDay: number | null,
    precision: ImportantDatePrecision = datePrecision,
  ) {
    emitCalendarDatePickerValue({
      nextCalendarType,
      nextYear,
      nextMonth,
      nextDay,
      precision,
      selectedYear,
      selectedMonth,
      selectedDay,
      onChange,
    });
  }

  return {
    handleTypeChange(valueToApply: string | number) {
      if (!isCalendarType(valueToApply) || datePrecision !== "full") {
        return;
      }

      const nextSystem = getCalendarSystem(valueToApply);
      const converted = nextSystem.fromGregorian(
        getCalendarSystem(calendarType).toGregorian({
          day: selectedDay,
          month: selectedMonth,
          year: selectedYear,
        }),
      );
      emit(valueToApply, converted.year, converted.month, converted.day);
    },

    handleGregorianChange(nextYear: number, nextMonth: number, nextDay: number) {
      emit("gregorian", nextYear, nextMonth, nextDay, "full");
    },

    handlePrecisionChange(valueToApply: string | number) {
      if (!isImportantDatePrecision(valueToApply)) {
        return;
      }

      if (valueToApply !== "full" && calendarType !== "gregorian") {
        const gregorianDate = getCalendarSystem(calendarType).toGregorian({
          day: selectedDay,
          month: selectedMonth,
          year: selectedYear,
        });
        emit(
          "gregorian",
          gregorianDate.year,
          gregorianDate.month,
          gregorianDate.day,
          valueToApply,
        );
        return;
      }

      emit(calendarType, displayYear, selectedMonth, selectedDay, valueToApply);
    },

    handleYearChange(nextYear: number) {
      if (nextYear === noYearValue) {
        emit(calendarType, null, selectedMonth, selectedDay, "month_day");
        return;
      }

      const validMonths = getCalendarSystem(calendarType).getMonths(nextYear);
      const validMonth = validMonths.some(
        (monthOption) => monthOption.value === selectedMonth,
      )
        ? selectedMonth
        : validMonths[0]?.value ?? 1;
      emit(calendarType, nextYear, validMonth, selectedDay);
    },

    handleMonthChange(nextMonth: number) {
      emit(calendarType, displayYear, nextMonth, selectedDay);
    },

    handleDayChange(nextDay: number) {
      emit(calendarType, displayYear, selectedMonth, nextDay);
    },
  };
}
