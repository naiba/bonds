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
import { api } from "@/api";
import type { QuickFact, APIError } from "@/api";
import { useTranslation } from "react-i18next";



export default function QuickFactsModule({
  vaultId,
  contactId,
  templateId = 1,
}: {
  vaultId: string | number;
  contactId: string | number;
  templateId?: number;
}) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [content, setContent] = useState("");
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "quickFacts", templateId];

  const { data: facts = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.quickFacts.contactsQuickFactsDetail(String(vaultId), String(contactId), templateId);
      return res.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => {
      const data = { content };
      if (editingId) {
        return api.quickFacts.contactsQuickFactsUpdate(String(vaultId), String(contactId), templateId, editingId, data);
      }
      return api.quickFacts.contactsQuickFactsCreate(String(vaultId), String(contactId), templateId, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.quickFacts.contactsQuickFactsDelete(String(vaultId), String(contactId), templateId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function resetForm() {
    setAdding(false);
    setEditingId(null);
    setContent("");
  }

  function startEdit(fact: QuickFact) {
    setEditingId(fact.id ?? null);
    setContent(fact.content ?? '');
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
            <Input placeholder={t("modules.quick_facts.content_placeholder")} value={content} onChange={(e) => setContent(e.target.value)} />
            <Space>
              <Button
                type="primary"
                onClick={() => saveMutation.mutate()}
                loading={saveMutation.isPending}
                disabled={!content.trim()}
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
              <Popconfirm key="d" title={t("modules.quick_facts.delete_confirm")} onConfirm={() => deleteMutation.mutate(fact.id!)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={<span style={{ fontWeight: 500 }}>{fact.content}</span>}
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
