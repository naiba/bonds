import { Card, Typography, Descriptions, Button, App, theme } from "antd";
import { LogoutOutlined, UserOutlined } from "@ant-design/icons";
import { useAuth } from "@/stores/auth";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title, Text } = Typography;

export default function Settings() {
  const { user, logout } = useAuth();
  const { modal } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  function handleLogout() {
    modal.confirm({
      title: t("settings.account.sign_out_confirm"),
      content: t("settings.account.sign_out_message"),
      okText: t("settings.account.sign_out"),
      okButtonProps: { danger: true },
      onOk: logout,
    });
  }

  const initials = [user?.first_name?.[0], user?.last_name?.[0]]
    .filter(Boolean)
    .join("")
    .toUpperCase();

  return (
    <div style={{ maxWidth: 640, margin: "0 auto" }}>
      <Title level={4} style={{ marginBottom: 4 }}>
        {t("settings.title")}
      </Title>
      <Text type="secondary" style={{ display: "block", marginBottom: 24 }}>
        {t("settings.description")}
      </Text>

      <Card>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 16,
            marginBottom: 24,
          }}
        >
          <div
            style={{
              width: 56,
              height: 56,
              borderRadius: "50%",
              background: token.colorPrimary,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              flexShrink: 0,
            }}
          >
            {initials ? (
              <span
                style={{
                  color: "#fff",
                  fontSize: 20,
                  fontWeight: 600,
                  lineHeight: 1,
                }}
              >
                {initials}
              </span>
            ) : (
              <UserOutlined style={{ color: "#fff", fontSize: 24 }} />
            )}
          </div>
          <div style={{ minWidth: 0 }}>
            <Text
              strong
              style={{
                fontSize: 18,
                display: "block",
                lineHeight: 1.3,
              }}
            >
              {user?.first_name} {user?.last_name}
            </Text>
            <Text type="secondary">{user?.email}</Text>
          </div>
        </div>

        <Descriptions
          column={1}
          bordered
          size="small"
          labelStyle={{
            fontWeight: 500,
            color: token.colorTextSecondary,
            width: 140,
          }}
          contentStyle={{
            color: token.colorText,
          }}
        >
          <Descriptions.Item label={t("settings.account.name")}>
            {user?.first_name} {user?.last_name}
          </Descriptions.Item>
          <Descriptions.Item label={t("settings.account.email")}>
            {user?.email}
          </Descriptions.Item>
          <Descriptions.Item label={t("settings.account.member_since")}>
            {user?.created_at
              ? dayjs(user.created_at).format("MMMM D, YYYY")
              : "—"}
          </Descriptions.Item>
        </Descriptions>

        <div style={{ marginTop: 32 }}>
          <Button
            danger
            icon={<LogoutOutlined />}
            onClick={handleLogout}
          >
            {t("settings.account.sign_out")}
          </Button>
        </div>
      </Card>
    </div>
  );
}
