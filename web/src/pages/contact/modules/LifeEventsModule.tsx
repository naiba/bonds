import { useState } from "react";
import {
  Card,
  Button,
  Modal,
  Form,
  Input,
  DatePicker,
  Popconfirm,
  App,
  Empty,
  Timeline,
  Typography,
  Collapse,
  Space,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { TimelineEvent as TEvent, APIError } from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Text } = Typography;

export default function LifeEventsModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [tlOpen, setTlOpen] = useState(false);
  const [leOpen, setLeOpen] = useState(false);
  const [selectedTimeline, setSelectedTimeline] = useState<number | null>(null);
  const [tlForm] = Form.useForm();
  const [leForm] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "timelineEvents"];

  const { data: timelines = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.lifeEvents.contactsTimelineEventsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const createTimelineMutation = useMutation({
    mutationFn: (values: { label: string; started_at: dayjs.Dayjs }) =>
      api.lifeEvents.contactsTimelineEventsCreate(String(vaultId), String(contactId), {
        label: values.label,
        started_at: values.started_at.format("YYYY-MM-DD"),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setTlOpen(false);
      tlForm.resetFields();
      message.success(t("modules.life_events.timeline_created"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteTimelineMutation = useMutation({
    mutationFn: (id: number) => api.lifeEvents.contactsTimelineEventsDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.life_events.timeline_deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const createLifeEventMutation = useMutation({
    mutationFn: (values: { label: string; happened_at: dayjs.Dayjs; description?: string }) => {
      if (!selectedTimeline) throw new Error("No timeline");
      return api.lifeEvents.contactsTimelineEventsLifeEventsCreate(String(vaultId), String(contactId), selectedTimeline, {
        summary: values.label,
        happened_at: values.happened_at.format("YYYY-MM-DD"),
        description: values.description,
        life_event_type_id: 1,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setLeOpen(false);
      leForm.resetFields();
      message.success(t("modules.life_events.event_added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteLifeEventMutation = useMutation({
    mutationFn: ({ timelineId, lifeEventId }: { timelineId: number; lifeEventId: number }) =>
      api.lifeEvents.contactsTimelineEventsLifeEventsDelete(String(vaultId), String(contactId), timelineId, lifeEventId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.life_events.event_deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  if (isLoading) return <Card loading />;

  const collapseItems = timelines.map((tl: TEvent) => ({
    key: tl.id,
    label: (
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <span style={{ fontWeight: 500 }}>{tl.label} — <span style={{ color: token.colorTextSecondary, fontWeight: 400 }}>{dayjs(tl.started_at).format("MMM YYYY")}</span></span>
        <Space>
          <Button
            type="text"
            size="small"
            icon={<PlusOutlined />}
            style={{ color: token.colorPrimary }}
            onClick={(e) => {
              e.stopPropagation();
               setSelectedTimeline(tl.id ?? null);
               setLeOpen(true);
            }}
          >
            {t("modules.life_events.event")}
          </Button>
          <Popconfirm
            title={t("modules.life_events.delete_timeline_confirm")}
            onConfirm={(e) => {
              e?.stopPropagation();
               deleteTimelineMutation.mutate(tl.id!);
            }}
            onCancel={(e) => e?.stopPropagation()}
          >
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={(e) => e.stopPropagation()}
            />
          </Popconfirm>
        </Space>
      </div>
    ),
    children: tl.life_events?.length ? (
      <Timeline
        items={tl.life_events.map((le) => ({
          color: token.colorPrimary,
          children: (
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
              <div>
                <strong style={{ fontWeight: 500 }}>{le.summary ?? le.description}</strong>
                <br />
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {dayjs(le.happened_at).format("MMM D, YYYY")}
                </Text>
                {le.description && (
                  <div style={{ marginTop: 4, color: token.colorTextSecondary }}>{le.description}</div>
                )}
              </div>
              <Popconfirm
                title={t("modules.life_events.delete_event_confirm")}
                onConfirm={() =>
                  deleteLifeEventMutation.mutate({
                    timelineId: tl.id!,
                    lifeEventId: le.id!,
                  })
                }
              >
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>
            </div>
          ),
        }))}
      />
    ) : (
      <Empty description={t("modules.life_events.no_events")} image={Empty.PRESENTED_IMAGE_SIMPLE} />
    ),
  }));

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.life_events.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        <Button type="text" icon={<PlusOutlined />} onClick={() => setTlOpen(true)} style={{ color: token.colorPrimary }}>
          {t("modules.life_events.new_timeline")}
        </Button>
      }
    >
      {timelines.length === 0 ? (
        <Empty description={t("modules.life_events.no_timelines")} />
      ) : (
        <Collapse items={collapseItems} />
      )}

      <Modal
        title={t("modules.life_events.timeline_modal")}
        open={tlOpen}
        onCancel={() => { setTlOpen(false); tlForm.resetFields(); }}
        onOk={() => tlForm.submit()}
        confirmLoading={createTimelineMutation.isPending}
      >
        <Form form={tlForm} layout="vertical" onFinish={(v) => createTimelineMutation.mutate(v)}>
          <Form.Item name="label" label={t("modules.life_events.label")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="started_at" label={t("modules.life_events.started_at")} rules={[{ required: true }]}>
            <DatePicker style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t("modules.life_events.event_modal")}
        open={leOpen}
        onCancel={() => { setLeOpen(false); leForm.resetFields(); }}
        onOk={() => leForm.submit()}
        confirmLoading={createLifeEventMutation.isPending}
      >
        <Form form={leForm} layout="vertical" onFinish={(v) => createLifeEventMutation.mutate(v)}>
          <Form.Item name="label" label={t("modules.life_events.label")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="happened_at" label={t("modules.life_events.happened_at")} rules={[{ required: true }]}>
            <DatePicker style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="description" label={t("common.description")}>
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
