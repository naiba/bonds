import { useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import {
  Card,
  Typography,
  List,
  Button,
  Popconfirm,
  App,
  Empty,
  Spin,
  Avatar,
  Tag,
  Dropdown,
} from "antd";
import {
  DisconnectOutlined,
  GithubOutlined,
  GoogleOutlined,
  LinkOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { OAuthProvider, APIError, InstanceInfo } from "@/api";
import { useDateFormat, formatDate } from "@/utils/dateFormat";

const { Title, Text } = Typography;

export default function OAuthProviders() {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const dateFormats = useDateFormat();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();

  useEffect(() => {
    const linked = searchParams.get("linked");
    const error = searchParams.get("error");
    if (linked) {
      message.success(t("settings.oauth.link_linked"));
      setSearchParams({}, { replace: true });
    } else if (error) {
      message.error(error);
      setSearchParams({}, { replace: true });
    }
  }, [searchParams, setSearchParams, message, t]);

  const { data: providers = [], isLoading } = useQuery({
    queryKey: ["settings", "oauth"],
    queryFn: async () => {
      const res = await api.oauth.oauthList();
      return res.data ?? [];
    },
  });

  const { data: instanceInfo } = useQuery({
    queryKey: ["instance-info"],
    queryFn: async () => {
      const res = await api.instance.infoList();
      return (res.data ?? null) as InstanceInfo | null;
    },
  });

  const availableProviders = instanceInfo?.oauth_providers ?? [];

  const handleLinkProvider = (provider: string) => {
    const jwt = localStorage.getItem("token") ?? "";
    const state = crypto.randomUUID();
    window.location.assign(
      `/api/auth/${provider}?mode=link&token=${encodeURIComponent(jwt)}&state=${encodeURIComponent(state)}`
    );
  };

  const unlinkMutation = useMutation({
    mutationFn: (driver: string) => api.oauth.oauthDelete(driver),
    onSuccess: () => {
      message.success(t("settings.oauth.unlinked"));
      queryClient.invalidateQueries({ queryKey: ["settings", "oauth"] });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const getIcon = (driver: string) => {
    switch (driver) {
      case "github":
        return <GithubOutlined />;
      case "google":
        return <GoogleOutlined />;
      default:
        return <LinkOutlined />;
    }
  };

  const getDisplayName = (driver: string) => {
    switch (driver) {
      case "github": return "GitHub";
      case "google": return "Google";
      case "openid-connect": return "SSO";
      default: return driver;
    }
  };

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 4 }}>
        <Title level={4} style={{ marginBottom: 0 }}>
          {t("settings.oauth.title")}
        </Title>
        {availableProviders.length === 1 ? (
          <Button
            icon={<PlusOutlined />}
            onClick={() => handleLinkProvider(availableProviders[0])}
          >
            {t("settings.oauth.link_new")}
          </Button>
        ) : availableProviders.length > 1 ? (
          <Dropdown
            menu={{
              items: availableProviders.map((p) => ({
                key: p,
                icon: getIcon(p),
                label: getDisplayName(p),
                onClick: () => handleLinkProvider(p),
              })),
            }}
          >
            <Button icon={<PlusOutlined />}>
              {t("settings.oauth.link_new")}
            </Button>
          </Dropdown>
        ) : null}
      </div>
      <Text type="secondary" style={{ display: "block", marginBottom: 24 }}>
        {t("settings.oauth.description")}
      </Text>

      <Card>
        {isLoading ? (
          <Spin />
        ) : providers.length === 0 ? (
          <Empty description={t("settings.oauth.no_providers")} />
        ) : (
          <List
            dataSource={providers}
            renderItem={(item: OAuthProvider) => (
              <List.Item
                actions={[
                  <Popconfirm
                    key="unlink"
                    title={t("settings.oauth.unlink_confirm")}
                    onConfirm={() => unlinkMutation.mutate(item.driver)}
                  >
                    <Button danger icon={<DisconnectOutlined />}>
                      {t("settings.oauth.unlink")}
                    </Button>
                  </Popconfirm>,
                ]}
              >
                <List.Item.Meta
                  avatar={
                    item.avatar_url ? (
                      <Avatar src={item.avatar_url} />
                    ) : (
                      <Avatar icon={getIcon(item.driver)} />
                    )
                  }
                  title={
                    <span>
                      {getDisplayName(item.driver)}
                    </span>
                  }
                  description={
                    <div
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: 8,
                        flexWrap: "wrap",
                      }}
                    >
                      <Text strong>{item.name}</Text>
                      {item.id && <Tag>{item.id}</Tag>}
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {t("settings.oauth.linked_at")}{" "}
                        {formatDate(item.created_at, dateFormats)}
                      </Text>
                    </div>
                  }
                />
              </List.Item>
            )}
          />
        )}
      </Card>
    </div>
  );
}
