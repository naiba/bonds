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
} from "antd";
import {
  DisconnectOutlined,
  GithubOutlined,
  GoogleOutlined,
  LinkOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { OAuthProvider, APIError } from "@/api";
import dayjs from "dayjs";

const { Title, Text } = Typography;

export default function OAuthProviders() {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  const { data: providers = [], isLoading } = useQuery({
    queryKey: ["settings", "oauth"],
    queryFn: async () => {
      const res = await api.oauth.oauthList();
      return res.data ?? [];
    },
  });

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
      <Title level={4} style={{ marginBottom: 4 }}>
        {t("settings.oauth.title")}
      </Title>
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
                        {dayjs(item.created_at).format("MMM D, YYYY")}
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
