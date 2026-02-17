import { useState } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { Card, Form, Input, Button, Typography, App, theme, Tooltip } from "antd";
import { UserOutlined, LockOutlined, SunOutlined, MoonOutlined, DesktopOutlined, GlobalOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { useTheme } from "@/stores/theme";
import type { ThemeMode } from "@/stores/theme";
import logoImg from "@/assets/logo.svg";
import { api } from "@/api";
import type { APIError } from "@/api";

const { Title, Text } = Typography;

export default function AcceptInvite() {
  const [loading, setLoading] = useState(false);
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t, i18n } = useTranslation();
  const { token: themeToken } = theme.useToken();
  const { themeMode, setThemeMode } = useTheme();
  const token = searchParams.get("token") ?? "";

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

  async function onFinish(values: {
    first_name: string;
    last_name?: string;
    password: string;
  }) {
    setLoading(true);
    try {
      await api.invitations.acceptCreate({ ...values, token });
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
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        background: `linear-gradient(145deg, ${themeToken.colorBgLayout} 0%, ${themeToken.colorPrimaryBg} 50%, ${themeToken.colorBgLayout} 100%)`,
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
          border: `1px solid ${themeToken.colorBorderSecondary}`,
          boxShadow: "0 8px 32px rgba(0,0,0,0.08), 0 2px 8px rgba(0,0,0,0.04)",
          borderRadius: themeToken.borderRadiusLG,
        }}
      >
        <div style={{ textAlign: "center", marginBottom: 32 }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: 10, marginBottom: 20 }}>
            <img src={logoImg} alt="Bonds" style={{ width: 36, height: 36, borderRadius: 10, flexShrink: 0 }} />
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
      <div style={{ textAlign: "center", marginTop: 24, color: themeToken.colorTextQuaternary, fontSize: 12 }}>
        © {new Date().getFullYear()}{" "}
        <a href="https://github.com/naiba/bonds" target="_blank" rel="noopener noreferrer" style={{ color: themeToken.colorTextTertiary }}>Bonds</a>
        {" by "}
        <a href="https://nai.ba" target="_blank" rel="noopener noreferrer" style={{ color: themeToken.colorTextTertiary }}>naiba</a>
      </div>
    </div>
  );
}
