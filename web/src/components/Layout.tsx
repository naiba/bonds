import { Outlet, useNavigate, useLocation, useParams } from "react-router-dom";
import {
  Layout as AntLayout,
  Dropdown,
  Avatar,
  theme,
  Button,
  Tooltip,
} from "antd";
import {
  SettingOutlined,
  TeamOutlined,
  LogoutOutlined,
  UserOutlined,
  CalendarOutlined,
  FileOutlined,
  BarChartOutlined,
  EditOutlined,
  UsergroupAddOutlined,
  UnorderedListOutlined,
  BellOutlined,
  ControlOutlined,
  UserSwitchOutlined,
  GlobalOutlined,
  LockOutlined,
  MailOutlined,
  SunOutlined,
  MoonOutlined,
  DesktopOutlined,
  CheckSquareOutlined,
  DashboardOutlined,
  RightOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { useAuth } from "@/stores/auth";
import { useTheme } from "@/stores/theme";
import type { ThemeMode } from "@/stores/theme";
import { useTranslation } from "react-i18next";
import SearchBar from "@/components/SearchBar";

const { Header, Content } = AntLayout;

export default function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const { id: vaultId } = useParams();
  const { token } = theme.useToken();
  const { t, i18n } = useTranslation();
  const { themeMode, setThemeMode } = useTheme();

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

  const isInVault = !!location.pathname.match(/^\/vaults\/[^/]+(\/|$)/);

  const vaultNavItems: { key: string; icon: React.ReactNode; label: string }[] = vaultId
    ? [
        { key: `/vaults/${vaultId}`, icon: <DashboardOutlined />, label: t("nav.dashboard") },
        { key: `/vaults/${vaultId}/contacts`, icon: <TeamOutlined />, label: t("nav.contacts") },
        { key: `/vaults/${vaultId}/journals`, icon: <EditOutlined />, label: t("nav.journal") },
        { key: `/vaults/${vaultId}/groups`, icon: <UsergroupAddOutlined />, label: t("nav.groups") },
        { key: `/vaults/${vaultId}/calendar`, icon: <CalendarOutlined />, label: t("nav.calendar") },
        { key: `/vaults/${vaultId}/tasks`, icon: <CheckSquareOutlined />, label: t("nav.tasks") },
        { key: `/vaults/${vaultId}/reports`, icon: <BarChartOutlined />, label: t("nav.reports") },
        { key: `/vaults/${vaultId}/files`, icon: <FileOutlined />, label: t("nav.files") },
        { key: `/vaults/${vaultId}/feed`, icon: <UnorderedListOutlined />, label: t("nav.feed") },
      ]
    : [];

  const activeVaultKey = vaultNavItems
    .slice()
    .sort((a, b) => b.key.length - a.key.length)
    .find((item) => location.pathname.startsWith(item.key))?.key ?? "";

  const userMenuItems: MenuProps["items"] = [
    { key: "/settings", icon: <SettingOutlined />, label: t("nav.account") },
    { key: "/settings/preferences", icon: <ControlOutlined />, label: t("nav.preferences") },
    { key: "/settings/notifications", icon: <BellOutlined />, label: t("nav.notifications") },
    { key: "/settings/personalize", icon: <EditOutlined />, label: t("nav.personalize") },
    { key: "/settings/users", icon: <UserSwitchOutlined />, label: t("nav.users") },
    { key: "/settings/2fa", icon: <LockOutlined />, label: t("nav.twoFactor") },
    { key: "/settings/invitations", icon: <MailOutlined />, label: t("nav.invitations") },
    { type: "divider" as const },
    { key: "logout", icon: <LogoutOutlined />, label: t("nav.logout"), danger: true },
  ];

  const nextThemeMode = () => {
    const idx = themeModeOrder.indexOf(themeMode);
    setThemeMode(themeModeOrder[(idx + 1) % themeModeOrder.length]);
  };

  const toggleLanguage = () => {
    const next = i18n.language?.startsWith("zh") ? "en" : "zh";
    i18n.changeLanguage(next);
  };

  const initials = user
    ? `${user.first_name.charAt(0)}${user.last_name.charAt(0)}`.toUpperCase()
    : "";

  return (
    <AntLayout style={{ minHeight: "100vh" }}>
      <div style={{ position: "sticky", top: 0, zIndex: 100 }}>
        <Header
          style={{
            background: token.colorBgContainer,
            borderBottom: isInVault ? "none" : `1px solid ${token.colorBorderSecondary}`,
            padding: "0 24px",
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            height: 48,
            lineHeight: "48px",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
            <div
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 6,
                cursor: "pointer",
                userSelect: "none",
                padding: "3px 10px",
                borderRadius: token.borderRadius,
                border: `1px solid ${token.colorBorderSecondary}`,
                background: token.colorBgLayout,
                fontSize: 13,
              }}
              onClick={() => navigate("/vaults")}
            >
              <span style={{ fontWeight: 600, color: token.colorText }}>
                {user?.first_name} {user?.last_name}
              </span>
              {isInVault && vaultId && (
                <>
                  <RightOutlined style={{ fontSize: 10, color: token.colorTextQuaternary }} />
                  <span style={{ color: token.colorTextSecondary }}>
                    {t("nav.vault")}
                  </span>
                </>
              )}
            </div>
            <SearchBar />
          </div>

          <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
            <Tooltip title={themeModeLabels[themeMode]}>
              <Button type="text" size="small" icon={themeModeIcons[themeMode]} onClick={nextThemeMode} />
            </Tooltip>
            <Tooltip title={i18n.language?.startsWith("zh") ? "English" : "中文"}>
              <Button type="text" size="small" icon={<GlobalOutlined />} onClick={toggleLanguage} />
            </Tooltip>
            <Dropdown
              menu={{
                items: userMenuItems,
                onClick: ({ key }) => {
                  if (key === "logout") {
                    logout();
                    navigate("/login");
                  } else {
                    navigate(key);
                  }
                },
              }}
              placement="bottomRight"
            >
              <div
                style={{
                  cursor: "pointer",
                  display: "flex",
                  alignItems: "center",
                  marginLeft: 4,
                  padding: "2px 4px",
                  borderRadius: token.borderRadius,
                }}
              >
                <Avatar
                  size={26}
                  icon={<UserOutlined />}
                  style={{ backgroundColor: token.colorPrimary, fontSize: 12 }}
                >
                  {initials}
                </Avatar>
              </div>
            </Dropdown>
          </div>
        </Header>

        {isInVault && vaultNavItems.length > 0 && (
          <nav
            style={{
              background: token.colorBgContainer,
              borderBottom: `1px solid ${token.colorBorderSecondary}`,
              padding: "0 24px",
              display: "flex",
              alignItems: "center",
              gap: 4,
              height: 40,
              overflowX: "auto",
            }}
          >
            {vaultNavItems.map((item) => {
              const isActive = item.key === activeVaultKey;
              return (
                <div
                  key={item.key}
                  onClick={() => navigate(item.key)}
                  style={{
                    display: "inline-flex",
                    alignItems: "center",
                    gap: 5,
                    padding: "4px 12px",
                    borderRadius: 6,
                    fontSize: 13,
                    fontWeight: isActive ? 500 : 400,
                    cursor: "pointer",
                    whiteSpace: "nowrap",
                    color: isActive ? "#fff" : token.colorText,
                    background: isActive ? token.colorPrimary : "transparent",
                    transition: "all 0.15s",
                  }}
                  onMouseEnter={(e) => {
                    if (!isActive) e.currentTarget.style.background = token.colorFillSecondary;
                  }}
                  onMouseLeave={(e) => {
                    if (!isActive) e.currentTarget.style.background = "transparent";
                  }}
                >
                  <span style={{ fontSize: 13 }}>{item.icon}</span>
                  <span>{item.label}</span>
                </div>
              );
            })}
          </nav>
        )}
      </div>

      <Content
        style={{
          padding: "24px 28px",
          background: token.colorBgLayout,
          minHeight: 280,
          overflow: "auto",
        }}
      >
        <div style={{ maxWidth: 1200, margin: "0 auto" }}>
          <Outlet />
        </div>
      </Content>
    </AntLayout>
  );
}
