import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import {
  Typography,
  Button,
  Table,
  theme,
  Tag,
} from "antd";
import {
  BellOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import { calendarApi } from "@/api/calendar";
import dayjs from "dayjs";

const { Title, Text } = Typography;

export default function VaultReminders() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const { data: reminders = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "reminders"],
    queryFn: async () => {
      const res = await calendarApi.getReminders(vaultId);
      return res.data.data ?? [];
    },
    enabled: !!vaultId,
  });

  return (
    <div style={{ maxWidth: 1000, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 24 }}>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <BellOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0 }}>{t("vault.reminders.title")}</Title>
      </div>

      <Table
        dataSource={reminders}
        rowKey="id"
        loading={isLoading}
        pagination={{ pageSize: 20 }}
        columns={[
          {
            title: t("vault.reminders.label"),
            dataIndex: "label",
            key: "label",
            render: (text) => <Text strong>{text}</Text>,
          },
          {
            title: t("common.contact"),
            key: "contact",
            render: (_, record) => (
              <a
                onClick={(e) => {
                  e.preventDefault();
                  navigate(`/vaults/${vaultId}/contacts/${record.contact_id}`);
                }}
              >
                {record.contact_first_name} {record.contact_last_name}
              </a>
            ),
          },
          {
            title: t("vault.reminders.date"),
            key: "date",
            render: (_, record) => {
              if (!record.year || !record.month || !record.day) return "-";
              const date = dayjs(`${record.year}-${record.month}-${record.day}`);
              return date.format("YYYY-MM-DD");
            },
            sorter: (a, b) => {
              if (!a.year) return -1;
              if (!b.year) return 1;
              const da = new Date(a.year, (a.month || 1) - 1, a.day || 1).getTime();
              const db = new Date(b.year, (b.month || 1) - 1, b.day || 1).getTime();
              return da - db;
            },
          },
          {
            title: t("vault.reminders.type"),
            dataIndex: "type",
            key: "type",
            render: (text) => <Tag>{text}</Tag>,
          },
        ]}
      />
    </div>
  );
}
