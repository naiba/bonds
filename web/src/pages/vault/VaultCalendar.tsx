import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  Calendar,
  Badge,
  theme,
  Modal,
  Empty,
} from "antd";
import {
  ArrowLeftOutlined,
  CalendarOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api";
import type { ImportantDate, Reminder } from "@/api";
import type {
  GithubComNaibaBondsInternalDtoCalendarDateItem as CalendarDateItem,
  GithubComNaibaBondsInternalDtoCalendarReminderItem as CalendarReminderItem,
} from "@/api/generated/data-contracts";
import type { Dayjs } from "dayjs";
import { useTranslation } from "react-i18next";

const { Title } = Typography;

interface CalendarItem {
  type: "date" | "reminder";
  label: string;
  dateStr: string;
  calendarType?: string;
}

export default function VaultCalendar() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const [selectedDate, setSelectedDate] = useState<string | null>(null);

  const { data: dates = [] } = useQuery({
    queryKey: ["vaults", vaultId, "calendar", "dates"],
    queryFn: async () => {
      const res = await api.calendar.calendarList(String(vaultId));
      return (res.data?.important_dates ?? []) as ImportantDate[];
    },
    enabled: !!vaultId,
  });

  const { data: reminders = [] } = useQuery({
    queryKey: ["vaults", vaultId, "calendar", "reminders"],
    queryFn: async () => {
      const res = await api.reminders.remindersList(String(vaultId));
      return (res.data ?? []) as Reminder[];
    },
    enabled: !!vaultId,
  });

  const { data: dayDetail } = useQuery({
    queryKey: ["vaults", vaultId, "calendar", "day", selectedDate],
    queryFn: async () => {
      const [y, m, d] = selectedDate!.split("-").map(Number);
      const res = await api.calendar.calendarYearsMonthsDaysDetail(String(vaultId), y, m, d);
      return res.data as { important_dates?: CalendarDateItem[]; reminders?: CalendarReminderItem[] } | undefined;
    },
    enabled: selectedDate !== null,
  });

  function toDateKey(year: number | null, month: number | null, day: number | null): string | null {
    if (!year || !month || !day) return null;
    return `${year}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
  }

  const itemsByDate = new Map<string, CalendarItem[]>();
  for (const d of dates) {
    const key = toDateKey(d.year ?? null, d.month ?? null, d.day ?? null);
    if (!key) continue;
    if (!itemsByDate.has(key)) itemsByDate.set(key, []);
    itemsByDate.get(key)!.push({ type: "date", label: d.label ?? '', dateStr: key, calendarType: d.calendar_type });
  }
  for (const r of reminders) {
    const key = toDateKey(r.year ?? null, r.month ?? null, r.day ?? null);
    if (!key) continue;
    if (!itemsByDate.has(key)) itemsByDate.set(key, []);
    itemsByDate.get(key)!.push({ type: "reminder", label: r.label ?? '', dateStr: key, calendarType: r.calendar_type });
  }

  function cellRender(date: Dayjs) {
    const key = date.format("YYYY-MM-DD");
    const items = itemsByDate.get(key);
    if (!items?.length) return null;

    return (
      <div onClick={() => setSelectedDate(key)} style={{ cursor: "pointer" }}>
        <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
          {items.map((item, i) => (
            <li key={i}>
              <Badge
                status={item.type === "date" ? "success" : "warning"}
                text={
                  <span style={{ fontSize: 11 }}>
                    {item.label}
                    {item.calendarType && item.calendarType !== "gregorian" && (
                      <span style={{ marginLeft: 2, color: "#fa541c", fontSize: 10 }}>🌙</span>
                    )}
                  </span>
                }
              />
            </li>
          ))}
        </ul>
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 24 }}>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <CalendarOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0 }}>{t("vault.calendar.title")}</Title>
      </div>

      <Card
        style={{
          boxShadow: token.boxShadowTertiary,
          borderRadius: token.borderRadiusLG,
          padding: 8,
        }}
      >
        <Calendar
          cellRender={(date) => cellRender(date as Dayjs)}
          onSelect={(date) => setSelectedDate((date as Dayjs).format("YYYY-MM-DD"))}
        />
      </Card>

      <Modal
        title={selectedDate ? `${t("vault.calendar.day_detail")} — ${selectedDate}` : ""}
        open={selectedDate !== null}
        onCancel={() => setSelectedDate(null)}
        footer={null}
      >
        {dayDetail ? (
          <div>
            {(dayDetail.important_dates ?? []).map((d: CalendarDateItem, i: number) => (
              <div key={`d-${i}`} style={{ marginBottom: 8 }}>
                <Badge status="success" text={d.label ?? ""} />
              </div>
            ))}
            {(dayDetail.reminders ?? []).map((r: CalendarReminderItem, i: number) => (
              <div key={`r-${i}`} style={{ marginBottom: 8 }}>
                <Badge status="warning" text={r.label ?? ""} />
              </div>
            ))}
            {!(dayDetail.important_dates?.length || dayDetail.reminders?.length) && (
              <Empty description={t("vault.calendar.no_events")} />
            )}
          </div>
        ) : (
          <Empty description={t("vault.calendar.no_events")} />
        )}
      </Modal>
    </div>
  );
}
