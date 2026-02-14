import { useNavigate } from "react-router-dom";
import { Card, Button, Typography, Spin, Empty, Row, Col, App } from "antd";
import { PlusOutlined, SafetyCertificateOutlined } from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { vaultsApi } from "@/api/vaults";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title, Text, Paragraph } = Typography;

export default function VaultList() {
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t } = useTranslation();

  const { data, isLoading } = useQuery({
    queryKey: ["vaults"],
    queryFn: async () => {
      const res = await vaultsApi.list();
      return res.data.data ?? [];
    },
    meta: {
      onError: () => message.error(t("vault.list.load_failed")),
    },
  });

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  const vaults = data ?? [];

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 24,
        }}
      >
        <Title level={4} style={{ margin: 0 }}>
          {t("vault.list.title")}
        </Title>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => navigate("/vaults/create")}
        >
          {t("vault.list.new_vault")}
        </Button>
      </div>

      {vaults.length === 0 ? (
        <Card>
          <Empty
            image={<SafetyCertificateOutlined style={{ fontSize: 48, color: "#d9d9d9" }} />}
            description={
              <div>
                <Text strong>{t("vault.list.no_vaults")}</Text>
                <br />
                <Text type="secondary">
                  {t("vault.list.no_vaults_desc")}
                </Text>
              </div>
            }
          >
            <Button type="primary" onClick={() => navigate("/vaults/create")}>
              {t("vault.list.create_vault")}
            </Button>
          </Empty>
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {vaults.map((vault) => (
            <Col xs={24} sm={12} lg={8} key={vault.id}>
              <Card
                hoverable
                onClick={() => navigate(`/vaults/${vault.id}`)}
                style={{ height: "100%" }}
              >
                <Title level={5} style={{ marginBottom: 8 }}>
                  {vault.name}
                </Title>
                {vault.description && (
                  <Paragraph
                    type="secondary"
                    ellipsis={{ rows: 2 }}
                    style={{ marginBottom: 12 }}
                  >
                    {vault.description}
                  </Paragraph>
                )}
                <Text type="secondary" style={{ fontSize: 12 }}>
                  Created {dayjs(vault.created_at).format("MMM D, YYYY")}
                </Text>
              </Card>
            </Col>
          ))}
        </Row>
      )}
    </div>
  );
}
