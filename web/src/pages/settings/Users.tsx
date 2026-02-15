import {
  Card,
  Typography,
  Table,
  Tag,
  Spin,
  Empty,
  theme,
} from "antd";
import { CrownOutlined } from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { settingsApi } from "@/api/settings";
import type { User } from "@/types/auth";
import dayjs from "dayjs";

const { Title, Text } = Typography;

const avatarColors = [
  "#5b8c5a", "#e8864b", "#5b7fb5", "#c75d8a",
  "#7b6bb5", "#d4a853", "#4ba8b5", "#b55b5b",
];
function getAvatarColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return avatarColors[Math.abs(hash) % avatarColors.length];
}

export default function Users() {
  const { t } = useTranslation();
  const { token } = theme.useToken();
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
      render: (_: unknown, record: User) => {
        const fullName = `${record.first_name} ${record.last_name}`;
        const initials = [record.first_name?.[0], record.last_name?.[0]]
          .filter(Boolean)
          .join("")
          .toUpperCase();
        const color = getAvatarColor(fullName);
        return (
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <div
              style={{
                width: 32,
                height: 32,
                borderRadius: "50%",
                background: color,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                flexShrink: 0,
                color: "#fff",
                fontSize: 13,
                fontWeight: 600,
              }}
            >
              {initials}
            </div>
            <span style={{ fontWeight: 500 }}>{fullName}</span>
          </div>
        );
      },
    },
    {
      title: t("settings.users.col_email"),
      dataIndex: "email",
      key: "email",
      render: (email: string) => (
        <Text type="secondary">{email}</Text>
      ),
    },
    {
      title: t("settings.users.col_role"),
      key: "role",
      render: (_: unknown, record: User) =>
        record.is_admin ? (
          <Tag color="blue" icon={<CrownOutlined />}>
            {t("settings.users.role_admin")}
          </Tag>
        ) : (
          <Tag color={token.colorBgLayout}>
            {t("settings.users.role_user")}
          </Tag>
        ),
    },
    {
      title: t("settings.users.col_joined"),
      dataIndex: "created_at",
      key: "created_at",
      render: (date: string) => (
        <Text type="secondary">{dayjs(date).format("MMM D, YYYY")}</Text>
      ),
    },
  ];

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <Title level={4} style={{ marginBottom: 4 }}>
        {t("settings.users.title")}
      </Title>
      <Text type="secondary" style={{ display: "block", marginBottom: 24 }}>
        {t("settings.users.description")}
      </Text>

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
