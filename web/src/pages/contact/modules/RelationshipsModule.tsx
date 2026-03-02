import { useState, useMemo } from "react";
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
import NetworkGraph from "@/components/NetworkGraph";
import type { Relationship, APIError } from "@/api";
import type {
  GithubComNaibaBondsInternalDtoRelationshipTypeWithGroupResponse,
  GithubComNaibaBondsInternalDtoCrossVaultContactItem,
} from "@/api";
import { useTranslation } from "react-i18next";

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
  // Cross-vault contacts query: returns contacts from ALL accessible vaults
  const qk = ["vaults", vaultId, "contacts", contactId, "relationships"];

  const { data: relationships = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.relationships.contactsRelationshipsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const { data: crossVaultContacts = [] } = useQuery({
    queryKey: ["relationships", "contacts"],
    queryFn: async () => {
      const res = await api.relationships.contactsList();
      return (res.data ?? []) as GithubComNaibaBondsInternalDtoCrossVaultContactItem[];
    },
  });

  // BUG FIX: Previously fetched relationship GROUP types (Love/Family/Friend/Work) via
  // personalizeDetail("relationship-types"), which queries the relationship_group_types table.
  // Users could only pick a group, not a specific type (Parent/Child/Sibling), causing
  // wrong relationship_type_id to be stored and incorrect labels on the graph.
  // Now fetches all actual RelationshipType records with group names for grouped select.
  const { data: relationshipTypes = [] } = useQuery({
    queryKey: ["personalize", "relationship-types", "all"],
    queryFn: async () => {
      const res = await api.relationshipTypes.personalizeRelationshipTypesAllList();
      return (res.data ?? []) as GithubComNaibaBondsInternalDtoRelationshipTypeWithGroupResponse[];
    },
  });

  // Track whether the selected contact lacks editor permission (one-way only)
  const selectedContactId = Form.useWatch("related_contact_id", form);
  const selectedContactOneWay = useMemo(() => {
    if (!selectedContactId) return false;
    const c = crossVaultContacts.find((x) => x.contact_id === selectedContactId);
    return c ? c.has_editor === false : false;
  }, [selectedContactId, crossVaultContacts]);

  // Build grouped options for the relationship type Select (OptGroup by group name).
  const typeSelectOptions = useMemo(() => {
    const groups = new Map<string, { value: number; label: string }[]>();
    for (const rt of relationshipTypes) {
      const groupName = rt.group_name ?? "";
      if (!groups.has(groupName)) groups.set(groupName, []);
      groups.get(groupName)!.push({ value: rt.id!, label: rt.name ?? "" });
    }
    return Array.from(groups.entries()).map(([group, options]) => ({
      label: group,
      options,
    }));
  }, [relationshipTypes]);

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

  // Group contacts by vault for OptGroup display, append one-way suffix for non-editor contacts
  const contactOptions = useMemo(() => {
    const groups = new Map<string, { value: string | undefined; label: string }[]>();
    for (const c of crossVaultContacts) {
      if (c.contact_id === contactId) continue;
      const vaultName = c.vault_name ?? "";
      if (!groups.has(vaultName)) groups.set(vaultName, []);
      const suffix = c.has_editor === false ? ` · ${t("modules.relationships.one_way_only")}` : "";
      groups.get(vaultName)!.push({ value: c.contact_id, label: `${c.contact_name ?? ""}${suffix}` });
    }
    return Array.from(groups.entries()).map(([group, options]) => ({
      label: group,
      options,
    }));
  }, [crossVaultContacts, contactId, t]);

  return (
    <>
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
                    relationship_type_id: r.relationship_type_id,
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
              // Cross-vault relationship: use related_contact_name from API response directly
              title={<span style={{ fontWeight: 500 }}>{r.related_contact_name ?? r.related_contact_id}</span>}
              description={
                <span>
                  <Tag color="blue">{r.relationship_type_name ?? ""}</Tag>
                  {r.related_vault_id !== String(vaultId) && r.related_vault_name && (
                    <Tag color="default">{r.related_vault_name}</Tag>
                  )}
                </span>
              }
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
          {selectedContactOneWay && (
            <div style={{ fontSize: 12, color: token.colorWarningText, marginTop: -16, marginBottom: 12 }}>
              {t("modules.relationships.one_way_hint")}
            </div>
          )}
          <Form.Item name="relationship_type_id" label={t("modules.relationships.relationship_type")} rules={[{ required: true }]}>
            <Select
              showSearch
              options={typeSelectOptions}
              filterOption={(input, option) =>
                (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <div style={{ marginTop: -12, marginBottom: 24 }}>
            <a onClick={() => window.open("/settings/personalize", "_blank")} style={{ fontSize: 12, color: token.colorPrimary }}>
              {t("modules.relationships.manage_types")}
            </a>
            <div style={{ fontSize: 12, color: token.colorTextSecondary, marginTop: 4 }}>
              {t("modules.relationships.manage_types_hint")}
            </div>
          </div>
        </Form>
      </Modal>
    </Card>

    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.relationships.graph_title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      style={{ marginTop: 16 }}
    >
      <NetworkGraph vaultId={String(vaultId)} contactId={String(contactId)} />
    </Card>
    </>
  );
}
