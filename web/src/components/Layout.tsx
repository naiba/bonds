import { useState } from "react";
import { Outlet, useNavigate, useLocation, useParams } from "react-router-dom";
import { Layout as AntLayout, Menu, Dropdown, Avatar, theme, Button } from "antd";
import {
  SafetyCertificateOutlined,
  SettingOutlined,
  TeamOutlined,
  LogoutOutlined,
  UserOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  BookOutlined,
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
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { useAuth } from "@/stores/auth";
import { useTranslation } from "react-i18next";
import SearchBar from "@/components/SearchBar";

const { Sider, Header, Content } = AntLayout;

export default function Layout() {
  const [collapsed, setCollapsed] = useState(false);
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const { id: vaultId } = useParams();
  const { token } = theme.useToken();
  const { t, i18n } = useTranslation();

  const isInVault = location.pathname.match(/^\/vaults\/\d+/);

  const settingsMenuItems: MenuProps["items"] = [
    {
      key: "/settings",
      icon: <SettingOutlined />,
      label: t("nav.account"),
    },
    {
      key: "/settings/preferences",
      icon: <ControlOutlined />,
      label: t("nav.preferences"),
    },
    {
      key: "/settings/notifications",
      icon: <BellOutlined />,
      label: t("nav.notifications"),
    },
    {
      key: "/settings/personalize",
      icon: <EditOutlined />,
      label: t("nav.personalize"),
    },
    {
      key: "/settings/users",
      icon: <UserSwitchOutlined />,
      label: t("nav.users"),
    },
    {
      key: "/settings/2fa",
      icon: <LockOutlined />,
      label: t("nav.twoFactor"),
    },
    {
      key: "/settings/invitations",
      icon: <MailOutlined />,
      label: t("nav.invitations"),
    },
  ];

  const topMenuItems: MenuProps["items"] = [
    {
      key: "/vaults",
      icon: <SafetyCertificateOutlined />,
      label: t("nav.vaults"),
    },
    {
      type: "group" as const,
      label: t("nav.settings"),
      children: settingsMenuItems,
    },
  ];

  const vaultMenuItems: MenuProps["items"] = vaultId
    ? [
        {
          key: `/vaults/${vaultId}/contacts`,
          icon: <TeamOutlined />,
          label: t("nav.contacts"),
        },
        {
          key: `/vaults/${vaultId}/journals`,
          icon: <EditOutlined />,
          label: t("nav.journal"),
        },
        {
          key: `/vaults/${vaultId}/groups`,
          icon: <UsergroupAddOutlined />,
          label: t("nav.groups"),
        },
        {
          key: `/vaults/${vaultId}/calendar`,
          icon: <CalendarOutlined />,
          label: t("nav.calendar"),
        },
        {
          key: `/vaults/${vaultId}/tasks`,
          icon: <BookOutlined />,
          label: t("nav.tasks"),
        },
        {
          key: `/vaults/${vaultId}/reports`,
          icon: <BarChartOutlined />,
          label: t("nav.reports"),
        },
        {
          key: `/vaults/${vaultId}/files`,
          icon: <FileOutlined />,
          label: t("nav.files"),
        },
        {
          key: `/vaults/${vaultId}/feed`,
          icon: <UnorderedListOutlined />,
          label: t("nav.feed"),
        },
      ]
    : [];

  const menuItems = isInVault
    ? [
        { type: "group" as const, label: t("nav.vault"), children: vaultMenuItems },
        { type: "divider" as const },
        ...topMenuItems,
      ]
    : topMenuItems;

  const selectedKey =
    menuItems
      .flatMap((item) => {
        if (item && "children" in item && item.children) {
          return item.children;
        }
        return [item];
      })
      .filter(Boolean)
      .map((item) => (item && "key" in item ? (item.key as string) : ""))
      .find((key) => key && location.pathname.startsWith(key)) ?? "/vaults";

  const userMenuItems: MenuProps["items"] = [
    {
      key: "language",
      icon: <GlobalOutlined />,
      label: i18n.language?.startsWith("zh") ? "English" : "中文",
      onClick: () => {
        const next = i18n.language?.startsWith("zh") ? "en" : "zh";
        i18n.changeLanguage(next);
      },
    },
    { type: "divider" as const },
    {
      key: "logout",
      icon: <LogoutOutlined />,
      label: t("nav.logout"),
      onClick: () => {
        logout();
        navigate("/login");
      },
    },
  ];

  const initials = user
    ? `${user.first_name.charAt(0)}${user.last_name.charAt(0)}`.toUpperCase()
    : "";

  return (
    <AntLayout style={{ minHeight: "100vh" }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={setCollapsed}
        trigger={null}
        breakpoint="lg"
        collapsedWidth={64}
        style={{
          borderRight: `1px solid ${token.colorBorderSecondary}`,
          background: token.colorBgContainer,
        }}
        width={220}
      >
        <div
          style={{
            height: 56,
            display: "flex",
            alignItems: "center",
            justifyContent: collapsed ? "center" : "flex-start",
            padding: collapsed ? 0 : "0 20px",
            fontWeight: 700,
            fontSize: 18,
            letterSpacing: "-0.02em",
            color: token.colorText,
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          {collapsed ? "B" : "Bonds"}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{
            border: "none",
            background: "transparent",
            padding: "8px 0",
          }}
        />
      </Sider>
      <AntLayout>
        <Header
          style={{
            background: token.colorBgContainer,
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
            padding: "0 24px",
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            height: 56,
            lineHeight: "56px",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <Button
              type="text"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={() => setCollapsed(!collapsed)}
            />
            <SearchBar />
          </div>
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <div
              style={{
                cursor: "pointer",
                display: "flex",
                alignItems: "center",
                gap: 8,
              }}
            >
              <span style={{ color: token.colorTextSecondary, fontSize: 14 }}>
                {user?.first_name} {user?.last_name}
              </span>
              <Avatar
                size={32}
                icon={<UserOutlined />}
                style={{
                  backgroundColor: token.colorPrimary,
                  fontSize: 13,
                }}
              >
                {initials}
              </Avatar>
            </div>
          </Dropdown>
        </Header>
        <Content
          style={{
            padding: 24,
            background: token.colorBgLayout,
            minHeight: 280,
            overflow: "auto",
          }}
        >
          <Outlet />
        </Content>
      </AntLayout>
    </AntLayout>
  );
}
