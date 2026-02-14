import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  List,
  Tag,
  Spin,
  Empty,
} from "antd";
import { ArrowLeftOutlined } from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import client from "@/api/client";
import type { APIResponse } from "@/types/api";
import type { FeedItem } from "@/types/modules";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";

dayjs.extend(relativeTime);

const { Title } = Typography;

export default function VaultFeed() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();

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
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}`)}
        style={{ marginBottom: 16 }}
      >
        {t("vault.feed.back")}
      </Button>

      <Title level={4}>{t("vault.feed.title")}</Title>

      <Card>
        <List
          dataSource={feed}
          locale={{ emptyText: <Empty description={t("vault.feed.no_activity")} /> }}
          renderItem={(item: FeedItem) => (
            <List.Item>
              <List.Item.Meta
                title={
                  <>
                    <Tag color="blue">{item.action}</Tag>
                    {item.contact_name && (
                      <a
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
                  </>
                }
                description={
                  <>
                    <div>{item.description}</div>
                    <div style={{ fontSize: 12, opacity: 0.5, marginTop: 4 }}>
                      {dayjs(item.happened_at).fromNow()}
                    </div>
                  </>
                }
              />
            </List.Item>
          )}
        />
      </Card>
    </div>
  );
}
