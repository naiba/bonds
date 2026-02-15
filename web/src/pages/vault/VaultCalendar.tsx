import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  Calendar,
  Badge,
} from "antd";
import { ArrowLeftOutlined } from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import client from "@/api/client";
import type { APIResponse } from "@/types/api";
import type { ImportantDate, Reminder } from "@/types/modules";
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

  const { data: dates = [] } = useQuery({
    queryKey: ["vaults", vaultId, "calendar", "dates"],
    queryFn: async () => {
      const res = await client.get<APIResponse<ImportantDate[]>>(
        `/vaults/${vaultId}/important-dates`,
      );
      return res.data.data ?? [];
    },
    enabled: !!vaultId,
  });

  const { data: reminders = [] } = useQuery({
    queryKey: ["vaults", vaultId, "calendar", "reminders"],
    queryFn: async () => {
      const res = await client.get<APIResponse<Reminder[]>>(
        `/vaults/${vaultId}/reminders`,
      );
      return res.data.data ?? [];
    },
    enabled: !!vaultId,
  });

  function toDateKey(year: number | null, month: number | null, day: number | null): string | null {
    if (!year || !month || !day) return null;
    return `${year}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
  }

  const itemsByDate = new Map<string, CalendarItem[]>();
  for (const d of dates) {
    const key = toDateKey(d.year, d.month, d.day);
    if (!key) continue;
    if (!itemsByDate.has(key)) itemsByDate.set(key, []);
    itemsByDate.get(key)!.push({ type: "date", label: d.label, dateStr: key, calendarType: d.calendar_type });
  }
  for (const r of reminders) {
    const key = toDateKey(r.year, r.month, r.day);
    if (!key) continue;
    if (!itemsByDate.has(key)) itemsByDate.set(key, []);
    itemsByDate.get(key)!.push({ type: "reminder", label: r.label, dateStr: key, calendarType: r.calendar_type });
  }

  function cellRender(date: Dayjs) {
    const key = date.format("YYYY-MM-DD");
    const items = itemsByDate.get(key);
    if (!items?.length) return null;

    return (
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
    );
  }

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}`)}
        style={{ marginBottom: 16 }}
      >
        {t("vault.calendar.back")}
      </Button>

      <Title level={4}>{t("vault.calendar.title")}</Title>

      <Card>
        <Calendar cellRender={(date) => cellRender(date as Dayjs)} />
      </Card>
    </div>
  );
}
