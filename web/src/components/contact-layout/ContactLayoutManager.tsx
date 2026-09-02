import { useMemo, useRef, useState } from "react";
import {
  App,
  Alert,
  Button,
  Card,
  Empty,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Switch,
  Tag,
  Typography,
} from "antd";
import {
  CopyOutlined,
  DeleteOutlined,
  DragOutlined,
  EditOutlined,
  PlusOutlined,
  SaveOutlined,
  StarOutlined,
  UndoOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
} from "@ant-design/icons";
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type {
  APIError,
  ContactLayout,
  ContactLayoutModuleDefinition,
  ContactLayoutTemplateSummary,
} from "@/api";

const { Text } = Typography;

type DraftModule = {
  key: string;
  name: string;
};

type DraftPage = {
  id?: number;
  clientKey: string;
  name: string;
  slug: string;
  type: string;
  visible: boolean;
  modules: DraftModule[];
};

function createDraftPages(layout: ContactLayout): DraftPage[] {
  return (layout.pages ?? []).map((page, pageIndex) => ({
    id: page.id,
    clientKey: `page-${page.id ?? pageIndex}`,
    name: page.name ?? "",
    slug: page.slug ?? `section-${pageIndex + 1}`,
    type: page.type ?? "",
    visible: page.visible !== false,
    modules: (page.modules ?? []).map((module) => ({
      key: module.key ?? "",
      name: module.name ?? module.key ?? "",
    })),
  }));
}

type ContactLayoutManagerProps = {
  vaultId: string;
  initialTemplateId?: number;
  onTemplateChange?: (templateId: number) => void;
};

function layoutQueryKey(vaultId: string, templateId: number) {
  return [
    "vaults",
    vaultId,
    "contact-layout",
    "templates",
    templateId,
  ] as const;
}

function templatesQueryKey(vaultId: string) {
  return ["vaults", vaultId, "contact-layout", "templates"] as const;
}

