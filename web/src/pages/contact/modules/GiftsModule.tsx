import { useMemo, useState } from "react";
import {
  App,
  Button,
  Card,
  Empty,
  Input,
  List,
  Popconfirm,
  Select,
  Space,
  Tag,
  Typography,
  theme,
} from "antd";
import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { APIError, CreateGiftRequest, Gift, PersonalizeItem, UpdateGiftRequest } from "@/api";

type GiftFormState = {
  name: string;
  type: string;
  description: string;
  giftOccasionId?: number;
  giftStateId?: number;
};

function emptyGiftForm(): GiftFormState {
  return {
    name: "",
    type: "given",
    description: "",
    giftOccasionId: undefined,
    giftStateId: undefined,
  };
}

function personalizeLabel(item: PersonalizeItem): string {
  return item.label || item.name || "";
}

function buildGiftRequest(form: GiftFormState): CreateGiftRequest | UpdateGiftRequest {
  const request: CreateGiftRequest | UpdateGiftRequest = {
    name: form.name.trim(),
    type: form.type,
    gift_occasion_id: form.giftOccasionId!,
    gift_state_id: form.giftStateId!,
  };
  const description = form.description.trim();
  if (description) request.description = description;
  return request;
}

export default function GiftsModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<GiftFormState>(emptyGiftForm);
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "gifts"];

  const { data: gifts = [], isLoading } = useQuery<Gift[]>({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.gifts.contactsGiftsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const { data: occasions = [], isLoading: isOccasionsLoading } = useQuery<PersonalizeItem[]>({
    queryKey: ["personalize", "gift-occasions"],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail("gift-occasions");
      return res.data ?? [];
    },
  });

  const { data: states = [], isLoading: isStatesLoading } = useQuery<PersonalizeItem[]>({
    queryKey: ["personalize", "gift-states"],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail("gift-states");
      return res.data ?? [];
    },
  });

  const occasionOptions = useMemo(
    () => occasions.filter((item) => item.id != null).map((item) => ({ value: item.id!, label: personalizeLabel(item) })),
    [occasions],
  );
  const stateOptions = useMemo(
    () => states.filter((item) => item.id != null).map((item) => ({ value: item.id!, label: personalizeLabel(item) })),
    [states],
  );

  const saveMutation = useMutation({
    mutationFn: () => {
      const request = buildGiftRequest(form);
      if (editingId) {
        return api.gifts.contactsGiftsUpdate(String(vaultId), String(contactId), editingId, request);
      }
      return api.gifts.contactsGiftsCreate(String(vaultId), String(contactId), request);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
      message.success(editingId ? t("modules.gifts.updated") : t("modules.gifts.added"));
    },
    onError: (err: APIError) => message.error(err.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.gifts.contactsGiftsDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.gifts.deleted"));
    },
    onError: (err: APIError) => message.error(err.message),
  });

  function resetForm() {
    setAdding(false);
    setEditingId(null);
    setForm(emptyGiftForm());
  }

  function startAdd() {
    setAdding(true);
    setEditingId(null);
    setForm(emptyGiftForm());
  }

  function startEdit(gift: Gift) {
    setAdding(false);
    setEditingId(gift.id ?? null);
    setForm({
      name: gift.name ?? "",
      type: gift.type || "given",
      description: gift.description ?? "",
      giftOccasionId: gift.gift_occasion_id,
      giftStateId: gift.gift_state_id,
    });
  }

  const showForm = adding || editingId !== null;
  const canSave = !!form.name.trim() && form.giftOccasionId != null && form.giftStateId != null;

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.gifts.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: "16px 24px" },
      }}
      extra={
        !showForm && (
          <Button type="text" icon={<PlusOutlined />} onClick={startAdd} style={{ color: token.colorPrimary }}>
            {t("modules.gifts.add")}
          </Button>
        )
      }
    >
      {showForm && (
        <div
          style={{
            marginBottom: 16,
            padding: 16,
            background: token.colorFillQuaternary,
            borderRadius: token.borderRadius,
          }}
        >
          <Space direction="vertical" style={{ width: "100%" }}>
            <Input
              aria-label={t("modules.gifts.name")}
              placeholder={t("modules.gifts.name_placeholder")}
              value={form.name}
              onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
            />
            <Select
              data-testid="gift-occasion-select"
              aria-label={t("modules.gifts.occasion")}
              placeholder={t("modules.gifts.occasion_placeholder")}
              value={form.giftOccasionId}
              onChange={(value) => setForm((current) => ({ ...current, giftOccasionId: value }))}
              options={occasionOptions}
              loading={isOccasionsLoading}
              showSearch
              optionFilterProp="label"
              style={{ width: "100%" }}
            />
            <Select
              data-testid="gift-state-select"
              aria-label={t("modules.gifts.state")}
              placeholder={t("modules.gifts.state_placeholder")}
              value={form.giftStateId}
              onChange={(value) => setForm((current) => ({ ...current, giftStateId: value }))}
              options={stateOptions}
              loading={isStatesLoading}
              showSearch
              optionFilterProp="label"
              style={{ width: "100%" }}
            />
            <Select
              aria-label={t("modules.gifts.type")}
              value={form.type}
              onChange={(value) => setForm((current) => ({ ...current, type: value }))}
              options={[
                { value: "given", label: t("modules.gifts.type_given") },
                { value: "received", label: t("modules.gifts.type_received") },
              ]}
              style={{ width: "100%" }}
            />
            <Input.TextArea
              aria-label={t("common.description")}
              placeholder={t("modules.gifts.description_placeholder")}
              rows={2}
              value={form.description}
              onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
            />
            <Space>
              <Button
                type="primary"
                onClick={() => saveMutation.mutate()}
                loading={saveMutation.isPending}
                disabled={!canSave}
                size="small"
              >
                {editingId ? t("common.update") : t("common.save")}
              </Button>
              <Button onClick={resetForm} size="small">
                {t("common.cancel")}
              </Button>
            </Space>
          </Space>
        </div>
      )}

      <List
        loading={isLoading}
        dataSource={gifts}
        locale={{ emptyText: <Empty description={t("modules.gifts.no_gifts")} /> }}
        split={false}
        renderItem={(gift) => (
          <List.Item
            style={{
              borderRadius: token.borderRadius,
              padding: "10px 12px",
              marginBottom: 4,
              transition: "background 0.2s",
            }}
            onMouseEnter={(event) => {
              event.currentTarget.style.background = token.colorFillQuaternary;
            }}
            onMouseLeave={(event) => {
              event.currentTarget.style.background = "transparent";
            }}
            actions={[
              <Button
                key="edit"
                type="text"
                size="small"
                icon={<EditOutlined />}
                aria-label={t("modules.gifts.edit")}
                onClick={() => startEdit(gift)}
              />,
              <Popconfirm
                key="delete"
                title={t("modules.gifts.delete_confirm")}
                onConfirm={() => deleteMutation.mutate(gift.id!)}
              >
                <Button
                  type="text"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  aria-label={t("modules.gifts.delete")}
                />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={<span style={{ fontWeight: 500 }}>{gift.name}</span>}
              description={
                <Space direction="vertical" size={4}>
                  <Space size={4} wrap>
                    {gift.gift_occasion_label && <Tag color="blue">{gift.gift_occasion_label}</Tag>}
                    {gift.gift_state_label && <Tag color="green">{gift.gift_state_label}</Tag>}
                    {gift.type && <Tag>{t(`modules.gifts.type_${gift.type}`, { defaultValue: gift.type })}</Tag>}
                  </Space>
                  {gift.description && (
                    <Typography.Text type="secondary">{gift.description}</Typography.Text>
                  )}
                </Space>
              }
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
