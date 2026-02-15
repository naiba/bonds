import { useState } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { Card, Form, Input, Button, Typography, App, theme } from "antd";
import { UserOutlined, LockOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { invitationsApi } from "@/api/invitations";
import type { APIError } from "@/types/api";

const { Title, Text } = Typography;

export default function AcceptInvite() {
  const [loading, setLoading] = useState(false);
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token: themeToken } = theme.useToken();
  const token = searchParams.get("token") ?? "";

  async function onFinish(values: {
    first_name: string;
    last_name?: string;
    password: string;
  }) {
    setLoading(true);
    try {
      await invitationsApi.accept({ ...values, token });
      message.success(t("acceptInvite.title"));
      navigate("/login");
    } catch (err) {
      const apiErr = err as APIError;
      message.error(apiErr.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: `linear-gradient(145deg, ${themeToken.colorBgLayout} 0%, ${themeToken.colorPrimaryBg} 50%, ${themeToken.colorBgLayout} 100%)`,
        padding: 16,
      }}
    >
      <Card
        style={{
          width: "100%",
          maxWidth: 420,
          border: `1px solid ${themeToken.colorBorderSecondary}`,
          boxShadow: "0 8px 32px rgba(0,0,0,0.08), 0 2px 8px rgba(0,0,0,0.04)",
          borderRadius: themeToken.borderRadiusLG,
        }}
      >
        <div style={{ textAlign: "center", marginBottom: 32 }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: 10, marginBottom: 20 }}>
            <span style={{
              display: "inline-flex",
              alignItems: "center",
              justifyContent: "center",
              width: 36,
              height: 36,
              borderRadius: 10,
              background: `linear-gradient(135deg, ${themeToken.colorPrimary}, ${themeToken.colorPrimaryBgHover})`,
              color: "#fff",
              fontSize: 17,
              fontWeight: 800,
              flexShrink: 0,
            }}>
              B
            </span>
            <span style={{
              fontWeight: 700,
              fontSize: 22,
              letterSpacing: "-0.02em",
              color: themeToken.colorPrimary,
            }}>
              Bonds
            </span>
          </div>
          <Title level={3} style={{ marginBottom: 4 }}>
            {t("acceptInvite.title")}
          </Title>
          <Text type="secondary">{t("acceptInvite.subtitle")}</Text>
        </div>

        <Form layout="vertical" onFinish={onFinish} size="large">
          <Form.Item
            name="first_name"
            rules={[{ required: true }]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder={t("acceptInvite.firstName")}
            />
          </Form.Item>

          <Form.Item name="last_name">
            <Input
              prefix={<UserOutlined />}
              placeholder={t("acceptInvite.lastName")}
            />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, min: 8 }]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder={t("acceptInvite.password")}
            />
          </Form.Item>

          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              block
            >
              {t("acceptInvite.submit")}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
