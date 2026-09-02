import { lazy, Suspense, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { formatContactName, useNameOrder } from "@/utils/nameFormat";
import { formatDateTime, useDateFormat } from "@/utils/dateFormat";
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
  Segmented,
  Skeleton,
} from "antd";
import {
  ArrowLeftOutlined,
  TeamOutlined,
  EnvironmentOutlined,
  CalendarOutlined,
  SmileOutlined,
  BarChartOutlined,
  GlobalOutlined,
  MessageOutlined,
  PieChartOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type {
  AddressContactItem,
  AddressReportItem,
  ImportantDateReportItem,
  MoodReportItem,
  MoodTrackingEvent,
} from "@/api";

// The map pulls in d3 and a world outline; both stay out of the initial bundle
// and load when the reports page is actually opened.
const ContactMap = lazy(() => import("@/components/ContactMap"));
const InteractionCadence = lazy(
  () => import("@/components/InteractionCadence"),
);
const DemographicsPanel = lazy(() => import("@/components/DemographicsPanel"));

/** How much history the cadence chart covers, in months. */
const CADENCE_WINDOWS = [12, 24, 60];

const { Title, Text } = Typography;

type AddressContactReportItem = AddressContactItem & {
  contact_name?: string | null;
  middle_name?: string | null;
  nickname?: string | null;
  maiden_name?: string | null;
  prefix?: string | null;
  suffix?: string | null;
};

type ImportantDateContactReportItem = ImportantDateReportItem & {
  contact_name?: string | null;
  middle_name?: string | null;
  nickname?: string | null;
  maiden_name?: string | null;
  prefix?: string | null;
  suffix?: string | null;
};

function getReportContactName(
  nameOrder: string,
  item: AddressContactReportItem | ImportantDateContactReportItem,
): string {
  const formattedName = item.contact_name?.trim();
  return formattedName || formatContactName(nameOrder, item);
}

export default function VaultReports() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const nameOrder = useNameOrder();
  const dateFormats = useDateFormat();

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
      return (res.data ?? []) as ImportantDateContactReportItem[];
    },
  });

  const { data: moodEntries = [] } = useQuery({
    queryKey: ["vault", vaultId, "reports", "mood"],
    queryFn: async () => {
      const res = await api.reports.reportsMoodTrackingEventsList(vaultId);
      return (res.data ?? []) as MoodReportItem[];
    },
  });

  const { data: moodHistory = [], isLoading: moodHistoryLoading } = useQuery({
    queryKey: ["vault", vaultId, "reports", "mood", "history"],
    queryFn: async () => {
      const res = await api.moodTracking.moodTrackingEventsList(vaultId);
      return (res.data ?? []) as MoodTrackingEvent[];
    },
  });

  const [cadenceMonths, setCadenceMonths] = useState(24);

  const { data: mapReport, isPending: mapPending } = useQuery({
    queryKey: ["vault", vaultId, "reports", "map"],
    queryFn: async () => {
      const res = await api.reports.reportsMapList(vaultId);
      return res.data;
    },
    enabled: !!vaultId,
  });

  const { data: demographics, isPending: demographicsPending } = useQuery({
    queryKey: ["vault", vaultId, "reports", "demographics"],
    queryFn: async () => {
      const res = await api.reports.reportsDemographicsList(vaultId);
      return res.data;
    },
    enabled: !!vaultId,
  });

  const { data: interactions, isPending: interactionsPending } = useQuery({
    queryKey: ["vault", vaultId, "reports", "interactions", cadenceMonths],
    queryFn: async () => {
      const res = await api.reports.reportsInteractionsList(vaultId, {
        months: cadenceMonths,
      });
      return res.data;
    },
    enabled: !!vaultId,
    // Keep the chart mounted while this vault's window changes, but never
    // carry contact names or interaction data across a vault switch.
    placeholderData: (previousData, previousQuery) =>
      previousQuery?.queryKey[1] === vaultId ? previousData : undefined,
  });

  const openContact = (contactId: string) =>
    navigate(`/vaults/${vaultId}/contacts/${contactId}`);

  const totalMoodEntries = moodEntries.reduce(
    (acc, curr) => acc + (curr.count || 0),
    0,
  );

  const statCards = [
    {
      icon: <TeamOutlined />,
      bg: token.colorPrimaryBg,
      color: token.colorPrimary,
      title: t("vault.reports.total_contacts"),
      value: reportOverview?.total_contacts ?? 0,
    },
    {
      icon: <EnvironmentOutlined />,
      bg: "rgba(250, 140, 22, 0.15)",
      color: "#fa8c16",
      title: t("vault.reports.total_addresses"),
      value: reportOverview?.total_addresses ?? 0,
    },
    {
      icon: <CalendarOutlined />,
      bg: "rgba(22, 119, 255, 0.15)",
      color: "#1677ff",
      title: t("vault.reports.total_dates"),
      value: reportOverview?.total_important_dates ?? 0,
    },
    {
      icon: <SmileOutlined />,
      bg: "rgba(82, 196, 26, 0.15)",
      color: "#52c41a",
      title: t("vault.reports.mood_entries"),
      value: reportOverview?.total_mood_entries ?? 0,
    },
  ];

  const AddressDrillDown = ({ record }: { record: AddressReportItem }) => {
    const { data: details = [], isLoading } = useQuery({
      queryKey: [
        "vault",
        vaultId,
        "reports",
        "addresses",
        "detail",
        record.country,
        record.city,
      ],
      queryFn: async () => {
        if (record.city) {
          const res = await api.reports.reportsAddressesCityDetail(
            vaultId,
            record.city,
          );
          return (res.data ?? []) as AddressContactReportItem[];
        } else if (record.country) {
          const res = await api.reports.reportsAddressesCountryDetail(
            vaultId,
            record.country,
          );
          return (res.data ?? []) as AddressContactReportItem[];
        }
        return [];
      },
      enabled: !!(record.country || record.city),
    });

    return (
      <Card
        size="small"
        title={t("vault.reports.contacts_in", {
          location: record.city || record.country,
        })}
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
                <a
                  onClick={() =>
                    navigate(`/vaults/${vaultId}/contacts/${item.contact_id}`)
                  }
                >
                  {getReportContactName(nameOrder, item)}
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
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          marginBottom: 24,
        }}
      >
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <BarChartOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0 }}>
          {t("vault.reports.title")}
        </Title>
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
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 12,
                  marginBottom: 12,
                }}
              >
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
        title={
          <span>
            <GlobalOutlined style={{ marginRight: 8 }} />
            {t("vault.reports.map.title")}
          </span>
        }
        style={{
          marginTop: 24,
          boxShadow: token.boxShadowTertiary,
          borderRadius: token.borderRadiusLG,
        }}
      >
        <Suspense fallback={<Skeleton active />}>
          <ContactMap
            points={mapReport?.points ?? []}
            countries={mapReport?.countries ?? []}
            attribution={mapReport?.attribution ?? []}
            onSelectContact={openContact}
            loading={mapPending}
          />
        </Suspense>
        {/* Coordinates only exist where geocoding was configured when the
            address was saved, so say how many are actually plotted. */}
        {(mapReport?.total_addresses ?? 0) > 0 && (
          <Text
            type="secondary"
            style={{ fontSize: 12, display: "block", marginTop: 4 }}
          >
            {t("vault.reports.map.geocoded_summary", {
              geocoded: mapReport?.geocoded_count ?? 0,
              total: mapReport?.total_addresses ?? 0,
            })}
          </Text>
        )}
      </Card>

      <Card
        title={
          <span>
            <MessageOutlined style={{ marginRight: 8 }} />
            {t("vault.reports.interactions.title")}
          </span>
        }
        extra={
          <Segmented
            size="small"
            aria-label={t("vault.reports.interactions.window_label")}
            value={cadenceMonths}
            onChange={(value) => setCadenceMonths(Number(value))}
            options={CADENCE_WINDOWS.map((months) => ({
              label: t("vault.reports.interactions.window_months", {
                count: months,
              }),
              value: months,
            }))}
          />
        }
        style={{
          marginTop: 16,
          boxShadow: token.boxShadowTertiary,
          borderRadius: token.borderRadiusLG,
        }}
      >
        <Suspense fallback={<Skeleton active />}>
          <InteractionCadence
            report={interactions}
            onSelectContact={openContact}
            loading={interactionsPending}
          />
        </Suspense>
      </Card>

      <Card
        title={
          <span>
            <PieChartOutlined style={{ marginRight: 8 }} />
            {t("vault.reports.demographics.title")}
          </span>
        }
        style={{
          marginTop: 16,
          boxShadow: token.boxShadowTertiary,
          borderRadius: token.borderRadiusLG,
        }}
      >
        <Suspense fallback={<Skeleton active />}>
          <DemographicsPanel
            report={demographics}
            loading={demographicsPending}
          />
        </Suspense>
      </Card>

      <Card
        title={t("vault.reports.address_distribution")}
        style={{
          marginTop: 16,
          boxShadow: token.boxShadowTertiary,
          borderRadius: token.borderRadiusLG,
        }}
      >
        {addresses.length > 0 ? (
          <Table
            dataSource={addresses}
            rowKey={(r) => (r.country || "") + (r.city || "")}
            pagination={{ pageSize: 5 }}
            expandable={{
              expandedRowRender: (record) => (
                <AddressDrillDown record={record} />
              ),
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
            style={{
              height: "100%",
              boxShadow: token.boxShadowTertiary,
              borderRadius: token.borderRadiusLG,
            }}
          >
            {importantDates.length > 0 ? (
              <Table
                dataSource={[...importantDates].sort((a, b) => {
                  if (a.month !== b.month)
                    return (a.month || 0) - (b.month || 0);
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
                      <a
                        onClick={() =>
                          navigate(
                            `/vaults/${vaultId}/contacts/${r.contact_id}`,
                          )
                        }
                      >
                        {getReportContactName(nameOrder, r)}
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
                    render: (_, r) => {
                      const date = [
                        r.year != null ? String(r.year) : null,
                        r.month != null
                          ? String(r.month).padStart(2, "0")
                          : null,
                        r.day != null ? String(r.day).padStart(2, "0") : null,
                      ]
                        .filter((part): part is string => part != null)
                        .join("-");
                      return <span>{date || "—"}</span>;
                    },
                  },
                  {
                    title: t("vault.reports.col_calendar"),
                    dataIndex: "calendar_type",
                    key: "calendar",
                    render: (val) =>
                      val === "lunar" ? (
                        <Tag color="purple">{t("calendar.lunar")}</Tag>
                      ) : (
                        <Tag>{t("calendar.gregorian")}</Tag>
                      ),
                  },
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
            style={{
              height: "100%",
              boxShadow: token.boxShadowTertiary,
              borderRadius: token.borderRadiusLG,
            }}
          >
            {totalMoodEntries > 0 ? (
              <div
                style={{
                  display: "flex",
                  flexDirection: "column",
                  gap: 16,
                  padding: "8px 0",
                }}
              >
                {moodEntries.map((mood, idx) => {
                  const percent =
                    totalMoodEntries > 0
                      ? Math.round(((mood.count || 0) / totalMoodEntries) * 100)
                      : 0;
                  return (
                    <div key={idx}>
                      <div
                        style={{
                          display: "flex",
                          justifyContent: "space-between",
                          marginBottom: 6,
                        }}
                      >
                        <div
                          style={{
                            display: "flex",
                            alignItems: "center",
                            gap: 8,
                          }}
                        >
                          <div
                            style={{
                              width: 10,
                              height: 10,
                              borderRadius: "50%",
                              backgroundColor:
                                mood.hex_color || token.colorTextSecondary,
                            }}
                          />
                          <Text strong style={{ fontSize: 14 }}>
                            {mood.parameter_label}
                          </Text>
                        </div>
                        <Text type="secondary">
                          {mood.count} ({percent}%)
                        </Text>
                      </div>
                      <div
                        style={{
                          height: 8,
                          width: "100%",
                          backgroundColor: token.colorFillSecondary,
                          borderRadius: 4,
                          overflow: "hidden",
                        }}
                      >
                        <div
                          style={{
                            height: "100%",
                            width: `${percent}%`,
                            backgroundColor:
                              mood.hex_color || token.colorPrimary,
                            borderRadius: 4,
                          }}
                        />
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

      <Card
        title={t("vault.reports.mood_history")}
        style={{
          marginTop: 16,
          boxShadow: token.boxShadowTertiary,
          borderRadius: token.borderRadiusLG,
        }}
      >
        <Table
          dataSource={moodHistory}
          loading={moodHistoryLoading}
          rowKey="id"
          size="small"
          pagination={{ pageSize: 10 }}
          locale={{ emptyText: t("vault.reports.no_mood_data") }}
          columns={[
            {
              title: t("vault.reports.col_date"),
              dataIndex: "rated_at",
              key: "rated_at",
              render: (value: string | undefined) =>
                value ? formatDateTime(value, dateFormats) : "—",
            },
            {
              title: t("vault.reports.col_mood"),
              key: "mood",
              render: (_, record: MoodTrackingEvent) => (
                <Tag color={record.hex_color || "default"}>
                  {record.parameter_label || "—"}
                </Tag>
              ),
            },
            {
              title: t("vault.reports.col_sleep"),
              dataIndex: "number_of_hours_slept",
              key: "number_of_hours_slept",
              render: (value: number | undefined) =>
                value != null
                  ? t("vault.reports.hours_value", { count: value })
                  : "—",
            },
            {
              title: t("vault.reports.col_note"),
              dataIndex: "note",
              key: "note",
              render: (value: string | undefined) => value || "—",
            },
          ]}
        />
      </Card>
    </div>
  );
}
