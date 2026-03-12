import { useState, useCallback } from "react";
import { useParams, useNavigate, Outlet } from "react-router-dom";
import { formatContactName, useNameOrder } from "@/utils/nameFormat";
import {
  Typography,
  Spin,
  Button,
  Dropdown,
  Modal,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  App,
  List,
  Tag,
  Empty,
  Segmented,
  theme,
  Tooltip,
  Radio,
  DatePicker,
  Select,
} from "antd";
import {
  PlusOutlined,
  SettingOutlined,
  EditOutlined,
  DeleteOutlined,
  CloudServerOutlined,
  ClockCircleOutlined,
  BellOutlined,
  CheckSquareOutlined,
  SmileOutlined,
} from "@ant-design/icons";
import ContactAvatar from "@/components/ContactAvatar";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, httpClient } from "@/api";
import type {
  FeedItem,
  PaginationMeta,
  LifeMetric,
  LifeMetricStats,
  LifeMetricMonthData,
  TimelineEvent,
  MoodTrackingParameterResponse,
  LifeEventCategoryResponse,
} from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";

dayjs.extend(relativeTime);

const { Title, Text } = Typography;

type DashboardTab = "activity" | "life_events" | "life_metrics";

// ─── Main Component ──────────────────────────────────────────────
export default function VaultDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const vaultId = id!;
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const nameOrder = useNameOrder();
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [form] = Form.useForm();
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  // ─── Core Queries ──────────────────────────────────────────────
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
      const res = await api.contacts.contactsList(String(vaultId), { per_page: 9999 });
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  const { data: mostConsulted = [] } = useQuery({
    queryKey: ["vaults", vaultId, "mostConsulted"],
    queryFn: async () => {
      const res = await api.search.searchMostConsultedList(String(vaultId));
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  // ─── Vault CRUD Mutations ─────────────────────────────────────
  const updateMutation = useMutation({
    mutationFn: (values: { name: string; description?: string }) =>
      api.vaults.vaultsUpdate(String(vaultId), values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId] });
      message.success(t("vault.detail.updated"));
      setEditModalOpen(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => api.vaults.vaultsDelete(String(vaultId)),
    onSuccess: () => {
      message.success(t("vault.detail.deleted"));
      navigate("/vaults");
    },
  });

  // ─── Tab State — persisted to backend ─────────────────────────
  const defaultTab = (vault?.default_activity_tab as DashboardTab) || "activity";
  const [activeTab, setActiveTab] = useState<DashboardTab | null>(null);
  const currentTab = activeTab ?? defaultTab;

  const handleTabChange = useCallback(
    (tab: DashboardTab) => {
      setActiveTab(tab);
      // Fire-and-forget: persist the tab preference
      httpClient.instance
        .put(`/vaults/${vaultId}/defaultTab`, { default_activity_tab: tab })
        .catch(() => {
          /* silent — non-critical */
        });
    },
    [vaultId],
  );

  // ─── Loading / Null Guard ─────────────────────────────────────
  if (vaultLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }
  if (!vault) return null;

  const recentContacts = (contacts ?? []).slice(0, 5);

  const settingsMenu = [
    {
      key: "edit",
      icon: <EditOutlined />,
      label: t("vault.detail.edit"),
      onClick: () => {
        form.setFieldsValue({ name: vault.name, description: vault.description });
        setEditModalOpen(true);
      },
    },
    {
      key: "settings",
      icon: <SettingOutlined />,
      label: t("vault_settings.title"),
      onClick: () => navigate(`/vaults/${vaultId}/settings`),
    },
    {
      key: "dav-sync",
      icon: <CloudServerOutlined />,
      label: t("vault.dav_subscriptions.title"),
      onClick: () => navigate(`/vaults/${vaultId}/dav-subscriptions`),
    },
    {
      key: "delete",
      danger: true,
      icon: <DeleteOutlined />,
      label: (
        <Popconfirm
          title={t("vault.detail.delete_confirm")}
          onConfirm={() => deleteMutation.mutate()}
          okText={t("common.delete")}
          cancelText={t("common.cancel")}
        >
          <div onClick={(e) => e.stopPropagation()}>{t("vault.detail.delete")}</div>
        </Popconfirm>
      ),
    },
  ];

  const segmentedOptions = [
    { label: t("vault.dashboard.activity_tab"), value: "activity" as const },
    { label: t("vault.dashboard.life_events_tab"), value: "life_events" as const },
    { label: t("vault.dashboard.life_metrics_tab"), value: "life_metrics" as const },
  ];

  return (
    <div style={{ maxWidth: 1280, margin: "0 auto" }}>
      {/* ─── Header ──────────────────────────────────────────── */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 20,
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <Title level={4} style={{ margin: 0 }}>
            {vault.name}
          </Title>
          <Dropdown menu={{ items: settingsMenu }} trigger={["click"]}>
            <Button type="text" icon={<SettingOutlined />} />
          </Dropdown>
        </div>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}/contacts/create`)}
        >
          {t("vault.detail.add_contact")}
        </Button>
      </div>

      {/* ─── 3-Column Dashboard ──────────────────────────────── */}
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "240px 1fr 320px",
          gap: 20,
          alignItems: "start",
        }}
        className="vault-dashboard-grid"
      >
        {/* ─── Left Sidebar ───────────────────────────────────── */}
        <div className="vault-dashboard-left" style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          <SidebarSection title={t("vault.dashboard.recent_contacts")}>
            {recentContacts.length === 0 ? (
              <Text type="secondary" style={{ fontSize: 13, padding: "8px 0" }}>
                {t("vault.dashboard.no_contacts")}
              </Text>
            ) : (
              <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {recentContacts.map((contact: any) => (
                  <div
                    key={contact.id}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 10,
                      padding: "6px 8px",
                      borderRadius: token.borderRadius,
                      cursor: "pointer",
                      transition: "background 0.15s",
                    }}
                    className="vault-sidebar-contact"
                    onClick={() => navigate(`/vaults/${vaultId}/contacts/${contact.id}`)}
                  >
                    <ContactAvatar
                      vaultId={vaultId}
                      contactId={contact.id}
                      firstName={contact.first_name}
                      lastName={contact.last_name}
                      size={28}
                    />
                    <Text style={{ fontSize: 13, flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {formatContactName(nameOrder, contact)}
                    </Text>
                  </div>
                ))}
              </div>
            )}
          </SidebarSection>

          <SidebarSection title={t("vault.dashboard.most_consulted")}>
            {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
            {(mostConsulted as any[]).length === 0 ? (
              <Text type="secondary" style={{ fontSize: 13, padding: "8px 0" }}>
                {t("vault.dashboard.no_consulted")}
              </Text>
            ) : (
              <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {(mostConsulted as any[]).map((item: any) => (
                  <div
                    key={item.contact_id}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 10,
                      padding: "6px 8px",
                      borderRadius: token.borderRadius,
                      cursor: "pointer",
                      transition: "background 0.15s",
                    }}
                    className="vault-sidebar-contact"
                    onClick={() => navigate(`/vaults/${vaultId}/contacts/${item.contact_id}`)}
                  >
                    <ContactAvatar
                      vaultId={vaultId}
                      contactId={item.contact_id}
                      firstName={item.first_name}
                      lastName={item.last_name}
                      size={28}
                    />
                    <Text style={{ fontSize: 13, flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {formatContactName(nameOrder, item)}
                    </Text>
                  </div>
                ))}
              </div>
            )}
          </SidebarSection>
        </div>

        {/* ─── Center Content ─────────────────────────────────── */}
        <div style={{ minWidth: 0 }}>
          <div style={{ marginBottom: 16 }}>
            <Segmented
              value={currentTab}
              onChange={(v) => handleTabChange(v as DashboardTab)}
              options={segmentedOptions}
              block
            />
          </div>
          <div
            style={{
              background: token.colorBgContainer,
              borderRadius: token.borderRadiusLG,
              boxShadow: token.boxShadowTertiary,
              minHeight: 200,
            }}
          >
            {currentTab === "activity" && <ActivityTab vaultId={vaultId} />}
            {currentTab === "life_events" && <LifeEventsTab vaultId={vaultId} userContactId={vault.user_contact_id} />}
            {currentTab === "life_metrics" && <LifeMetricsTab vaultId={vaultId} />}
          </div>
        </div>

        {/* ─── Right Sidebar ──────────────────────────────────── */}
        <div className="vault-dashboard-right" style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          <MoodRecordingWidget vaultId={vaultId} userContactId={vault.user_contact_id} />
          <UpcomingRemindersWidget vaultId={vaultId} />
          <DueTasksWidget vaultId={vaultId} />
        </div>
      </div>

      <Outlet />

      {/* ─── Edit Vault Modal ─────────────────────────────────── */}
      <Modal
        title={t("vault.detail.edit")}
        open={editModalOpen}
        onCancel={() => setEditModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={updateMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => updateMutation.mutate(v)}>
          <Form.Item
            name="name"
            label={t("vault.create.name_label")}
            rules={[{ required: true, message: t("vault.create.name_required") }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="description" label={t("vault.create.description_label")}>
            <Input.TextArea />
          </Form.Item>
        </Form>
      </Modal>

      {/* ─── Responsive CSS ───────────────────────────────────── */}
      <style>{`
        .vault-sidebar-contact:hover {
          background: ${token.colorFillQuaternary};
        }
        /* Tablet: hide left sidebar */
        @media (max-width: 1023px) {
          .vault-dashboard-grid {
            grid-template-columns: 1fr 320px !important;
          }
          .vault-dashboard-left {
            display: none !important;
          }
        }
        /* Mobile: single column */
        @media (max-width: 767px) {
          .vault-dashboard-grid {
            grid-template-columns: 1fr !important;
          }
          .vault-dashboard-right {
            order: -1;
          }
        }
      `}</style>
    </div>
  );
}

// ─── Sidebar Section ─────────────────────────────────────────────
function SidebarSection({ title, children }: { title: string; children: React.ReactNode }) {
  const { token } = theme.useToken();
  return (
    <div
      style={{
        background: token.colorBgContainer,
        borderRadius: token.borderRadiusLG,
        boxShadow: token.boxShadowTertiary,
        padding: "14px 16px",
      }}
    >
      <Text strong style={{ fontSize: 13, display: "block", marginBottom: 10, color: token.colorTextSecondary }}>
        {title}
      </Text>
      {children}
    </div>
  );
}

// ─── Activity Tab ────────────────────────────────────────────────
function ActivityTab({ vaultId }: { vaultId: string }) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const navigate = useNavigate();
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
    queryKey: ["vaults", vaultId, "feed", page],
    queryFn: async () => {
      const res = await api.feed.feedList(String(vaultId), { page, per_page: 15 });
      const newItems = (res.data ?? []) as FeedItem[];
      const meta = res.meta as PaginationMeta | undefined;
      setAllItems((prev) => (page === 1 ? newItems : [...prev, ...newItems]));
      setHasMore(meta ? meta.page! < meta.total_pages! : newItems.length >= 15);
      return newItems;
    },
    enabled: !!vaultId,
  });

  if (isLoading && page === 1) {
    return (
      <div style={{ textAlign: "center", padding: 48 }}>
        <Spin />
      </div>
    );
  }

  return (
    <div style={{ padding: "8px 0" }}>
      <List
        dataSource={allItems}
        locale={{
          emptyText: (
            <Empty description={t("empty.feed")} style={{ padding: 32 }} />
          ),
        }}
        renderItem={(item: FeedItem, index: number) => (
          <List.Item
            style={{
              margin: "0 16px",
              paddingLeft: 20,
              borderLeft: `2px solid ${index === 0 ? token.colorPrimary : token.colorBorderSecondary}`,
              position: "relative",
            }}
          >
            <div
              style={{
                position: "absolute",
                left: -5,
                top: 18,
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
                    color={getActionColor(item.action ?? "")}
                    style={{ borderRadius: 12, fontSize: 11, margin: 0 }}
                  >
                    {item.action}
                  </Tag>
                  {item.contact_id && (
                    <a
                      style={{ fontWeight: 600 }}
                      onClick={() => navigate(`/vaults/${vaultId}/contacts/${item.contact_id}`)}
                    >
                      {item.contact_name || item.contact_id}
                    </a>
                  )}
                </div>
              }
              description={
                <>
                  {item.description && (
                    <Text type="secondary" style={{ display: "block", marginTop: 4 }}>
                      {item.description}
                    </Text>
                  )}
                  <div style={{ display: "flex", alignItems: "center", gap: 4, marginTop: 6 }}>
                    <ClockCircleOutlined style={{ fontSize: 11, color: token.colorTextQuaternary }} />
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {dayjs(item.created_at).fromNow()}
                    </Text>
                  </div>
                </>
              }
            />
          </List.Item>
        )}
      />
      {hasMore && allItems.length > 0 && (
        <div style={{ textAlign: "center", padding: "12px 0" }}>
          <Button onClick={() => setPage((p) => p + 1)} loading={isFetching}>
            {t("common.load_more")}
          </Button>
        </div>
      )}
    </div>
  );
}

// ─── Life Events Tab ─────────────────────────────────────────────
function LifeEventsTab({ vaultId, userContactId }: { vaultId: string; userContactId?: string }) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [allTimelines, setAllTimelines] = useState<TimelineEvent[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [addForm] = Form.useForm();
  const [selectedCategoryId, setSelectedCategoryId] = useState<number | null>(null);

  const { isLoading, isFetching } = useQuery({
    queryKey: ["vaults", vaultId, "dashboardLifeEvents", page],
    queryFn: async () => {
      const res = await api.lifeEvents.dashboardLifeEventsList(String(vaultId), {
        page,
        per_page: 15,
      });
      const newItems = (res.data ?? []) as TimelineEvent[];
      const meta = res.meta as PaginationMeta | undefined;
      setAllTimelines((prev) => (page === 1 ? newItems : [...prev, ...newItems]));
      setHasMore(meta ? meta.page! < meta.total_pages! : newItems.length >= 15);
      return newItems;
    },
    enabled: !!vaultId,
  });

  const { data: lifeEventCategories = [] } = useQuery({
    queryKey: ["vaults", vaultId, "settings", "lifeEventCategories"],
    queryFn: async () => {
      const res = await api.vaultSettings.settingsLifeEventCategoriesList(String(vaultId));
      return (res.data ?? []) as LifeEventCategoryResponse[];
    },
    enabled: !!vaultId && addModalOpen,
  });

  const filteredTypes = lifeEventCategories.find((c) => c.id === selectedCategoryId)?.types ?? [];

  const addLifeEventMutation = useMutation({
    mutationFn: async (values: { life_event_type_id: number; happened_at: dayjs.Dayjs; summary?: string; description?: string }) => {
      const dateStr = values.happened_at.toISOString();
      const timelineRes = await api.lifeEvents.contactsTimelineEventsCreate(
        String(vaultId),
        userContactId!,
        { started_at: dateStr, label: values.summary || undefined },
      );
      const timelineId = timelineRes.data?.id;
      if (!timelineId) throw new Error("Failed to create timeline event");
      await api.lifeEvents.contactsTimelineEventsLifeEventsCreate(
        String(vaultId),
        userContactId!,
        timelineId,
        {
          life_event_type_id: values.life_event_type_id,
          happened_at: dateStr,
          summary: values.summary || undefined,
          description: values.description || undefined,
        },
      );
    },
    onSuccess: () => {
      message.success(t("vault.dashboard.life_event_added"));
      setAddModalOpen(false);
      addForm.resetFields();
      setSelectedCategoryId(null);
      setPage(1);
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "dashboardLifeEvents"] });
    },
  });

  if (isLoading && page === 1) {
    return (
      <div style={{ textAlign: "center", padding: 48 }}>
        <Spin />
      </div>
    );
  }

  return (
    <div style={{ padding: "16px 20px" }}>
      {!userContactId ? (
        <Text type="secondary" style={{ fontSize: 13 }}>
          {t("vault.dashboard.life_events_not_available")}
        </Text>
      ) : (
        <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: 12 }}>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              addForm.resetFields();
              setSelectedCategoryId(null);
              setAddModalOpen(true);
            }}
            size="small"
          >
            {t("vault.dashboard.add_life_event")}
          </Button>
        </div>
      )}

      {allTimelines.length === 0 ? (
        <Empty description={t("vault.dashboard.no_life_events")} style={{ padding: 16 }} />
      ) : (
        <>
          {allTimelines.map((tl) => (
            <div key={tl.id} style={{ marginBottom: 20 }}>
              <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 10 }}>
                <Text strong style={{ fontSize: 14 }}>
                  {tl.label}
                </Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {dayjs(tl.started_at).format("MMM YYYY")}
                </Text>
              </div>
              {tl.life_events && tl.life_events.length > 0 ? (
                <div
                  style={{
                    borderLeft: `2px solid ${token.colorBorderSecondary}`,
                    marginLeft: 4,
                    paddingLeft: 16,
                  }}
                >
                  {tl.life_events.map((le) => (
                    <div key={le.id} style={{ marginBottom: 12, position: "relative" }}>
                      <div
                        style={{
                          position: "absolute",
                          left: -21,
                          top: 6,
                          width: 8,
                          height: 8,
                          borderRadius: "50%",
                          background: token.colorPrimary,
                        }}
                      />
                      <Text style={{ fontWeight: 500, fontSize: 13 }}>
                        {le.summary ?? le.description}
                      </Text>
                      <br />
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {dayjs(le.happened_at).format("MMM D, YYYY")}
                      </Text>
                      {le.description && le.summary && (
                        <div style={{ marginTop: 2, color: token.colorTextSecondary, fontSize: 12 }}>
                          {le.description}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {t("modules.life_events.no_events")}
                </Text>
              )}
            </div>
          ))}
          {hasMore && allTimelines.length > 0 && (
            <div style={{ textAlign: "center", paddingBottom: 8 }}>
              <Button onClick={() => setPage((p) => p + 1)} loading={isFetching}>
                {t("common.load_more")}
              </Button>
            </div>
          )}
        </>
      )}

      <Modal
        title={t("vault.dashboard.add_life_event")}
        open={addModalOpen}
        onCancel={() => {
          setAddModalOpen(false);
          addForm.resetFields();
          setSelectedCategoryId(null);
        }}
        onOk={() => addForm.submit()}
        confirmLoading={addLifeEventMutation.isPending}
      >
        <Form
          form={addForm}
          layout="vertical"
          initialValues={{ happened_at: dayjs() }}
          onFinish={(values) => addLifeEventMutation.mutate(values)}
        >
          <Form.Item
            name="category_id"
            label={t("vault.dashboard.select_category")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Select
              placeholder={t("vault.dashboard.select_category")}
              onChange={(v: number) => {
                setSelectedCategoryId(v);
                addForm.setFieldValue("life_event_type_id", undefined);
              }}
              options={lifeEventCategories.map((c) => ({ label: c.label, value: c.id }))}
            />
          </Form.Item>
          <Form.Item
            name="life_event_type_id"
            label={t("vault.dashboard.select_type")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Select
              placeholder={t("vault.dashboard.select_type")}
              disabled={!selectedCategoryId}
              options={filteredTypes.map((tp) => ({ label: tp.label, value: tp.id }))}
            />
          </Form.Item>
          <Form.Item
            name="happened_at"
            label={t("vault.dashboard.life_event_date")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <DatePicker style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="summary" label={t("vault.dashboard.life_event_summary")}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label={t("vault.dashboard.life_event_description")}>
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

// ─── Life Metrics Tab ────────────────────────────────────────────
function LifeMetricsTab({ vaultId }: { vaultId: string }) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [editingMetric, setEditingMetric] = useState<LifeMetric | null>(null);
  const [expandedMetricId, setExpandedMetricId] = useState<number | null>(null);
  const [incrementedId, setIncrementedId] = useState<number | null>(null);
  const [form] = Form.useForm();

  const { data: metrics = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "lifeMetrics"],
    queryFn: async () => {
      const res = await api.lifeMetrics.lifeMetricsList(String(vaultId));
      return (res.data ?? []) as LifeMetric[];
    },
    enabled: !!vaultId,
  });

  const createMutation = useMutation({
    mutationFn: (values: { label: string }) =>
      api.lifeMetrics.lifeMetricsCreate(String(vaultId), values),
    onSuccess: () => {
      message.success(t("vault.dashboard.metric_created"));
      setCreateOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "lifeMetrics"] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: (values: { id: number; label: string }) =>
      api.lifeMetrics.lifeMetricsUpdate(String(vaultId), values.id, { label: values.label }),
    onSuccess: () => {
      message.success(t("vault.dashboard.metric_updated"));
      setCreateOpen(false);
      setEditingMetric(null);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "lifeMetrics"] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.lifeMetrics.lifeMetricsDelete(String(vaultId), id),
    onSuccess: () => {
      message.success(t("vault.dashboard.metric_deleted"));
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "lifeMetrics"] });
    },
  });

  const incrementMutation = useMutation({
    mutationFn: (id: number) => api.lifeMetrics.lifeMetricsIncrementCreate(String(vaultId), id),
    onSuccess: (_data, id) => {
      message.success(t("vault.dashboard.metric_incremented"));
      setIncrementedId(id);
      setTimeout(() => setIncrementedId(null), 1200);
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "lifeMetrics"] });
      // Refresh detail if expanded
      if (expandedMetricId === id) {
        queryClient.invalidateQueries({
          queryKey: ["vaults", vaultId, "lifeMetrics", id, "detail"],
        });
      }
    },
  });

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 48 }}>
        <Spin />
      </div>
    );
  }

  return (
    <div style={{ padding: "16px 20px" }}>
      <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: 12 }}>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingMetric(null);
            form.resetFields();
            setCreateOpen(true);
          }}
          size="small"
        >
          {t("vault.dashboard.track_new_metric")}
        </Button>
      </div>

      {metrics.length === 0 ? (
        <Empty description={t("vault.dashboard.no_metrics")} style={{ padding: 24 }} />
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          {metrics.map((metric) => (
            <MetricCard
              key={metric.id}
              metric={metric}
              vaultId={vaultId}
              token={token}
              isIncremented={incrementedId === metric.id}
              isExpanded={expandedMetricId === metric.id}
              onIncrement={() => incrementMutation.mutate(metric.id!)}
              onToggleExpand={() =>
                setExpandedMetricId((prev) => (prev === metric.id ? null : metric.id!))
              }
              onEdit={() => {
                setEditingMetric(metric);
                form.setFieldsValue({ label: metric.label });
                setCreateOpen(true);
              }}
              onDelete={() => {
                Modal.confirm({
                  title: t("common.confirmDelete"),
                  content: t("common.confirmDeleteDescription"),
                  okText: t("common.delete"),
                  cancelText: t("common.cancel"),
                  okType: "danger",
                  onOk: () => deleteMutation.mutate(metric.id!),
                });
              }}
            />
          ))}
        </div>
      )}

      <Modal
        title={editingMetric ? t("vault.lifeMetrics.edit") : t("vault.lifeMetrics.create")}
        open={createOpen}
        onCancel={() => {
          setCreateOpen(false);
          setEditingMetric(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => {
            if (editingMetric) {
              updateMutation.mutate({ id: editingMetric.id!, label: values.label });
            } else {
              createMutation.mutate(values);
            }
          }}
        >
          <Form.Item
            name="label"
            label={t("vault.lifeMetrics.label")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

// ─── Metric Card ─────────────────────────────────────────────────
function MetricCard({
  metric,
  vaultId,
  token: themeToken,
  isIncremented,
  isExpanded,
  onIncrement,
  onToggleExpand,
  onEdit,
  onDelete,
}: {
  metric: LifeMetric;
  vaultId: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  token: any;
  isIncremented: boolean;
  isExpanded: boolean;
  onIncrement: () => void;
  onToggleExpand: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const { t } = useTranslation();
  const stats = metric.stats as LifeMetricStats | undefined;

  return (
    <div
      style={{
        border: `1px solid ${themeToken.colorBorderSecondary}`,
        borderRadius: themeToken.borderRadius,
        padding: "12px 16px",
        transition: "box-shadow 0.2s",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 12, flex: 1, minWidth: 0 }}>
          <Text strong style={{ fontSize: 14 }}>
            {metric.label}
          </Text>
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
            {stats && (
              <>
                <Tag style={{ margin: 0, cursor: "pointer", fontSize: 11 }} onClick={onToggleExpand}>
                  {stats.weekly_events ?? 0}/{t("vault.dashboard.events_this_week")}
                </Tag>
                <Tag style={{ margin: 0, fontSize: 11 }}>
                  {stats.monthly_events ?? 0}/{t("vault.dashboard.events_this_month")}
                </Tag>
                <Tag style={{ margin: 0, fontSize: 11 }}>
                  {stats.yearly_events ?? 0}/{t("vault.dashboard.events_this_year")}
                </Tag>
              </>
            )}
          </div>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
          <Tooltip title="+1">
            <Button
              type="primary"
              size="small"
              onClick={onIncrement}
              style={{
                borderRadius: 16,
                fontWeight: 600,
                minWidth: 40,
              }}
            >
              {isIncremented ? "🤭" : "+1"}
            </Button>
          </Tooltip>
          <Dropdown
            menu={{
              items: [
                { key: "edit", label: t("common.edit"), icon: <EditOutlined />, onClick: onEdit },
                {
                  key: "delete",
                  label: t("common.delete"),
                  icon: <DeleteOutlined />,
                  danger: true,
                  onClick: onDelete,
                },
              ],
            }}
            trigger={["click"]}
          >
            <Button type="text" size="small" style={{ color: themeToken.colorTextSecondary }}>
              ···
            </Button>
          </Dropdown>
        </div>
      </div>

      {isExpanded && <MetricBarChart vaultId={vaultId} metricId={metric.id!} token={themeToken} />}
    </div>
  );
}

// ─── Metric Bar Chart ────────────────────────────────────────────
function MetricBarChart({
  vaultId,
  metricId,
  token: themeToken,
}: {
  vaultId: string;
  metricId: number;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  token: any;
}) {
  const currentYear = dayjs().year();
  const [year, setYear] = useState(currentYear);

  const { data: detail } = useQuery({
    queryKey: ["vaults", vaultId, "lifeMetrics", metricId, "detail", year],
    queryFn: async () => {
      const res = await api.lifeMetrics.lifeMetricsDetailList(String(vaultId), metricId, { year });
      return res.data;
    },
    enabled: !!vaultId && !!metricId,
  });

  const months = (detail?.months ?? []) as LifeMetricMonthData[];
  const maxEvents = detail?.max_events ?? Math.max(...months.map((m) => m.events ?? 0), 1);

  return (
    <div style={{ marginTop: 12, paddingTop: 12, borderTop: `1px solid ${themeToken.colorBorderSecondary}` }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 8 }}>
        <Button size="small" type="text" onClick={() => setYear((y) => y - 1)}>
          ←
        </Button>
        <Text strong style={{ fontSize: 12 }}>
          {year}
        </Text>
        <Button size="small" type="text" onClick={() => setYear((y) => y + 1)} disabled={year >= currentYear}>
          →
        </Button>
      </div>
      <div style={{ display: "flex", gap: 3, alignItems: "flex-end", height: 80 }}>
        {months.map((m) => {
          const height = maxEvents > 0 ? Math.max(((m.events ?? 0) / maxEvents) * 100, 2) : 2;
          return (
            <Tooltip key={m.month} title={`${m.friendly_name}: ${m.events ?? 0}`}>
              <div
                style={{
                  flex: 1,
                  display: "flex",
                  flexDirection: "column",
                  alignItems: "center",
                  gap: 2,
                }}
              >
                <div
                  style={{
                    width: "100%",
                    height: `${height}%`,
                    minHeight: 2,
                    background:
                      (m.events ?? 0) > 0 ? themeToken.colorPrimary : themeToken.colorFillSecondary,
                    borderRadius: 2,
                    transition: "height 0.3s",
                  }}
                />
                <Text style={{ fontSize: 9, color: themeToken.colorTextQuaternary }}>
                  {(m.friendly_name ?? "").slice(0, 3)}
                </Text>
              </div>
            </Tooltip>
          );
        })}
      </div>
    </div>
  );
}

// ─── Mood Recording Widget ───────────────────────────────────────
function MoodRecordingWidget({ vaultId, userContactId }: { vaultId: string; userContactId?: string }) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const { message } = App.useApp();
  const [selectedMoodId, setSelectedMoodId] = useState<number | null>(null);
  const [showDatePicker, setShowDatePicker] = useState(false);
  const [showNote, setShowNote] = useState(false);
  const [showSleep, setShowSleep] = useState(false);
  const [moodDate, setMoodDate] = useState<dayjs.Dayjs | null>(null);
  const [moodNote, setMoodNote] = useState("");
  const [hoursSlept, setHoursSlept] = useState<number | null>(null);

  const { data: moodParams = [] } = useQuery({
    queryKey: ["vaults", vaultId, "settings", "moodParams"],
    queryFn: async () => {
      const res = await api.vaultSettings.settingsMoodParamsList(String(vaultId));
      return (res.data ?? []) as MoodTrackingParameterResponse[];
    },
    enabled: !!vaultId,
  });

  const recordMutation = useMutation({
    mutationFn: (data: { mood_tracking_parameter_id: number; rated_at: string; note?: string; number_of_hours_slept?: number }) =>
      api.moodTracking.contactsMoodTrackingEventsCreate(String(vaultId), userContactId!, data),
    onSuccess: () => {
      message.success(t("vault.dashboard.mood_recorded"));
      setSelectedMoodId(null);
      setShowDatePicker(false);
      setShowNote(false);
      setShowSleep(false);
      setMoodDate(null);
      setMoodNote("");
      setHoursSlept(null);
    },
  });

  const handleRecord = () => {
    if (!selectedMoodId || !userContactId) return;
    const data: { mood_tracking_parameter_id: number; rated_at: string; note?: string; number_of_hours_slept?: number } = {
      mood_tracking_parameter_id: selectedMoodId,
      rated_at: (moodDate ?? dayjs()).toISOString(),
    };
    if (moodNote.trim()) data.note = moodNote.trim();
    if (hoursSlept != null) data.number_of_hours_slept = hoursSlept;
    recordMutation.mutate(data);
  };

  return (
    <div
      style={{
        background: token.colorBgContainer,
        borderRadius: token.borderRadiusLG,
        boxShadow: token.boxShadowTertiary,
        padding: "14px 16px",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 12 }}>
        <SmileOutlined style={{ color: token.colorWarning, fontSize: 15 }} />
        <Text strong style={{ fontSize: 13, color: token.colorTextSecondary }}>
          {t("vault.dashboard.mood_title")}
        </Text>
      </div>

      {!userContactId ? (
        <Text type="secondary" style={{ fontSize: 13 }}>
          {t("vault.dashboard.mood_not_available")}
        </Text>
      ) : moodParams.length === 0 ? (
        <div style={{ textAlign: "center", padding: "16px 0", color: token.colorTextSecondary, fontSize: 13 }}>
          <SmileOutlined style={{ fontSize: 28, opacity: 0.3, display: "block", marginBottom: 8 }} />
          {t("vault.dashboard.mood_how_are_you")}
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          <Radio.Group
            value={selectedMoodId}
            onChange={(e) => setSelectedMoodId(e.target.value as number)}
            style={{ display: "flex", flexDirection: "column", gap: 6 }}
          >
            {moodParams.map((param) => (
              <Radio key={param.id} value={param.id} style={{ fontSize: 13 }}>
                <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                  <span
                    style={{
                      width: 10,
                      height: 10,
                      borderRadius: "50%",
                      background: param.hex_color ?? token.colorPrimary,
                      display: "inline-block",
                      flexShrink: 0,
                    }}
                  />
                  {param.label}
                </span>
              </Radio>
            ))}
          </Radio.Group>

          <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
            {!showDatePicker && (
              <Button type="link" size="small" style={{ padding: 0, fontSize: 12 }} onClick={() => setShowDatePicker(true)}>
                {t("vault.dashboard.mood_change_date")}
              </Button>
            )}
            {!showNote && (
              <Button type="link" size="small" style={{ padding: 0, fontSize: 12 }} onClick={() => setShowNote(true)}>
                {t("vault.dashboard.mood_add_note")}
              </Button>
            )}
            {!showSleep && (
              <Button type="link" size="small" style={{ padding: 0, fontSize: 12 }} onClick={() => setShowSleep(true)}>
                {t("vault.dashboard.mood_hours_slept")}
              </Button>
            )}
          </div>

          {showDatePicker && (
            <DatePicker
              value={moodDate}
              onChange={(d) => setMoodDate(d)}
              style={{ width: "100%" }}
              size="small"
            />
          )}
          {showNote && (
            <Input.TextArea
              value={moodNote}
              onChange={(e) => setMoodNote(e.target.value)}
              placeholder={t("modules.mood_tracking.note_placeholder")}
              rows={2}
              size="small"
            />
          )}
          {showSleep && (
            <InputNumber
              value={hoursSlept}
              onChange={(v) => setHoursSlept(v)}
              min={0}
              max={24}
              placeholder={t("vault.dashboard.mood_hours_slept")}
              style={{ width: "100%" }}
              size="small"
            />
          )}

          <Button
            type="primary"
            size="small"
            block
            disabled={!selectedMoodId}
            loading={recordMutation.isPending}
            onClick={handleRecord}
          >
            {t("vault.dashboard.mood_record")}
          </Button>
        </div>
      )}
    </div>
  );
}

// ─── Upcoming Reminders Widget ───────────────────────────────────
function UpcomingRemindersWidget({ vaultId }: { vaultId: string }) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const navigate = useNavigate();

  const { data: reminders = [] } = useQuery({
    queryKey: ["vaults", vaultId, "reminders"],
    queryFn: async () => {
      const res = await api.reminders.remindersList(String(vaultId));
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const upcoming = (reminders as any[]).slice(0, 5);

  return (
    <div
      style={{
        background: token.colorBgContainer,
        borderRadius: token.borderRadiusLG,
        boxShadow: token.boxShadowTertiary,
        padding: "14px 16px",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 12 }}>
        <BellOutlined style={{ color: token.colorWarning, fontSize: 15 }} />
        <Text strong style={{ fontSize: 13, color: token.colorTextSecondary }}>
          {t("vault.dashboard.upcoming_reminders")}
        </Text>
      </div>

      {upcoming.length === 0 ? (
        <Text type="secondary" style={{ fontSize: 13 }}>
          {t("vault.dashboard.no_reminders")}
        </Text>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
          {upcoming.map((r: any) => (
            <div key={r.id} style={{ display: "flex", justifyContent: "space-between", fontSize: 13 }}>
              <Text style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {r.label}
              </Text>
              <Text type="secondary" style={{ fontSize: 12, flexShrink: 0, marginLeft: 8 }}>
                {r.month && r.day ? `${r.month}/${r.day}` : ""}
              </Text>
            </div>
          ))}
          <Button
            type="link"
            size="small"
            style={{ padding: 0, fontSize: 12 }}
            onClick={() => navigate(`/vaults/${vaultId}/reminders`)}
          >
            {t("vault.dashboard.view_all")} →
          </Button>
        </div>
      )}
    </div>
  );
}

// ─── Due Tasks Widget ────────────────────────────────────────────
function DueTasksWidget({ vaultId }: { vaultId: string }) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const navigate = useNavigate();

  const { data: tasks = [] } = useQuery({
    queryKey: ["vaults", vaultId, "all-tasks"],
    queryFn: async () => {
      const res = await api.vaultTasks.tasksList(String(vaultId));
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  // Filter to incomplete tasks with a due date within 30 days
  const now = dayjs();
  const cutoff = now.add(30, "day");
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const dueTasks = (tasks as any[])
    .filter(
      (t) => !t.completed && t.due_at && dayjs(t.due_at).isBefore(cutoff),
    )
    .sort((a, b) => dayjs(a.due_at).valueOf() - dayjs(b.due_at).valueOf())
    .slice(0, 5);

  return (
    <div
      style={{
        background: token.colorBgContainer,
        borderRadius: token.borderRadiusLG,
        boxShadow: token.boxShadowTertiary,
        padding: "14px 16px",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 12 }}>
        <CheckSquareOutlined style={{ color: token.colorSuccess, fontSize: 15 }} />
        <Text strong style={{ fontSize: 13, color: token.colorTextSecondary }}>
          {t("vault.dashboard.due_tasks")}
        </Text>
      </div>

      {dueTasks.length === 0 ? (
        <Text type="secondary" style={{ fontSize: 13 }}>
          {t("vault.dashboard.no_tasks")}
        </Text>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
          {dueTasks.map((task: any) => (
            <div key={task.id} style={{ display: "flex", justifyContent: "space-between", fontSize: 13 }}>
              <Text style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {task.label}
              </Text>
              <Text type="secondary" style={{ fontSize: 12, flexShrink: 0, marginLeft: 8 }}>
                {dayjs(task.due_at).format("M/D")}
              </Text>
            </div>
          ))}
          <Button
            type="link"
            size="small"
            style={{ padding: 0, fontSize: 12 }}
            onClick={() => navigate(`/vaults/${vaultId}/tasks`)}
          >
            {t("vault.dashboard.view_all")} →
          </Button>
        </div>
      )}
    </div>
  );
}
