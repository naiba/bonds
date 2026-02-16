import { useState } from "react";
import {
  Card,
  List,
  Button,
  Modal,
  Form,
  Select,
  Popconfirm,
  App,
  Tag,
  Empty,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined, UserOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Relationship, Contact, APIError } from "@/api";
import { useTranslation } from "react-i18next";

const relationshipTypes = [
  "partner",
  "spouse",
  "child",
  "parent",
  "sibling",
  "friend",
  "colleague",
  "mentor",
  "other",
];

export default function RelationshipsModule({
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
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "relationships"];

  const { data: relationships = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.relationships.contactsRelationshipsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const { data: contacts = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts"],
    queryFn: async () => {
      const res = await api.contacts.contactsList(String(vaultId));
      return res.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: (values: { related_contact_id: string; relationship_type_id: number }) => {
      if (editingId) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        return api.relationships.contactsRelationshipsUpdate(String(vaultId), String(contactId), editingId, values as any);
      }
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return api.relationships.contactsRelationshipsCreate(String(vaultId), String(contactId), values as any);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      setEditingId(null);
      form.resetFields();
      message.success(editingId ? t("modules.relationships.updated") : t("modules.relationships.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.relationships.contactsRelationshipsDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.relationships.removed"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const contactOptions = contacts
    .filter((c: Contact) => c.id !== contactId)
    .map((c: Contact) => ({
      value: c.id,
      label: `${c.first_name} ${c.last_name}`.trim(),
    }));

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.relationships.title")}</span>}
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
          {t("modules.relationships.add")}
        </Button>
      }
    >
      <List
        loading={isLoading}
        dataSource={relationships}
        locale={{ emptyText: <Empty description={t("modules.relationships.no_relationships")} /> }}
        split={false}
        renderItem={(r: Relationship) => (
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
                  setEditingId(r.id!);
                  form.setFieldsValue({
                    related_contact_id: r.related_contact_id,
                    relationship_type: r.relationship_type?.name,
                  });
                  setOpen(true);
                }}
              />,
              <Popconfirm key="d" title={t("modules.relationships.remove_confirm")} onConfirm={() => deleteMutation.mutate(r.id!)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={<UserOutlined style={{ fontSize: 18, color: token.colorPrimary }} />}
              title={<span style={{ fontWeight: 500 }}>{r.related_contact?.first_name} {r.related_contact?.last_name}</span>}
              description={<Tag color="blue">{r.relationship_type ? r.relationship_type.name : ''}</Tag>}
            />
          </List.Item>
        )}
      />

      <Modal
        title={editingId ? t("modules.relationships.edit") : t("modules.relationships.modal_title")}
        open={open}
        onCancel={() => { setOpen(false); setEditingId(null); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => createMutation.mutate(v)}>
          <Form.Item name="related_contact_id" label={t("modules.relationships.contact")} rules={[{ required: true }]}>
            <Select
              showSearch
              options={contactOptions}
              filterOption={(input, option) =>
                (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
              }
              placeholder={t("modules.relationships.select_contact")}
            />
          </Form.Item>
          <Form.Item name="relationship_type" label={t("modules.relationships.relationship_type")} rules={[{ required: true }]}>
            <Select
              options={relationshipTypes.map((t) => ({ value: t, label: t.charAt(0).toUpperCase() + t.slice(1) }))}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
