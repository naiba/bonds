import { Card, Typography, Descriptions, Button, Divider, App } from "antd";
import { useAuth } from "@/stores/auth";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title } = Typography;

export default function Settings() {
  const { user, logout } = useAuth();
  const { modal } = App.useApp();
  const { t } = useTranslation();

  function handleLogout() {
    modal.confirm({
      title: t("settings.account.sign_out_confirm"),
      content: t("settings.account.sign_out_message"),
      okText: t("settings.account.sign_out"),
      okButtonProps: { danger: true },
      onOk: logout,
    });
  }

  return (
    <div style={{ maxWidth: 640, margin: "0 auto" }}>
      <Title level={4} style={{ marginBottom: 24 }}>
        {t("settings.title")}
      </Title>

      <Card title={t("settings.account.title")}>
        <Descriptions column={1}>
          <Descriptions.Item label={t("settings.account.name")}>
            {user?.first_name} {user?.last_name}
          </Descriptions.Item>
          <Descriptions.Item label={t("settings.account.email")}>{user?.email}</Descriptions.Item>
          <Descriptions.Item label={t("settings.account.member_since")}>
            {user?.created_at
              ? dayjs(user.created_at).format("MMMM D, YYYY")
              : "—"}
          </Descriptions.Item>
        </Descriptions>

        <Divider />

        <Button danger onClick={handleLogout}>
          {t("settings.account.sign_out")}
        </Button>
      </Card>
    </div>
  );
}
