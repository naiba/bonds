import {
  List,
  Typography,
  Tag,
  Empty,
  theme,
  Card,
  Spin,
} from "antd";
import {
  HistoryOutlined,
  ClockCircleOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { contactExtraApi } from "@/api/contactExtra";
import type { FeedItem } from "@/types/modules";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";

dayjs.extend(relativeTime);

const { Title, Text } = Typography;

interface FeedModuleProps {
  vaultId: string;
  contactId: string;
}

export default function FeedModule({ vaultId, contactId }: FeedModuleProps) {
  const { t } = useTranslation();
  const { token } = theme.useToken();

  function getActionColor(action: string): string {
    if (action.includes("created")) return "green";
    if (action.includes("updated")) return "blue";
    if (action.includes("deleted")) return "red";
    return "default";
  }

  const { data: feed = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "feed"],
    queryFn: async () => {
      const res = await contactExtraApi.getFeed(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 40 }}>
        <Spin />
      </div>
    );
  }

  return (
    <Card
      title={
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <HistoryOutlined style={{ color: token.colorPrimary }} />
          <Title level={5} style={{ margin: 0 }}>
            {t("contact.detail.feed.title")}
          </Title>
        </div>
      }
    >
      <List
        dataSource={feed}
        locale={{ emptyText: <Empty description={t("contact.detail.feed.no_activity")} /> }}
        renderItem={(item: FeedItem, index: number) => (
          <List.Item
            style={{
              paddingLeft: 20,
              borderLeft: `2px solid ${index === 0 ? token.colorPrimary : token.colorBorderSecondary}`,
              position: "relative",
              paddingBottom: 24,
            }}
          >
            <div
              style={{
                position: "absolute",
                left: -5,
                top: 22,
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
                  <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
                    <ClockCircleOutlined style={{ fontSize: 11, color: token.colorTextQuaternary }} />
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {dayjs(item.happened_at).fromNow()}
                    </Text>
                  </div>
                </div>
              }
              description={
                item.description && (
                  <Text style={{ display: "block", marginTop: 4 }}>
                    {item.description}
                  </Text>
                )
              }
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
