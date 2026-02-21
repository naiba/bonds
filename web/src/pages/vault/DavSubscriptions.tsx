import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Typography,
  Button,
  Table,
  theme,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  Drawer,
  App,
  Popconfirm,
  Space,
  Alert,
  Tooltip,
  Empty,
  Card,
} from "antd";
import {
  ArrowLeftOutlined,
  PlusOutlined,
  SyncOutlined,
  FileTextOutlined,
  EditOutlined,
  DeleteOutlined,
  CloudServerOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  PlayCircleOutlined,
} from "@ant-design/icons";
import dayjs from "dayjs";
import { api } from "@/api";
import { useAuth } from "@/stores/auth";
import type {
  DavSubscription,
  DavSyncLog,
  CreateDavSubscriptionRequest,
  UpdateDavSubscriptionRequest,
  TestDavConnectionResponse,
  APIError,
  PaginationMeta,
} from "@/api";

const { Title, Text, Paragraph } = Typography;

const SYNC_WAY_PUSH = 1;
const SYNC_WAY_PULL = 2;
const SYNC_WAY_BOTH = 3;

const FREQUENCY_OPTIONS = [30, 60, 180, 360, 720, 1440];

const LOG_ACTION_COLORS: Record<string, string> = {
  created: "green",
  updated: "blue",
  deleted: "red",
  error: "red",
  pushed: "cyan",
  skipped: "orange",
  conflict_local_wins: "gold",
  skipped_push_origin: "orange",
  push_deleted: "magenta",
};

