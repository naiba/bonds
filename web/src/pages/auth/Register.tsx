import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Card, Form, Input, Button, Typography, App, theme } from "antd";
import { MailOutlined, LockOutlined, UserOutlined } from "@ant-design/icons";
import { useAuth } from "@/stores/auth";
import { useTranslation } from "react-i18next";
import type { RegisterRequest } from "@/types/auth";
import type { APIError } from "@/types/api";

const { Title, Text } = Typography;

export default function Register() {
  const [loading, setLoading] = useState(false);
  const { register } = useAuth();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  async function onFinish(values: RegisterRequest) {
    setLoading(true);
    try {
      await register(values);
      navigate("/vaults", { replace: true });
    } catch (err) {
      const apiErr = err as APIError;
      message.error(apiErr.message || t("auth.register.failed"));
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
        background: token.colorBgLayout,
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
            {t("auth.register.title")}
          </Title>
          <Text type="secondary">{t("auth.register.subtitle")}</Text>
        </div>

        <Form layout="vertical" onFinish={onFinish} size="large">
          <div style={{ display: "flex", gap: 12 }}>
            <Form.Item
              name="first_name"
              style={{ flex: 1 }}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <Input prefix={<UserOutlined />} placeholder={t("auth.register.first_name_placeholder")} />
            </Form.Item>
            <Form.Item
              name="last_name"
              style={{ flex: 1 }}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <Input placeholder={t("auth.register.last_name_placeholder")} />
            </Form.Item>
          </div>

          <Form.Item
            name="email"
            rules={[
              { required: true, message: t("auth.register.email_required") },
              { type: "email", message: t("auth.register.email_invalid") },
            ]}
          >
            <Input prefix={<MailOutlined />} placeholder={t("auth.register.email_placeholder")} />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[
              { required: true, message: t("auth.register.password_required") },
              { min: 8, message: t("auth.register.password_min") },
            ]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder={t("auth.register.password_placeholder")}
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 16 }}>
            <Button type="primary" htmlType="submit" loading={loading} block>
              {t("auth.register.submit")}
            </Button>
          </Form.Item>
        </Form>

        <div style={{ textAlign: "center" }}>
          <Text type="secondary">
            {t("auth.register.has_account")} <Link to="/login">{t("auth.register.sign_in")}</Link>
          </Text>
        </div>
      </Card>
    </div>
  );
}
