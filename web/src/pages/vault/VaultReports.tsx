import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  Row,
  Col,
  Statistic,
  Empty,
  theme,
} from "antd";
import {
  ArrowLeftOutlined,
  TeamOutlined,
  EnvironmentOutlined,
  CalendarOutlined,
  SmileOutlined,
  BarChartOutlined,
} from "@ant-design/icons";

import { useTranslation } from "react-i18next";

const { Title } = Typography;

export default function VaultReports() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const statCards: { icon: React.ReactNode; bg: string; color: string; title: string; value: number }[] = [
    { icon: <TeamOutlined />, bg: token.colorPrimaryBg, color: token.colorPrimary, title: t("vault.reports.contacts"), value: 0 },
    { icon: <EnvironmentOutlined />, bg: "#fff7e6", color: "#fa8c16", title: t("vault.reports.addresses"), value: 0 },
    { icon: <CalendarOutlined />, bg: "#e6f4ff", color: "#1677ff", title: t("vault.reports.important_dates"), value: 0 },
    { icon: <SmileOutlined />, bg: "#f6ffed", color: "#52c41a", title: t("vault.reports.mood_entries"), value: 0 },
  ];

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 24 }}>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <BarChartOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0 }}>{t("vault.reports.title")}</Title>
      </div>

      <Row gutter={[16, 16]}>
        {statCards.map((s, i) => (
          <Col xs={12} sm={6} key={i}>
            <Card
              style={{
                boxShadow: token.boxShadowTertiary,
                borderRadius: token.borderRadiusLG,
              }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 12 }}>
                <div
                  style={{
                    width: 40,
                    height: 40,
                    borderRadius: "50%",
                    background: s.bg,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    fontSize: 18,
                    color: s.color,
                  }}
                >
                  {s.icon}
                </div>
              </div>
              <Statistic title={s.title} value={s.value} />
            </Card>
          </Col>
        ))}
      </Row>

      <Card
        title={t("vault.reports.address_distribution")}
        style={{ marginTop: 24, boxShadow: token.boxShadowTertiary, borderRadius: token.borderRadiusLG }}
      >
        <Empty
          description={t("vault.reports.no_address_data")}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </Card>

      <Card
        title={t("vault.reports.important_dates_overview")}
        style={{ marginTop: 16, boxShadow: token.boxShadowTertiary, borderRadius: token.borderRadiusLG }}
      >
        <Empty
          description={t("vault.reports.no_date_data")}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </Card>

      <Card
        title={t("vault.reports.mood_trends")}
        style={{ marginTop: 16, boxShadow: token.boxShadowTertiary, borderRadius: token.borderRadiusLG }}
      >
        <Empty
          description={t("vault.reports.no_mood_data")}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </Card>
    </div>
  );
}
