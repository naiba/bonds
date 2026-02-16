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
  Tag,
  Select,
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined, RightOutlined, DownOutlined, AppstoreOutlined, ArrowUpOutlined, ArrowDownOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { PersonalizeItem, APIError } from "@/api";

const { Title, Text } = Typography;

const sectionKeys = [
  "genders", "pronouns", "address-types", "pet-categories",
  "contact-info-types", "relationship-types", "templates", "modules",
  "currencies", "religions", "call-reasons",
  "gift-occasions", "gift-states", "post-templates", "group-types",
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
  "post-templates": "settings.personalize.post_templates",
  "group-types": "settings.personalize.group_types",
};

interface SubItemConfig {
  labelKey: string;
  addKey: string;
  fields: Array<{ key: string; placeholder: string }>;
  list: (parentId: number) => Promise<{ data?: Array<Record<string, unknown>> }>;
  create: (parentId: number, body: Record<string, string>) => Promise<unknown>;
  update: (parentId: number, itemId: number, body: Record<string, string>) => Promise<unknown>;
  remove: (parentId: number, itemId: number) => Promise<unknown>;
  position?: (parentId: number, itemId: number, position: number) => Promise<unknown>;
}

const subItemConfigs: Record<string, SubItemConfig> = {
  "templates": {
    labelKey: "settings.personalize.pages",
    addKey: "settings.personalize.add_page",
    fields: [{ key: "name", placeholder: "common.name" }],
    list: (id) => api.templatePages.personalizeTemplatesPagesList(id),
    create: (id, b) => api.templatePages.personalizeTemplatesPagesCreate(id, { name: b.name, slug: b.name.toLowerCase().replace(/\s+/g, "-") }),
    update: (id, itemId, b) => api.templatePages.personalizeTemplatesPagesUpdate(id, itemId, { name: b.name }),
    remove: (id, itemId) => api.templatePages.personalizeTemplatesPagesDelete(id, itemId),
  },
  "post-templates": {
    labelKey: "settings.personalize.sections",
    addKey: "settings.personalize.add_section",
    fields: [{ key: "label", placeholder: "common.label" }],
    list: (id) => api.postTemplateSections.personalizePostTemplatesSectionsList(id),
    create: (id, b) => api.postTemplateSections.personalizePostTemplatesSectionsCreate(id, { label: b.label }),
    update: (id, itemId, b) => api.postTemplateSections.personalizePostTemplatesSectionsUpdate(id, itemId, { label: b.label }),
    remove: (id, itemId) => api.postTemplateSections.personalizePostTemplatesSectionsDelete(id, itemId),
    position: (id, itemId, pos) => api.postTemplateSections.personalizePostTemplatesSectionsPositionCreate(id, itemId, { position: pos }),
  },
  "group-types": {
    labelKey: "settings.personalize.roles",
    addKey: "settings.personalize.add_role",
    fields: [{ key: "label", placeholder: "common.label" }],
    list: (id) => api.groupTypeRoles.personalizeGroupTypesRolesList(id),
    create: (id, b) => api.groupTypeRoles.personalizeGroupTypesRolesCreate(id, { label: b.label }),
    update: (id, itemId, b) => api.groupTypeRoles.personalizeGroupTypesRolesUpdate(id, itemId, { label: b.label }),
    remove: (id, itemId) => api.groupTypeRoles.personalizeGroupTypesRolesDelete(id, itemId),
    position: (id, itemId, pos) => api.groupTypeRoles.personalizeGroupTypesRolesPositionCreate(id, itemId, { position: pos }),
  },
  "call-reasons": {
    labelKey: "settings.personalize.reasons",
    addKey: "settings.personalize.add_reason",
    fields: [{ key: "label", placeholder: "common.label" }],
    list: (id) => api.callReasons.personalizeCallReasonsReasonsList(id),
    create: (id, b) => api.callReasons.personalizeCallReasonsReasonsCreate(id, { label: b.label }),
    update: (id, itemId, b) => api.callReasons.personalizeCallReasonsReasonsUpdate(id, itemId, { label: b.label }),
    remove: (id, itemId) => api.callReasons.personalizeCallReasonsReasonsDelete(id, itemId),
  },
  "relationship-types": {
    labelKey: "settings.personalize.types",
    addKey: "settings.personalize.add_type",
    fields: [
      { key: "name", placeholder: "common.name" },
      { key: "name_reverse_relationship", placeholder: "settings.personalize.name_reverse" },
    ],
    list: (id) => api.relationshipTypes.personalizeRelationshipTypesTypesList(id),
    create: (id, b) => api.relationshipTypes.personalizeRelationshipTypesTypesCreate(id, { name: b.name, name_reverse_relationship: b.name_reverse_relationship }),
    update: (id, itemId, b) => api.relationshipTypes.personalizeRelationshipTypesTypesUpdate(id, itemId, { name: b.name, name_reverse_relationship: b.name_reverse_relationship }),
    remove: (id, itemId) => api.relationshipTypes.personalizeRelationshipTypesTypesDelete(id, itemId),
  },
};

