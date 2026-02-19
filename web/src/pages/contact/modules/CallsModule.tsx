import { useState, useCallback } from "react";
import {
  Card,
  List,
  Button,
  Modal,
  Form,
  Input,
  DatePicker,
  Select,
  InputNumber,
  Popconfirm,
  App,
  Tag,
  Empty,
  theme,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  PhoneOutlined,
  EditOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Call, PaginationMeta, APIError } from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const typeColor: Record<string, string> = {
  incoming: "green",
  outgoing: "blue",
  missed: "red",
};

export default function CallsModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [page, setPage] = useState(1);
  const [allCalls, setAllCalls] = useState<Call[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "calls"];

  const resetPagination = useCallback(() => {
    setPage(1);
    setAllCalls([]);
  }, []);

  const callTypes = [
    { value: "incoming", label: t("modules.calls.type_incoming") },
    { value: "outgoing", label: t("modules.calls.type_outgoing") },
    { value: "missed", label: t("modules.calls.type_missed") },
  ];

  const { isLoading, isFetching } = useQuery({
    queryKey: [...qk, page],
    queryFn: async () => {
      const res = await api.calls.contactsCallsList(String(vaultId), String(contactId), { page, per_page: 15 });
      const newItems = (res.data ?? []) as Call[];
      const meta = res.meta as PaginationMeta | undefined;
      setAllCalls(prev => page === 1 ? newItems : [...prev, ...newItems]);
      setHasMore(meta ? meta.page! < meta.total_pages! : newItems.length >= 15);
      return newItems;
    },
  });

  const saveMutation = useMutation({
    mutationFn: (values: {
      called_at: dayjs.Dayjs;
      duration?: number;
      type: string;
      description?: string;
    }) => {
      const data = {
        called_at: values.called_at.toISOString(),
        duration: values.duration,
        type: values.type,
        description: values.description,
        who_initiated: values.type === "outgoing" ? "me" : "contact",
      };
      
      if (editingId) {
        return api.calls.contactsCallsUpdate(String(vaultId), String(contactId), editingId, data);
      }
      return api.calls.contactsCallsCreate(String(vaultId), String(contactId), data);
    },
    onSuccess: () => {
      resetPagination();
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      setEditingId(null);
      form.resetFields();
      message.success(editingId ? t("modules.calls.updated") : t("modules.calls.logged"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.calls.contactsCallsDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      resetPagination();
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.calls.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.calls.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        <Button 
          type="text" 
          icon={<PlusOutlined />} 
          onClick={() => {
            setEditingId(null);
            form.resetFields();
            setOpen(true);
          }} 
          style={{ color: token.colorPrimary }}
        >
          {t("modules.calls.log_call")}
        </Button>
      }
    >
      <List
        loading={isLoading && page === 1}
        dataSource={allCalls}
        locale={{ emptyText: <Empty description={t("modules.calls.no_calls")} /> }}
        split={false}
        renderItem={(c: Call) => (
          <List.Item
            style={{
              borderRadius: token.borderRadius,
              padding: '10px 12px',
              marginBottom: 4,
              transition: 'background 0.2s',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.background = token.colorFillQuaternary; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; }}
            actions={[
              <Button
                key="edit"
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => {
                  setEditingId(c.id!);
                  form.setFieldsValue({
                    ...c,
                    called_at: dayjs(c.called_at),
                  });
                  setOpen(true);
                }}
              />,
              <Popconfirm key="d" title={t("modules.calls.delete_confirm")} onConfirm={() => deleteMutation.mutate(c.id!)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={<PhoneOutlined style={{ fontSize: 18, color: token.colorPrimary }} />}
              title={
                <>
                    <Tag color={typeColor[c.type!] ?? "default"}>{c.type}</Tag>
                  <span style={{ fontWeight: 400, color: token.colorTextSecondary }}>{dayjs(c.called_at).format("MMM D, YYYY h:mm A")}</span>
                </>
              }
              description={
                <span style={{ color: token.colorTextTertiary }}>
                  {c.duration != null && <span>{c.duration} min · </span>}
                  {c.description}
                </span>
              }
            />
          </List.Item>
        )}
      />
      {hasMore && allCalls.length > 0 && (
        <div style={{ textAlign: "center", marginTop: 12 }}>
          <Button onClick={() => setPage(p => p + 1)} loading={isFetching}>
            {t("common.load_more")}
          </Button>
        </div>
      )}

      <Modal
        title={editingId ? t("modules.calls.edit_call") : t("modules.calls.modal_title")}
        open={open}
        onCancel={() => { setOpen(false); setEditingId(null); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => saveMutation.mutate(v)}>
          <Form.Item name="called_at" label={t("modules.calls.date_time")} rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="type" label={t("modules.calls.type")} rules={[{ required: true }]}>
            <Select options={callTypes} />
          </Form.Item>
          <Form.Item name="duration" label={t("modules.calls.duration")}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="description" label={t("modules.calls.notes")}>
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
