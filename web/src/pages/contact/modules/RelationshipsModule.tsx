import { useState, useMemo } from "react";
import {
  Card,
  List,
  Button,
  Input,
  Modal,
  Form,
  Select,
  Radio,
  Popconfirm,
  App,
  Tag,
  Empty,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined, UserOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import { Link } from "react-router-dom";
import NetworkGraph from "@/components/NetworkGraph";
import type { Relationship, APIError } from "@/api";
import type {
  GithubComNaibaBondsInternalDtoCreateRelationshipRequest,
  GithubComNaibaBondsInternalDtoRelationshipTypeWithGroupResponse,
  GithubComNaibaBondsInternalDtoCrossVaultContactItem,
  GithubComNaibaBondsInternalDtoUpdateRelationshipRequest,
} from "@/api";
import { useTranslation } from "react-i18next";

const RELATIONSHIP_TARGET_KINDS = {
  existing: "existing",
  external: "external",
} as const;

type RelationshipTargetKind = (typeof RELATIONSHIP_TARGET_KINDS)[keyof typeof RELATIONSHIP_TARGET_KINDS];

type RelationshipFormValues = {
  readonly target_kind?: RelationshipTargetKind;
  readonly related_contact_id?: string;
  readonly external_contact_name?: string;
  readonly relationship_type_id?: number;
};

export default function RelationshipsModule({
  vaultId,
  contactId,
  currentContactName,
}: {
  vaultId: string | number;
  contactId: string | number;
  currentContactName?: string;
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
  const selectedTargetKind = Form.useWatch("target_kind", form) ?? RELATIONSHIP_TARGET_KINDS.existing;
  const selectedRelationshipTypeId = Form.useWatch("relationship_type_id", form) as number | undefined;

  const { data: relationships = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.relationships.contactsRelationshipsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const { data: crossVaultContacts = [] } = useQuery({
    queryKey: ["relationships", "contacts"],
    queryFn: async (): Promise<GithubComNaibaBondsInternalDtoCrossVaultContactItem[]> => {
      const res = await api.relationships.contactsList();
      return res.data ?? [];
    },
  });

  // BUG FIX: Previously fetched relationship GROUP types (Love/Family/Friend/Work) via
  // personalizeDetail("relationship-types"), which queries the relationship_group_types table.
  // Users could only pick a group, not a specific type (Parent/Child/Sibling), causing
  // wrong relationship_type_id to be stored and incorrect labels on the graph.
  // Now fetches all actual RelationshipType records with group names for grouped select.
  const { data: relationshipTypes = [] } = useQuery({
    queryKey: ["personalize", "relationship-types", "all"],
    queryFn: async (): Promise<GithubComNaibaBondsInternalDtoRelationshipTypeWithGroupResponse[]> => {
      const res = await api.relationshipTypes.personalizeRelationshipTypesAllList();
      return res.data ?? [];
    },
  });

  // Track whether the selected contact lacks editor permission (one-way only)
  const selectedContactId = Form.useWatch("related_contact_id", form);
  const selectedContactOneWay = useMemo(() => {
    if (selectedTargetKind !== RELATIONSHIP_TARGET_KINDS.existing || !selectedContactId) return false;
    const c = crossVaultContacts.find((x) => x.contact_id === selectedContactId);
    return c ? c.has_editor === false : false;
  }, [selectedTargetKind, selectedContactId, crossVaultContacts]);

  const selectedRelationshipTypeName = useMemo(() => {
    if (selectedRelationshipTypeId == null) return "";
    const selectedRelationshipType = relationshipTypes.find((relationshipType) => relationshipType.id === selectedRelationshipTypeId);
    return selectedRelationshipType?.name ?? "";
  }, [relationshipTypes, selectedRelationshipTypeId]);

  const selectedContactName = useMemo(() => {
    if (selectedTargetKind !== RELATIONSHIP_TARGET_KINDS.existing || !selectedContactId) return "";
    const selectedContact = crossVaultContacts.find((contact) => contact.contact_id === selectedContactId);
    return selectedContact?.contact_name ?? "";
  }, [crossVaultContacts, selectedContactId, selectedTargetKind]);

  const relationshipDirectionHint = useMemo(() => {
    if (!currentContactName || !selectedRelationshipTypeName || !selectedContactName) {
      return t("modules.relationships.direction_hint");
    }
    return t("modules.relationships.direction_hint_named", {
      currentContactName,
      relationshipTypeName: selectedRelationshipTypeName,
      selectedContactName,
    });
  }, [currentContactName, selectedContactName, selectedRelationshipTypeName, t]);

  // Build grouped options for the relationship type Select (OptGroup by group name).
  const typeSelectOptions = useMemo(() => {
    const groups = new Map<string, { value: number; label: string }[]>();
    for (const rt of relationshipTypes) {
      if (rt.id == null) continue;
      const groupName = rt.group_name ?? "";
      const option = { value: rt.id, label: rt.name ?? "" };
      const existingOptions = groups.get(groupName);
      if (existingOptions) {
        existingOptions.push(option);
      } else {
        groups.set(groupName, [option]);
      }
    }
    return Array.from(groups.entries()).map(([group, options]) => ({
      label: group,
      options,
    }));
  }, [relationshipTypes]);

  const createMutation = useMutation({
    mutationFn: (request: GithubComNaibaBondsInternalDtoCreateRelationshipRequest) =>
      api.relationships.contactsRelationshipsCreate(String(vaultId), String(contactId), request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      setEditingId(null);
      form.resetFields();
      message.success(editingId ? t("modules.relationships.updated") : t("modules.relationships.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, request }: { readonly id: number; readonly request: GithubComNaibaBondsInternalDtoUpdateRelationshipRequest }) =>
      api.relationships.contactsRelationshipsUpdate(String(vaultId), String(contactId), id, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      setEditingId(null);
      form.resetFields();
      message.success(t("modules.relationships.updated"));
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
    const groups = new Map<string, { value: string; label: string }[]>();
    for (const c of crossVaultContacts) {
      if (!c.contact_id || c.contact_id === String(contactId)) continue;
      const vaultName = c.vault_name ?? "";
      const suffix = c.has_editor === false ? ` · ${t("modules.relationships.one_way_only")}` : "";
      const option = { value: c.contact_id, label: `${c.contact_name ?? ""}${suffix}` };
      const existingOptions = groups.get(vaultName);
      if (existingOptions) {
        existingOptions.push(option);
      } else {
        groups.set(vaultName, [option]);
      }
    }
    return Array.from(groups.entries()).map(([group, options]) => ({
      label: group,
      options,
    }));
  }, [crossVaultContacts, contactId, t]);

  function handleRelationshipSubmit(values: RelationshipFormValues) {
    if (values.relationship_type_id == null) return;

    if (editingId != null) {
      if (!values.related_contact_id) return;
      const request: GithubComNaibaBondsInternalDtoUpdateRelationshipRequest = {
        relationship_type_id: values.relationship_type_id,
        related_contact_id: values.related_contact_id,
      };
      updateMutation.mutate({ id: editingId, request });
      return;
    }

    if (selectedTargetKind === RELATIONSHIP_TARGET_KINDS.external) {
      const externalName = values.external_contact_name?.trim();
      if (!externalName) return;
      const request: GithubComNaibaBondsInternalDtoCreateRelationshipRequest = {
        relationship_type_id: values.relationship_type_id,
        external_contact_name: externalName,
      };
      createMutation.mutate(request);
      return;
    }

    if (!values.related_contact_id) return;
    const request: GithubComNaibaBondsInternalDtoCreateRelationshipRequest = {
      relationship_type_id: values.relationship_type_id,
      related_contact_id: values.related_contact_id,
    };
    createMutation.mutate(request);
  }

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
             form.setFieldsValue({
               target_kind: RELATIONSHIP_TARGET_KINDS.existing,
               related_contact_id: undefined,
               external_contact_name: undefined,
               relationship_type_id: undefined,
             });
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
                  if (r.id == null) return;
                  setEditingId(r.id);
                  form.setFieldsValue({
                    target_kind: RELATIONSHIP_TARGET_KINDS.existing,
                    related_contact_id: r.related_contact_id,
                    relationship_type_id: r.relationship_type_id,
                    external_contact_name: undefined,
                  });
                  setOpen(true);
                }}
              />,
              <Popconfirm
                key="d"
                title={t("modules.relationships.remove_confirm")}
                onConfirm={() => {
                  if (r.id != null) deleteMutation.mutate(r.id);
                }}
              >
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={<UserOutlined style={{ fontSize: 18, color: token.colorPrimary }} />}
              // Fix #60: make relationship contact names clickable links to navigate to their profile.
              // Uses related_vault_id for cross-vault relationships, falls back to current vaultId.
              title={
                <Link
                  to={`/vaults/${r.related_vault_id || String(vaultId)}/contacts/${r.related_contact_id}`}
                  style={{ fontWeight: 500, color: token.colorPrimary }}
                >
                  {r.related_contact_name ?? r.related_contact_id}
                </Link>
              }
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
        confirmLoading={createMutation.isPending || updateMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={handleRelationshipSubmit}>
          {!editingId && (
            <Form.Item name="target_kind" label={t("modules.relationships.target_kind")} initialValue={RELATIONSHIP_TARGET_KINDS.existing}>
              <Radio.Group
                optionType="button"
                buttonStyle="solid"
                onChange={() => {
                  form.setFieldsValue({ related_contact_id: undefined, external_contact_name: undefined });
                }}
              >
                <Radio.Button value={RELATIONSHIP_TARGET_KINDS.existing}>{t("modules.relationships.target_existing")}</Radio.Button>
                <Radio.Button value={RELATIONSHIP_TARGET_KINDS.external}>{t("modules.relationships.target_external")}</Radio.Button>
              </Radio.Group>
            </Form.Item>
          )}
          {(editingId || selectedTargetKind === RELATIONSHIP_TARGET_KINDS.existing) && (
            <Form.Item name="related_contact_id" label={t("modules.relationships.contact")} rules={[{ required: true }]}>
              <Select
                showSearch
                options={contactOptions}
                filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
                placeholder={t("modules.relationships.select_contact")}
              />
            </Form.Item>
          )}
          {!editingId && selectedTargetKind === RELATIONSHIP_TARGET_KINDS.external && (
            <Form.Item
              name="external_contact_name"
              label={t("modules.relationships.external_contact")}
              rules={[{ required: true, whitespace: true }]}
            >
              <Input placeholder={t("modules.relationships.external_contact_placeholder")} />
            </Form.Item>
          )}
          {selectedContactOneWay && (
            <div style={{ fontSize: 12, color: token.colorWarningText, marginTop: -16, marginBottom: 12 }}>
              {t("modules.relationships.one_way_hint")}
            </div>
          )}
          <Form.Item name="relationship_type_id" label={t("modules.relationships.relationship_type")} rules={[{ required: true }]}> 
            <Select
              showSearch
              options={typeSelectOptions}
              filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
            />
          </Form.Item>
          <div style={{ fontSize: 12, color: token.colorTextSecondary, marginTop: -16, marginBottom: 12 }}>
            {relationshipDirectionHint}
          </div>
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
