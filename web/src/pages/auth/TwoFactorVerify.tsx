import { useState } from "react";
import { useNavigate, Navigate, Link } from "react-router-dom";
import { Form, Input, Button, Typography, App, theme, Card } from "antd";
import { SafetyOutlined, LockOutlined } from "@ant-design/icons";
import { useAuth } from "@/stores/auth";
import { useTranslation } from "react-i18next";
import logoImg from "@/assets/logo.svg";
import type { APIError } from "@/api";

const { Title, Text } = Typography;

export default function TwoFactorVerify() {
  const [loading, setLoading] = useState(false);
  const [useRecoveryCode, setUseRecoveryCode] = useState(false);
  const { twoFactorPending, tempToken, verifyTwoFactor, logout } = useAuth();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token: colorToken } = theme.useToken();

  if (!twoFactorPending || !tempToken) {
    return <Navigate to="/login" replace />;
  }

  async function onFinish(values: { code: string }) {
    setLoading(true);
    try {
      await verifyTwoFactor(values.code);
      navigate("/vaults", { replace: true });
    } catch (err) {
      const apiErr = err as APIError;
      message.error(apiErr.message || t("auth.two_factor_verify.failed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        background: `linear-gradient(145deg, ${colorToken.colorBgLayout} 0%, ${colorToken.colorPrimaryBg} 50%, ${colorToken.colorBgLayout} 100%)`,
        padding: 16,
      }}
    >
      <Card
        style={{
          width: "100%",
          maxWidth: 420,
          border: `1px solid ${colorToken.colorBorderSecondary}`,
          boxShadow: "0 8px 32px rgba(0,0,0,0.08), 0 2px 8px rgba(0,0,0,0.04)",
          borderRadius: colorToken.borderRadiusLG,
        }}
      >
        <div style={{ textAlign: "center", marginBottom: 32 }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: 10, marginBottom: 20 }}>
            <img src={logoImg} alt="Bonds" style={{ width: 36, height: 36, borderRadius: 10, flexShrink: 0 }} />
            <span style={{
              fontWeight: 700,
              fontSize: 22,
              letterSpacing: "-0.02em",
              color: colorToken.colorPrimary,
            }}>
              Bonds
            </span>
          </div>
          <SafetyOutlined style={{ fontSize: 40, color: colorToken.colorPrimary, marginBottom: 12 }} />
          <Title level={3} style={{ marginBottom: 4 }}>
            {t("auth.two_factor_verify.title")}
          </Title>
          <Text type="secondary">{t("auth.two_factor_verify.subtitle")}</Text>
        </div>

        <Form layout="vertical" onFinish={onFinish} size="large">
          {!useRecoveryCode ? (
            <Form.Item
              name="code"
              rules={[{ required: true, message: t("auth.two_factor_verify.code_required") }]}
            >
              <Input
                prefix={<LockOutlined />}
                placeholder={t("auth.two_factor_verify.code_placeholder")}
                maxLength={6}
                autoComplete="one-time-code"
                inputMode="numeric"
              />
            </Form.Item>
          ) : (
            <Form.Item
              name="code"
              rules={[{ required: true, message: t("auth.two_factor_verify.code_required") }]}
            >
              <Input
                prefix={<LockOutlined />}
                placeholder={t("auth.two_factor_verify.recovery_code_placeholder")}
                autoComplete="off"
              />
            </Form.Item>
          )}

          <Form.Item style={{ marginBottom: 16 }}>
            <Button type="primary" htmlType="submit" loading={loading} block>
              {t("auth.two_factor_verify.submit")}
            </Button>
          </Form.Item>
        </Form>

        <div style={{ textAlign: "center" }}>
          <Button
            type="link"
            onClick={() => setUseRecoveryCode(!useRecoveryCode)}
            style={{ padding: 0, marginBottom: 8 }}
          >
            {t("auth.two_factor_verify.use_recovery_code")}
          </Button>
        </div>

        <div style={{ textAlign: "center", marginTop: 8 }}>
          <Link to="/login" onClick={() => logout()}>
            {t("auth.two_factor_verify.back_to_login")}
          </Link>
        </div>
      </Card>
    </div>
  );
}
