import { useState } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { Card, Form, Input, Button, Typography, App, theme } from "antd";
import { UserOutlined, LockOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { invitationsApi } from "@/api/invitations";
import type { APIError } from "@/types/api";

const { Title } = Typography;

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
        background: themeToken.colorBgLayout,
        padding: 16,
      }}
    >
      <Card
        style={{
          width: "100%",
          maxWidth: 400,
          boxShadow: "0 2px 8px rgba(0,0,0,0.06)",
        }}
      >
        <div style={{ textAlign: "center", marginBottom: 32 }}>
          <Title level={3} style={{ marginBottom: 4 }}>
            {t("acceptInvite.title")}
          </Title>
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