export default function DavSubscriptions() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const { message } = App.useApp();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();

  const [modalOpen, setModalOpen] = useState(false);
  const [editingSubscription, setEditingSubscription] = useState<DavSubscription | null>(null);
  const [testResult, setTestResult] = useState<TestDavConnectionResponse | null>(null);
  const [testLoading, setTestLoading] = useState(false);

  const [logsDrawerOpen, setLogsDrawerOpen] = useState(false);
  const [logsSubscription, setLogsSubscription] = useState<DavSubscription | null>(null);
  const [logsPage, setLogsPage] = useState(1);
  const [accumulatedLogs, setAccumulatedLogs] = useState<DavSyncLog[]>([]);

  const { data: subscriptions = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "dav-subscriptions"],
    queryFn: async () => {
      const res = await api.davSubscriptions.davSubscriptionsList(vaultId);
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  const { data: logsData } = useQuery({
    queryKey: ["vaults", vaultId, "dav-subscriptions", logsSubscription?.id, "logs", logsPage],
    queryFn: async () => {
      const res = await api.davSubscriptions.davSubscriptionsLogsList(
        vaultId,
        logsSubscription!.id!,
        { page: logsPage, per_page: 20 },
      );
      const newLogs = (res.data ?? []) as DavSyncLog[];
      setAccumulatedLogs((prev) => logsPage === 1 ? newLogs : [...prev, ...newLogs]);
      return { logs: newLogs, meta: res.meta as PaginationMeta | undefined };
    },
    enabled: !!logsSubscription?.id,
  });

  const createMutation = useMutation({
    mutationFn: (data: CreateDavSubscriptionRequest) =>
      api.davSubscriptions.davSubscriptionsCreate(vaultId, data),
    onSuccess: () => {
      message.success(t("vault.dav_subscriptions.created"));
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "dav-subscriptions"] });
      setModalOpen(false);
      form.resetFields();
      setTestResult(null);
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateMutation = useMutation({
    mutationFn: ({ subId, data }: { subId: string; data: UpdateDavSubscriptionRequest }) =>
      api.davSubscriptions.davSubscriptionsUpdate(vaultId, subId, data),
    onSuccess: () => {
      message.success(t("vault.dav_subscriptions.updated"));
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "dav-subscriptions"] });
      setModalOpen(false);
      setEditingSubscription(null);
      form.resetFields();
      setTestResult(null);
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (subId: string) =>
      api.davSubscriptions.davSubscriptionsDelete(vaultId, subId),
    onSuccess: () => {
      message.success(t("vault.dav_subscriptions.deleted"));
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "dav-subscriptions"] });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const syncMutation = useMutation({
    mutationFn: (subId: string) =>
      api.davSubscriptions.davSubscriptionsSyncCreate(vaultId, subId),
    onSuccess: () => {
      message.success(t("vault.dav_subscriptions.sync_triggered"));
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "dav-subscriptions"] });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const openCreateModal = () => {
    setEditingSubscription(null);
    setTestResult(null);
    form.resetFields();
    form.setFieldsValue({ sync_way: SYNC_WAY_PULL, frequency: 180 });
    setModalOpen(true);
  };

  const openEditModal = (record: DavSubscription) => {
    setEditingSubscription(record);
    setTestResult(null);
    form.resetFields();
    form.setFieldsValue({
      uri: record.uri,
      username: record.username,
      sync_way: record.sync_way,
      frequency: record.frequency,
      active: record.active,
    });
    setModalOpen(true);
  };

  const openLogsDrawer = (record: DavSubscription) => {
    setLogsSubscription(record);
    setLogsPage(1);
    setAccumulatedLogs([]);
    setLogsDrawerOpen(true);
  };

  const handleModalOk = async () => {
    const values = await form.validateFields();
    if (editingSubscription) {
      const data: UpdateDavSubscriptionRequest = {
        uri: values.uri,
        username: values.username,
        sync_way: values.sync_way,
        frequency: values.frequency,
        active: values.active,
      };
      if (values.password) {
        data.password = values.password;
      }
      updateMutation.mutate({ subId: editingSubscription.id!, data });
    } else {
      createMutation.mutate(values as CreateDavSubscriptionRequest);
    }
  };

  const handleTestConnection = async () => {
    try {
      const values = await form.validateFields(["uri", "username", "password"]);
      setTestLoading(true);
      setTestResult(null);
      const res = await api.davSubscriptions.davSubscriptionsTestCreate(vaultId, {
        uri: values.uri,
        username: values.username,
        password: values.password,
      });
      setTestResult(res.data as TestDavConnectionResponse);
    } catch {
      setTestResult({ success: false, error: "Validation failed" });
    } finally {
      setTestLoading(false);
    }
  };

  const syncWayLabel = (val?: number) => {
    switch (val) {
      case SYNC_WAY_PUSH: return t("vault.dav_subscriptions.sync_push");
      case SYNC_WAY_PULL: return t("vault.dav_subscriptions.sync_pull");
      case SYNC_WAY_BOTH: return t("vault.dav_subscriptions.sync_both");
      default: return "-";
    }
  };

  const syncWayColor = (val?: number) => {
    switch (val) {
      case SYNC_WAY_PUSH: return "orange";
      case SYNC_WAY_PULL: return "blue";
      case SYNC_WAY_BOTH: return "purple";
      default: return "default";
    }
  };

  const allLogs = accumulatedLogs;
  const logsMeta = logsData?.meta;
  const hasMoreLogs = logsMeta ? logsPage < (logsMeta.total_pages ?? 1) : false;

  return (
    <div style={{ maxWidth: 1000, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 24 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate(`/vaults/${vaultId}`)}
            style={{ color: token.colorTextSecondary }}
          />
          <CloudServerOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
          <Title level={4} style={{ margin: 0 }}>{t("vault.dav_subscriptions.title")}</Title>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
          {t("vault.dav_subscriptions.add")}
        </Button>
      </div>

      <Card
        title={t("vault.dav_subscriptions.server_urls")}
        size="small"
        style={{ marginBottom: 24 }}
      >
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <div>
            <Text type="secondary" style={{ fontSize: 12, display: "block", marginBottom: 4 }}>
              {t("vault.dav_subscriptions.carddav_url")}
            </Text>
            <Paragraph
              copyable={{ tooltips: [t("common.copy"), t("vault.dav_subscriptions.copy_success")] }}
              style={{ margin: 0, background: token.colorFillQuaternary, padding: "6px 12px", borderRadius: token.borderRadius, fontFamily: "monospace", fontSize: 13 }}
            >
              {`${window.location.origin}/dav/addressbooks/${user?.email ?? ""}/`}
            </Paragraph>
          </div>
          <div>
            <Text type="secondary" style={{ fontSize: 12, display: "block", marginBottom: 4 }}>
              {t("vault.dav_subscriptions.caldav_url")}
            </Text>
            <Paragraph
              copyable={{ tooltips: [t("common.copy"), t("vault.dav_subscriptions.copy_success")] }}
              style={{ margin: 0, background: token.colorFillQuaternary, padding: "6px 12px", borderRadius: token.borderRadius, fontFamily: "monospace", fontSize: 13 }}
            >
              {`${window.location.origin}/dav/calendars/${user?.email ?? ""}/`}
            </Paragraph>
          </div>
          <Alert
            type="info"
            showIcon
            message={t("vault.dav_subscriptions.auth_note")}
            description={t("vault.dav_subscriptions.auth_note_text")}
            style={{ marginTop: 4 }}
          />
        </div>
      </Card>

      {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
      <Table<any>
        dataSource={subscriptions}
        rowKey="id"
        loading={isLoading}
        pagination={false}
        locale={{ emptyText: <Empty description={t("vault.dav_subscriptions.no_subscriptions")} /> }}
        columns={[
          {
            title: t("vault.dav_subscriptions.uri"),
            dataIndex: "uri",
            key: "uri",
            ellipsis: true,
            render: (text: string) => (
              <Tooltip title={text}>
                <Text strong style={{ maxWidth: 250, display: "inline-block" }}>{text}</Text>
              </Tooltip>
            ),
          },
          {
            title: t("vault.dav_subscriptions.username"),
            dataIndex: "username",
            key: "username",
          },
          {
            title: t("vault.dav_subscriptions.sync_way"),
            dataIndex: "sync_way",
            key: "sync_way",
            render: (val: number) => <Tag color={syncWayColor(val)}>{syncWayLabel(val)}</Tag>,
          },
          {
            title: t("vault.dav_subscriptions.frequency"),
            dataIndex: "frequency",
            key: "frequency",
            render: (val: number) => t("vault.dav_subscriptions.frequency_minutes", { count: val }),
          },
          {
            title: t("vault.dav_subscriptions.status"),
            dataIndex: "active",
            key: "active",
            render: (val: boolean) => (
              <Tag
                color={val ? "success" : "error"}
                icon={val ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
              >
                {val ? t("common.active") : t("common.disabled")}
              </Tag>
            ),
          },
          {
            title: t("vault.dav_subscriptions.last_synced"),
            dataIndex: "last_synchronized_at",
            key: "last_synced",
            render: (val: string) =>
              val ? dayjs(val).format("YYYY-MM-DD HH:mm") : t("vault.dav_subscriptions.never_synced"),
          },
          {
            title: "",
            key: "actions",
            width: 180,
            render: (_: unknown, record: DavSubscription) => (
              <Space size="small">
                <Tooltip title={t("vault.dav_subscriptions.sync_now")}>
                  <Button
                    type="text"
                    size="small"
                    icon={<PlayCircleOutlined />}
                    loading={syncMutation.isPending}
                    onClick={() => syncMutation.mutate(record.id!)}
                  />
                </Tooltip>
                <Tooltip title={t("vault.dav_subscriptions.sync_logs")}>
                  <Button
                    type="text"
                    size="small"
                    icon={<FileTextOutlined />}
                    onClick={() => openLogsDrawer(record)}
                  />
                </Tooltip>
                <Tooltip title={t("vault.dav_subscriptions.edit")}>
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    onClick={() => openEditModal(record)}
                  />
                </Tooltip>
                <Popconfirm
                  title={t("vault.dav_subscriptions.delete_confirm")}
                  onConfirm={() => deleteMutation.mutate(record.id!)}
                >
                  <Button
                    type="text"
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                  />
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title={editingSubscription ? t("vault.dav_subscriptions.edit") : t("vault.dav_subscriptions.add")}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false);
          setEditingSubscription(null);
          setTestResult(null);
          form.resetFields();
        }}
        onOk={handleModalOk}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="uri"
            label={t("vault.dav_subscriptions.uri")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Input placeholder={t("vault.dav_subscriptions.uri_placeholder")} />
          </Form.Item>
          <Form.Item
            name="username"
            label={t("vault.dav_subscriptions.username")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="password"
            label={t("vault.dav_subscriptions.password")}
            rules={editingSubscription ? [] : [{ required: true, message: t("common.required") }]}
          >
            <Input.Password
              placeholder={editingSubscription ? t("vault.dav_subscriptions.password_keep") : undefined}
            />
          </Form.Item>
          <Form.Item
            name="sync_way"
            label={t("vault.dav_subscriptions.sync_way")}
            initialValue={SYNC_WAY_PULL}
          >
            <Select>
              <Select.Option value={SYNC_WAY_PULL}>{t("vault.dav_subscriptions.sync_pull")}</Select.Option>
              <Select.Option value={SYNC_WAY_PUSH}>{t("vault.dav_subscriptions.sync_push")}</Select.Option>
              <Select.Option value={SYNC_WAY_BOTH}>{t("vault.dav_subscriptions.sync_both")}</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="frequency"
            label={t("vault.dav_subscriptions.frequency")}
            initialValue={180}
          >
            <Select>
              {FREQUENCY_OPTIONS.map((mins) => (
                <Select.Option key={mins} value={mins}>
                  {t("vault.dav_subscriptions.frequency_minutes", { count: mins })}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          {editingSubscription && (
            <Form.Item
              name="active"
              label={t("vault.dav_subscriptions.status")}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          )}

          <Button
            icon={<SyncOutlined spin={testLoading} />}
            loading={testLoading}
            onClick={handleTestConnection}
            style={{ marginBottom: 16 }}
          >
            {t("vault.dav_subscriptions.test_connection")}
          </Button>

          {testResult && (
            <Alert
              type={testResult.success ? "success" : "error"}
              showIcon
              message={
                testResult.success
                  ? t("vault.dav_subscriptions.test_success")
                  : t("vault.dav_subscriptions.test_failed")
              }
              description={
                testResult.success
                  ? t("vault.dav_subscriptions.address_books_found", {
                      count: testResult.address_books?.length ?? 0,
                    })
                  : testResult.error
              }
              style={{ marginBottom: 0 }}
            />
          )}
        </Form>
      </Modal>

      <Drawer
        title={`${t("vault.dav_subscriptions.sync_logs")} - ${logsSubscription?.uri ?? ""}`}
        placement="right"
        width={700}
        open={logsDrawerOpen}
        onClose={() => {
          setLogsDrawerOpen(false);
          setLogsSubscription(null);
          setLogsPage(1);
          setAccumulatedLogs([]);
        }}
      >
        {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
        <Table<any>
          dataSource={allLogs}
          rowKey="id"
          pagination={false}
          locale={{ emptyText: <Empty description={t("vault.dav_subscriptions.no_logs")} /> }}
          columns={[
            {
              title: t("vault.dav_subscriptions.log_time"),
              dataIndex: "created_at",
              key: "created_at",
              width: 160,
              render: (val: string) => val ? dayjs(val).format("YYYY-MM-DD HH:mm:ss") : "-",
            },
            {
              title: t("vault.dav_subscriptions.log_action"),
              dataIndex: "action",
              key: "action",
              width: 140,
              render: (val: string) => (
                <Tag color={LOG_ACTION_COLORS[val] ?? "default"}>{val}</Tag>
              ),
            },
            {
              title: t("vault.dav_subscriptions.log_contact"),
              dataIndex: "contact_id",
              key: "contact_id",
              ellipsis: true,
              render: (val: string) => val ? (
                <Tooltip title={val}>
                  <a onClick={() => navigate(`/vaults/${vaultId}/contacts/${val}`)}>
                    {val.slice(0, 8)}...
                  </a>
                </Tooltip>
              ) : "-",
            },
            {
              title: t("vault.dav_subscriptions.log_remote_uri"),
              dataIndex: "distant_uri",
              key: "distant_uri",
              ellipsis: true,
              render: (val: string) => val ? (
                <Tooltip title={val}>
                  <Text style={{ maxWidth: 150, display: "inline-block" }}>{val}</Text>
                </Tooltip>
              ) : "-",
            },
            {
              title: t("vault.dav_subscriptions.log_error"),
              dataIndex: "error",
              key: "error",
              ellipsis: true,
              render: (val: string) => val ? (
                <Tooltip title={val}>
                  <Text type="danger" style={{ maxWidth: 150, display: "inline-block" }}>{val}</Text>
                </Tooltip>
              ) : "-",
            },
          ]}
        />
        {hasMoreLogs && (
          <div style={{ textAlign: "center", marginTop: 16 }}>
            <Button onClick={() => setLogsPage((p) => p + 1)}>
              {t("common.load_more")}
            </Button>
          </div>
        )}
      </Drawer>
    </div>
  );
}
