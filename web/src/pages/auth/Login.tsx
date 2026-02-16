import { useState } from "react";
import { Link, useNavigate, useLocation } from "react-router-dom";
import { Card, Form, Input, Button, Typography, App, Divider, theme } from "antd";
import { MailOutlined, LockOutlined, GithubOutlined, GoogleOutlined } from "@ant-design/icons";
import { useAuth } from "@/stores/auth";
import { useTranslation } from "react-i18next";
import type { LoginRequest, APIError } from "@/api";

const { Title, Text } = Typography;

export default function Login() {
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

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
        background: `linear-gradient(145deg, ${token.colorBgLayout} 0%, ${token.colorPrimaryBg} 50%, ${token.colorBgLayout} 100%)`,
        padding: 16,
      }}
    >
      <Card
        style={{
          width: "100%",
          maxWidth: 420,
          border: `1px solid ${token.colorBorderSecondary}`,
          boxShadow: "0 8px 32px rgba(0,0,0,0.08), 0 2px 8px rgba(0,0,0,0.04)",
          borderRadius: token.borderRadiusLG,
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
              background: `linear-gradient(135deg, ${token.colorPrimary}, ${token.colorPrimaryBgHover})`,
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
              color: token.colorPrimary,
            }}>
              Bonds
            </span>
          </div>
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
        <div style={{ display: "flex", gap: 12 }}>
          <Button
            icon={<GithubOutlined />}
            href="/api/auth/github"
            style={{
              flex: 1,
              borderColor: token.colorBorderSecondary,
            }}
          >
            {t("oauth.github")}
          </Button>
          <Button
            icon={<GoogleOutlined />}
            href="/api/auth/google"
            style={{
              flex: 1,
              borderColor: token.colorBorderSecondary,
            }}
          >
            {t("oauth.google")}
          </Button>
        </div>

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
