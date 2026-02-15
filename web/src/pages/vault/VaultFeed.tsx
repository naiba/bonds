import { useParams, useNavigate } from "react-router-dom";
import {
  Typography,
  Button,
  List,
  Tag,
  Spin,
  Empty,
  theme,
} from "antd";
import {
  ArrowLeftOutlined,
  HistoryOutlined,
  ClockCircleOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import client from "@/api/client";
import type { APIResponse } from "@/types/api";
import type { FeedItem } from "@/types/modules";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";

dayjs.extend(relativeTime);

const { Title, Text } = Typography;

export default function VaultFeed() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  function getActionColor(action: string): string {
    if (action.includes("created")) return "green";
    if (action.includes("updated")) return "blue";
    if (action.includes("deleted")) return "red";
    return "default";
  }

  const { data: feed = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "feed"],
    queryFn: async () => {
      const res = await client.get<APIResponse<FeedItem[]>>(
        `/vaults/${vaultId}/feed`,
      );
      return res.data.data ?? [];
    },
    enabled: !!vaultId,
  });

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 24 }}>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <HistoryOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0 }}>{t("vault.feed.title")}</Title>
      </div>

      <div
        style={{
          background: token.colorBgContainer,
          borderRadius: token.borderRadiusLG,
          boxShadow: token.boxShadowTertiary,
          padding: "8px 0",
        }}
      >
        <List
          dataSource={feed}
          locale={{ emptyText: <Empty description={t("vault.feed.no_activity")} style={{ padding: 32 }} /> }}
          renderItem={(item: FeedItem, index: number) => (
            <List.Item
              style={{
                margin: "0 16px",
                paddingLeft: 20,
                borderLeft: `2px solid ${index === 0 ? token.colorPrimary : token.colorBorderSecondary}`,
                position: "relative",
              }}
            >
              <div
                style={{
                  position: "absolute",
                  left: -5,
                  top: 18,
                  width: 8,
                  height: 8,
                  borderRadius: "50%",
                  background: index === 0 ? token.colorPrimary : token.colorBorder,
                }}
              />
              <List.Item.Meta
                title={
                  <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
                    <Tag
                      color={getActionColor(item.action)}
                      style={{ borderRadius: 12, fontSize: 11, margin: 0 }}
                    >
                      {item.action}
                    </Tag>
                    {item.contact_name && (
                      <a
                        style={{ fontWeight: 600 }}
                        onClick={() =>
                          item.contact_id &&
                          navigate(
                            `/vaults/${vaultId}/contacts/${item.contact_id}`,
                          )
                        }
                      >
                        {item.contact_name}
                      </a>
                    )}
                  </div>
                }
                description={
                  <>
                    {item.description && (
                      <Text type="secondary" style={{ display: "block", marginTop: 4 }}>
                        {item.description}
                      </Text>
                    )}
                    <div style={{ display: "flex", alignItems: "center", gap: 4, marginTop: 6 }}>
                      <ClockCircleOutlined style={{ fontSize: 11, color: token.colorTextQuaternary }} />
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {dayjs(item.happened_at).fromNow()}
                      </Text>
                    </div>
                  </>
                }
              />
            </List.Item>
          )}
        />
      </div>
    </div>
  );
}
