import { useParams, useNavigate, Outlet } from "react-router-dom";
import { Card, Typography, Spin, Statistic, Row, Col, Button, Descriptions, theme } from "antd";
import { TeamOutlined, PlusOutlined } from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title } = Typography;

export default function VaultDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const vaultId = id!;
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const { data: vault, isLoading: vaultLoading } = useQuery({
    queryKey: ["vaults", vaultId],
    queryFn: async () => {
      const res = await api.vaults.vaultsDetail(String(vaultId));
      return res.data!;
    },
    enabled: !!vaultId,
  });

  const { data: contacts } = useQuery({
    queryKey: ["vaults", vaultId, "contacts"],
    queryFn: async () => {
      const res = await api.contacts.contactsList(String(vaultId));
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  if (vaultLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!vault) return null;

  const contactCount = contacts?.length ?? 0;
  const recentContacts = (contacts ?? []).slice(0, 5);

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
          {vault.name}
        </Title>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}/contacts/create`)}
        >
          {t("vault.detail.add_contact")}
        </Button>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card title={t("vault.detail.overview")} style={{ marginBottom: 16 }}>
            <Descriptions column={1}>
              {vault.description && (
                <Descriptions.Item label={t("vault.detail.description")}>
                  {vault.description}
                </Descriptions.Item>
              )}
              <Descriptions.Item label={t("common.created")}>
                {dayjs(vault.created_at).format("MMMM D, YYYY")}
              </Descriptions.Item>
              <Descriptions.Item label={t("common.last_updated")}>
                {dayjs(vault.updated_at).format("MMMM D, YYYY")}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title={t("vault.detail.recent_contacts")}>
            {recentContacts.length === 0 ? (
              <div style={{ textAlign: "center", padding: 24 }}>
                <TeamOutlined
                  style={{ fontSize: 32, color: token.colorTextQuaternary, marginBottom: 8 }}
                />
                <div style={{ color: token.colorTextSecondary }}>{t("vault.detail.no_contacts")}</div>
              </div>
            ) : (
              <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {recentContacts.map((contact: any) => (
                  <Card
                    key={contact.id}
                    size="small"
                    hoverable
                    onClick={() =>
                      navigate(
                        `/vaults/${vaultId}/contacts/${contact.id}`,
                      )
                    }
                    style={{ cursor: "pointer" }}
                  >
                    <div
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                      }}
                    >
                      <span>
                        {contact.first_name} {contact.last_name}
                      </span>
                      <span style={{ color: token.colorTextSecondary, fontSize: 12 }}>
                        {dayjs(contact.updated_at).format("MMM D")}
                      </span>
                    </div>
                  </Card>
                ))}
              </div>
            )}
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card>
            <Statistic
              title={t("vault.detail.total_contacts")}
              value={contactCount}
              prefix={<TeamOutlined />}
            />
            <Button
              type="link"
              style={{ padding: 0, marginTop: 8 }}
              onClick={() => navigate(`/vaults/${vaultId}/contacts`)}
            >
              {t("vault.detail.view_all_contacts")}
            </Button>
          </Card>
        </Col>
      </Row>

      <Outlet />
    </div>
  );
}
