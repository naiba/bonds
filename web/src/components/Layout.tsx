import React from "react";
import { Outlet, useNavigate, useLocation, useParams } from "react-router-dom";
import { formatContactName, formatContactInitials, useNameOrder } from "@/utils/nameFormat";
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
  CrownOutlined,
  CloudServerOutlined,
  LinkOutlined,
  HeartOutlined,
  KeyOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { useAuth } from "@/stores/auth";
import { useTheme } from "@/stores/theme";
import type { ThemeMode } from "@/stores/theme";
import { useTranslation } from "react-i18next";
import SearchBar from "@/components/SearchBar";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api";

const { Header, Content } = AntLayout;

export default function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const { id: vaultId } = useParams();
  const { token } = theme.useToken();
  const nameOrder = useNameOrder();
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
        { key: `/vaults/${vaultId}/reminders`, icon: <BellOutlined />, label: t("nav.reminders") },
        { key: `/vaults/${vaultId}/life-metrics`, icon: <HeartOutlined />, label: t("nav.lifeMetrics") },
        { key: `/vaults/${vaultId}/dav-subscriptions`, icon: <CloudServerOutlined />, label: t("nav.davSubscriptions") },
      ]
    : [];

  // Grouped nav: Core | Content | Management | Activity
  // Groups separated by thin dividers for visual hierarchy
  const vaultNavGroups: { key: string; icon: React.ReactNode; label: string }[][] = vaultId
    ? [
        // Core
        [
          { key: `/vaults/${vaultId}`, icon: <DashboardOutlined />, label: t("nav.dashboard") },
          { key: `/vaults/${vaultId}/contacts`, icon: <TeamOutlined />, label: t("nav.contacts") },
        ],
        // Content
        [
          { key: `/vaults/${vaultId}/journals`, icon: <EditOutlined />, label: t("nav.journal") },
          { key: `/vaults/${vaultId}/groups`, icon: <UsergroupAddOutlined />, label: t("nav.groups") },
          { key: `/vaults/${vaultId}/calendar`, icon: <CalendarOutlined />, label: t("nav.calendar") },
        ],
        // Management
        [
          { key: `/vaults/${vaultId}/tasks`, icon: <CheckSquareOutlined />, label: t("nav.tasks") },
          { key: `/vaults/${vaultId}/reports`, icon: <BarChartOutlined />, label: t("nav.reports") },
          { key: `/vaults/${vaultId}/files`, icon: <FileOutlined />, label: t("nav.files") },
        ],
        // Activity
        [
          { key: `/vaults/${vaultId}/feed`, icon: <UnorderedListOutlined />, label: t("nav.feed") },
          { key: `/vaults/${vaultId}/reminders`, icon: <BellOutlined />, label: t("nav.reminders") },
          { key: `/vaults/${vaultId}/life-metrics`, icon: <HeartOutlined />, label: t("nav.lifeMetrics") },
          { key: `/vaults/${vaultId}/dav-subscriptions`, icon: <CloudServerOutlined />, label: t("nav.davSubscriptions") },
        ],
      ]
    : [];

  const activeVaultKey = vaultNavItems
    .slice()
    .sort((a, b) => b.key.length - a.key.length)
    .find((item) => location.pathname.startsWith(item.key))?.key ?? "";

  
  const { data: currentVault } = useQuery({
    queryKey: ["vaults", vaultId],
    queryFn: async () => {
      const res = await api.vaults.vaultsDetail(String(vaultId));
      return res.data;
    },
    enabled: !!vaultId,
  });

  const userMenuItems: MenuProps["items"] = [
    { key: "/settings", icon: <SettingOutlined />, label: t("nav.account") },
    { key: "/settings/preferences", icon: <ControlOutlined />, label: t("nav.preferences") },
    { key: "/settings/notifications", icon: <BellOutlined />, label: t("nav.notifications") },
    { key: "/settings/personalize", icon: <EditOutlined />, label: t("nav.personalize") },
    { key: "/settings/users", icon: <UserSwitchOutlined />, label: t("nav.users") },
    { key: "/settings/2fa", icon: <LockOutlined />, label: t("nav.twoFactor") },
    { key: "/settings/invitations", icon: <MailOutlined />, label: t("nav.invitations") },
    { key: "/settings/webauthn", icon: <LockOutlined />, label: t("nav.webauthn") },
    { key: "/settings/oauth", icon: <LinkOutlined />, label: t("nav.oauth") },
    { key: "/settings/storage", icon: <CloudServerOutlined />, label: t("nav.storage") },
    { key: "/settings/tokens", icon: <KeyOutlined />, label: t("nav.api_tokens") },
    ...(user?.is_instance_administrator
      ? [
          { type: "divider" as const },
          { key: "/admin/users", icon: <CrownOutlined />, label: t("nav.admin") },
        ]
      : []),
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

  const initials = user ? formatContactInitials(nameOrder, user) : "";

  return (
    <AntLayout style={{ minHeight: "100vh" }}>
      <div style={{ position: "sticky", top: 0, zIndex: 100 }}>
        <Header
          style={{
            background: token.Layout?.headerBg || token.colorBgContainer,
            borderBottom: isInVault ? "none" : `1px solid ${token.colorBorderSecondary}`,
            padding: "0 16px",
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            height: 56,
            lineHeight: "56px",
            /* Fix: prevent header from exceeding viewport width on mobile */
            maxWidth: "100vw",
            overflow: "hidden",
          }}
        >
          {/* Left side: breadcrumb + search — allow shrinking on mobile */}
          <div style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 0, flex: 1 }}>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 4,
                cursor: "pointer",
                userSelect: "none",
                padding: "4px 8px",
                borderRadius: token.borderRadiusSM,
                fontSize: 13,
                transition: "background 0.2s",
                flexShrink: 1,
                minWidth: 0,
              }}
              className="nav-breadcrumb-trigger"
              onClick={() => navigate("/vaults")}
            >
              <span style={{ fontWeight: 600, color: token.colorText, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", flexShrink: 1, maxWidth: 120 }}>
                {formatContactName(nameOrder, user ?? {})}
              </span>
              {isInVault && vaultId && (
                <>
                  <span style={{ color: token.colorTextQuaternary, fontSize: 11, margin: "0 2px", fontWeight: 400 }}>/</span>
                  <span style={{ color: token.colorTextSecondary, fontWeight: 500, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", flex: 1, minWidth: 0, maxWidth: "max-content" }}>
                    {currentVault?.name || t("nav.vault")}
                  </span>
                </>
              )}
            </div>
            <SearchBar />
          </div>

          {/* Right side: actions — never shrink */}
          <div style={{ display: "flex", alignItems: "center", gap: 4, flexShrink: 0 }}>
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
              background: token.Layout?.headerBg || token.colorBgContainer,
              borderBottom: `1px solid ${token.colorBorderSecondary}`,
              padding: "0 16px",
              display: "flex",
              alignItems: "center",
              gap: 2,
              height: 52,
              overflowX: "auto",
            }}
          >
            {vaultNavGroups.map((group, groupIdx) => (
              <React.Fragment key={groupIdx}>
                {groupIdx > 0 && (
                  <div
                    className="vault-nav-divider"
                    style={{ background: token.colorBorderSecondary }}
                  />
                )}
                {group.map((item) => {
                  const isActive = item.key === activeVaultKey;
                  return (
                    <div
                      key={item.key}
                      className={`vault-nav-pill${isActive ? " vault-nav-pill--active" : ""}`}
                      style={!isActive ? { color: token.colorText } : undefined}
                      onClick={() => navigate(item.key)}
                    >
                      <span>{item.icon}</span>
                      <span>{item.label}</span>
                    </div>
                  );
                })}
              </React.Fragment>
            ))}
          </nav>
        )}
      </div>

      <Content
        style={{
          padding: "24px 16px",
          background: token.colorBgLayout,
          minHeight: 280,
          overflow: "auto",
        }}
      >
        <div style={{ maxWidth: 1200, margin: "0 auto" }}>
          <Outlet />
        </div>
      </Content>

      <div
        style={{
          textAlign: "center",
          padding: "16px 24px",
          background: token.colorBgLayout,
          color: token.colorTextQuaternary,
          fontSize: 12,
        }}
      >
        © {new Date().getFullYear()}{" "}
        <a href="https://github.com/naiba/bonds" target="_blank" rel="noopener noreferrer" style={{ color: token.colorTextTertiary }}>Bonds</a>
        {` ${__APP_VERSION__}`}
        {" by "}
        <a href="https://nai.ba" target="_blank" rel="noopener noreferrer" style={{ color: token.colorTextTertiary }}>naiba</a>
      </div>
    </AntLayout>
  );
}
