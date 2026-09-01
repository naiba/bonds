import { DatePicker, Segmented, Typography } from "antd";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import type { Dayjs } from "dayjs";
import { getCalendarSystem, supportedCalendarTypes } from "@/utils/calendar";
import CalendarDatePickerControls from "./CalendarDatePickerControls";
import { createCalendarDatePickerHandlers } from "./calendarDatePickerHandlers";
import { formatCalendarDatePickerPreview } from "./calendarDatePickerPreview";
import { inferPrecisionFromValue } from "./calendarDatePickerValue";
import type {
  CalendarDatePickerValue,
  ImportantDatePrecision,
} from "./calendarDatePickerValue";

export type {
  CalendarDatePickerValue,
  ImportantDatePrecision,
} from "./calendarDatePickerValue";

const { Text } = Typography;

const NO_YEAR_VALUE = -1;
interface CalendarDatePickerProps {
  value?: CalendarDatePickerValue | null;
  onChange?: (value: CalendarDatePickerValue | null) => void;
  enableAlternativeCalendar?: boolean;
  enableNoYear?: boolean;
  enableDatePrecision?: boolean;
  allowedDatePrecisions?: readonly ImportantDatePrecision[];
  allowClear?: boolean;
  showToday?: boolean;
  maxDate?: Dayjs;
}

function buildDayOptions(
  totalDays: number,
): Array<{ value: number; label: string }> {
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
      label: dayjs()
        .month(month - 1)
        .format("MMMM"),
    });
  }
  return options;
}

function optionsThroughValue(
  options: Array<{ value: number; label: string }>,
  maximumValue: number,
) {
  const maximumIndex = options.findIndex(
    (option) => option.value === maximumValue,
  );
  return maximumIndex < 0 ? options : options.slice(0, maximumIndex + 1);
}

function clampValueToMaximum(
  value: CalendarDatePickerValue | null,
  maximum: CalendarDatePickerValue | null,
): CalendarDatePickerValue | null {
  if (
    value == null ||
    maximum == null ||
    value.year == null ||
    value.datePrecision === "month_day"
  ) {
    return value;
  }
  const maximumYear = maximum.year!;
  const maximumMonth = maximum.month!;
  const maximumDay = maximum.day!;
  let afterMaximum = value.year > maximumYear;
  if (!afterMaximum && value.year === maximumYear && value.month != null) {
    const months = getCalendarSystem(value.calendarType).getMonths(value.year);
    const valueMonthIndex = months.findIndex(
      (month) => month.value === value.month,
    );
    const maximumMonthIndex = months.findIndex(
      (month) => month.value === maximumMonth,
    );
    afterMaximum = valueMonthIndex > maximumMonthIndex;
    if (
      !afterMaximum &&
      valueMonthIndex === maximumMonthIndex &&
      value.day != null
    ) {
      afterMaximum = value.day > maximumDay;
    }
  }
  if (!afterMaximum) {
    return value;
  }
  if (value.datePrecision === "year") {
    return { ...value, year: maximumYear, month: null, day: null };
  }
  if (value.datePrecision === "month") {
    return { ...value, year: maximumYear, month: maximumMonth, day: null };
  }
  return {
    ...value,
    year: maximumYear,
    month: maximumMonth,
    day: maximumDay,
    datePrecision: "full",
  };
}

