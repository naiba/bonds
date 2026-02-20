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
  Table,
  Tag,
} from "antd";
import {
  ArrowLeftOutlined,
  TeamOutlined,
  EnvironmentOutlined,
  CalendarOutlined,
  SmileOutlined,
  BarChartOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { 
  AddressReportItem, 
  ImportantDateReportItem, 
  MoodReportItem,
  AddressContactItem 
} from "@/api";

const { Title, Text } = Typography;

export default function VaultReports() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  // Queries
  const { data: reportOverview } = useQuery({
    queryKey: ["vault", vaultId, "reports", "overview"],
    queryFn: async () => {
      const res = await api.reports.reportsOverviewList(String(vaultId));
      return res.data;
    },
    enabled: !!vaultId,
  });

  const { data: addresses = [] } = useQuery({
    queryKey: ["vault", vaultId, "reports", "addresses"],
    queryFn: async () => {
      const res = await api.reports.reportsAddressesList(vaultId);
      return (res.data ?? []) as AddressReportItem[];
    },
  });

  const { data: importantDates = [] } = useQuery({
    queryKey: ["vault", vaultId, "reports", "importantDates"],
    queryFn: async () => {
      const res = await api.reports.reportsImportantDatesList(vaultId);
      return (res.data ?? []) as ImportantDateReportItem[];
    },
  });

  const { data: moodEntries = [] } = useQuery({
    queryKey: ["vault", vaultId, "reports", "mood"],
    queryFn: async () => {
      const res = await api.reports.reportsMoodTrackingEventsList(vaultId);
      return (res.data ?? []) as MoodReportItem[];
    },
  });

  const totalMoodEntries = moodEntries.reduce((acc, curr) => acc + (curr.count || 0), 0);

  const statCards = [
    { icon: <TeamOutlined />, bg: token.colorPrimaryBg, color: token.colorPrimary, title: t("vault.reports.total_contacts"), value: reportOverview?.total_contacts ?? 0 },
    { icon: <EnvironmentOutlined />, bg: "#fff7e6", color: "#fa8c16", title: t("vault.reports.total_addresses"), value: reportOverview?.total_addresses ?? 0 },
    { icon: <CalendarOutlined />, bg: "#e6f4ff", color: "#1677ff", title: t("vault.reports.total_dates"), value: reportOverview?.total_important_dates ?? 0 },
    { icon: <SmileOutlined />, bg: "#f6ffed", color: "#52c41a", title: t("vault.reports.mood_entries"), value: reportOverview?.total_mood_entries ?? 0 },
  ];

  const AddressDrillDown = ({ record }: { record: AddressReportItem }) => {
    const { data: details = [], isLoading } = useQuery({
      queryKey: ["vault", vaultId, "reports", "addresses", "detail", record.country, record.city],
      queryFn: async () => {
        if (record.city) {
          const res = await api.reports.reportsAddressesCityDetail(vaultId, record.city);
          return (res.data ?? []) as AddressContactItem[];
        } else if (record.country) {
          const res = await api.reports.reportsAddressesCountryDetail(vaultId, record.country);
          return (res.data ?? []) as AddressContactItem[];
        }
        return [];
      },
      enabled: !!(record.country || record.city),
    });

    return (
      <Card 
        size="small" 
        title={t("vault.reports.contacts_in", { location: record.city || record.country })}
        style={{ margin: 16 }}
      >
        <Table
          dataSource={details}
          loading={isLoading}
          rowKey="contact_id"
          pagination={false}
          size="small"
          columns={[
            {
              title: t("vault.reports.col_contact"),
              key: "name",
              render: (_, item) => (
                <a onClick={() => navigate(`/vaults/${vaultId}/contacts/${item.contact_id}`)}>
                  {item.first_name} {item.last_name}
                </a>
              ),
            },
            {
              title: t("vault.reports.col_city"),
              dataIndex: "city",
              key: "city",
            },
            {
              title: t("vault.reports.col_province"),
              dataIndex: "province",
              key: "province",
            },
          ]}
        />
      </Card>
    );
  };

  return (
    <div style={{ maxWidth: 960, margin: "0 auto", paddingBottom: 48 }}>
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
              styles={{ body: { padding: 20 } }}
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
        {addresses.length > 0 ? (
          <Table
            dataSource={addresses}
            rowKey={(r) => (r.country || "") + (r.city || "")}
            pagination={{ pageSize: 5 }}
            expandable={{
              expandedRowRender: (record) => <AddressDrillDown record={record} />,
            }}
            columns={[
              {
                title: t("vault.reports.col_country"),
                dataIndex: "country",
                key: "country",
              },
              {
                title: t("vault.reports.col_province"),
                dataIndex: "province",
                key: "province",
              },
              {
                title: t("vault.reports.col_city"),
                dataIndex: "city",
                key: "city",
              },
              {
                title: t("vault.reports.col_count"),
                dataIndex: "count",
                key: "count",
                sorter: (a, b) => (a.count || 0) - (b.count || 0),
                defaultSortOrder: "descend",
              },
            ]}
          />
        ) : (
          <Empty
            description={t("vault.reports.no_address_data")}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        )}
      </Card>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} md={12}>
          <Card
            title={t("vault.reports.important_dates_overview")}
            style={{ height: "100%", boxShadow: token.boxShadowTertiary, borderRadius: token.borderRadiusLG }}
          >
            {importantDates.length > 0 ? (
              <Table
                dataSource={[...importantDates].sort((a, b) => {
                  if (a.month !== b.month) return (a.month || 0) - (b.month || 0);
                  return (a.day || 0) - (b.day || 0);
                })}
                rowKey={(r) => `${r.contact_id}-${r.label}-${r.month}-${r.day}`}
                pagination={{ pageSize: 5, size: "small" }}
                size="small"
                columns={[
                  {
                    title: t("vault.reports.col_contact"),
                    key: "contact",
                    render: (_, r) => (
                      <a onClick={() => navigate(`/vaults/${vaultId}/contacts/${r.contact_id}`)}>
                        {r.first_name} {r.last_name}
                      </a>
                    ),
                  },
                  {
                    title: t("vault.reports.col_label"),
                    dataIndex: "label",
                    key: "label",
                  },
                  {
                    title: t("vault.reports.col_date"),
                    key: "date",
                    render: (_, r) => (
                      <span>
                        {r.year ? r.year + "-" : ""}{String(r.month).padStart(2, '0')}-{String(r.day).padStart(2, '0')}
                      </span>
                    ),
                  },
                  {
                    title: t("vault.reports.col_calendar"),
                    dataIndex: "calendar_type",
                    key: "calendar",
                    render: (val) => val === 'lunar' ? <Tag color="purple">{t("calendar.lunar")}</Tag> : <Tag>{t("calendar.gregorian")}</Tag>
                  }
                ]}
              />
            ) : (
              <Empty
                description={t("vault.reports.no_date_data")}
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              />
            )}
          </Card>
        </Col>

        <Col xs={24} md={12}>
          <Card
            title={t("vault.reports.mood_trends")}
            style={{ height: "100%", boxShadow: token.boxShadowTertiary, borderRadius: token.borderRadiusLG }}
          >
            {moodEntries.length > 0 ? (
              <div style={{ display: "flex", flexDirection: "column", gap: 16, padding: "8px 0" }}>
                {moodEntries.map((mood, idx) => {
                  const percent = Math.round(((mood.count || 0) / totalMoodEntries) * 100);
                  return (
                    <div key={idx}>
                      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 6 }}>
                        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                          <div style={{ width: 10, height: 10, borderRadius: "50%", backgroundColor: mood.hex_color || token.colorTextSecondary }} />
                          <Text strong style={{ fontSize: 14 }}>{mood.parameter_label}</Text>
                        </div>
                        <Text type="secondary">{mood.count} ({percent}%)</Text>
                      </div>
                      <div style={{ 
                        height: 8, 
                        width: "100%", 
                        backgroundColor: token.colorFillSecondary, 
                        borderRadius: 4,
                        overflow: "hidden" 
                      }}>
                        <div style={{ 
                          height: "100%", 
                          width: `${percent}%`, 
                          backgroundColor: mood.hex_color || token.colorPrimary,
                          borderRadius: 4
                        }} />
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <Empty
                description={t("vault.reports.no_mood_data")}
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
}