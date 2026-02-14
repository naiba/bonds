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
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { contactInfoApi } from "@/api/contactInfo";
import type { ContactInfo } from "@/types/modules";
import type { APIError } from "@/types/api";
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
  const qk = ["vaults", vaultId, "contacts", contactId, "contact-info"];

  const { data: items = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await contactInfoApi.list(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => {
      const data = { type, label, value };
      if (editingId) {
        return contactInfoApi.update(vaultId, contactId, editingId, data);
      }
      return contactInfoApi.create(vaultId, contactId, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
      message.success(editingId ? t("modules.contact_info.updated") : t("modules.contact_info.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => contactInfoApi.delete(vaultId, contactId, id),
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
    setEditingId(item.id);
    setType(item.type);
    setLabel(item.label);
    setValue(item.value);
    setAdding(false);
  }

  const showForm = adding || editingId !== null;

  return (
    <Card
      title={t("modules.contact_info.title")}
      extra={
        !showForm && (
          <Button type="link" icon={<PlusOutlined />} onClick={() => setAdding(true)}>
            {t("modules.contact_info.add")}
          </Button>
        )
      }
    >
      {showForm && (
        <div style={{ marginBottom: 16 }}>
          <Space direction="vertical" style={{ width: "100%" }}>
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
              >
                {editingId ? t("common.update") : t("common.save")}
              </Button>
              <Button onClick={resetForm}>{t("common.cancel")}</Button>
            </Space>
          </Space>
        </div>
      )}

      <List
        loading={isLoading}
        dataSource={items}
        locale={{ emptyText: <Empty description={t("modules.contact_info.no_info")} /> }}
        renderItem={(item: ContactInfo) => (
          <List.Item
            actions={[
              <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => startEdit(item)} />,
              <Popconfirm key="d" title={t("modules.contact_info.delete_confirm")} onConfirm={() => deleteMutation.mutate(item.id)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={
                <>
                  <Tag>{item.type}</Tag> {item.label}
                </>
              }
              description={item.value}
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
