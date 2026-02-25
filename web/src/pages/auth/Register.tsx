import { useState, useEffect } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Form, Input, Button, Typography, App, theme, Tooltip, Spin } from "antd";
import { MailOutlined, LockOutlined, UserOutlined, SunOutlined, MoonOutlined, DesktopOutlined, GlobalOutlined } from "@ant-design/icons";
import { useAuth } from "@/stores/auth";
import logoImg from "@/assets/logo.svg";
import { useTranslation } from "react-i18next";
import { useTheme } from "@/stores/theme";
import type { ThemeMode } from "@/stores/theme";
import type { RegisterRequest, APIError } from "@/api";
import { api } from "@/api";

const { Title, Text } = Typography;

/* ---- CSS keyframes injected once (shared id with Login) ---- */
const styleId = "bonds-auth-animations";
if (typeof document !== "undefined" && !document.getElementById(styleId)) {
  const style = document.createElement("style");
  style.id = styleId;
  style.textContent = `
    @keyframes bondsFieldFadeIn {
      from { opacity: 0; transform: translateY(12px); }
      to   { opacity: 1; transform: translateY(0); }
    }
    @keyframes bondsHeroPulse {
      0%, 100% { transform: scale(1); opacity: 0.5; }
      50%      { transform: scale(1.08); opacity: 0.7; }
    }
    @keyframes bondsHeroDrift {
      0%, 100% { transform: translate(0, 0) rotate(0deg); }
      33%      { transform: translate(12px, -8px) rotate(2deg); }
      66%      { transform: translate(-6px, 6px) rotate(-1deg); }
    }
    @keyframes bondsLeafFloat {
      0%, 100% { transform: translateY(0) rotate(0deg); }
      50%      { transform: translateY(-10px) rotate(5deg); }
    }
    @media (max-width: 768px) {
      .bonds-auth-hero { display: none !important; }
      .bonds-auth-wrapper { grid-template-columns: 1fr !important; }
    }
  `;
  document.head.appendChild(style);
}

