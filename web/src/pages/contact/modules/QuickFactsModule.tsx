import { useState } from "react";
import {
  Card,
  List,
  Button,
  Input,
  Space,
  Popconfirm,
  App,
  Empty,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import client from "@/api/client";
import type { APIResponse } from "@/types/api";
import type { QuickFact } from "@/types/modules";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";

const quickFactsApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<QuickFact[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/quick-facts`,
    );
  },
  create(vaultId: string | number, contactId: string | number, data: { label: string; value: string }) {
    return client.post<APIResponse<QuickFact>>(
      `/vaults/${vaultId}/contacts/${contactId}/quick-facts`,
      data,
    );
  },
  update(vaultId: string | number, contactId: string | number, id: number, data: { label: string; value: string }) {
    return client.put<APIResponse<QuickFact>>(
      `/vaults/${vaultId}/contacts/${contactId}/quick-facts/${id}`,
      data,
    );
  },
  delete(vaultId: string | number, contactId: string | number, id: number) {
    return client.delete(`/vaults/${vaultId}/contacts/${contactId}/quick-facts/${id}`);
  },
};

export default function QuickFactsModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [label, setLabel] = useState("");
  const [value, setValue] = useState("");
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "quick-facts"];

  const { data: facts = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await quickFactsApi.list(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => {
      const data = { label, value };
      if (editingId) {
        return quickFactsApi.update(vaultId, contactId, editingId, data);
      }
      return quickFactsApi.create(vaultId, contactId, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => quickFactsApi.delete(vaultId, contactId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function resetForm() {
    setAdding(false);
    setEditingId(null);
    setLabel("");
    setValue("");
  }

  function startEdit(fact: QuickFact) {
    setEditingId(fact.id);
    setLabel(fact.label);
    setValue(fact.value);
    setAdding(false);
  }

  const showForm = adding || editingId !== null;

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.quick_facts.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        !showForm && (
          <Button type="text" icon={<PlusOutlined />} onClick={() => setAdding(true)} style={{ color: token.colorPrimary }}>
            {t("modules.quick_facts.add")}
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
            <Input placeholder={t("modules.quick_facts.label_placeholder")} value={label} onChange={(e) => setLabel(e.target.value)} />
            <Input placeholder={t("modules.quick_facts.value_placeholder")} value={value} onChange={(e) => setValue(e.target.value)} />
            <Space>
              <Button
                type="primary"
                onClick={() => saveMutation.mutate()}
                loading={saveMutation.isPending}
                disabled={!label.trim() || !value.trim()}
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
        dataSource={facts}
        locale={{ emptyText: <Empty description={t("modules.quick_facts.no_facts")} /> }}
        split={false}
        renderItem={(fact: QuickFact) => (
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
              <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => startEdit(fact)} />,
              <Popconfirm key="d" title={t("modules.quick_facts.delete_confirm")} onConfirm={() => deleteMutation.mutate(fact.id)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={<span style={{ fontWeight: 500 }}>{fact.label}</span>}
              description={<span style={{ color: token.colorTextSecondary }}>{fact.value}</span>}
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
