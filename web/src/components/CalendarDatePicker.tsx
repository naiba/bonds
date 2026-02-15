import { useState, useEffect, useMemo } from "react";
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

export interface CalendarDatePickerValue {
  calendarType: CalendarType;
  day: number;
  month: number;
  year: number;
}

interface CalendarDatePickerProps {
  value?: CalendarDatePickerValue;
  onChange?: (value: CalendarDatePickerValue) => void;
}

export default function CalendarDatePicker({
  value,
  onChange,
}: CalendarDatePickerProps) {
  const { t } = useTranslation();
  const now = dayjs();

  const [calendarType, setCalendarType] = useState<CalendarType>(
    value?.calendarType ?? "gregorian"
  );
  const [year, setYear] = useState(value?.year ?? now.year());
  const [month, setMonth] = useState(value?.month ?? now.month() + 1);
  const [day, setDay] = useState(value?.day ?? now.date());

  useEffect(() => {
    if (value) {
      setCalendarType(value.calendarType);
      setYear(value.year);
      setMonth(value.month);
      setDay(value.day);
    }
  }, [value]);

  const system = getCalendarSystem(calendarType);
  const months = useMemo(() => system.getMonths(year), [system, year]);
  const daysInMonth = useMemo(
    () => system.getDaysInMonth(year, month),
    [system, year, month]
  );
  const [minYear, maxYear] = system.getYearRange();

  const yearOptions = useMemo(() => {
    const opts = [];
    for (let y = minYear; y <= maxYear; y++) {
      opts.push({ value: y, label: String(y) });
    }
    return opts;
  }, [minYear, maxYear]);

  const dayOptions = useMemo(() => {
    const opts = [];
    for (let d = 1; d <= daysInMonth; d++) {
      opts.push({ value: d, label: String(d) });
    }
    return opts;
  }, [daysInMonth]);

  function emit(ct: CalendarType, y: number, m: number, d: number) {
    const maxD = getCalendarSystem(ct).getDaysInMonth(y, m);
    const safeDay = d > maxD ? maxD : d;
    onChange?.({ calendarType: ct, year: y, month: m, day: safeDay });
  }

  function handleTypeChange(val: string | number) {
    const newType = val as CalendarType;
    setCalendarType(newType);
    const newSystem = getCalendarSystem(newType);
    const converted = newSystem.fromGregorian(
      getCalendarSystem(calendarType).toGregorian({ day, month, year })
    );
    setYear(converted.year);
    setMonth(converted.month);
    setDay(converted.day);
    emit(newType, converted.year, converted.month, converted.day);
  }

  function handleGregorianChange(d: Dayjs | null) {
    if (!d) return;
    setYear(d.year());
    setMonth(d.month() + 1);
    setDay(d.date());
    emit("gregorian", d.year(), d.month() + 1, d.date());
  }

  function handleYearChange(y: number) {
    setYear(y);
    const maxM = system.getMonths(y);
    const validMonth = maxM.some((mo) => mo.value === month) ? month : maxM[0]?.value ?? 1;
    setMonth(validMonth);
    emit(calendarType, y, validMonth, day);
  }

  function handleMonthChange(m: number) {
    setMonth(m);
    emit(calendarType, year, m, day);
  }

  function handleDayChange(d: number) {
    setDay(d);
    emit(calendarType, year, month, d);
  }

  const previewText = useMemo(() => {
    if (calendarType === "gregorian") {
      const lunarSys = getCalendarSystem("lunar");
      const lunar = lunarSys.fromGregorian({ day, month, year });
      return `${t("calendar.lunar")}: ${lunarSys.formatDate(lunar)}`;
    }
    const gd = system.toGregorian({ day, month, year });
    return `${t("calendar.gregorian")}: ${gd.year}-${String(gd.month).padStart(2, "0")}-${String(gd.day).padStart(2, "0")}`;
  }, [calendarType, day, month, year, system, t]);

  const segmentOptions = supportedCalendarTypes.map((ct) => ({
    value: ct,
    label: t(getCalendarSystem(ct).labelKey),
  }));

  return (
    <div>
      <Segmented
        options={segmentOptions}
        value={calendarType}
        onChange={handleTypeChange}
        style={{ marginBottom: 8 }}
        block
      />

      {calendarType === "gregorian" ? (
        <DatePicker
          value={dayjs(`${year}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`)}
          onChange={handleGregorianChange}
          style={{ width: "100%" }}
        />
      ) : (
        <Space.Compact style={{ width: "100%" }}>
          <Select
            showSearch
            value={year}
            onChange={handleYearChange}
            options={yearOptions}
            style={{ width: "35%" }}
            placeholder={t("calendar.year")}
          />
          <Select
            value={month}
            onChange={handleMonthChange}
            options={months}
            style={{ width: "35%" }}
            placeholder={t("calendar.month")}
          />
          <Select
            value={day}
            onChange={handleDayChange}
            options={dayOptions}
            style={{ width: "30%" }}
            placeholder={t("calendar.day")}
          />
        </Space.Compact>
      )}

      <Text type="secondary" style={{ fontSize: 12, marginTop: 4, display: "block" }}>
        {previewText}
      </Text>
    </div>
  );
}