export default function CalendarDatePicker({
  value,
  onChange,
  enableAlternativeCalendar = false,
  enableNoYear = false,
  enableDatePrecision = false,
  allowedDatePrecisions = ["full", "month", "year", "month_day"],
  allowClear,
  showToday = false,
  maxDate,
}: CalendarDatePickerProps) {
  const { t } = useTranslation();
  const now = dayjs();
  const effectiveDefaultDate = maxDate?.isBefore(now, "day") ? maxDate : now;

  const calendarType = value?.calendarType ?? "gregorian";
  const inferredPrecision = inferPrecisionFromValue(value ?? undefined);
  const datePrecision = allowedDatePrecisions.includes(inferredPrecision)
    ? inferredPrecision
    : (allowedDatePrecisions[0] ?? "full");
  const usesPrecisionLayout = enableDatePrecision;
  const selectedYear = value?.year ?? effectiveDefaultDate.year();
  const selectedMonth = value?.month ?? effectiveDefaultDate.month() + 1;
  const selectedDay = value?.day ?? effectiveDefaultDate.date();
  const displayYear = datePrecision === "month_day" ? null : selectedYear;
  const controlYear =
    value == null
      ? undefined
      : datePrecision === "month_day"
        ? null
        : (value.year ?? undefined);
  const controlMonth = value?.month ?? undefined;
  const controlDay = value?.day ?? undefined;
  const hasCompleteSelection =
    value?.day != null && value.month != null && value.year != null;

  const calendarSystem = getCalendarSystem(calendarType);
  const maximumCalendarDate = maxDate
    ? calendarSystem.fromGregorian({
        year: maxDate.year(),
        month: maxDate.month() + 1,
        day: maxDate.date(),
      })
    : null;
  const allSelectableMonths = calendarSystem.getMonths(selectedYear);
  const selectableMonths =
    maximumCalendarDate?.year === selectedYear
      ? optionsThroughValue(allSelectableMonths, maximumCalendarDate.month)
      : allSelectableMonths;
  const allSelectableDays = buildDayOptions(
    calendarSystem.getDaysInMonth(selectedYear, selectedMonth),
  );
  const selectableDays =
    maximumCalendarDate?.year === selectedYear &&
    maximumCalendarDate.month === selectedMonth
      ? optionsThroughValue(allSelectableDays, maximumCalendarDate.day)
      : allSelectableDays;
  const [minYear, maxYear] = calendarSystem.getYearRange();

  const yearOptions = (() => {
    const options: Array<{ value: number; label: string }> = [];
    if (enableNoYear) {
      options.push({ value: NO_YEAR_VALUE, label: t("calendar.no_year") });
    }
    const lastYear = Math.min(maxYear, maximumCalendarDate?.year ?? maxYear);
    for (let year = minYear; year <= lastYear; year += 1) {
      options.push({ value: year, label: String(year) });
    }
    return options;
  })();

  const allGregorianMonthOptions = buildGregorianMonthOptions();
  const gregorianMonthOptions =
    maximumCalendarDate?.year === selectedYear
      ? optionsThroughValue(allGregorianMonthOptions, maximumCalendarDate.month)
      : allGregorianMonthOptions;
  const gregorianDayOptions = (() => {
    const referenceYear = datePrecision === "month_day" ? 2000 : selectedYear;
    const options = buildDayOptions(
      dayjs(
        `${referenceYear}-${String(selectedMonth).padStart(2, "0")}-01`,
      ).daysInMonth(),
    );
    return maximumCalendarDate?.year === selectedYear &&
      maximumCalendarDate.month === selectedMonth
      ? optionsThroughValue(options, maximumCalendarDate.day)
      : options;
  })();

  const constrainedOnChange = (nextValue: CalendarDatePickerValue | null) =>
    onChange?.(
      clampValueToMaximum(
        nextValue,
        maximumCalendarDate
          ? { calendarType, ...maximumCalendarDate, datePrecision: "full" }
          : null,
      ),
    );

  const handlers = createCalendarDatePickerHandlers({
    calendarType,
    datePrecision,
    displayYear,
    selectedYear,
    selectedMonth,
    selectedDay,
    noYearValue: NO_YEAR_VALUE,
    onChange: constrainedOnChange,
  });

  const previewText = hasCompleteSelection
    ? formatCalendarDatePickerPreview({
        calendarType,
        datePrecision,
        selectedDay,
        selectedMonth,
        selectedYear,
        gregorianLabel: t("calendar.gregorian"),
        lunarLabel: t("calendar.lunar"),
      })
    : "";

  const calendarTypeOptions = supportedCalendarTypes.map((type) => ({
    value: type,
    label: t(getCalendarSystem(type).labelKey),
  }));

  const fieldControls = (
    <CalendarDatePickerControls
      showPrecisionSelector={enableDatePrecision}
      availablePrecisions={allowedDatePrecisions}
      usesPrecisionLayout={usesPrecisionLayout}
      datePrecision={datePrecision}
      displayYear={controlYear}
      selectedMonth={controlMonth}
      selectedDay={controlDay}
      yearOptions={yearOptions}
      monthOptions={
        calendarType === "gregorian" ? gregorianMonthOptions : selectableMonths
      }
      dayOptions={
        calendarType === "gregorian" ? gregorianDayOptions : selectableDays
      }
      noYearValue={NO_YEAR_VALUE}
      yearPlaceholder={t("calendar.year")}
      monthPlaceholder={t("calendar.month")}
      dayPlaceholder={t("calendar.day")}
      showToday={showToday}
      todayLabel={t("calendar.today")}
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
      onToday={() => {
        const today = calendarSystem.fromGregorian({
          year: now.year(),
          month: now.month() + 1,
          day: now.date(),
        });
        handlers.handleToday(today.year, today.month, today.day);
      }}
    />
  );

  const handleDatePickerChange = (nextDate: Dayjs | null) => {
    if (!nextDate) {
      handlers.handleClear();
      return;
    }
    handlers.handleGregorianChange(
      nextDate.year(),
      nextDate.month() + 1,
      nextDate.date(),
    );
  };

  const pickerValue = hasCompleteSelection
    ? dayjs(
        `${selectedYear}-${String(selectedMonth).padStart(2, "0")}-${String(selectedDay).padStart(2, "0")}`,
      )
    : null;

  if (!enableAlternativeCalendar) {
    if (enableNoYear || enableDatePrecision) {
      return fieldControls;
    }

    return (
      <DatePicker
        value={pickerValue}
        onChange={handleDatePickerChange}
        allowClear={allowClear}
        showNow={showToday}
        maxDate={maxDate}
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
          value={pickerValue}
          onChange={handleDatePickerChange}
          allowClear={allowClear}
          showNow={showToday}
          maxDate={maxDate}
          style={{ width: "100%" }}
        />
      ) : (
        fieldControls
      )}

      {previewText && (
        <Text
          type="secondary"
          style={{ fontSize: 12, marginTop: 4, display: "block" }}
        >
          {previewText}
        </Text>
      )}
    </div>
  );
}
