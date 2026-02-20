import { useState } from "react";
import {
  Card,
  Typography,
  Button,
  List,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  Popconfirm,
  App,
  Tag,
  Empty,
  Spin,
  theme,
  Drawer,
  Space,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  MailOutlined,
  SendOutlined,
  ApiOutlined,
  HistoryOutlined,
  ThunderboltOutlined,
  CheckCircleOutlined,
  SafetyCertificateOutlined,
  NotificationOutlined,
  BellOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { NotificationChannel, APIError } from "@/api";
import type { GithubComNaibaBondsInternalDtoNotificationLogResponse as NotificationLog } from "@/api/generated/data-contracts";

const { Title, Text } = Typography;

const channelTypes = [
  { label: "Email", value: "email" },
  { label: "Telegram", value: "telegram" },
  { label: "Ntfy", value: "ntfy" },
  { label: "Gotify", value: "gotify" },
  { label: "Webhook", value: "webhook" },
];

const channelIconMap: Record<
  string,
  { icon: React.ReactNode; color: string; bg: string }
> = {
  email: { icon: <MailOutlined />, color: "#1677ff", bg: "#e6f4ff" },
  telegram: { icon: <SendOutlined />, color: "#0088cc", bg: "#e6f7ff" },
  ntfy: { icon: <NotificationOutlined />, color: "#52c41a", bg: "#f6ffed" },
  gotify: { icon: <BellOutlined />, color: "#fa8c16", bg: "#fff7e6" },
  webhook: { icon: <ApiOutlined />, color: "#722ed1", bg: "#f9f0ff" },
};

export default function Notifications() {
  const [open, setOpen] = useState(false);
  const [logsChannelId, setLogsChannelId] = useState<number | null>(null);
  const [verifyingChannelId, setVerifyingChannelId] = useState<number | null>(null);
  const [verifyToken, setVerifyToken] = useState("");
  const [form] = Form.useForm();
  const selectedType = Form.useWatch("type", form) as string | undefined;
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["settings", "notifications"];

  const { data: channels = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.notifications.notificationsList();
      return res.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: (values: { type: "email" | "telegram" | "ntfy" | "gotify" | "webhook"; label: string; content: string }) =>
      api.notifications.notificationsCreate(values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      form.resetFields();
      message.success(t("settings.notifications.created"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const toggleMutation = useMutation({
    mutationFn: (channel: NotificationChannel) =>
      api.notifications.notificationsToggleUpdate(channel.id!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: qk }),
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.notifications.notificationsDelete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("settings.notifications.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const testMutation = useMutation({
    mutationFn: (id: number) => api.notifications.notificationsTestCreate(id),
    onSuccess: () => message.success(t("settings.notifications.test_success")),
    onError: (e: APIError) => message.error(e.message),
  });

  const verifyMutation = useMutation({
    mutationFn: ({ id, token }: { id: number; token: string }) =>
      api.notifications.notificationsVerifyDetail(id, token),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setVerifyingChannelId(null);
      setVerifyToken("");
      message.success(t("settings.notifications.verify_success"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const { data: logs = [] } = useQuery({
    queryKey: ["notifications", "logs", logsChannelId],
    queryFn: async () => {
      const res = await api.notifications.notificationsLogsList(logsChannelId!);
      return (res.data ?? []) as NotificationLog[];
    },
    enabled: logsChannelId !== null,
  });

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 640, margin: "0 auto" }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          marginBottom: 24,
        }}
      >
        <div>
          <Title level={4} style={{ marginBottom: 4 }}>
            {t("settings.notifications.title")}
          </Title>
          <Text type="secondary">
            {t("settings.notifications.description")}
          </Text>
        </div>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setOpen(true)}
          style={{ flexShrink: 0, marginTop: 4 }}
        >
          {t("settings.notifications.add_channel")}
        </Button>
      </div>

      <Card>
        <List
          dataSource={channels}
          locale={{
            emptyText: (
              <Empty
                description={t("settings.notifications.no_channels")}
              />
            ),
          }}
          renderItem={(ch: NotificationChannel) => {
            const iconInfo = channelIconMap[ch.type!] ?? {
              icon: <ApiOutlined />,
              color: token.colorTextSecondary,
              bg: token.colorBgLayout,
            };
            return (
              <List.Item
                actions={[
                  <Switch
                    key="toggle"
                    checked={ch.active}
                    onChange={() => toggleMutation.mutate(ch)}
                  />,
                  ...(!ch.verified_at
                    ? [
                        <Button
                          key="verify"
                          type="text"
                          size="small"
                          icon={<SafetyCertificateOutlined />}
                          onClick={() => {
                            setVerifyingChannelId(ch.id!);
                            setVerifyToken("");
                          }}
                          title={t("settings.notifications.verify_action")}
                        />,
                      ]
                    : []),
                  <Button
                    key="test"
                    type="text"
                    size="small"
                    icon={<ThunderboltOutlined />}
                    loading={testMutation.isPending && testMutation.variables === ch.id}
                    onClick={() => testMutation.mutate(ch.id!)}
                    title={t("settings.notifications.test_send")}
                  />,
                  <Button
                    key="logs"
                    type="text"
                    size="small"
                    icon={<HistoryOutlined />}
                    onClick={() => setLogsChannelId(ch.id!)}
                    title={t("settings.notifications.view_logs")}
                  />,
                  <Popconfirm
                    key="d"
                    title={t("settings.notifications.delete_confirm")}
                    onConfirm={() => deleteMutation.mutate(ch.id!)}
                  >
                    <Button
                      type="text"
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                    />
                  </Popconfirm>,
                ]}
              >
                <List.Item.Meta
                  avatar={
                    <div
                      style={{
                        width: 40,
                        height: 40,
                        borderRadius: 10,
                        background: iconInfo.bg,
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        fontSize: 18,
                        color: iconInfo.color,
                      }}
                    >
                      {iconInfo.icon}
                    </div>
                  }
                  title={
                    <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
                      {ch.label}
                      <Tag style={{ marginInlineStart: 0 }}>{ch.type}</Tag>
                      {!ch.active && (
                        <Tag color="default">{t("common.disabled")}</Tag>
                      )}
                      {ch.verified_at ? (
                        <Tag color="success" icon={<CheckCircleOutlined />}>{t("settings.notifications.verified")}</Tag>
                      ) : (
                        <Tag color="warning">{t("settings.notifications.unverified")}</Tag>
                      )}
                    </span>
                  }
                  description={ch.content}
                />
              </List.Item>
            );
          }}
        />
      </Card>

      <Modal
        title={t("settings.notifications.modal_title")}
        open={open}
        onCancel={() => {
          setOpen(false);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => createMutation.mutate(v)}
          initialValues={{ type: "email" }}
        >
          <Form.Item
            name="type"
            label={t("settings.notifications.type")}
            rules={[{ required: true }]}
          >
            <Select options={channelTypes} />
          </Form.Item>
          <Form.Item
            name="label"
            label={t("settings.notifications.label")}
            rules={[{ required: true }]}
          >
            <Input placeholder={t("settings.notifications.label_placeholder")} />
          </Form.Item>
          <Form.Item
            name="content"
            label={
              selectedType === "email"
                ? t("settings.notifications.destination")
                : t("settings.notifications.shoutrrr_url")
            }
            extra={
              selectedType === "telegram"
                ? t("settings.notifications.telegram_help")
                : selectedType === "ntfy"
                  ? t("settings.notifications.ntfy_help")
                  : selectedType === "gotify"
                    ? t("settings.notifications.gotify_help")
                    : selectedType === "webhook"
                      ? t("settings.notifications.webhook_help")
                      : null
            }
            rules={[{ required: true }]}
          >
            <Input
              placeholder={
                selectedType === "email"
                  ? "user@example.com"
                  : selectedType === "telegram"
                    ? "telegram://token@telegram?channels=chatid"
                    : selectedType === "ntfy"
                      ? "ntfy://ntfy.sh/your-topic"
                      : selectedType === "gotify"
                        ? "gotify://your-server/apptoken"
                        : selectedType === "webhook"
                          ? "generic+https://example.com/webhook"
                          : t("settings.notifications.destination_placeholder")
              }
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t("settings.notifications.verify_modal_title")}
        open={verifyingChannelId !== null}
        onCancel={() => {
          setVerifyingChannelId(null);
          setVerifyToken("");
        }}
        onOk={() => {
          if (verifyToken.trim() && verifyingChannelId) {
            verifyMutation.mutate({ id: verifyingChannelId, token: verifyToken.trim() });
          }
        }}
        confirmLoading={verifyMutation.isPending}
        okButtonProps={{ disabled: !verifyToken.trim() }}
      >
        <Input
          placeholder={t("settings.notifications.verify_token_placeholder")}
          value={verifyToken}
          onChange={(e) => setVerifyToken(e.target.value)}
          onPressEnter={() => {
            if (verifyToken.trim() && verifyingChannelId) {
              verifyMutation.mutate({ id: verifyingChannelId, token: verifyToken.trim() });
            }
          }}
          autoFocus
        />
      </Modal>

      <Drawer
        title={t("settings.notifications.logs_title")}
        open={logsChannelId !== null}
        onClose={() => setLogsChannelId(null)}
        width={480}
      >
        {logs.length === 0 ? (
          <Empty description={t("settings.notifications.no_logs")} />
        ) : (
          <List
            dataSource={logs}
            renderItem={(log: NotificationLog) => (
              <List.Item>
                <Space direction="vertical" size={2} style={{ width: "100%" }}>
                  <span style={{ fontSize: 12, color: token.colorTextSecondary }}>
                    {log.sent_at ? new Date(log.sent_at).toLocaleString() : log.created_at}
                  </span>
                  {log.subject_line && (
                    <span style={{ fontWeight: 500 }}>{log.subject_line}</span>
                  )}
                  {log.error ? (
                    <Tag color="error" style={{ whiteSpace: "normal", height: "auto" }}>
                      {log.error}
                    </Tag>
                  ) : (
                    <Tag color="success">{t("common.active")}</Tag>
                  )}
                </Space>
              </List.Item>
            )}
          />
        )}
      </Drawer>
    </div>
  );
}
