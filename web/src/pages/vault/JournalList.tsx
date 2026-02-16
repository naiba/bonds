import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Typography,
  Button,
  List,
  Modal,
  Form,
  Input,
  Popconfirm,
  App,
  Empty,
  Spin,
  theme,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  ArrowLeftOutlined,
  BookOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Journal, APIError } from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title, Text } = Typography;

export default function JournalList() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "journals"];

  const { data: journals = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.journals.journalsList(String(vaultId));
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  const createMutation = useMutation({
    mutationFn: (values: { name: string; description?: string }) =>
      api.journals.journalsCreate(String(vaultId), values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      form.resetFields();
      message.success(t("vault.journals.created_success"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (journalId: number) => api.journals.journalsDelete(String(vaultId), Number(journalId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("vault.journals.deleted_success"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateMutation = useMutation({
    mutationFn: (values: { name: string; description?: string }) =>
      api.journals.journalsUpdate(String(vaultId), editingId!, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      setEditingId(null);
      form.resetFields();
      message.success(t("vault.journals.updated"));
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
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 24 }}>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <BookOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0, flex: 1 }}>{t("vault.journals.title")}</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          {t("vault.journals.new_journal")}
        </Button>
      </div>

      <div
        style={{
          background: token.colorBgContainer,
          borderRadius: token.borderRadiusLG,
          boxShadow: token.boxShadowTertiary,
          padding: "8px 0",
        }}
      >
        <List
          dataSource={journals}
          locale={{ emptyText: <Empty description={t("vault.journals.no_journals")} style={{ padding: 32 }} /> }}
          renderItem={(journal: Journal) => (
            <List.Item
              style={{
                borderLeft: `3px solid ${token.colorPrimary}`,
                marginLeft: 16,
                marginRight: 16,
                marginBottom: 8,
                paddingLeft: 16,
                borderRadius: `0 ${token.borderRadius}px ${token.borderRadius}px 0`,
                background: token.colorFillQuaternary,
                cursor: "pointer",
              }}
              actions={[
                <Button
                  key="e"
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={(e) => {
                    e.stopPropagation();
                    setEditingId(journal.id!);
                    form.setFieldsValue({ name: journal.name, description: journal.description });
                    setOpen(true);
                  }}
                />,
                <Popconfirm
                  key="d"
                  title={t("vault.journals.delete_confirm")}
                  onConfirm={(e) => { e?.stopPropagation(); deleteMutation.mutate(journal.id!); }}
                >
                  <Button
                    type="text"
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={(e) => e.stopPropagation()}
                  />
                </Popconfirm>,
              ]}
              onClick={() => navigate(`/vaults/${vaultId}/journals/${journal.id}`)}
            >
              <List.Item.Meta
                title={
                  <Text strong style={{ fontSize: 16 }}>
                    {journal.name}
                  </Text>
                }
                description={
                  <>
                    {journal.description && (
                      <Text type="secondary" style={{ display: "block", marginBottom: 4 }}>
                        {journal.description}
                      </Text>
                    )}
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      Created {dayjs(journal.created_at).format("MMM D, YYYY")}
                    </Text>
                  </>
                }
              />
            </List.Item>
          )}
        />
      </div>

      <Modal
        title={editingId ? t("vault.journals.edit_title") : t("vault.journals.modal_title")}
        open={open}
        onCancel={() => { setOpen(false); setEditingId(null); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={editingId ? updateMutation.isPending : createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => editingId ? updateMutation.mutate(v) : createMutation.mutate(v)}>
          <Form.Item name="name" label={t("common.name")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label={t("common.description")}>
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
