import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  Row,
  Col,
  Statistic,
  Empty,
} from "antd";
import {
  ArrowLeftOutlined,
  TeamOutlined,
  EnvironmentOutlined,
  CalendarOutlined,
  SmileOutlined,
} from "@ant-design/icons";

import { useTranslation } from "react-i18next";

const { Title } = Typography;

export default function VaultReports() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}`)}
        style={{ marginBottom: 16 }}
      >
        {t("vault.reports.back")}
      </Button>

      <Title level={4}>{t("vault.reports.title")}</Title>

      <Row gutter={[16, 16]}>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic title={t("vault.reports.contacts")} value={0} prefix={<TeamOutlined />} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic title={t("vault.reports.addresses")} value={0} prefix={<EnvironmentOutlined />} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic title={t("vault.reports.important_dates")} value={0} prefix={<CalendarOutlined />} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic title={t("vault.reports.mood_entries")} value={0} prefix={<SmileOutlined />} />
          </Card>
        </Col>
      </Row>

      <Card title={t("vault.reports.address_distribution")} style={{ marginTop: 24 }}>
        <Empty description={t("vault.reports.no_address_data")} />
      </Card>

      <Card title={t("vault.reports.important_dates_overview")} style={{ marginTop: 16 }}>
        <Empty description={t("vault.reports.no_date_data")} />
      </Card>

      <Card title={t("vault.reports.mood_trends")} style={{ marginTop: 16 }}>
        <Empty description={t("vault.reports.no_mood_data")} />
      </Card>
    </div>
  );
}
