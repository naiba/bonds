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
} from "antd";
import { PlusOutlined, DeleteOutlined, UserOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { relationshipsApi } from "@/api/relationships";
import { contactsApi } from "@/api/contacts";
import type { Relationship } from "@/types/modules";
import type { Contact } from "@/types/contact";
import type { APIError } from "@/types/api";
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
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["vaults", vaultId, "contacts", contactId, "relationships"];

  const { data: relationships = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await relationshipsApi.list(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const { data: contacts = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts"],
    queryFn: async () => {
      const res = await contactsApi.list(vaultId);
      return res.data.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: (values: { related_contact_id: number; relationship_type: string }) =>
      relationshipsApi.create(vaultId, contactId, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      form.resetFields();
      message.success(t("modules.relationships.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => relationshipsApi.delete(vaultId, contactId, id),
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
      title={t("modules.relationships.title")}
      extra={
        <Button type="link" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          {t("modules.relationships.add")}
        </Button>
      }
    >
      <List
        loading={isLoading}
        dataSource={relationships}
        locale={{ emptyText: <Empty description={t("modules.relationships.no_relationships")} /> }}
        renderItem={(r: Relationship) => (
          <List.Item
            actions={[
              <Popconfirm key="d" title={t("modules.relationships.remove_confirm")} onConfirm={() => deleteMutation.mutate(r.id)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={<UserOutlined style={{ fontSize: 20 }} />}
              title={r.related_contact_name}
              description={<Tag color="blue">{r.relationship_type}</Tag>}
            />
          </List.Item>
        )}
      />

      <Modal
        title={t("modules.relationships.modal_title")}
        open={open}
        onCancel={() => { setOpen(false); form.resetFields(); }}
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
