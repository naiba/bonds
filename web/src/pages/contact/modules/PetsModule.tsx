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
      title={t("modules.pets.title")}
      extra={
        !showForm && (
          <Button type="link" icon={<PlusOutlined />} onClick={() => setAdding(true)}>
            {t("modules.pets.add")}
          </Button>
        )
      }
    >
      {showForm && (
        <div style={{ marginBottom: 16 }}>
          <Space direction="vertical" style={{ width: "100%" }}>
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
        dataSource={pets}
        locale={{ emptyText: <Empty description={t("modules.pets.no_pets")} /> }}
        renderItem={(pet: Pet) => (
          <List.Item
            actions={[
              <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => startEdit(pet)} />,
              <Popconfirm key="d" title={t("modules.pets.delete_confirm")} onConfirm={() => deleteMutation.mutate(pet.id)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={pet.name}
              description={<Tag>{pet.category}</Tag>}
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