function SubItemsPanel({ parentId, sectionKey }: { parentId: number; sectionKey: string }) {
  const config = subItemConfigs[sectionKey];
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [formValues, setFormValues] = useState<Record<string, string>>({});
  const [expandedModulePageId, setExpandedModulePageId] = useState<number | null>(null);
  const showModules = sectionKey === "templates";
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["settings", "personalize", sectionKey, "sub-items", parentId];

  const { data: items = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await config.list(parentId);
      return (res.data ?? []) as Array<Record<string, unknown>>;
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => {
      if (editingId) {
        return config.update(parentId, editingId, formValues);
      }
      return config.create(parentId, formValues);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (itemId: number) => config.remove(parentId, itemId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: qk }),
    onError: (e: APIError) => message.error(e.message),
  });

  const subPositionMutation = useMutation({
    mutationFn: ({ itemId, position }: { itemId: number; position: number }) => {
      if (!config.position) throw new Error("No position API");
      return config.position(parentId, itemId, position);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: qk }),
    onError: (e: APIError) => message.error(e.message),
  });

  function resetForm() {
    setAdding(false);
    setEditingId(null);
    setFormValues({});
  }

  function startEdit(item: Record<string, unknown>) {
    setEditingId((item.id as number) ?? null);
    const vals: Record<string, string> = {};
    for (const f of config.fields) {
      vals[f.key] = String(item[f.key] ?? "");
    }
    setFormValues(vals);
    setAdding(false);
  }

  function updateField(key: string, value: string) {
    setFormValues((prev) => ({ ...prev, [key]: value }));
  }

  const hasValues = config.fields.every((f) => (formValues[f.key] ?? "").trim());
  const showForm = adding || editingId !== null;
  const hasPosition = !!config.position;

  function getDisplayLabel(item: Record<string, unknown>): string {
    const primary = String(item[config.fields[0].key] ?? item.label ?? "");
    if (sectionKey === "relationship-types" && item.name_reverse_relationship) {
      return `${primary} ↔ ${item.name_reverse_relationship}`;
    }
    return primary;
  }

  return (
    <div style={{ paddingLeft: 16, paddingTop: 8, paddingBottom: 4, borderLeft: "2px solid #f0f0f0" }}>
      <Text type="secondary" style={{ fontSize: 12, marginBottom: 8, display: "block" }}>
        {t(config.labelKey)}
      </Text>

      {!showForm && (
        <Button
          type="dashed"
          size="small"
          icon={<PlusOutlined />}
          onClick={() => setAdding(true)}
          style={{ marginBottom: 8 }}
          block
        >
          {t(config.addKey)}
        </Button>
      )}

      {showForm && (
        <div style={{ marginBottom: 8 }}>
          <Space.Compact style={{ width: "100%" }}>
            {config.fields.map((f) => (
              <Input
                key={f.key}
                size="small"
                placeholder={t(f.placeholder)}
                value={formValues[f.key] ?? ""}
                onChange={(e) => updateField(f.key, e.target.value)}
                onPressEnter={() => hasValues && saveMutation.mutate()}
              />
            ))}
            <Button
              type="primary"
              size="small"
              onClick={() => hasValues && saveMutation.mutate()}
              loading={saveMutation.isPending}
            >
              {editingId ? t("common.update") : t("common.add")}
            </Button>
          </Space.Compact>
          <Button type="text" size="small" onClick={resetForm} style={{ marginTop: 2 }}>
            {t("common.cancel")}
          </Button>
        </div>
      )}

      <List
        loading={isLoading}
        dataSource={items}
        locale={{ emptyText: <Empty description={t("settings.personalize.no_items")} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
        size="small"
        renderItem={(item: Record<string, unknown>, index: number) => (
          <div>
            <List.Item
              style={{ padding: "4px 0" }}
              actions={[
                ...(hasPosition ? [
                  <Button
                    key="up"
                    type="text"
                    size="small"
                    icon={<ArrowUpOutlined />}
                    title={t("settings.personalize.move_up")}
                    disabled={index === 0}
                    onClick={() => subPositionMutation.mutate({ itemId: item.id as number, position: index - 1 })}
                  />,
                  <Button
                    key="down"
                    type="text"
                    size="small"
                    icon={<ArrowDownOutlined />}
                    title={t("settings.personalize.move_down")}
                    disabled={index === items.length - 1}
                    onClick={() => subPositionMutation.mutate({ itemId: item.id as number, position: index + 1 })}
                  />,
                ] : []),
                ...(showModules ? [
                  <Button
                    key="modules"
                    type="text"
                    size="small"
                    icon={<AppstoreOutlined />}
                    title={t("settings.personalize.page_modules")}
                    onClick={() => setExpandedModulePageId(expandedModulePageId === (item.id as number) ? null : (item.id as number))}
                    style={expandedModulePageId === (item.id as number) ? { color: '#1677ff' } : undefined}
                  />,
                ] : []),
                <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => startEdit(item)} />,
                <Popconfirm key="d" title={t("settings.personalize.delete_confirm")} onConfirm={() => deleteMutation.mutate(item.id as number)}>
                  <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>,
              ]}
            >
              <span style={{ fontSize: 13 }}>{getDisplayLabel(item)}</span>
            </List.Item>
            {showModules && expandedModulePageId === (item.id as number) && (
              <ModulesPanel templateId={parentId} pageId={item.id as number} />
            )}
          </div>
        )}
      />
    </div>
  );
}

