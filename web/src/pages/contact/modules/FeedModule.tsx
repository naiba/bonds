import { useState, useCallback } from "react";
import {
  List,
  Typography,
  Tag,
  Empty,
  theme,
  Card,
  Spin,
  Button,
} from "antd";
import {
  HistoryOutlined,
  ClockCircleOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api";
import type { FeedItem, PaginationMeta } from "@/api";
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
  const [page, setPage] = useState(1);
  const [allItems, setAllItems] = useState<FeedItem[]>([]);
  const [hasMore, setHasMore] = useState(true);

  function getActionColor(action: string): string {
    if (action.includes("created")) return "green";
    if (action.includes("updated")) return "blue";
    if (action.includes("deleted")) return "red";
    return "default";
  }

  const { isLoading, isFetching } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "feed", page],
    queryFn: async () => {
      const res = await api.feed.contactsFeedList(String(vaultId), String(contactId), { page, per_page: 15 });
      const newItems = (res.data ?? []) as FeedItem[];
      const meta = res.meta as PaginationMeta | undefined;
      setAllItems(prev => page === 1 ? newItems : [...prev, ...newItems]);
      setHasMore(meta ? meta.page! < meta.total_pages! : newItems.length >= 15);
      return newItems;
    },
  });

  const handleLoadMore = useCallback(() => {
    setPage(p => p + 1);
  }, []);

  if (isLoading && page === 1) {
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
        dataSource={allItems}
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
                    color={getActionColor(item.action ?? '')}
                    style={{ borderRadius: 12, fontSize: 11, margin: 0 }}
                  >
                    {item.action}
                  </Tag>
                  <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
                    <ClockCircleOutlined style={{ fontSize: 11, color: token.colorTextQuaternary }} />
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {dayjs(item.created_at).fromNow()}
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
      {hasMore && allItems.length > 0 && (
        <div style={{ textAlign: "center", marginTop: 12 }}>
          <Button onClick={handleLoadMore} loading={isFetching}>
            {t("common.load_more")}
          </Button>
        </div>
      )}
    </Card>
  );
}
