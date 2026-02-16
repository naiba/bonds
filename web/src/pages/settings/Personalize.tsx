import { useState } from "react";
import {
  Card,
  Typography,
  Button,
  List,
  Collapse,
  Input,
  Space,
  Popconfirm,
  App,
  Empty,
  Badge,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { PersonalizeItem, APIError } from "@/api";

const { Title, Text } = Typography;

const sectionKeys = [
  "genders", "pronouns", "address-types", "pet-categories",
  "contact-info-types", "relationship-types", "templates", "modules",
  "currencies", "religions", "call-reasons",
  "gift-occasions", "gift-states",
];

const sectionI18nMap: Record<string, string> = {
  "genders": "settings.personalize.genders",
  "pronouns": "settings.personalize.pronouns",
  "address-types": "settings.personalize.address_types",
  "pet-categories": "settings.personalize.pet_categories",
  "contact-info-types": "settings.personalize.contact_info_types",
  "relationship-types": "settings.personalize.relationship_types",
  "templates": "settings.personalize.templates",
  "modules": "settings.personalize.modules_label",
  "currencies": "settings.personalize.currencies",
  "religions": "settings.personalize.religions",
  "call-reasons": "settings.personalize.call_reasons",
  "life-event-categories": "settings.personalize.life_event_categories",
  "gift-occasions": "settings.personalize.gift_occasions",
  "gift-states": "settings.personalize.gift_states",
};

function SectionPanel({ sectionKey }: { sectionKey: string }) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [label, setLabel] = useState("");
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["settings", "personalize", sectionKey];

  const { data: items = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail(sectionKey);
      return res.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => {
      if (editingId) {
        return api.personalize.personalizeUpdate(sectionKey, editingId, { label });
      }
      return api.personalize.personalizeCreate(sectionKey, { label });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.personalize.personalizeDelete(sectionKey, id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: qk }),
    onError: (e: APIError) => message.error(e.message),
  });

  function resetForm() {
    setAdding(false);
    setEditingId(null);
    setLabel("");
  }

  function startEdit(item: PersonalizeItem) {
    setEditingId(item.id ?? null);
    setLabel(item.label ?? '');
    setAdding(false);
  }

  const showForm = adding || editingId !== null;

  return (
    <div>
      {!showForm && (
        <Button
          type="dashed"
          icon={<PlusOutlined />}
          onClick={() => setAdding(true)}
          style={{ marginBottom: 12 }}
          block
        >
          {t("settings.personalize.add_item")}
        </Button>
      )}

      {showForm && (
        <div style={{ marginBottom: 12 }}>
          <Space.Compact style={{ width: "100%" }}>
            <Input
              placeholder={t("common.label")}
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              onPressEnter={() => label.trim() && saveMutation.mutate()}
            />
            <Button
              type="primary"
              onClick={() => label.trim() && saveMutation.mutate()}
              loading={saveMutation.isPending}
            >
              {editingId ? t("common.update") : t("common.add")}
            </Button>
          </Space.Compact>
          <Button type="text" size="small" onClick={resetForm} style={{ marginTop: 4 }}>
            {t("common.cancel")}
          </Button>
        </div>
      )}

      <List
        loading={isLoading}
        dataSource={items}
        locale={{ emptyText: <Empty description={t("settings.personalize.no_items")} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
        size="small"
        renderItem={(item: PersonalizeItem) => (
          <List.Item
            actions={[
              <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => startEdit(item)} />,
              <Popconfirm key="d" title={t("settings.personalize.delete_confirm")} onConfirm={() => deleteMutation.mutate(item.id!)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <span>
              {item.label}
            </span>
          </List.Item>
        )}
      />
    </div>
  );
}

function SectionCollapseLabel({
  sectionKey,
  label,
}: {
  sectionKey: string;
  label: string;
}) {
  const { data: items = [] } = useQuery({
    queryKey: ["settings", "personalize", sectionKey],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail(sectionKey);
      return res.data ?? [];
    },
  });

  return (
    <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
      <span style={{ fontWeight: 500 }}>{label}</span>
      <Badge
        count={items.length}
        showZero
        color="#d9d9d9"
        style={{ color: "#666", fontSize: 11 }}
      />
    </span>
  );
}

export default function Personalize() {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const collapseItems = sectionKeys.map((key) => ({
    key,
    label: (
      <SectionCollapseLabel
        sectionKey={key}
        label={t(sectionI18nMap[key])}
      />
    ),
    children: <SectionPanel sectionKey={key} />,
  }));

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <Title level={4} style={{ marginBottom: 4 }}>
        {t("settings.personalize.title")}
      </Title>
      <Text type="secondary" style={{ display: "block", marginBottom: 24 }}>
        {t("settings.personalize.description")}
      </Text>

      <Card
        styles={{
          body: { padding: 0 },
        }}
      >
        <Collapse
          items={collapseItems}
          bordered={false}
          style={{ background: token.colorBgContainer }}
        />
      </Card>
    </div>
  );
}
