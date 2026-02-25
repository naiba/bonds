import { Card, Typography, Progress, Spin, Statistic, Row, Col, theme } from "antd";
import { DatabaseOutlined, HddOutlined } from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import { filesize } from "filesize";

const { Title, Text } = Typography;

export default function StorageInfo() {
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const { data: usage, isLoading } = useQuery({
    queryKey: ["settings", "storage"],
    queryFn: async () => {
      const res = await api.settings.storageList();
      return res.data;
    },
  });

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!usage) return null;

  const percent = Math.round((usage.used_bytes / usage.limit_bytes) * 100);
  const used = filesize(usage.used_bytes) as string;
  const limit = filesize(usage.limit_bytes) as string;
  const remaining = filesize(usage.limit_bytes - usage.used_bytes) as string;

  return (
    <div style={{ maxWidth: 640, margin: "0 auto" }}>
      <Title level={4} style={{ marginBottom: 4 }}>
        {t("settings.storage.title")}
      </Title>
      <Text type="secondary" style={{ display: "block", marginBottom: 24 }}>
        {t("settings.storage.description")}
      </Text>

      <Card>
        <div style={{ textAlign: "center", marginBottom: 32 }}>
          <Progress
            type="dashboard"
            percent={percent}
            status={percent > 90 ? "exception" : "normal"}
            strokeWidth={10}
            size={200}
          />
          <div style={{ marginTop: 16 }}>
            <Title level={3} style={{ margin: 0 }}>
              {used} / {limit}
            </Title>
            <Text type="secondary">{t("settings.storage.used")}</Text>
          </div>
        </div>

        <Row gutter={16}>
          <Col span={12}>
            <Card style={{ background: token.colorBgLayout }}>
              <Statistic
                title={t("settings.storage.remaining")}
                value={remaining}
                prefix={<HddOutlined />}
              />
            </Card>
          </Col>
          <Col span={12}>
            <Card style={{ background: token.colorBgLayout }}>
              <Statistic
                title={t("settings.storage.usage_percent")}
                value={percent}
                suffix="%"
                prefix={<DatabaseOutlined />}
                valueStyle={{ color: percent > 90 ? "#cf1322" : "inherit" }}
              />
            </Card>
          </Col>
        </Row>
      </Card>
    </div>
  );
}
