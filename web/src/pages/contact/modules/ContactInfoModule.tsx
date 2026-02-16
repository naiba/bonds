import { useState } from "react";
import {
  Card,
  List,
  Button,
  Input,
  Select,
  Space,
  Popconfirm,
  App,
  Tag,
  Empty,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { ContactInfo, APIError } from "@/api";
import { useTranslation } from "react-i18next";

const infoTypes = [
  { value: "email", label: "Email" },
  { value: "phone", label: "Phone" },
  { value: "facebook", label: "Facebook" },
  { value: "twitter", label: "Twitter" },
  { value: "linkedin", label: "LinkedIn" },
  { value: "instagram", label: "Instagram" },
  { value: "whatsapp", label: "WhatsApp" },
  { value: "telegram", label: "Telegram" },
  { value: "other", label: "Other" },
];

export default function ContactInfoModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [type, setType] = useState("email");
  const [label, setLabel] = useState("");
  const [value, setValue] = useState("");
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "contactInformation"];

  const { data: items = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.contactInformation.contactsContactInformationList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => {
      const payload = { data: value, kind: label || type, type_id: Math.max(1, infoTypes.findIndex((t) => t.value === type) + 1) };
      if (editingId) {
        return api.contactInformation.contactsContactInformationUpdate(String(vaultId), String(contactId), editingId, payload);
      }
      return api.contactInformation.contactsContactInformationCreate(String(vaultId), String(contactId), payload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
      message.success(editingId ? t("modules.contact_info.updated") : t("modules.contact_info.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.contactInformation.contactsContactInformationDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.contact_info.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function resetForm() {
    setAdding(false);
    setEditingId(null);
    setType("email");
    setLabel("");
    setValue("");
  }

  function startEdit(item: ContactInfo) {
    setEditingId(item.id ?? null);
    setType(item.kind ?? 'email');
    setLabel(item.kind ?? '');
    setValue(item.data ?? '');
    setAdding(false);
  }

  const showForm = adding || editingId !== null;

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.contact_info.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        !showForm && (
          <Button type="text" icon={<PlusOutlined />} onClick={() => setAdding(true)} style={{ color: token.colorPrimary }}>
            {t("modules.contact_info.add")}
          </Button>
        )
      }
    >
      {showForm && (
        <div style={{
          marginBottom: 16,
          padding: 16,
          background: token.colorFillQuaternary,
          borderRadius: token.borderRadius,
        }}>
          <Space orientation="vertical" style={{ width: "100%" }}>
            <Select
              value={type}
              onChange={setType}
              options={infoTypes}
              style={{ width: "100%" }}
            />
            <Input
              placeholder={t("modules.contact_info.label_placeholder")}
              value={label}
              onChange={(e) => setLabel(e.target.value)}
            />
            <Input
              placeholder={t("modules.contact_info.value_placeholder")}
              value={value}
              onChange={(e) => setValue(e.target.value)}
            />
            <Space>
              <Button
                type="primary"
                onClick={() => saveMutation.mutate()}
                loading={saveMutation.isPending}
                disabled={!value.trim()}
                size="small"
              >
                {editingId ? t("common.update") : t("common.save")}
              </Button>
              <Button onClick={resetForm} size="small">{t("common.cancel")}</Button>
            </Space>
          </Space>
        </div>
      )}

      <List
        loading={isLoading}
        dataSource={items}
        locale={{ emptyText: <Empty description={t("modules.contact_info.no_info")} /> }}
        split={false}
        renderItem={(item: ContactInfo) => (
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
              <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => startEdit(item)} />,
              <Popconfirm key="d" title={t("modules.contact_info.delete_confirm")} onConfirm={() => deleteMutation.mutate(item.id!)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={
                <span style={{ fontWeight: 500 }}>
                  <Tag>{item.kind}</Tag> {item.kind}
                </span>
              }
              description={<span style={{ color: token.colorTextSecondary }}>{item.data}</span>}
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