export default function Register() {
  const [loading, setLoading] = useState(false);
  const [registrationEnabled, setRegistrationEnabled] = useState(true);
  const [loadingInstanceInfo, setLoadingInstanceInfo] = useState(true);
  const { register } = useAuth();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t, i18n } = useTranslation();
  const { token } = theme.useToken();
  const { themeMode, setThemeMode } = useTheme();

  useEffect(() => {
    let cancelled = false;
    api.instance
      .infoList()
      .then((res) => {
        if (!cancelled) {
          setRegistrationEnabled(res.data?.registration_enabled !== false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setRegistrationEnabled(true);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingInstanceInfo(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

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

  /* ---- Stagger animation helper ---- */
  const fadeIn = (index: number): React.CSSProperties => ({
    opacity: 0,
    animation: "bondsFieldFadeIn 0.5s ease forwards",
    animationDelay: `${index * 0.07}s`,
  });

  if (loadingInstanceInfo) {
    return (
      <div
        style={{
          minHeight: "100vh",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          background: token.colorBgContainer,
        }}
      >
        <Spin />
      </div>
    );
  }

  if (!registrationEnabled) {
    return (
      <div
        style={{
          minHeight: "100vh",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          background: token.colorBgContainer,
          padding: 16,
        }}
      >
        <div style={{
          width: "100%", maxWidth: 420, textAlign: "center",
          padding: 40, borderRadius: 16,
          border: `1px solid ${token.colorBorderSecondary}`,
          background: token.colorBgElevated,
        }}>
          <Title level={3} style={{ marginBottom: 4 }}>
            Registration Disabled
          </Title>
          <Text type="secondary">User registration is currently disabled. Please contact an administrator.</Text>
          <div style={{ marginTop: 24 }}>
            <Link to="/login">
              <Button type="primary">{t("auth.register.sign_in")}</Button>
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div
      className="bonds-auth-wrapper"
      style={{
        minHeight: "100vh",
        display: "grid",
        gridTemplateColumns: "1fr 1fr",
      }}
    >
      {/* ====== HERO SIDE ====== */}
      <div
        className="bonds-auth-hero"
        style={{
          position: "relative",
          overflow: "hidden",
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          alignItems: "center",
          padding: "64px 48px",
          background: "linear-gradient(160deg, #3a6347 0%, #4a7c59 35%, #5e9a6f 65%, #3d6a4c 100%)",
        }}
      >
        {/* Decorative background orbs */}
        <div style={{
          position: "absolute", top: "-10%", left: "-10%", width: "55%", height: "55%",
          borderRadius: "50%",
          background: "radial-gradient(circle, rgba(255,255,255,0.08) 0%, transparent 70%)",
          animation: "bondsHeroPulse 8s ease-in-out infinite",
          pointerEvents: "none",
        }} />
        <div style={{
          position: "absolute", bottom: "-15%", right: "-8%", width: "50%", height: "60%",
          borderRadius: "50%",
          background: "radial-gradient(circle, rgba(255,255,255,0.06) 0%, transparent 70%)",
          animation: "bondsHeroPulse 10s ease-in-out infinite 2s",
          pointerEvents: "none",
        }} />
        <div style={{
          position: "absolute", top: "20%", right: "15%", width: "30%", height: "30%",
          borderRadius: "50%",
          background: "radial-gradient(circle, rgba(212,168,83,0.1) 0%, transparent 70%)",
          animation: "bondsHeroDrift 14s ease-in-out infinite",
          pointerEvents: "none",
        }} />

        {/* Grid pattern overlay */}
        <div style={{
          position: "absolute", inset: 0, opacity: 0.04, pointerEvents: "none",
          backgroundImage: `
            linear-gradient(rgba(255,255,255,0.5) 1px, transparent 1px),
            linear-gradient(90deg, rgba(255,255,255,0.5) 1px, transparent 1px)
          `,
          backgroundSize: "60px 60px",
        }} />

        {/* Decorative leaf shapes — CSS only */}
        <div style={{
          position: "absolute", top: "15%", right: "10%", width: 36, height: 36,
          borderRadius: "50% 0 50% 0", border: "2px solid rgba(255,255,255,0.15)",
          transform: "rotate(-30deg)", animation: "bondsLeafFloat 6s ease-in-out infinite",
          pointerEvents: "none",
        }} />
        <div style={{
          position: "absolute", bottom: "22%", right: "20%", width: 24, height: 24,
          borderRadius: "50% 0 50% 0", border: "2px solid rgba(255,255,255,0.1)",
          transform: "rotate(20deg)", animation: "bondsLeafFloat 8s ease-in-out infinite 1s",
          pointerEvents: "none",
        }} />
        <div style={{
          position: "absolute", top: "40%", left: "8%", width: 28, height: 28,
          borderRadius: "0 50% 0 50%", border: "2px solid rgba(255,255,255,0.12)",
          transform: "rotate(-15deg)", animation: "bondsLeafFloat 7s ease-in-out infinite 0.5s",
          pointerEvents: "none",
        }} />
        <div style={{
          position: "absolute", bottom: "35%", left: "22%", width: 20, height: 20,
          borderRadius: "50% 0 50% 0", background: "rgba(255,255,255,0.06)",
          transform: "rotate(-45deg)", animation: "bondsLeafFloat 9s ease-in-out infinite 2s",
          pointerEvents: "none",
        }} />

        {/* Diamond accents */}
        <div style={{
          position: "absolute", top: "60%", right: "12%", width: 16, height: 16,
          border: "1.5px solid rgba(212,168,83,0.25)", transform: "rotate(45deg)",
          pointerEvents: "none",
        }} />
        <div style={{
          position: "absolute", top: "25%", left: "15%", width: 12, height: 12,
          border: "1.5px solid rgba(255,255,255,0.1)", transform: "rotate(45deg)",
          pointerEvents: "none",
        }} />

        {/* Hero content */}
        <div style={{ position: "relative", zIndex: 1, textAlign: "center", maxWidth: 380 }}>
          <div style={{
            display: "inline-flex", alignItems: "center", gap: 14, marginBottom: 32,
            padding: "10px 22px", borderRadius: 50,
            background: "rgba(255,255,255,0.1)", backdropFilter: "blur(12px)",
            border: "1px solid rgba(255,255,255,0.15)",
          }}>
            <img src={logoImg} alt="Bonds" style={{ width: 32, height: 32, borderRadius: 8 }} />
            <span style={{
              fontSize: 20, fontWeight: 700, color: "#fff",
              letterSpacing: "0.12em", textTransform: "uppercase",
            }}>
              Bonds
            </span>
          </div>

          <h1 style={{
            fontSize: 34, fontWeight: 400, color: "#fff", lineHeight: 1.3, fontFamily: "\x27Playfair Display\x27, serif",
            margin: "0 0 16px 0", letterSpacing: "-0.01em",
          }}>
            {t("auth.register.hero_title", { defaultValue: "Begin your journey of meaningful connections" })}
          </h1>
          <p style={{
            fontSize: 15, color: "rgba(255,255,255,0.7)", lineHeight: 1.7,
            margin: 0, fontWeight: 400,
          }}>
            {t("auth.register.hero_subtitle", { defaultValue: "A private, personal space to cherish the people in your life." })}
          </p>

          {/* Decorative line */}
          <div style={{
            width: 48, height: 2, background: "rgba(212,168,83,0.5)",
            margin: "28px auto 0", borderRadius: 1,
          }} />
        </div>
      </div>

      {/* ====== FORM SIDE ====== */}
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          alignItems: "center",
          padding: "64px 40px",
          background: token.colorBgContainer,
          position: "relative",
          overflowY: "auto",
        }}
      >
        {/* Theme & language toggles */}
        <div style={{ position: "absolute", top: 16, right: 16, display: "flex", gap: 4 }}>
          <Tooltip title={themeModeLabels[themeMode]}>
            <Button type="text" size="small" icon={themeModeIcons[themeMode]} onClick={nextThemeMode} />
          </Tooltip>
          <Tooltip title={i18n.language?.startsWith("zh") ? "English" : "中文"}>
            <Button type="text" size="small" icon={<GlobalOutlined />} onClick={toggleLanguage} />
          </Tooltip>
        </div>

        <div style={{ width: "100%", maxWidth: 400 }}>
          {/* Header */}
          <div style={{ marginBottom: 36, ...fadeIn(0) }}>
            <Title level={2} style={{ marginBottom: 8, fontFamily: "\x27Playfair Display\x27, serif" }}>
              {t("auth.register.title")}
            </Title>
            <Text type="secondary">{t("auth.register.subtitle")}</Text>
          </div>

          {/* Form */}
          <Form layout="vertical" onFinish={onFinish} size="large">
            <div style={{ display: "flex", gap: 12, ...fadeIn(1) }}>
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

            <div style={fadeIn(2)}>
              <Form.Item
                name="email"
                rules={[
                  { required: true, message: t("auth.register.email_required") },
                  { type: "email", message: t("auth.register.email_invalid") },
                ]}
              >
                <Input prefix={<MailOutlined />} placeholder={t("auth.register.email_placeholder")} />
              </Form.Item>
            </div>

            <div style={fadeIn(3)}>
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
            </div>

            <div style={fadeIn(4)}>
              <Form.Item style={{ marginBottom: 16 }}>
                <Button type="primary" htmlType="submit" loading={loading} block>
                  {t("auth.register.submit")}
                </Button>
              </Form.Item>
            </div>
          </Form>

          <div style={{ textAlign: "center", ...fadeIn(5) }}>
            <Text type="secondary">
              {t("auth.register.has_account")} <Link to="/login">{t("auth.register.sign_in")}</Link>
            </Text>
          </div>
        </div>

        {/* Footer */}
        <div style={{
          textAlign: "center", marginTop: 48, color: token.colorTextQuaternary, fontSize: 12,
          position: "absolute", bottom: 20, left: 0, right: 0,
        }}>
          © {new Date().getFullYear()}{" "}
          <a href="https://github.com/naiba/bonds" target="_blank" rel="noopener noreferrer" style={{ color: token.colorTextTertiary }}>Bonds</a>
          {" by "}
          <a href="https://nai.ba" target="_blank" rel="noopener noreferrer" style={{ color: token.colorTextTertiary }}>naiba</a>
        </div>
      </div>
    </div>
  );
}
