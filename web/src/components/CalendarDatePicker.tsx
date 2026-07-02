import { DatePicker, Segmented, Typography } from "antd";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import type { Dayjs } from "dayjs";
import { getCalendarSystem, supportedCalendarTypes } from "@/utils/calendar";
import CalendarDatePickerControls from "./CalendarDatePickerControls";
import { createCalendarDatePickerHandlers } from "./calendarDatePickerHandlers";
import { formatCalendarDatePickerPreview } from "./calendarDatePickerPreview";
import { inferPrecisionFromValue } from "./calendarDatePickerValue";
import type { CalendarDatePickerValue } from "./calendarDatePickerValue";

export type { CalendarDatePickerValue, ImportantDatePrecision } from "./calendarDatePickerValue";

const { Text } = Typography;

const NO_YEAR_VALUE = -1;
interface CalendarDatePickerProps {
  value?: CalendarDatePickerValue;
  onChange?: (value: CalendarDatePickerValue) => void;
  enableAlternativeCalendar?: boolean;
  enableNoYear?: boolean;
  enableDatePrecision?: boolean;
}

function buildDayOptions(totalDays: number): Array<{ value: number; label: string }> {
  const options = [];
  for (let day = 1; day <= totalDays; day += 1) {
    options.push({ value: day, label: String(day) });
  }
  return options;
}

function buildGregorianMonthOptions(): Array<{ value: number; label: string }> {
  const options = [];
  for (let month = 1; month <= 12; month += 1) {
    options.push({
      value: month,
      label: dayjs().month(month - 1).format("MMMM"),
    });
  }
  return options;
}

export default function CalendarDatePicker({
  value,
  onChange,
  enableAlternativeCalendar = false,
  enableNoYear = false,
  enableDatePrecision = false,
}: CalendarDatePickerProps) {
  const { t } = useTranslation();
  const now = dayjs();

  const calendarType = value?.calendarType ?? "gregorian";
  const datePrecision = inferPrecisionFromValue(value);
  const usesPrecisionLayout = enableDatePrecision;
  const selectedYear = value?.year ?? now.year();
  const selectedMonth = value?.month ?? now.month() + 1;
  const selectedDay = value?.day ?? now.date();
  const displayYear = datePrecision === "month_day" ? null : selectedYear;

  const calendarSystem = getCalendarSystem(calendarType);
  const selectableMonths = calendarSystem.getMonths(selectedYear);
  const selectableDays = buildDayOptions(
    calendarSystem.getDaysInMonth(selectedYear, selectedMonth),
  );
  const [minYear, maxYear] = calendarSystem.getYearRange();

  const yearOptions = (() => {
    const options: Array<{ value: number; label: string }> = [];
    if (enableNoYear) {
      options.push({ value: NO_YEAR_VALUE, label: t("calendar.no_year") });
    }
    for (let year = minYear; year <= maxYear; year += 1) {
      options.push({ value: year, label: String(year) });
    }
    return options;
  })();

  const gregorianMonthOptions = buildGregorianMonthOptions();
  const gregorianDayOptions = (() => {
    const referenceYear = datePrecision === "month_day"
      ? 2000
      : selectedYear;
    return buildDayOptions(
      dayjs(`${referenceYear}-${String(selectedMonth).padStart(2, "0")}-01`).daysInMonth(),
    );
  })();

  const handlers = createCalendarDatePickerHandlers({
    calendarType,
    datePrecision,
    displayYear,
    selectedYear,
    selectedMonth,
    selectedDay,
    noYearValue: NO_YEAR_VALUE,
    onChange,
  });

  const previewText = formatCalendarDatePickerPreview({
    calendarType,
    datePrecision,
    selectedDay,
    selectedMonth,
    selectedYear,
    gregorianLabel: t("calendar.gregorian"),
    lunarLabel: t("calendar.lunar"),
  });

  const calendarTypeOptions = supportedCalendarTypes.map((type) => ({
    value: type,
    label: t(getCalendarSystem(type).labelKey),
  }));

  const fieldControls = (
    <CalendarDatePickerControls
      showPrecisionSelector={enableDatePrecision}
      usesPrecisionLayout={usesPrecisionLayout}
      datePrecision={datePrecision}
      displayYear={displayYear}
      selectedMonth={selectedMonth}
      selectedDay={selectedDay}
      yearOptions={yearOptions}
      monthOptions={calendarType === "gregorian" ? gregorianMonthOptions : selectableMonths}
      dayOptions={calendarType === "gregorian" ? gregorianDayOptions : selectableDays}
      noYearValue={NO_YEAR_VALUE}
      yearPlaceholder={t("calendar.year")}
      monthPlaceholder={t("calendar.month")}
      dayPlaceholder={t("calendar.day")}
      precisionLabels={{
        full: t("calendar.date_precision.full"),
        month: t("calendar.date_precision.month"),
        year: t("calendar.date_precision.year"),
        monthDay: t("calendar.date_precision.month_day"),
      }}
      onPrecisionChange={handlers.handlePrecisionChange}
      onYearChange={handlers.handleYearChange}
      onMonthChange={handlers.handleMonthChange}
      onDayChange={handlers.handleDayChange}
    />
  );

  const handleDatePickerChange = (nextDate: Dayjs | null) => {
    if (!nextDate) {
      return;
    }
    handlers.handleGregorianChange(
      nextDate.year(),
      nextDate.month() + 1,
      nextDate.date(),
    );
  };

  if (!enableAlternativeCalendar) {
    if (enableNoYear || enableDatePrecision) {
      return fieldControls;
    }

    return (
      <DatePicker
        value={dayjs(`${selectedYear}-${String(selectedMonth).padStart(2, "0")}-${String(selectedDay).padStart(2, "0")}`)}
        onChange={handleDatePickerChange}
        style={{ width: "100%" }}
      />
    );
  }

  return (
    <div>
      <Segmented
        options={calendarTypeOptions}
        value={calendarType}
        onChange={handlers.handleTypeChange}
        style={{ marginBottom: 8 }}
        disabled={usesPrecisionLayout && datePrecision !== "full"}
        block
      />

      {calendarType === "gregorian" && !enableNoYear && !enableDatePrecision ? (
        <DatePicker
          value={dayjs(`${selectedYear}-${String(selectedMonth).padStart(2, "0")}-${String(selectedDay).padStart(2, "0")}`)}
          onChange={handleDatePickerChange}
          style={{ width: "100%" }}
        />
      ) : (
        fieldControls
      )}

      {previewText && (
        <Text type="secondary" style={{ fontSize: 12, marginTop: 4, display: "block" }}>
          {previewText}
        </Text>
      )}
    </div>
  );
}
