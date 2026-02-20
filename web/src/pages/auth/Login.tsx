import { useState, useEffect } from "react";
import { Link, useNavigate, useLocation } from "react-router-dom";
import { Card, Form, Input, Button, Typography, App, Divider, theme, Tooltip } from "antd";
import { MailOutlined, LockOutlined, GithubOutlined, GoogleOutlined, KeyOutlined, SunOutlined, MoonOutlined, DesktopOutlined, GlobalOutlined, LinkOutlined } from "@ant-design/icons";
import { useAuth } from "@/stores/auth";
import logoImg from "@/assets/logo.svg";
import { useTranslation } from "react-i18next";
import { useTheme } from "@/stores/theme";
import type { ThemeMode } from "@/stores/theme";
import { api } from "@/api";
import type { LoginRequest, APIError, InstanceInfo } from "@/api";
import { startAuthentication, browserSupportsWebAuthn } from "@simplewebauthn/browser";
import type { PublicKeyCredentialRequestOptionsJSON } from "@simplewebauthn/browser";
import { httpClient } from "@/api";

const { Title, Text } = Typography;

export default function Login() {
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const { message } = App.useApp();
  const { t, i18n } = useTranslation();
  const { token } = theme.useToken();
  const { themeMode, setThemeMode } = useTheme();
  const [isWebAuthnSupported, setIsWebAuthnSupported] = useState(false);
  const [instanceInfo, setInstanceInfo] = useState<InstanceInfo | null>(null);

  const themeModeOrder: ThemeMode[] = ["light", "dark", "system"];
  const themeModeIcons: Record<ThemeMode, React.ReactNode> = {
    light: <SunOutlined />,
    dark: <MoonOutlined />,
    system: <DesktopOutlined />,
  };
  const themeModeLabels: Record<ThemeMode, string> = {
    light: t("theme.light"),
    dark: t("theme.dark"),
    system: t("theme.system"),
  };
  const nextThemeMode = () => {
    const idx = themeModeOrder.indexOf(themeMode);
    setThemeMode(themeModeOrder[(idx + 1) % themeModeOrder.length]);
  };
  const toggleLanguage = () => {
    const next = i18n.language?.startsWith("zh") ? "en" : "zh";
    i18n.changeLanguage(next);
  };

  useEffect(() => {
    setIsWebAuthnSupported(browserSupportsWebAuthn());
    api.instance.infoList()
      .then(res => setInstanceInfo((res.data ?? null) as InstanceInfo | null))
      .catch(() => {});
  }, []);

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

  async function handleWebAuthnLogin() {
    try {
      // 1. Get options from server
      const beginRes = await api.webauthn.webauthnLoginBeginCreate();
      const options = beginRes.data!.publicKey;

      // 2. Authenticate with browser
      const asseResp = await startAuthentication({ optionsJSON: options as unknown as PublicKeyCredentialRequestOptionsJSON });

      // 3. Verify with server — send assertion as raw body via httpClient
      //    (generated client doesn't declare a body param for this endpoint)
      const verifyRes = await httpClient.instance.post<{
        success: boolean;
        data: { token: string; user: { id: string; email: string } };
      }>("/auth/webauthn/login/finish", asseResp);
      
      const auth = verifyRes.data.data;
      localStorage.setItem("token", auth.token);
      // Force reload to update auth state since we bypassed the store login method
      window.location.href = from;
      
    } catch (error) {
      console.error(error);
      message.error(t("auth.login.passkey_failed"));
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
        background: `linear-gradient(145deg, ${token.colorBgLayout} 0%, ${token.colorPrimaryBg} 50%, ${token.colorBgLayout} 100%)`,
        padding: 16,
        position: "relative",
      }}
    >
      <div style={{ position: "absolute", top: 16, right: 16, display: "flex", gap: 4 }}>
        <Tooltip title={themeModeLabels[themeMode]}>
          <Button type="text" size="small" icon={themeModeIcons[themeMode]} onClick={nextThemeMode} />
        </Tooltip>
        <Tooltip title={i18n.language?.startsWith("zh") ? "English" : "中文"}>
          <Button type="text" size="small" icon={<GlobalOutlined />} onClick={toggleLanguage} />
        </Tooltip>
      </div>
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
            <img src={logoImg} alt="Bonds" style={{ width: 36, height: 36, borderRadius: 10, flexShrink: 0 }} />
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

        {isWebAuthnSupported && instanceInfo?.webauthn_enabled && (
          <div style={{ marginBottom: 24 }}>
            <Button 
              block 
              icon={<KeyOutlined />} 
              onClick={handleWebAuthnLogin}
              style={{ borderColor: token.colorBorderSecondary }}
            >
              {t("auth.login.passkey")}
            </Button>
          </div>
        )}

        {(instanceInfo?.oauth_providers ?? []).length > 0 && (
          <>
            <Divider>{t("oauth.continueWith")}</Divider>
            <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
              {(instanceInfo?.oauth_providers ?? []).map(name => {
                const icon = name === "github" ? <GithubOutlined /> : name === "google" ? <GoogleOutlined /> : <LinkOutlined />;
                const label = name === "github" ? t("oauth.github") : name === "google" ? t("oauth.google") : name === "openid-connect" ? t("oauth.sso") : name;
                return (
                  <Button
                    key={name}
                    icon={icon}
                    href={`/api/auth/${name}`}
                    style={{ flex: 1, minWidth: 120, borderColor: token.colorBorderSecondary }}
                  >
                    {label}
                  </Button>
                );
              })}
            </div>
          </>
        )}

        {(instanceInfo?.registration_enabled !== false) && (
          <div style={{ textAlign: "center", marginTop: 16 }}>
            <Text type="secondary">
              {t("auth.login.no_account")}{" "}
              <Link to="/register">{t("auth.login.create_one")}</Link>
            </Text>
          </div>
        )}
      </Card>
      <div style={{ textAlign: "center", marginTop: 24, color: token.colorTextQuaternary, fontSize: 12 }}>
        © {new Date().getFullYear()}{" "}
        <a href="https://github.com/naiba/bonds" target="_blank" rel="noopener noreferrer" style={{ color: token.colorTextTertiary }}>Bonds</a>
        {instanceInfo?.version && ` ${instanceInfo.version}`}
        {" by "}
        <a href="https://nai.ba" target="_blank" rel="noopener noreferrer" style={{ color: token.colorTextTertiary }}>naiba</a>
      </div>
    </div>
  );
}
