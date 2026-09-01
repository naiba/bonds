import { Button, Segmented, Select, Space } from "antd";
import type { ImportantDatePrecision } from "./calendarDatePickerValue";

type SelectOption = {
  readonly value: number;
  readonly label: string;
};

interface CalendarDatePickerControlsProps {
  readonly showPrecisionSelector: boolean;
  readonly availablePrecisions: readonly ImportantDatePrecision[];
  readonly usesPrecisionLayout: boolean;
  readonly datePrecision: ImportantDatePrecision;
  readonly displayYear: number | null | undefined;
  readonly selectedMonth: number | undefined;
  readonly selectedDay: number | undefined;
  readonly yearOptions: SelectOption[];
  readonly monthOptions: SelectOption[];
  readonly dayOptions: SelectOption[];
  readonly noYearValue: number;
  readonly yearPlaceholder: string;
  readonly monthPlaceholder: string;
  readonly dayPlaceholder: string;
  readonly showToday: boolean;
  readonly todayLabel: string;
  readonly precisionLabels: {
    readonly full: string;
    readonly month: string;
    readonly year: string;
    readonly monthDay: string;
  };
  readonly onPrecisionChange: (value: string | number) => void;
  readonly onYearChange: (value: number) => void;
  readonly onMonthChange: (value: number) => void;
  readonly onDayChange: (value: number) => void;
  readonly onToday: () => void;
}

export default function CalendarDatePickerControls({
  showPrecisionSelector,
  availablePrecisions,
  usesPrecisionLayout,
  datePrecision,
  displayYear,
  selectedMonth,
  selectedDay,
  yearOptions,
  monthOptions,
  dayOptions,
  noYearValue,
  yearPlaceholder,
  monthPlaceholder,
  dayPlaceholder,
  showToday,
  todayLabel,
  precisionLabels,
  onPrecisionChange,
  onYearChange,
  onMonthChange,
  onDayChange,
  onToday,
}: CalendarDatePickerControlsProps) {
  return (
    <>
      {showPrecisionSelector && (
        <Segmented
          options={availablePrecisions.map((precision) => ({
            value: precision,
            label:
              precision === "full"
                ? precisionLabels.full
                : precision === "month"
                  ? precisionLabels.month
                  : precision === "year"
                    ? precisionLabels.year
                    : precisionLabels.monthDay,
          }))}
          value={datePrecision}
          onChange={onPrecisionChange}
          style={{ marginBottom: 8 }}
          block
        />
      )}

      <Space.Compact style={{ width: "100%" }}>
        {(!usesPrecisionLayout || datePrecision !== "month_day") && (
          <Select
            showSearch
            value={displayYear === null ? noYearValue : displayYear}
            onChange={onYearChange}
            options={yearOptions}
            style={{ flex: 1, minWidth: 0 }}
            placeholder={yearPlaceholder}
          />
        )}
        {(!usesPrecisionLayout || datePrecision !== "year") && (
          <Select
            value={selectedMonth}
            onChange={onMonthChange}
            options={monthOptions}
            style={{ flex: 1, minWidth: 0 }}
            placeholder={monthPlaceholder}
          />
        )}
        {(!usesPrecisionLayout ||
          datePrecision === "full" ||
          datePrecision === "month_day") && (
          <Select
            value={selectedDay}
            onChange={onDayChange}
            options={dayOptions}
            style={{ flex: 1, minWidth: 0 }}
            placeholder={dayPlaceholder}
          />
        )}
      </Space.Compact>
      {showToday && (
        <Button
          type="link"
          size="small"
          onClick={onToday}
          style={{ paddingInline: 0 }}
        >
          {todayLabel}
        </Button>
      )}
    </>
  );
}