export default function ContactLayoutManager({
  vaultId,
  initialTemplateId,
  onTemplateChange,
}: ContactLayoutManagerProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [selectedTemplateId, setSelectedTemplateId] = useState<number | null>(
    initialTemplateId ?? null,
  );
  const [createOpen, setCreateOpen] = useState(false);
  const [newTemplateName, setNewTemplateName] = useState("");
  const [renameOpen, setRenameOpen] = useState(false);
  const [renameValue, setRenameValue] = useState("");

  const { data: templates = [], isLoading: templatesLoading } = useQuery<
    ContactLayoutTemplateSummary[]
  >({
    queryKey: templatesQueryKey(vaultId),
    queryFn: async () =>
      (await api.contactLayouts.contactLayoutTemplatesList(vaultId)).data ?? [],
  });

  const activeTemplateId =
    selectedTemplateId ??
    (initialTemplateId &&
    templates.some((item) => item.id === initialTemplateId)
      ? initialTemplateId
      : templates.find((item) => item.is_default)?.id) ??
    templates[0]?.id;

  const { data: moduleDefinitions = [] } = useQuery<
    ContactLayoutModuleDefinition[]
  >({
    queryKey: ["vaults", vaultId, "contact-layout", "modules"],
    queryFn: async () =>
      (await api.contactLayouts.contactLayoutModulesList(vaultId)).data ?? [],
  });

  const { data: layout, isLoading: layoutLoading } = useQuery<ContactLayout>({
    queryKey: layoutQueryKey(vaultId, activeTemplateId ?? 0),
    queryFn: async () =>
      (
        await api.contactLayouts.contactLayoutTemplatesDetail(
          vaultId,
          activeTemplateId!,
        )
      ).data!,
    enabled: activeTemplateId != null,
  });

  const invalidateTemplates = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: templatesQueryKey(vaultId) }),
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId] }),
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts"],
      }),
    ]);
  };

  const createMutation = useMutation({
    mutationFn: () =>
      api.contactLayouts.contactLayoutTemplatesCreate(vaultId, {
        name: newTemplateName.trim(),
        source_template_id: activeTemplateId,
      }),
    onSuccess: async (response) => {
      const createdId = response.data?.id;
      await invalidateTemplates();
      if (createdId != null) {
        setSelectedTemplateId(createdId);
        onTemplateChange?.(createdId);
      }
      setCreateOpen(false);
      setNewTemplateName("");
      message.success(t("contact.layout.created"));
    },
    onError: (error: APIError) => message.error(error.message),
  });

  const renameMutation = useMutation({
    mutationFn: () =>
      api.contactLayouts.contactLayoutTemplatesUpdate(
        vaultId,
        activeTemplateId!,
        { name: renameValue.trim() },
      ),
    onSuccess: async () => {
      await invalidateTemplates();
      await queryClient.invalidateQueries({
        queryKey: layoutQueryKey(vaultId, activeTemplateId!),
      });
      setRenameOpen(false);
      message.success(t("common.saved"));
    },
    onError: (error: APIError) => message.error(error.message),
  });

  const defaultMutation = useMutation({
    mutationFn: () =>
      api.contactLayouts.contactLayoutTemplatesDefaultUpdate(
        vaultId,
        activeTemplateId!,
      ),
    onSuccess: async () => {
      await invalidateTemplates();
      message.success(t("contact.layout.default_updated"));
    },
    onError: (error: APIError) => message.error(error.message),
  });

  const deleteMutation = useMutation({
    mutationFn: () =>
      api.contactLayouts.contactLayoutTemplatesDelete(
        vaultId,
        activeTemplateId!,
      ),
    onSuccess: async () => {
      setSelectedTemplateId(null);
      await invalidateTemplates();
      message.success(t("common.deleted"));
    },
    onError: (error: APIError) => message.error(error.message),
  });

  if (templatesLoading) {
    return <Spin />;
  }

  if (templates.length === 0 || activeTemplateId == null) {
    return <Empty description={t("contact.layout.no_templates")} />;
  }

  const selectedTemplate = templates.find(
    (template) => template.id === activeTemplateId,
  );

  return (
    <Space direction="vertical" size="middle" style={{ width: "100%" }}>
      <div
        style={{
          display: "flex",
          gap: 8,
          alignItems: "center",
          flexWrap: "wrap",
        }}
      >
        <Select
          aria-label={t("contact.layout.template")}
          value={activeTemplateId}
          style={{ minWidth: 220, flex: 1 }}
          options={templates.map((template) => ({
            value: template.id,
            label: (
              <Space size={4}>
                <span>{template.name}</span>
                {template.is_default && (
                  <Tag color="green">{t("contact.layout.default")}</Tag>
                )}
              </Space>
            ),
          }))}
          onChange={(templateId) => {
            setSelectedTemplateId(templateId);
            onTemplateChange?.(templateId);
          }}
        />
        <Button icon={<CopyOutlined />} onClick={() => setCreateOpen(true)}>
          {t("contact.layout.duplicate")}
        </Button>
        <Button
          icon={<EditOutlined />}
          onClick={() => {
            setRenameValue(selectedTemplate?.name ?? "");
            setRenameOpen(true);
          }}
        >
          {t("contact.layout.rename")}
        </Button>
        {!selectedTemplate?.is_default && (
          <Button
            icon={<StarOutlined />}
            loading={defaultMutation.isPending}
            onClick={() => defaultMutation.mutate()}
          >
            {t("contact.layout.make_default")}
          </Button>
        )}
        {selectedTemplate?.can_be_deleted && (
          <Popconfirm
            title={t("contact.layout.delete_confirm")}
            description={
              selectedTemplate.contact_count
                ? t("contact.layout.in_use", {
                    count: selectedTemplate.contact_count,
                  })
                : undefined
            }
            disabled={(selectedTemplate.contact_count ?? 0) > 0}
            onConfirm={() => deleteMutation.mutate()}
          >
            <Button
              danger
              icon={<DeleteOutlined />}
              disabled={(selectedTemplate.contact_count ?? 0) > 0}
            />
          </Popconfirm>
        )}
      </div>

      {selectedTemplate && (selectedTemplate.contact_count ?? 0) > 0 && (
        <Alert
          type="info"
          showIcon
          message={t("contact.layout.shared_notice", {
            count: selectedTemplate.contact_count,
          })}
        />
      )}

      {layoutLoading || !layout ? (
        <div style={{ textAlign: "center", padding: 32 }}>
          <Spin />
        </div>
      ) : (
        <LayoutDraftEditor
          key={`${layout.id}:${layout.revision}`}
          vaultId={vaultId}
          layout={layout}
          moduleDefinitions={moduleDefinitions}
        />
      )}

      <Modal
        title={t("contact.layout.duplicate")}
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => createMutation.mutate()}
        okButtonProps={{ disabled: newTemplateName.trim().length === 0 }}
        confirmLoading={createMutation.isPending}
      >
        <Input
          autoFocus
          value={newTemplateName}
          onChange={(event) => setNewTemplateName(event.target.value)}
          placeholder={t("contact.layout.template_name")}
          onPressEnter={() => newTemplateName.trim() && createMutation.mutate()}
        />
      </Modal>

      <Modal
        title={t("contact.layout.rename")}
        open={renameOpen}
        onCancel={() => setRenameOpen(false)}
        onOk={() => renameMutation.mutate()}
        okButtonProps={{ disabled: renameValue.trim().length === 0 }}
        confirmLoading={renameMutation.isPending}
      >
        <Input
          autoFocus
          value={renameValue}
          onChange={(event) => setRenameValue(event.target.value)}
          onPressEnter={() => renameValue.trim() && renameMutation.mutate()}
        />
      </Modal>
    </Space>
  );
}

