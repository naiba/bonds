import { useState } from "react";
import { Link, useNavigate, useLocation } from "react-router-dom";
import { Card, Form, Input, Button, Typography, App, Divider, Space } from "antd";
import { MailOutlined, LockOutlined, GithubOutlined, GoogleOutlined } from "@ant-design/icons";
import { useAuth } from "@/stores/auth";
import { useTranslation } from "react-i18next";
import type { LoginRequest } from "@/types/auth";
import type { APIError } from "@/types/api";

const { Title, Text } = Typography;

export default function Login() {
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const { message } = App.useApp();
  const { t } = useTranslation();

  const from = (location.state as { from?: { pathname: string } })?.from
    ?.pathname ?? "/vaults";

  async function onFinish(values: LoginRequest) {
    setLoading(true);
    try {
      await login(values);
      navigate(from, { replace: true });
    } catch (err) {
      const apiErr = err as APIError;
      message.error(apiErr.message || t("auth.login.failed"));
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
        background: "#f5f5f5",
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
            {t("auth.login.title")}
          </Title>
          <Text type="secondary">{t("auth.login.subtitle")}</Text>
        </div>

        <Form layout="vertical" onFinish={onFinish} size="large">
          <Form.Item
            name="email"
            rules={[
              { required: true, message: t("auth.login.email_required") },
              { type: "email", message: t("auth.login.email_invalid") },
            ]}
          >
            <Input prefix={<MailOutlined />} placeholder={t("auth.login.email_placeholder")} />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[
              { required: true, message: t("auth.login.password_required") },
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder={t("auth.login.password_placeholder")} />
          </Form.Item>

          <Form.Item style={{ marginBottom: 16 }}>
            <Button type="primary" htmlType="submit" loading={loading} block>
              {t("auth.login.submit")}
            </Button>
          </Form.Item>
        </Form>

        <Divider>{t("oauth.continueWith")}</Divider>
        <Space direction="vertical" style={{ width: "100%" }}>
          <Button block icon={<GithubOutlined />} href="/api/auth/github">
            {t("oauth.github")}
          </Button>
          <Button block icon={<GoogleOutlined />} href="/api/auth/google">
            {t("oauth.google")}
          </Button>
        </Space>

        <div style={{ textAlign: "center", marginTop: 16 }}>
          <Text type="secondary">
            {t("auth.login.no_account")}{" "}
            <Link to="/register">{t("auth.login.create_one")}</Link>
          </Text>
        </div>
      </Card>
    </div>
  );
}