function ModulesPanel({ templateId, pageId }: { templateId: number; pageId: number }) {
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const [selectedModuleId, setSelectedModuleId] = useState<number | null>(null);

  const modulesQk = ["settings", "personalize", "templates", templateId, "pages", pageId, "modules"];

  const { data: pageModules = [], isLoading: loadingPageModules } = useQuery({
    queryKey: modulesQk,
    queryFn: async () => {
      const res = await api.templatePages.personalizeTemplatesPagesModulesList(templateId, pageId);
      return (res.data ?? []) as Array<Record<string, unknown>>;
    },
  });

  const { data: availableModules = [] } = useQuery({
    queryKey: ["settings", "personalize", "modules"],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail("modules");
      return res.data ?? [];
    },
  });

  const addModuleMutation = useMutation({
    mutationFn: (moduleId: number) =>
      api.templatePages.personalizeTemplatesPagesModulesCreate(templateId, pageId, { module_id: moduleId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: modulesQk });
      setSelectedModuleId(null);
      message.success(t("common.created"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const removeModuleMutation = useMutation({
    mutationFn: (moduleId: number) =>
      api.templatePages.personalizeTemplatesPagesModulesDelete(templateId, pageId, moduleId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: modulesQk });
      message.success(t("common.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const assignedModuleIds = new Set(pageModules.map((m: Record<string, unknown>) => m.module_id as number));
  const unassignedModules = availableModules.filter((m: PersonalizeItem) => !assignedModuleIds.has(m.id!));

  return (
    <div style={{ paddingLeft: 24, paddingTop: 8, paddingBottom: 4, borderLeft: "2px solid #e8e8e8" }}>
      <Text type="secondary" style={{ fontSize: 12, marginBottom: 8, display: "flex", alignItems: "center", gap: 4 }}>
        <AppstoreOutlined /> {t("settings.personalize.page_modules")}
      </Text>

      {loadingPageModules ? null : pageModules.length === 0 ? (
        <Text type="secondary" style={{ fontSize: 12, fontStyle: "italic", display: "block", marginBottom: 8 }}>
          {t("settings.personalize.no_modules")}
        </Text>
      ) : (
        <div style={{ display: "flex", flexWrap: "wrap", gap: 4, marginBottom: 8 }}>
          {pageModules.map((mod) => (
            <Tag
              key={mod.module_id as number}
              closable
              onClose={(e) => {
                e.preventDefault();
                removeModuleMutation.mutate(mod.module_id as number);
              }}
            >
              {(mod.module_name as string) ?? `Module #${mod.module_id}`}
            </Tag>
          ))}
        </div>
      )}

      {unassignedModules.length > 0 && (
        <Space.Compact size="small" style={{ width: "100%" }}>
          <Select
            size="small"
            style={{ flex: 1 }}
            placeholder={t("settings.personalize.available_modules")}
            value={selectedModuleId}
            onChange={(val) => setSelectedModuleId(val)}
            options={unassignedModules.map((m: PersonalizeItem) => ({ label: m.label ?? m.name, value: m.id }))}
          />
          <Button
            type="primary"
            size="small"
            icon={<PlusOutlined />}
            disabled={selectedModuleId === null}
            loading={addModuleMutation.isPending}
            onClick={() => { if (selectedModuleId !== null) addModuleMutation.mutate(selectedModuleId); }}
          >
            {t("settings.personalize.add_module")}
          </Button>
        </Space.Compact>
      )}
    </div>
  );
}

function SectionPanel({ sectionKey }: { sectionKey: string }) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [label, setLabel] = useState("");
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["settings", "personalize", sectionKey];
  const hasSubItems = sectionKey in subItemConfigs;

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

  const positionMutation = useMutation({
    mutationFn: ({ id, position }: { id: number; position: number }) =>
      api.personalize.personalizePositionCreate(sectionKey, id, { position }),
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

  function toggleExpand(id: number) {
    setExpandedId((prev) => (prev === id ? null : id));
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
        renderItem={(item: PersonalizeItem, index: number) => (
          <div>
            <List.Item
              actions={[
                <Button
                  key="up"
                  type="text"
                  size="small"
                  icon={<ArrowUpOutlined />}
                  title={t("settings.personalize.move_up")}
                  disabled={index === 0}
                  onClick={() => positionMutation.mutate({ id: item.id!, position: index - 1 })}
                />,
                <Button
                  key="down"
                  type="text"
                  size="small"
                  icon={<ArrowDownOutlined />}
                  title={t("settings.personalize.move_down")}
                  disabled={index === items.length - 1}
                  onClick={() => positionMutation.mutate({ id: item.id!, position: index + 1 })}
                />,
                <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => startEdit(item)} />,
                <Popconfirm key="d" title={t("settings.personalize.delete_confirm")} onConfirm={() => deleteMutation.mutate(item.id!)}>
                  <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>,
              ]}
            >
              <span style={{ display: "flex", alignItems: "center", gap: 6 }}>
                {hasSubItems && (
                  <Button
                    type="text"
                    size="small"
                    icon={expandedId === item.id ? <DownOutlined /> : <RightOutlined />}
                    onClick={() => toggleExpand(item.id!)}
                    style={{ padding: 0, width: 20, height: 20, minWidth: 20 }}
                  />
                )}
                {item.label}
              </span>
            </List.Item>
            {hasSubItems && expandedId === item.id && (
              <SubItemsPanel parentId={item.id!} sectionKey={sectionKey} />
            )}
          </div>
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
