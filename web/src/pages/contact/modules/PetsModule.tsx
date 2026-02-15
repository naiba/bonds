import { useState } from "react";
import {
  Card,
  List,
  Button,
  Input,
  Space,
  Popconfirm,
  App,
  Tag,
  Empty,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { petsApi } from "@/api/pets";
import type { Pet } from "@/types/modules";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";

export default function PetsModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [name, setName] = useState("");
  const [category, setCategory] = useState("");
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "pets"];

  const { data: pets = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await petsApi.list(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => {
      const data = { name, category };
      if (editingId) {
        return petsApi.update(vaultId, contactId, editingId, data);
      }
      return petsApi.create(vaultId, contactId, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
      message.success(editingId ? t("modules.pets.updated") : t("modules.pets.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => petsApi.delete(vaultId, contactId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.pets.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function resetForm() {
    setAdding(false);
    setEditingId(null);
    setName("");
    setCategory("");
  }

  function startEdit(pet: Pet) {
    setEditingId(pet.id);
    setName(pet.name);
    setCategory(pet.category);
    setAdding(false);
  }

  const showForm = adding || editingId !== null;

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.pets.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        !showForm && (
          <Button type="text" icon={<PlusOutlined />} onClick={() => setAdding(true)} style={{ color: token.colorPrimary }}>
            {t("modules.pets.add")}
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
            <Input
              placeholder={t("modules.pets.name_placeholder")}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <Input
              placeholder={t("modules.pets.category_placeholder")}
              value={category}
              onChange={(e) => setCategory(e.target.value)}
            />
            <Space>
              <Button
                type="primary"
                onClick={() => saveMutation.mutate()}
                loading={saveMutation.isPending}
                disabled={!name.trim() || !category.trim()}
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
        dataSource={pets}
        locale={{ emptyText: <Empty description={t("modules.pets.no_pets")} /> }}
        split={false}
        renderItem={(pet: Pet) => (
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
              <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => startEdit(pet)} />,
              <Popconfirm key="d" title={t("modules.pets.delete_confirm")} onConfirm={() => deleteMutation.mutate(pet.id)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={<span style={{ fontWeight: 500 }}>{pet.name}</span>}
              description={<Tag>{pet.category}</Tag>}
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
