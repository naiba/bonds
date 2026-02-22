import { useEffect, useState } from "react";
import { useSearchParams, useNavigate, Navigate } from "react-router-dom";
import { Button, Card, Typography, Space, Spin, App, theme } from "antd";
import { MailOutlined, CheckCircleOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/stores/auth";
import { api } from "@/api";
import logoImg from "@/assets/logo.svg";

const { Title, Text, Paragraph } = Typography;

export default function VerifyEmail() {
  const { t } = useTranslation();
  const { token: colorToken } = theme.useToken();
  const { user, logout } = useAuth();
  const { message } = App.useApp();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const verifyToken = searchParams.get("token");

  const [verifying, setVerifying] = useState(false);
  const [verified, setVerified] = useState(false);
  const [resending, setResending] = useState(false);

  useEffect(() => {
    if (!verifyToken) return;
    setVerifying(true);
    api.auth
      .verifyEmailCreate({ token: verifyToken })
      .then(() => {
        setVerified(true);
        message.success(t("verify_email.success"));
        setTimeout(() => navigate("/vaults", { replace: true }), 2000);
      })
      .catch(() => {
        message.error(t("verify_email.invalid_token"));
      })
      .finally(() => setVerifying(false));
  }, [verifyToken, message, navigate, t]);

  const handleResend = async () => {
    setResending(true);
    try {
      await api.auth.resendVerificationCreate();
      message.success(t("verify_email.resent"));
    } catch {
      message.error(t("verify_email.resend_failed"));
    } finally {
      setResending(false);
    }
  };

  if (user?.email_verified_at) {
    return <Navigate to="/vaults" replace />;
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
          <Title level={3} style={{ marginBottom: 4 }}>
            {t("verify_email.title")}
          </Title>
        </div>

        {verifying && (
          <div style={{ textAlign: "center", padding: "32px 0" }}>
            <Spin size="large" />
          </div>
        )}

        {verified && (
          <div style={{ textAlign: "center", padding: "32px 0" }}>
            <Space direction="vertical" align="center" size="middle">
              <CheckCircleOutlined style={{ fontSize: 48, color: colorToken.colorSuccess }} />
              <Text>{t("verify_email.success")}</Text>
            </Space>
          </div>
        )}

        {!verifying && !verified && (
          <div style={{ textAlign: "center" }}>
            <MailOutlined style={{ fontSize: 48, color: colorToken.colorPrimary, marginBottom: 16 }} />
            <Paragraph type="secondary" style={{ marginBottom: 24 }}>
              {t("verify_email.description")}
            </Paragraph>
            <Space direction="vertical" style={{ width: "100%" }} size="middle">
              <Button type="primary" loading={resending} onClick={handleResend} block>
                {t("verify_email.resend")}
              </Button>
              <Button type="link" onClick={logout}>
                {t("verify_email.logout")}
              </Button>
            </Space>
          </div>
        )}
      </Card>
    </div>
  );
}
