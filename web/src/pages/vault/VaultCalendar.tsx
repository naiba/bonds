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
import dayjs from "dayjs";
import { useTranslation } from "react-i18next";

const { Title } = Typography;

interface CalendarItem {
  type: "date" | "reminder";
  label: string;
  date: string;
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

  const itemsByDate = new Map<string, CalendarItem[]>();
  for (const d of dates) {
    const key = dayjs(d.date).format("YYYY-MM-DD");
    if (!itemsByDate.has(key)) itemsByDate.set(key, []);
    itemsByDate.get(key)!.push({ type: "date", label: d.label, date: d.date });
  }
  for (const r of reminders) {
    const key = dayjs(r.date).format("YYYY-MM-DD");
    if (!itemsByDate.has(key)) itemsByDate.set(key, []);
    itemsByDate.get(key)!.push({ type: "reminder", label: r.label, date: r.date });
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
              text={<span style={{ fontSize: 11 }}>{item.label}</span>}
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
