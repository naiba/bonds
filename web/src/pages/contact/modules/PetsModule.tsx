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
  Modal,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Pet, PetCategory, APIError } from "@/api";
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
  const [selectedCategoryId, setSelectedCategoryId] = useState<number | null>(
    null,
  );
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "pets"];

  const { data: pets = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.pets.contactsPetsList(
        String(vaultId),
        String(contactId),
      );
      return res.data ?? [];
    },
  });

  const { data: petCategories = [], isLoading: isCategoriesLoading } = useQuery<
    PetCategory[]
  >({
    queryKey: ["pet-categories"],
    queryFn: async (): Promise<PetCategory[]> => {
      const res = await api.pets.petCategoriesList();
      return res.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => {
      // Category labels like "Dog" must resolve through the account-backed selector, not Number(text) -> 0.
      const data = { name: name.trim(), pet_category_id: selectedCategoryId! };
      if (editingId) {
        return api.pets.contactsPetsUpdate(
          String(vaultId),
          String(contactId),
          editingId,
          data,
        );
      }
      return api.pets.contactsPetsCreate(
        String(vaultId),
        String(contactId),
        data,
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
      message.success(
        editingId ? t("modules.pets.updated") : t("modules.pets.added"),
      );
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) =>
      api.pets.contactsPetsDelete(String(vaultId), String(contactId), id),
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
    setSelectedCategoryId(null);
  }

  function startEdit(pet: Pet) {
    setEditingId(pet.id ?? null);
    setName(pet.name ?? "");
    setSelectedCategoryId(pet.pet_category_id ?? null);
    setAdding(false);
  }

  const showForm = adding || editingId !== null;

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.pets.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: "16px 24px" },
      }}
      extra={
        !showForm && (
          <Button
            type="text"
            icon={<PlusOutlined />}
            onClick={() => setAdding(true)}
            style={{ color: token.colorPrimary }}
          >
            {t("modules.pets.add")}
          </Button>
        )
      }
    >
      <List
        loading={isLoading}
        dataSource={pets}
        locale={{
          emptyText: <Empty description={t("modules.pets.no_pets")} />,
        }}
        split={false}
        renderItem={(pet: Pet) => (
          <List.Item
            style={{
              borderRadius: token.borderRadius,
              padding: "10px 12px",
              marginBottom: 4,
              transition: "background 0.2s",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = token.colorFillQuaternary;
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = "transparent";
            }}
            actions={[
              <Button
                key="e"
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => startEdit(pet)}
              />,
              <Popconfirm
                key="d"
                title={t("modules.pets.delete_confirm")}
                onConfirm={() => deleteMutation.mutate(pet.id!)}
              >
                <Button
                  type="text"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={<span style={{ fontWeight: 500 }}>{pet.name}</span>}
              description={<Tag>{pet.pet_category_name ?? ""}</Tag>}
            />
          </List.Item>
        )}
      />
      <Modal
        title={
          editingId ? t("modules.pets.modal_edit") : t("modules.pets.modal_add")
        }
        open={showForm}
        onCancel={resetForm}
        onOk={() => saveMutation.mutate()}
        okText={editingId ? t("common.update") : t("common.save")}
        cancelText={t("common.cancel")}
        confirmLoading={saveMutation.isPending}
        okButtonProps={{
          disabled: !name.trim() || selectedCategoryId === null,
        }}
        destroyOnHidden
      >
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Input
            placeholder={t("modules.pets.name_placeholder")}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <Select
            placeholder={t("modules.pets.category_placeholder")}
            value={selectedCategoryId ?? undefined}
            onChange={(value) => setSelectedCategoryId(value)}
            options={petCategories.map((category) => ({
              value: category.id,
              label: category.name,
            }))}
            loading={isCategoriesLoading}
            showSearch
            optionFilterProp="label"
            style={{ width: "100%" }}
          />
        </Space>
      </Modal>
    </Card>
  );
}