function LayoutDraftEditor({
  vaultId,
  layout,
  moduleDefinitions,
}: {
  vaultId: string;
  layout: ContactLayout;
  moduleDefinitions: ContactLayoutModuleDefinition[];
}) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [pages, setPages] = useState<DraftPage[]>(() =>
    createDraftPages(layout),
  );
  const [newPageOpen, setNewPageOpen] = useState(false);
  const [newPageName, setNewPageName] = useState("");
  const nextNewPageKey = useRef(0);
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  const assignedKeys = useMemo(
    () =>
      new Set(pages.flatMap((page) => page.modules.map((item) => item.key))),
    [pages],
  );

  const saveMutation = useMutation({
    mutationFn: () =>
      api.contactLayouts.contactLayoutTemplatesLayoutUpdate(
        vaultId,
        layout.id!,
        {
          expected_revision: layout.revision!,
          pages: pages.map((page) => ({
            id: page.id,
            name: page.name.trim(),
            slug: page.slug,
            type: page.type || undefined,
            visible: page.visible,
            modules: page.modules.map((module) => ({ key: module.key })),
          })),
        },
      ),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: layoutQueryKey(vaultId, layout.id!),
        }),
        queryClient.invalidateQueries({
          queryKey: templatesQueryKey(vaultId),
        }),
        queryClient.invalidateQueries({
          queryKey: ["vaults", vaultId, "contacts"],
        }),
      ]);
      message.success(t("contact.layout.saved"));
    },
    onError: (error: APIError) => {
      message.error(error.message);
      queryClient.invalidateQueries({
        queryKey: layoutQueryKey(vaultId, layout.id!),
      });
    },
  });

  const updatePage = (pageIndex: number, next: Partial<DraftPage>) => {
    setPages((current) =>
      current.map((page, index) =>
        index === pageIndex ? { ...page, ...next } : page,
      ),
    );
  };

  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (!over || active.id === over.id) return;
    setPages((current) => {
      const oldIndex = current.findIndex(
        (page) => page.clientKey === active.id,
      );
      const newIndex = current.findIndex((page) => page.clientKey === over.id);
      return oldIndex < 0 || newIndex < 0
        ? current
        : arrayMove(current, oldIndex, newIndex);
    });
  };

  const addPage = () => {
    const baseSlug =
      newPageName
        .trim()
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-|-$/g, "") || "section";
    let slug = baseSlug;
    let suffix = 2;
    const slugs = new Set(pages.map((page) => page.slug));
    while (slugs.has(slug)) {
      slug = `${baseSlug}-${suffix}`;
      suffix += 1;
    }
    setPages((current) => [
      ...current,
      {
        clientKey: `new-${nextNewPageKey.current++}`,
        name: newPageName.trim(),
        slug,
        type: "",
        visible: true,
        modules: [],
      },
    ]);
    setNewPageName("");
    setNewPageOpen(false);
  };

  return (
    <Space direction="vertical" size="middle" style={{ width: "100%" }}>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={onDragEnd}
      >
        <SortableContext
          items={pages.map((page) => page.clientKey)}
          strategy={verticalListSortingStrategy}
        >
          {pages.map((page, pageIndex) => (
            <SortablePage
              key={page.clientKey}
              page={page}
              pageIndex={pageIndex}
              pageCount={pages.length}
              availableModules={moduleDefinitions.filter(
                (definition) =>
                  definition.key && !assignedKeys.has(definition.key),
              )}
              allPages={pages}
              onChange={(next) => updatePage(pageIndex, next)}
              onDelete={() =>
                setPages((current) =>
                  current.filter((_, index) => index !== pageIndex),
                )
              }
              onMoveModule={(moduleIndex, targetPageIndex) => {
                setPages((current) => {
                  const next = current.map((item) => ({
                    ...item,
                    modules: [...item.modules],
                  }));
                  const [module] = next[pageIndex]!.modules.splice(
                    moduleIndex,
                    1,
                  );
                  if (module) next[targetPageIndex]!.modules.push(module);
                  return next;
                });
              }}
            />
          ))}
        </SortableContext>
      </DndContext>

      <Button
        type="dashed"
        block
        icon={<PlusOutlined />}
        onClick={() => setNewPageOpen(true)}
      >
        {t("contact.layout.add_section")}
      </Button>

      <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
        <Button
          icon={<UndoOutlined />}
          onClick={() => setPages(createDraftPages(layout))}
        >
          {t("contact.layout.reset")}
        </Button>
        <Button
          type="primary"
          icon={<SaveOutlined />}
          loading={saveMutation.isPending}
          disabled={pages.length === 0 || pages.every((page) => !page.visible)}
          onClick={() => saveMutation.mutate()}
        >
          {t("common.save")}
        </Button>
      </div>

      <Modal
        title={t("contact.layout.add_section")}
        open={newPageOpen}
        onCancel={() => setNewPageOpen(false)}
        onOk={addPage}
        okButtonProps={{ disabled: newPageName.trim().length === 0 }}
      >
        <Input
          autoFocus
          value={newPageName}
          onChange={(event) => setNewPageName(event.target.value)}
          placeholder={t("contact.layout.section_name")}
          onPressEnter={() => newPageName.trim() && addPage()}
        />
      </Modal>
    </Space>
  );
}

