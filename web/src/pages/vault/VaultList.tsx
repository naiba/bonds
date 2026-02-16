import { useNavigate } from "react-router-dom";
import { Card, Button, Typography, Spin, Empty, Row, Col, App, theme } from "antd";
import { PlusOutlined, SafetyCertificateOutlined } from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";

dayjs.extend(relativeTime);

const { Title, Text, Paragraph } = Typography;

export default function VaultList() {
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const { data, isLoading } = useQuery({
    queryKey: ["vaults"],
    queryFn: async () => {
      const res = await api.vaults.vaultsList();
      return res.data ?? [];
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
          background: `linear-gradient(135deg, ${token.colorPrimaryBg}, ${token.colorPrimaryBgHover})`,
          borderRadius: token.borderRadiusLG,
          padding: "28px 32px",
          marginBottom: 28,
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "flex-start",
          }}
        >
          <div>
            <Title level={3} style={{ margin: 0 }}>
              {t("vault.list.title")}
            </Title>
            <Text
              type="secondary"
              style={{ marginTop: 6, display: "block", fontSize: 14 }}
            >
              {t("vault.list.no_vaults_desc")}
            </Text>
          </div>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            size="large"
            onClick={() => navigate("/vaults/create")}
          >
            {t("vault.list.new_vault")}
          </Button>
        </div>
      </div>

      {vaults.length === 0 ? (
        <Card
          style={{
            textAlign: "center",
            padding: "48px 24px",
            borderStyle: "dashed",
            borderColor: token.colorBorderSecondary,
          }}
        >
          <Empty
            image={
              <div
                style={{
                  width: 88,
                  height: 88,
                  borderRadius: "50%",
                  background: token.colorPrimaryBg,
                  display: "inline-flex",
                  alignItems: "center",
                  justifyContent: "center",
                  marginBottom: 8,
                }}
              >
                <SafetyCertificateOutlined
                  style={{ fontSize: 40, color: token.colorPrimary }}
                />
              </div>
            }
            description={
              <div style={{ marginTop: 8 }}>
                <Text strong style={{ fontSize: 16, display: "block", marginBottom: 6 }}>
                  {t("vault.list.no_vaults")}
                </Text>
                <Text type="secondary">
                  {t("vault.list.no_vaults_desc")}
                </Text>
              </div>
            }
          >
            <Button
              type="primary"
              size="large"
              icon={<PlusOutlined />}
              onClick={() => navigate("/vaults/create")}
              style={{ marginTop: 8 }}
            >
              {t("vault.list.create_vault")}
            </Button>
          </Empty>
        </Card>
      ) : (
        <Row gutter={[20, 20]}>
          {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
          {vaults.map((vault: any) => (
            <Col xs={24} sm={12} lg={8} key={vault.id}>
              <Card
                hoverable
                onClick={() => navigate(`/vaults/${vault.id}`)}
                style={{
                  height: "100%",
                  borderTop: `3px solid ${token.colorPrimary}`,
                  cursor: "pointer",
                }}
                styles={{
                  body: { padding: "24px 24px 20px" },
                }}
              >
                <div style={{ display: "flex", gap: 14, alignItems: "flex-start" }}>
                  <div
                    style={{
                      width: 42,
                      height: 42,
                      borderRadius: "50%",
                      background: token.colorPrimaryBg,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      flexShrink: 0,
                    }}
                  >
                    <SafetyCertificateOutlined
                      style={{ fontSize: 20, color: token.colorPrimary }}
                    />
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <Title level={5} style={{ margin: 0, marginBottom: 4 }}>
                      {vault.name}
                    </Title>
                    {vault.description && (
                      <Paragraph
                        type="secondary"
                        ellipsis={{ rows: 2 }}
                        style={{ marginBottom: 0, fontSize: 13 }}
                      >
                        {vault.description}
                      </Paragraph>
                    )}
                  </div>
                </div>
                <div
                  style={{
                    marginTop: 16,
                    paddingTop: 12,
                    borderTop: `1px solid ${token.colorBorderSecondary}`,
                  }}
                >
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {dayjs(vault.created_at).fromNow()}
                  </Text>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      )}
    </div>
  );
}
