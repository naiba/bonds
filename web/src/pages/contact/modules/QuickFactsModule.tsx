import { useState } from "react";
import type { ReactNode } from "react";
import {
  App,
  Button,
  Card,
  Empty,
  Image,
  Input,
  InputNumber,
  Popconfirm,
  Select,
  Space,
  Tooltip,
  Typography,
  Upload,
  theme,
} from "antd";
import type { UploadProps } from "antd";
import {
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  FileOutlined,
  PlusOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { APIError, CreateQuickFactRequest, QuickFact, QuickFactGroup, UpdateQuickFactRequest } from "@/api";
import { formatDate, useDateFormat } from "@/utils/dateFormat";

const QUICK_FACT_FIELD_TYPES = ["text", "number", "date", "select", "photo", "document"] as const;
type QuickFactFieldType = (typeof QUICK_FACT_FIELD_TYPES)[number];

function isQuickFactFieldType(value: string | undefined): value is QuickFactFieldType {
  return QUICK_FACT_FIELD_TYPES.some((fieldType) => fieldType === value);
}

function normalizeQuickFactFieldType(value: string | undefined): QuickFactFieldType {
  return isQuickFactFieldType(value) ? value : "text";
}

function isQuickFactFileFieldType(fieldType: QuickFactFieldType) {
  return fieldType === "photo" || fieldType === "document";
}

function formatFileSize(bytes: number | undefined) {
  if (bytes === undefined) return "";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function QuickFactsModule({
  vaultId,
  contactId,
  readOnly = false,
}: {
  vaultId: string | number;
  contactId: string | number;
  readOnly?: boolean;
}) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [selectedTemplateId, setSelectedTemplateId] = useState<number | null>(null);
  const [textValue, setTextValue] = useState("");
  const [numberValue, setNumberValue] = useState<number | null>(null);
  const [isCollapsed, setIsCollapsed] = useState(false);

  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const dateFormat = useDateFormat();
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
  const selectedGroup = editableGroups.find((group) => group.template_id === selectedTemplateId) ?? null;
  const selectedFieldType = normalizeQuickFactFieldType(selectedGroup?.field_type);
  const showForm = !readOnly && (adding || editingId !== null);

  const saveMutation = useMutation({
    mutationFn: () => {
      if (!selectedGroup?.template_id) {
        return Promise.reject({ message: t("modules.quick_facts.select_category_required") });
      }
      const data = buildScalarRequest(selectedGroup);
      if (editingId !== null) {
        return api.quickFacts.contactsQuickFactsUpdate(
          String(vaultId),
          String(contactId),
          selectedGroup.template_id,
          editingId,
          data,
        );
      }
      return api.quickFacts.contactsQuickFactsCreate(String(vaultId), String(contactId), selectedGroup.template_id, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const fileMutation = useMutation({
    mutationFn: ({ templateId, factId, file }: { templateId: number; factId: number | null; file: File }) => {
      if (factId !== null) {
        return api.quickFacts.contactsQuickFactsFileUpdate(String(vaultId), String(contactId), templateId, factId, { file });
      }
      return api.quickFacts.contactsQuickFactsFileCreate(String(vaultId), String(contactId), templateId, { file });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.quick_facts.file_uploaded"));
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
    setTextValue("");
    setNumberValue(null);
  }

  function applyDefaultValue(group: QuickFactGroup | null) {
    const fieldType = normalizeQuickFactFieldType(group?.field_type);
    const defaultValue = group?.default_value ?? "";
    if (fieldType === "number") {
      const parsed = Number(defaultValue);
      setNumberValue(Number.isFinite(parsed) ? parsed : null);
      setTextValue("");
      return;
    }
    setNumberValue(null);
    setTextValue(defaultValue);
  }

  function startAdd() {
    const firstGroup = editableGroups[0] ?? null;
    setAdding(true);
    setEditingId(null);
    setSelectedTemplateId(firstGroup?.template_id ?? null);
    applyDefaultValue(firstGroup);
  }

  function startEdit(fact: QuickFact) {
    const templateId = fact.vault_quick_facts_template_id ?? null;
    const group = editableGroups.find((candidate) => candidate.template_id === templateId) ?? null;
    const fieldType = normalizeQuickFactFieldType(group?.field_type ?? fact.field_type);
    setEditingId(fact.id ?? null);
    setSelectedTemplateId(templateId);
    setAdding(false);

    if (fieldType === "number") {
      setNumberValue(fact.value_number ?? null);
      setTextValue("");
      return;
    }
    setNumberValue(null);
    setTextValue(fact.value_text ?? fact.value_date ?? fact.value_option ?? fact.content ?? "");
  }

  function selectTemplate(templateId: number) {
    const group = editableGroups.find((candidate) => candidate.template_id === templateId) ?? null;
    setSelectedTemplateId(templateId);
    applyDefaultValue(group);
  }

  function buildScalarRequest(group: QuickFactGroup): CreateQuickFactRequest | UpdateQuickFactRequest {
    const fieldType = normalizeQuickFactFieldType(group.field_type);
    switch (fieldType) {
      case "number":
        return { value_number: numberValue ?? undefined };
      case "date":
        return { value_date: textValue };
      case "select":
        return { value_option: textValue };
      case "text":
        return { value_text: textValue };
      case "photo":
      case "document":
        return {};
    }
  }

  function canSaveScalar() {
    if (!selectedGroup || isQuickFactFileFieldType(selectedFieldType)) return false;
    if (selectedFieldType === "number") return numberValue !== null;
    return textValue.trim().length > 0;
  }

  function groupLabel(group: QuickFactGroup) {
    return group.template_label || t("modules.quick_facts.untitled_category");
  }

  function formatQuickFactDate(value: string | undefined) {
    if (!value) return "";
    return formatDate(`${value}T00:00:00`, { ...dateFormat, tz: undefined });
  }

  function downloadUrl(fileId: number) {
    return `/api/vaults/${vaultId}/files/${fileId}/download?token=${localStorage.getItem("token")}`;
  }

  function uploadFile(file: File) {
    if (!selectedGroup?.template_id) {
      message.error(t("modules.quick_facts.select_category_required"));
      return;
    }
    if (!isQuickFactFileFieldType(selectedFieldType)) {
      message.error(t("modules.quick_facts.unsupported_field_type"));
      return;
    }
    fileMutation.mutate({ templateId: selectedGroup.template_id, factId: editingId, file });
  }

  const beforeUpload: UploadProps["beforeUpload"] = (file) => {
    uploadFile(file);
    return false;
  };

  function renderValueInput(group: QuickFactGroup): ReactNode {
    const fieldType = normalizeQuickFactFieldType(group.field_type);
    if (fieldType === "number") {
      return (
        <InputNumber
          style={{ width: "100%" }}
          placeholder={t("modules.quick_facts.number_placeholder")}
          value={numberValue}
          onChange={(value) => setNumberValue(typeof value === "number" ? value : null)}
        />
      );
    }
    if (fieldType === "date") {
      return (
        <Input
          type="date"
          placeholder={t("modules.quick_facts.date_placeholder")}
          value={textValue}
          onChange={(event) => setTextValue(event.target.value)}
        />
      );
    }
    if (fieldType === "select") {
      return (
        <Select
          placeholder={t("modules.quick_facts.select_placeholder")}
          value={textValue || undefined}
          onChange={setTextValue}
          options={(group.select_options ?? []).map((option) => ({ label: option, value: option }))}
        />
      );
    }
    if (isQuickFactFileFieldType(fieldType)) {
      return (
        <Upload
          beforeUpload={beforeUpload}
          showUploadList={false}
          multiple={false}
          accept={fieldType === "photo" ? "image/*" : undefined}
        >
          <Button icon={<UploadOutlined />} loading={fileMutation.isPending}>
            {editingId !== null
              ? t("modules.quick_facts.replace_file")
              : t(fieldType === "photo" ? "modules.quick_facts.upload_photo" : "modules.quick_facts.upload_document")}
          </Button>
        </Upload>
      );
    }
    return (
      <Input
        placeholder={t("modules.quick_facts.text_placeholder")}
        value={textValue}
        onChange={(event) => setTextValue(event.target.value)}
      />
    );
  }

  function renderFactValue(fact: QuickFact, group: QuickFactGroup): ReactNode {
    const fieldType = normalizeQuickFactFieldType(group.field_type ?? fact.field_type);
    if (fieldType === "photo") {
      return fact.file?.id ? (
        <Image
          width={96}
          height={96}
          src={downloadUrl(fact.file.id)}
          style={{ objectFit: "cover", borderRadius: token.borderRadius }}
        />
      ) : (
        <Typography.Text type="secondary">{t("modules.quick_facts.file_missing")}</Typography.Text>
      );
    }
    if (fieldType === "document") {
      return fact.file?.id ? (
        <Space direction="vertical" size={0}>
          <Button type="link" icon={<FileOutlined />} href={downloadUrl(fact.file.id)} target="_blank" style={{ padding: 0 }}>
            {fact.file.name ?? t("modules.quick_facts.download_file")}
          </Button>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {[fact.file.mime_type, formatFileSize(fact.file.size)].filter(Boolean).join(" · ")}
          </Typography.Text>
        </Space>
      ) : (
        <Typography.Text type="secondary">{t("modules.quick_facts.file_missing")}</Typography.Text>
      );
    }
    if (fieldType === "date") {
      return <Typography.Text style={{ fontWeight: 500 }}>{formatQuickFactDate(fact.value_date ?? fact.content)}</Typography.Text>;
    }
    if (fieldType === "number") {
      return <Typography.Text style={{ fontWeight: 500 }}>{fact.value_number ?? fact.content ?? t("modules.quick_facts.empty_value")}</Typography.Text>;
    }
    if (fieldType === "select") {
      return <Typography.Text style={{ fontWeight: 500 }}>{fact.value_option ?? fact.content ?? t("modules.quick_facts.empty_value")}</Typography.Text>;
    }
    return <Typography.Text style={{ fontWeight: 500 }}>{fact.value_text ?? fact.content ?? t("modules.quick_facts.empty_value")}</Typography.Text>;
  }

  if (readOnly && !isLoading && !hasFacts) return null;

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.quick_facts.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: isCollapsed ? 0 : "16px 24px", display: isCollapsed ? "none" : "block" },
      }}
      extra={
        !readOnly && (
          <Space>
            <Tooltip title={t("modules.quick_facts.toggle")}>
              <Button
                type="text"
                icon={isCollapsed ? <EyeInvisibleOutlined /> : <EyeOutlined />}
                onClick={() => toggleMutation.mutate()}
              />
            </Tooltip>
            {!showForm && (
              <Button type="link" icon={<PlusOutlined />} onClick={startAdd} disabled={editableGroups.length === 0}>
                {t("modules.quick_facts.add")}
              </Button>
            )}
          </Space>
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
            <Select
              placeholder={t("modules.quick_facts.category_placeholder")}
              value={selectedTemplateId ?? undefined}
              onChange={selectTemplate}
              disabled={editingId !== null}
              options={editableGroups.map((group) => ({
                label: groupLabel(group),
                value: group.template_id,
              }))}
            />
            {selectedGroup && renderValueInput(selectedGroup)}
            {selectedGroup?.help_text && <Typography.Text type="secondary">{selectedGroup.help_text}</Typography.Text>}
            <Space>
              {!isQuickFactFileFieldType(selectedFieldType) && (
                <Button
                  type="primary"
                  onClick={() => saveMutation.mutate()}
                  loading={saveMutation.isPending}
                  disabled={!canSaveScalar() || !selectedTemplateId}
                  size="small"
                >
                  {editingId ? t("common.update") : t("common.save")}
                </Button>
              )}
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
                    onMouseEnter={(event) => {
                      event.currentTarget.style.background = token.colorFillQuaternary;
                    }}
                    onMouseLeave={(event) => {
                      event.currentTarget.style.background = "transparent";
                    }}
                  >
                    <div style={{ minWidth: 0, flex: 1 }}>{renderFactValue(fact, group)}</div>
                    {!readOnly && (
                      <Space size={0}>
                        {fact.file?.id && (
                          <Button
                            type="text"
                            size="small"
                            icon={<DownloadOutlined />}
                            href={downloadUrl(fact.file.id)}
                            target="_blank"
                          />
                        )}
                        <Button type="text" size="small" icon={<EditOutlined />} onClick={() => startEdit(fact)} />
                        <Popconfirm title={t("modules.quick_facts.delete_confirm")} onConfirm={() => deleteMutation.mutate(fact)}>
                          <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                        </Popconfirm>
                      </Space>
                    )}
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
