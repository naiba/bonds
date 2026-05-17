import { useState } from "react";
import {
  App,
  Button,
  Card,
  Empty,
  Input,
  Popconfirm,
  Select,
  Space,
  Tooltip,
  Typography,
  theme,
} from "antd";
import {
  DeleteOutlined,
  EditOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { APIError, QuickFact, QuickFactGroup } from "@/api";

export default function QuickFactsModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [selectedTemplateId, setSelectedTemplateId] = useState<number | null>(null);
  const [content, setContent] = useState("");
  const [isCollapsed, setIsCollapsed] = useState(false);

  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "quickFacts"];

  const { data: groups = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.quickFacts.contactsQuickFactsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const quickFactGroups: QuickFactGroup[] = groups;

  const editableGroups = quickFactGroups.filter((group) => group.template_id !== undefined);
  const visibleGroups = quickFactGroups.filter((group) => (group.facts?.length ?? 0) > 0);
  const hasFacts = visibleGroups.length > 0;

  const saveMutation = useMutation({
    mutationFn: () => {
      if (!selectedTemplateId) {
        return Promise.reject({ message: t("modules.quick_facts.select_category_required") });
      }
      const data = { content };
      if (editingId) {
        return api.quickFacts.contactsQuickFactsUpdate(
          String(vaultId),
          String(contactId),
          selectedTemplateId,
          editingId,
          data,
        );
      }
      return api.quickFacts.contactsQuickFactsCreate(String(vaultId), String(contactId), selectedTemplateId, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (fact: QuickFact) => {
      if (!fact.id || !fact.vault_quick_facts_template_id) {
        return Promise.reject({ message: t("modules.quick_facts.missing_fact_reference") });
      }
      return api.quickFacts.contactsQuickFactsDelete(
        String(vaultId),
        String(contactId),
        fact.vault_quick_facts_template_id,
        fact.id,
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const toggleMutation = useMutation({
    mutationFn: () => api.quickFacts.contactsQuickFactsToggleUpdate(String(vaultId), String(contactId)),
    onSuccess: () => {
      setIsCollapsed(!isCollapsed);
      message.success(t("modules.quick_facts.toggled"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function resetForm() {
    setAdding(false);
    setEditingId(null);
    setSelectedTemplateId(null);
    setContent("");
  }

  function startAdd() {
    setAdding(true);
    setEditingId(null);
    setSelectedTemplateId(editableGroups[0]?.template_id ?? null);
    setContent("");
  }

  function startEdit(fact: QuickFact) {
    setEditingId(fact.id ?? null);
    setSelectedTemplateId(fact.vault_quick_facts_template_id ?? null);
    setContent(fact.content ?? "");
    setAdding(false);
  }

  function groupLabel(group: QuickFactGroup) {
    return group.template_label || t("modules.quick_facts.untitled_category");
  }

  const showForm = adding || editingId !== null;

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.quick_facts.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: isCollapsed ? 0 : "16px 24px", display: isCollapsed ? "none" : "block" },
      }}
      extra={
        <Space>
          <Tooltip title={t("modules.quick_facts.toggle")}>
            <Button
              type="text"
              icon={isCollapsed ? <EyeInvisibleOutlined /> : <EyeOutlined />}
              onClick={() => toggleMutation.mutate()}
            />
          </Tooltip>
          {!showForm && (
            <Button type="text" icon={<PlusOutlined />} onClick={startAdd} style={{ color: token.colorPrimary }}>
              {t("modules.quick_facts.add")}
            </Button>
          )}
        </Space>
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
            <Select
              placeholder={t("modules.quick_facts.category_placeholder")}
              value={selectedTemplateId ?? undefined}
              onChange={setSelectedTemplateId}
              disabled={editingId !== null}
              options={editableGroups.map((group) => ({
                label: groupLabel(group),
                value: group.template_id,
              }))}
            />
            <Input
              placeholder={t("modules.quick_facts.content_placeholder")}
              value={content}
              onChange={(e) => setContent(e.target.value)}
            />
            <Space>
              <Button
                type="primary"
                onClick={() => saveMutation.mutate()}
                loading={saveMutation.isPending}
                disabled={!content.trim() || !selectedTemplateId}
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

      {isLoading ? (
        <Typography.Text type="secondary">{t("common.loading")}</Typography.Text>
      ) : !hasFacts ? (
        <Empty description={t("modules.quick_facts.no_facts")} />
      ) : (
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          {visibleGroups.map((group) => (
            <div key={group.template_id}>
              <Typography.Text
                type="secondary"
                style={{ fontSize: 12, fontWeight: 600, textTransform: "uppercase", letterSpacing: 0.4 }}
              >
                {groupLabel(group)}
              </Typography.Text>
              <Space direction="vertical" size={4} style={{ width: "100%", marginTop: 8 }}>
                {(group.facts ?? []).map((fact) => (
                  <div
                    key={fact.id}
                    style={{
                      borderRadius: token.borderRadius,
                      padding: "10px 12px",
                      transition: "background 0.2s",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "space-between",
                      gap: 12,
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.background = token.colorFillQuaternary;
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.background = "transparent";
                    }}
                  >
                    <Typography.Text style={{ fontWeight: 500 }}>{fact.content}</Typography.Text>
                    <Space size={0}>
                      <Button type="text" size="small" icon={<EditOutlined />} onClick={() => startEdit(fact)} />
                      <Popconfirm title={t("modules.quick_facts.delete_confirm")} onConfirm={() => deleteMutation.mutate(fact)}>
                        <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                      </Popconfirm>
                    </Space>
                  </div>
                ))}
              </Space>
            </div>
          ))}
        </Space>
      )}
    </Card>
  );
}
