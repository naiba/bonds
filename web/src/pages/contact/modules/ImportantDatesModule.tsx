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
import { importantDatesApi } from "@/api/importantDates";
import type { ImportantDate } from "@/types/modules";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const typeColor: Record<string, string> = {
  birthday: "magenta",
  anniversary: "gold",
  death: "default",
  first_met: "cyan",
  other: "blue",
};

export default function ImportantDatesModule({
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
  const qk = ["vaults", vaultId, "contacts", contactId, "important-dates"];

  const dateTypes = [
    { value: "birthday", label: t("modules.important_dates.type_birthday") },
    { value: "anniversary", label: t("modules.important_dates.type_anniversary") },
    { value: "death", label: t("modules.important_dates.type_death") },
    { value: "first_met", label: t("modules.important_dates.type_first_met") },
    { value: "other", label: t("modules.important_dates.type_other") },
  ];

  const { data: dates = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await importantDatesApi.list(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: (values: { label: string; date: dayjs.Dayjs; type: string }) => {
      const data = {
        label: values.label,
        date: values.date.format("YYYY-MM-DD"),
        type: values.type,
      };
      if (editingId) {
        return importantDatesApi.update(vaultId, contactId, editingId, data);
      }
      return importantDatesApi.create(vaultId, contactId, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      closeModal();
      message.success(editingId ? t("modules.important_dates.updated") : t("modules.important_dates.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => importantDatesApi.delete(vaultId, contactId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.important_dates.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function openEdit(d: ImportantDate) {
    setEditingId(d.id);
    form.setFieldsValue({ label: d.label, date: dayjs(d.date), type: d.type });
    setOpen(true);
  }

  function closeModal() {
    setOpen(false);
    setEditingId(null);
    form.resetFields();
  }

  return (
    <Card
      title={t("modules.important_dates.title")}
      extra={
        <Button type="link" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          {t("modules.important_dates.add")}
        </Button>
      }
    >
      <List
        loading={isLoading}
        dataSource={dates}
        locale={{ emptyText: <Empty description={t("modules.important_dates.no_dates")} /> }}
        renderItem={(d: ImportantDate) => (
          <List.Item
            actions={[
              <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => openEdit(d)} />,
              <Popconfirm key="d" title={t("modules.important_dates.delete_confirm")} onConfirm={() => deleteMutation.mutate(d.id)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={d.label}
              description={
                <>
                  {dayjs(d.date).format("MMM D, YYYY")}{" "}
                  <Tag color={typeColor[d.type] ?? "default"}>{d.type}</Tag>
                </>
              }
            />
          </List.Item>
        )}
      />

      <Modal
        title={editingId ? t("modules.important_dates.modal_edit") : t("modules.important_dates.modal_add")}
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => saveMutation.mutate(v)}>
          <Form.Item name="label" label={t("modules.important_dates.label")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="date" label={t("modules.important_dates.date")} rules={[{ required: true }]}>
            <DatePicker style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="type" label={t("modules.important_dates.type")} rules={[{ required: true }]}>
            <Select options={dateTypes} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
