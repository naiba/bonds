import { useMemo } from "react";
import { DatePicker, Select, Segmented, Typography, Space } from "antd";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import type { Dayjs } from "dayjs";
import {
  supportedCalendarTypes,
  getCalendarSystem,
} from "@/utils/calendar";
import type { CalendarType } from "@/utils/calendar";

const { Text } = Typography;

const NO_YEAR_VALUE = -1;

export interface CalendarDatePickerValue {
  calendarType: CalendarType;
  day: number;
  month: number;
  year: number | null;
}

interface CalendarDatePickerProps {
  value?: CalendarDatePickerValue;
  onChange?: (value: CalendarDatePickerValue) => void;
  enableAlternativeCalendar?: boolean;
  /** Issue #76: allow dates without a year (e.g. nameday, anniversary with unknown year) */
  enableNoYear?: boolean;
}

export default function CalendarDatePicker({
  value,
  onChange,
  enableAlternativeCalendar = false,
  enableNoYear = false,
}: CalendarDatePickerProps) {
  const { t } = useTranslation();
  const now = dayjs();

  const calendarType = value?.calendarType ?? "gregorian";
  const displayYear = value?.year;
  const effectiveYear = value?.year ?? now.year();
  const month = value?.month ?? (now.month() + 1);
  const day = value?.day ?? now.date();

  const system = getCalendarSystem(calendarType);
  const months = useMemo(() => system.getMonths(effectiveYear), [system, effectiveYear]);
  const daysInMonth = useMemo(
    () => system.getDaysInMonth(effectiveYear, month),
    [system, effectiveYear, month]
  );
  const [minYear, maxYear] = system.getYearRange();

  const yearOptions = useMemo(() => {
    const opts: Array<{ value: number; label: string }> = [];
    if (enableNoYear) {
      opts.push({ value: NO_YEAR_VALUE, label: t("calendar.no_year") });
    }
    for (let y = minYear; y <= maxYear; y++) {
      opts.push({ value: y, label: String(y) });
    }
    return opts;
  }, [minYear, maxYear, enableNoYear, t]);

  const dayOptions = useMemo(() => {
    const opts = [];
    for (let d = 1; d <= daysInMonth; d++) {
      opts.push({ value: d, label: String(d) });
    }
    return opts;
  }, [daysInMonth]);

  function emit(ct: CalendarType, y: number | null, m: number, d: number) {
    if (y === null) {
      onChange?.({ calendarType: ct, year: null, month: m, day: d });
      return;
    }
    const maxD = getCalendarSystem(ct).getDaysInMonth(y, m);
    const safeDay = d > maxD ? maxD : d;
    onChange?.({ calendarType: ct, year: y, month: m, day: safeDay });
  }

  function handleTypeChange(val: string | number) {
    const newType = val as CalendarType;
    if (value?.year === null) {
      emit(newType, null, value.month ?? 1, value.day ?? 1);
      return;
    }
    const newSystem = getCalendarSystem(newType);
    const converted = newSystem.fromGregorian(
      getCalendarSystem(calendarType).toGregorian({ day, month, year: effectiveYear })
    );
    emit(newType, converted.year, converted.month, converted.day);
  }

  function handleGregorianChange(d: Dayjs | null) {
    if (!d) return;
    emit("gregorian", d.year(), d.month() + 1, d.date());
  }

  function handleYearChange(y: number) {
    if (y === NO_YEAR_VALUE) {
      emit(calendarType, null, month, day);
      return;
    }
    const maxM = system.getMonths(y);
    const validMonth = maxM.some((mo) => mo.value === month) ? month : maxM[0]?.value ?? 1;
    emit(calendarType, y, validMonth, day);
  }

  function handleMonthChange(m: number) {
    emit(calendarType, displayYear ?? null, m, day);
  }

  function handleDayChange(d: number) {
    emit(calendarType, displayYear ?? null, month, d);
  }

  const previewText = useMemo(() => {
    if (displayYear === null) {
      return t("calendar.no_year");
    }
    if (calendarType === "gregorian") {
      const lunarSys = getCalendarSystem("lunar");
      const lunar = lunarSys.fromGregorian({ day, month, year: effectiveYear });
      return `${t("calendar.lunar")}: ${lunarSys.formatDate(lunar)}`;
    }
    const gd = system.toGregorian({ day, month, year: effectiveYear });
    return `${t("calendar.gregorian")}: ${gd.year}-${String(gd.month).padStart(2, "0")}-${String(gd.day).padStart(2, "0")}`;
  }, [calendarType, day, month, displayYear, effectiveYear, system, t]);

  const segmentOptions = supportedCalendarTypes.map((ct) => ({
    value: ct,
    label: t(getCalendarSystem(ct).labelKey),
  }));

  const gregorianMonths = useMemo(() => {
    const m = [];
    for (let i = 1; i <= 12; i++) {
      m.push({ value: i, label: dayjs().month(i - 1).format("MMMM") });
    }
    return m;
  }, []);

  const gregorianDaysInMonth = useMemo(() => {
    return dayjs(`${effectiveYear}-${String(month).padStart(2, "0")}-01`).daysInMonth();
  }, [effectiveYear, month]);

  const gregorianDayOptions = useMemo(() => {
    const opts = [];
    for (let d = 1; d <= gregorianDaysInMonth; d++) {
      opts.push({ value: d, label: String(d) });
    }
    return opts;
  }, [gregorianDaysInMonth]);

  const selectDropdowns = (
    <Space.Compact style={{ width: "100%" }}>
      <Select
        showSearch
        value={displayYear === null ? NO_YEAR_VALUE : displayYear}
        onChange={handleYearChange}
        options={yearOptions}
        style={{ width: "35%" }}
        placeholder={t("calendar.year")}
      />
      <Select
        value={month}
        onChange={handleMonthChange}
        options={calendarType === "gregorian" ? gregorianMonths : months}
        style={{ width: "35%" }}
        placeholder={t("calendar.month")}
      />
      <Select
        value={day}
        onChange={handleDayChange}
        options={calendarType === "gregorian" ? gregorianDayOptions : dayOptions}
        style={{ width: "30%" }}
        placeholder={t("calendar.day")}
      />
    </Space.Compact>
  );

  if (!enableAlternativeCalendar) {
    if (enableNoYear) {
      return selectDropdowns;
    }
    return (
      <DatePicker
        value={dayjs(`${effectiveYear}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`)}
        onChange={handleGregorianChange}
        style={{ width: "100%" }}
      />
    );
  }

  return (
    <div>
      <Segmented
        options={segmentOptions}
        value={calendarType}
        onChange={handleTypeChange}
        style={{ marginBottom: 8 }}
        block
      />

      {calendarType === "gregorian" && !enableNoYear ? (
        <DatePicker
          value={dayjs(`${effectiveYear}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`)}
          onChange={handleGregorianChange}
          style={{ width: "100%" }}
        />
      ) : selectDropdowns}

      <Text type="secondary" style={{ fontSize: 12, marginTop: 4, display: "block" }}>
        {previewText}
      </Text>
    </div>
  );
}