function SortablePage({
  page,
  pageIndex,
  pageCount,
  availableModules,
  allPages,
  onChange,
  onDelete,
  onMoveModule,
}: {
  page: DraftPage;
  pageIndex: number;
  pageCount: number;
  availableModules: ContactLayoutModuleDefinition[];
  allPages: DraftPage[];
  onChange: (next: Partial<DraftPage>) => void;
  onDelete: () => void;
  onMoveModule: (moduleIndex: number, targetPageIndex: number) => void;
}) {
  const { t } = useTranslation();
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: page.clientKey });
  const [moduleToAdd, setModuleToAdd] = useState<string | null>(null);

  return (
    <Card
      ref={setNodeRef}
      style={{
        transform: CSS.Transform.toString(transform),
        transition,
        opacity: isDragging ? 0.6 : 1,
      }}
      title={
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <Button
            type="text"
            icon={<DragOutlined />}
            aria-label={t("contact.layout.drag_section")}
            {...attributes}
            {...listeners}
            style={{ cursor: "grab" }}
          />
          <Input
            variant="borderless"
            value={page.name}
            onChange={(event) => onChange({ name: event.target.value })}
            aria-label={t("contact.layout.section_name")}
            style={{ fontWeight: 600, maxWidth: 320 }}
          />
          {!page.visible && <Tag>{t("contact.layout.hidden")}</Tag>}
        </div>
      }
      extra={
        <Space>
          <Switch
            checked={page.visible}
            checkedChildren={t("contact.layout.visible")}
            unCheckedChildren={t("contact.layout.hidden")}
            onChange={(visible) => onChange({ visible })}
          />
          <Popconfirm
            title={t("contact.layout.delete_section_confirm")}
            disabled={pageCount <= 1}
            onConfirm={onDelete}
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              disabled={pageCount <= 1}
            />
          </Popconfirm>
        </Space>
      }
    >
      {page.modules.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={t("contact.layout.empty_section")}
        />
      ) : (
        <Space direction="vertical" size="small" style={{ width: "100%" }}>
          {page.modules.map((module, moduleIndex) => (
            <div
              key={module.key}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 8,
                padding: "6px 8px",
                border: "1px solid rgba(0,0,0,0.08)",
                borderRadius: 8,
              }}
            >
              <Text style={{ flex: 1 }}>{module.name}</Text>
              <Button
                type="text"
                size="small"
                icon={<ArrowUpOutlined />}
                aria-label={t("settings.personalize.move_up")}
                disabled={moduleIndex === 0}
                onClick={() =>
                  onChange({
                    modules: arrayMove(
                      page.modules,
                      moduleIndex,
                      moduleIndex - 1,
                    ),
                  })
                }
              />
              <Button
                type="text"
                size="small"
                icon={<ArrowDownOutlined />}
                aria-label={t("settings.personalize.move_down")}
                disabled={moduleIndex === page.modules.length - 1}
                onClick={() =>
                  onChange({
                    modules: arrayMove(
                      page.modules,
                      moduleIndex,
                      moduleIndex + 1,
                    ),
                  })
                }
              />
              {allPages.length > 1 && (
                <Select
                  size="small"
                  aria-label={t("contact.layout.move_to_section")}
                  placeholder={t("contact.layout.move")}
                  style={{ width: 130 }}
                  options={allPages
                    .map((item, index) => ({ label: item.name, value: index }))
                    .filter((item) => item.value !== pageIndex)}
                  onChange={(targetPageIndex) =>
                    onMoveModule(moduleIndex, targetPageIndex)
                  }
                />
              )}
              <Button
                type="text"
                danger
                size="small"
                icon={<DeleteOutlined />}
                aria-label={t("common.delete")}
                onClick={() =>
                  onChange({
                    modules: page.modules.filter(
                      (_, index) => index !== moduleIndex,
                    ),
                  })
                }
              />
            </div>
          ))}
        </Space>
      )}

      {availableModules.length > 0 && (
        <Space.Compact style={{ width: "100%", marginTop: 12 }}>
          <Select
            value={moduleToAdd}
            onChange={setModuleToAdd}
            placeholder={t("contact.layout.add_module")}
            style={{ flex: 1 }}
            options={availableModules.map((definition) => ({
              label: definition.name,
              value: definition.key,
            }))}
          />
          <Button
            icon={<PlusOutlined />}
            disabled={!moduleToAdd}
            onClick={() => {
              const definition = availableModules.find(
                (item) => item.key === moduleToAdd,
              );
              if (!definition?.key) return;
              onChange({
                modules: [
                  ...page.modules,
                  {
                    key: definition.key,
                    name: definition.name ?? definition.key,
                  },
                ],
              });
              setModuleToAdd(null);
            }}
          >
            {t("common.add")}
          </Button>
        </Space.Compact>
      )}
    </Card>
  );
}
