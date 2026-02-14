import {
  Card,
  Typography,
  Table,
  Tag,
  Spin,
  Empty,
} from "antd";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { settingsApi } from "@/api/settings";
import type { User } from "@/types/auth";
import dayjs from "dayjs";

const { Title } = Typography;

export default function Users() {
  const { t } = useTranslation();
  const { data: users = [], isLoading } = useQuery({
    queryKey: ["settings", "users"],
    queryFn: async () => {
      const res = await settingsApi.listUsers();
      return res.data.data ?? [];
    },
  });

  const columns = [
    {
      title: t("settings.users.col_name"),
      key: "name",
      render: (_: unknown, record: User) =>
        `${record.first_name} ${record.last_name}`,
    },
    {
      title: t("settings.users.col_email"),
      dataIndex: "email",
      key: "email",
    },
    {
      title: t("settings.users.col_role"),
      key: "role",
      render: (_: unknown, record: User) =>
        record.is_admin ? <Tag color="blue">{t("settings.users.role_admin")}</Tag> : <Tag>{t("settings.users.role_user")}</Tag>,
    },
    {
      title: t("settings.users.col_joined"),
      dataIndex: "created_at",
      key: "created_at",
      render: (date: string) => dayjs(date).format("MMM D, YYYY"),
    },
  ];

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <Title level={4} style={{ marginBottom: 24 }}>
        {t("settings.users.title")}
      </Title>

      <Card>
        {isLoading ? (
          <Spin />
        ) : users.length === 0 ? (
          <Empty description={t("settings.users.no_users")} />
        ) : (
          <Table
            dataSource={users}
            columns={columns}
            rowKey="id"
            pagination={false}
          />
        )}
      </Card>
    </div>
  );
}
