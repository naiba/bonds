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
} from "antd";
import { PlusOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { settingsApi } from "@/api/settings";
import type { NotificationChannel } from "@/types/modules";
import type { APIError } from "@/types/api";

const { Title } = Typography;

const channelTypes = [
  { value: "email", label: "Email" },
  { value: "telegram", label: "Telegram" },
  { value: "slack", label: "Slack" },
];

export default function Notifications() {
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["settings", "notification-channels"];

  const { data: channels = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await settingsApi.listNotificationChannels();
      return res.data.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: (values: { type: string; label: string; content: string }) =>
      settingsApi.createNotificationChannel(values),
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
      settingsApi.updateNotificationChannel(channel.id, {
        active: !channel.active,
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: qk }),
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => settingsApi.deleteNotificationChannel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("settings.notifications.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
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
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>{t("settings.notifications.title")}</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          {t("settings.notifications.add_channel")}
        </Button>
      </div>

      <Card>
        <List
          dataSource={channels}
          locale={{ emptyText: <Empty description={t("settings.notifications.no_channels")} /> }}
          renderItem={(ch: NotificationChannel) => (
            <List.Item
              actions={[
                <Switch
                  key="toggle"
                  checked={ch.active}
                  onChange={() => toggleMutation.mutate(ch)}
                  size="small"
                />,
                <Popconfirm
                  key="d"
                  title={t("settings.notifications.delete_confirm")}
                  onConfirm={() => deleteMutation.mutate(ch.id)}
                >
                  <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>,
              ]}
            >
              <List.Item.Meta
                title={
                  <>
                    {ch.label} <Tag>{ch.type}</Tag>
                    {!ch.active && <Tag color="default">{t("common.disabled")}</Tag>}
                  </>
                }
                description={ch.content}
              />
            </List.Item>
          )}
        />
      </Card>

      <Modal
        title={t("settings.notifications.modal_title")}
        open={open}
        onCancel={() => { setOpen(false); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => createMutation.mutate(v)}>
          <Form.Item name="type" label={t("settings.notifications.type")} rules={[{ required: true }]}>
            <Select options={channelTypes} />
          </Form.Item>
          <Form.Item name="label" label={t("settings.notifications.label")} rules={[{ required: true }]}>
            <Input placeholder={t("settings.notifications.label_placeholder")} />
          </Form.Item>
          <Form.Item name="content" label={t("settings.notifications.destination")} rules={[{ required: true }]}>
            <Input placeholder={t("settings.notifications.destination_placeholder")} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
