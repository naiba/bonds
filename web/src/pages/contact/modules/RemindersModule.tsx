import { useState } from "react";
import {
  Card,
  List,
  Button,
  Modal,
  Form,
  Input,
  DatePicker,
  Select,
  Popconfirm,
  App,
  Tag,
  Empty,
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { remindersApi } from "@/api/reminders";
import type { Reminder } from "@/types/modules";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const freqColor: Record<string, string> = {
  one_time: "blue",
  weekly: "green",
  monthly: "orange",
  yearly: "purple",
};

export default function RemindersModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["vaults", vaultId, "contacts", contactId, "reminders"];

  const frequencyOptions = [
    { value: "one_time", label: t("modules.reminders.freq_one_time") },
    { value: "weekly", label: t("modules.reminders.freq_weekly") },
    { value: "monthly", label: t("modules.reminders.freq_monthly") },
    { value: "yearly", label: t("modules.reminders.freq_yearly") },
  ];

  const { data: reminders = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await remindersApi.list(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: (values: { label: string; date: dayjs.Dayjs; frequency: string }) => {
      const data = {
        label: values.label,
        date: values.date.format("YYYY-MM-DD"),
        frequency: values.frequency,
      };
      if (editingId) {
        return remindersApi.update(vaultId, contactId, editingId, data);
      }
      return remindersApi.create(vaultId, contactId, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      closeModal();
      message.success(editingId ? t("modules.reminders.updated") : t("modules.reminders.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => remindersApi.delete(vaultId, contactId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.reminders.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function openEdit(r: Reminder) {
    setEditingId(r.id);
    form.setFieldsValue({
      label: r.label,
      date: dayjs(r.date),
      frequency: r.frequency,
    });
    setOpen(true);
  }

  function closeModal() {
    setOpen(false);
    setEditingId(null);
    form.resetFields();
  }

  return (
    <Card
      title={t("modules.reminders.title")}
      extra={
        <Button type="link" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          {t("modules.reminders.add")}
        </Button>
      }
    >
      <List
        loading={isLoading}
        dataSource={reminders}
        locale={{ emptyText: <Empty description={t("modules.reminders.no_reminders")} /> }}
        renderItem={(r: Reminder) => (
          <List.Item
            actions={[
              <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => openEdit(r)} />,
              <Popconfirm key="d" title={t("modules.reminders.delete_confirm")} onConfirm={() => deleteMutation.mutate(r.id)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={r.label}
              description={
                <>
                  {dayjs(r.date).format("MMM D, YYYY")}{" "}
                  <Tag color={freqColor[r.frequency] ?? "default"}>{r.frequency}</Tag>
                </>
              }
            />
          </List.Item>
        )}
      />

      <Modal
        title={editingId ? t("modules.reminders.modal_edit") : t("modules.reminders.modal_add")}
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => saveMutation.mutate(v)}>
          <Form.Item name="label" label={t("modules.reminders.label")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="date" label={t("modules.reminders.date")} rules={[{ required: true }]}>
            <DatePicker style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="frequency" label={t("modules.reminders.frequency")} rules={[{ required: true }]}>
            <Select options={frequencyOptions} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
