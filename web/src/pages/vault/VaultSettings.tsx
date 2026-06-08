import { useEffect, useState } from "react";
import { formatContactName, useNameOrder } from "@/utils/nameFormat";
import type { ContactNameFields } from "@/utils/nameFormat";
import { useParams, Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import {
  Typography,
  Card,
  Tabs,
  Form,
  Input,
  Button,
  Switch,
  List,
  Space,
  App,
  Popconfirm,
  Tag,
  Select,
  ColorPicker,
  Collapse,
  Upload,
  Spin,
  Alert,
  theme,
  Radio,
} from "antd";
import {
  SaveOutlined,
  DeleteOutlined,
  EditOutlined,
  UserAddOutlined,
  ArrowLeftOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  InboxOutlined,
} from "@ant-design/icons";
import type { TabsProps } from "antd";
import { api } from "@/api";
import type {
  APIError,
  PersonalizeItem,
  UpdateVaultSettingsRequest,
  VaultUserResponse,
  LabelResponse,
  LifeEventCategoryResponse,
  LifeEventCategoryTypeResponse,
} from "@/api";
import type {
  GithubComNaibaBondsInternalDtoMonicaImportResponse,
  GithubComNaibaBondsInternalDtoCSVImportResponse,
} from "@/api/generated/data-contracts";
import VaultCompanies from "./VaultCompanies";
import { getReadableLabelTagColors } from "@/utils/labelColor";

const { Title, Text } = Typography;
const { Option } = Select;

const NAME_ORDER_PRESETS = [
  "%first_name% %last_name%",
  "%last_name% %first_name%",
  "%first_name% %last_name% {nickname? (%nickname%)}",
  "%nickname%",
] as const;

const CUSTOM_SENTINEL = "__custom__";

const SAMPLE_CONTACT: ContactNameFields = {
  first_name: "James",
  last_name: "Bond",
  middle_name: "Herbert",
  nickname: "007",
  maiden_name: "",
  prefix: "",
  suffix: "",
};


export default function VaultSettings() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const nameOrder = useNameOrder();
  const { token } = theme.useToken();

  const [activeTab, setActiveTab] = useState("general");
  const [nameOrderMode, setNameOrderMode] = useState<"global" | "override">("global");
  const [nameOrderTemplate, setNameOrderTemplate] = useState<string>(NAME_ORDER_PRESETS[0]);
  const [customNameOrderTemplate, setCustomNameOrderTemplate] = useState("");

  const { data: vaultSettings } = useQuery({
    queryKey: ["vault", vaultId, "settings"],
    queryFn: async () => {
      const res = await api.vaultSettings.settingsList(String(vaultId));
      return res.data;
    },
    enabled: !!vaultId,
  });

  useEffect(() => {
    if (!vaultSettings) return;

    const vaultNameOrder = vaultSettings.name_order;
    const hasOverride = vaultNameOrder !== null && vaultNameOrder !== undefined;
    const isPreset = hasOverride && (NAME_ORDER_PRESETS as readonly string[]).includes(vaultNameOrder);

    setNameOrderMode(hasOverride ? "override" : "global");
    setNameOrderTemplate(hasOverride ? (isPreset ? vaultNameOrder : CUSTOM_SENTINEL) : NAME_ORDER_PRESETS[0]);
    setCustomNameOrderTemplate(hasOverride && !isPreset ? vaultNameOrder : "");
  }, [vaultSettings]);

  const { data: personalizeTemplates } = useQuery<PersonalizeItem[]>({
    queryKey: ["settings", "personalize", "templates"],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail("templates");
      return res.data ?? [];
    },
  });

  const updateSettingsMutation = useMutation({
    mutationFn: (data: UpdateVaultSettingsRequest) =>
      api.vaultSettings.settingsUpdate(String(vaultId), data),
    onSuccess: () => {
      message.success(t("common.saved"));
      queryClient.invalidateQueries({ queryKey: ["vault", vaultId] });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateNameOrderMutation = useMutation({
    mutationFn: (nextNameOrder: string | undefined) =>
      api.vaultSettings.settingsNameOrderUpdate(String(vaultId), { name_order: nextNameOrder }),
    onSuccess: () => {
      message.success(t("common.saved"));
      queryClient.invalidateQueries({ queryKey: ["vault", vaultId, "settings"] });
      queryClient.invalidateQueries({ queryKey: ["vault", vaultId] });
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId] });
      queryClient.invalidateQueries({ queryKey: ["vaults"] });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateTemplateMutation = useMutation({
    mutationFn: (templateId: number) =>
      api.vaultSettings.settingsTemplateUpdate(String(vaultId), { default_template_id: templateId }),
    onSuccess: () => {
      message.success(t("vault_settings.template_updated"));
      queryClient.invalidateQueries({ queryKey: ["vault", vaultId] });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateTabVisibilityMutation = useMutation({
    mutationFn: (data: Record<string, boolean>) =>
      api.vaultSettings.settingsVisibilityUpdate(String(vaultId), data),
    onSuccess: () => {
      message.success(t("common.saved"));
      queryClient.invalidateQueries({ queryKey: ["vault", vaultId] });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const positionMutation = useMutation({
    mutationFn: async ({ entityType, id, position, categoryId }: { entityType: string; id: number; position: number; categoryId?: number }): Promise<void> => {
      const vid = String(vaultId);
      switch (entityType) {
        case "lifeEventCategories":
          await api.vaultSettings.settingsLifeEventCategoriesPositionCreate(vid, id, { position });
          break;
        case "lifeEventTypes":
          await api.vaultSettings.settingsLifeEventCategoriesLifeEventTypesPositionCreate(vid, categoryId!, id, { position });
          break;
        case "moodParams":
          await api.vaultSettings.settingsMoodParamsPositionCreate(vid, id, { position });
          break;
        case "quickFactTemplates":
          await api.vaultSettings.settingsQuickFactTemplatesPositionCreate(vid, id, { position });
          break;
      }
    },
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ["vault", vaultId] });
      if (vars.entityType === "lifeEventCategories" || vars.entityType === "lifeEventTypes") {
        queryClient.invalidateQueries({ queryKey: ["vault", vaultId, "lifeEventCategories"] });
      } else {
        queryClient.invalidateQueries({ queryKey: ["vault", vaultId, vars.entityType] });
      }
    },
    onError: (e: APIError) => message.error(e.message),
  });

  // --- Components for each tab ---


  const GeneralTab = () => {
    const [form] = Form.useForm();

    const presetLabels: Record<string, string> = {
      [NAME_ORDER_PRESETS[0]]: t("settings.preferences.name_order_first_last"),
      [NAME_ORDER_PRESETS[1]]: t("settings.preferences.name_order_last_first"),
      [NAME_ORDER_PRESETS[2]]: t("settings.preferences.name_order_first_last_nickname"),
      [NAME_ORDER_PRESETS[3]]: t("settings.preferences.name_order_nickname"),
    };

    const presetExamples: Record<string, string> = {
      [NAME_ORDER_PRESETS[0]]: "James Bond",
      [NAME_ORDER_PRESETS[1]]: "Bond James",
      [NAME_ORDER_PRESETS[2]]: "James Bond (007)",
      [NAME_ORDER_PRESETS[3]]: "007",
    };

    if (!vaultSettings) return null;

    const hasNameOrderOverride = vaultSettings.name_order !== null && vaultSettings.name_order !== undefined;
    const activeNameOrderTemplate = nameOrderTemplate === CUSTOM_SENTINEL ? customNameOrderTemplate : nameOrderTemplate;
    const canSaveNameOrder = nameOrderMode === "override" && activeNameOrderTemplate.trim().length > 0;


    return (
      <Space direction="vertical" style={{ width: "100%" }}>
        <Card title={t("vault_settings.general")}>
          <Form
            form={form}
            layout="vertical"
            initialValues={{
              name: vaultSettings.name,
              description: vaultSettings.description,
            }}
            onFinish={(values) => updateSettingsMutation.mutate(values)}
          >
            <Form.Item
              name="name"
              label={t("vault_settings.name")}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="description"
              label={t("vault_settings.description_label")}
            >
              <Input.TextArea rows={3} />
            </Form.Item>
            <Form.Item>
              <Button
                type="primary"
                htmlType="submit"
                icon={<SaveOutlined />}
                loading={updateSettingsMutation.isPending}
              >
                {t("common.save")}
              </Button>
            </Form.Item>
          </Form>
        </Card>

        <Card title={t("vault_settings.default_template")}>
          <Text type="secondary" style={{ display: "block", marginBottom: 12 }}>
            {t("vault_settings.template_description")}
          </Text>
          <Select
            style={{ width: "100%" }}
            value={vaultSettings.default_template_id}
            onChange={(value) => updateTemplateMutation.mutate(value)}
            loading={updateTemplateMutation.isPending}
          >
            {personalizeTemplates?.map((tpl) => (
              <Option key={tpl.id} value={tpl.id}>
                {tpl.label}
              </Option>
            ))}
          </Select>
        </Card>
        <Card title={t("vault_settings.name_order_override_title")}>
          <Text type="secondary" style={{ display: "block", marginBottom: 24 }}>
            {t("vault_settings.name_order_override_description")}
          </Text>

          <Radio.Group
            value={nameOrderMode}
            onChange={(e) => setNameOrderMode(e.target.value)}
            style={{ display: "flex", flexDirection: "column", gap: 16, marginBottom: 24 }}
          >
            <Radio value="global">
              <span style={{ fontWeight: 500 }}>{t("vault_settings.name_order_global")}</span>
              <div style={{ color: token.colorTextSecondary, fontSize: 13, marginTop: 4 }}>
                {t("vault_settings.name_order_global_help", { template: nameOrder })}
              </div>
            </Radio>
            <Radio value="override">
              <span style={{ fontWeight: 500 }}>{t("vault_settings.name_order_override")}</span>
            </Radio>
          </Radio.Group>

          {nameOrderMode === "global" && hasNameOrderOverride && (
            <Button
              loading={updateNameOrderMutation.isPending}
              onClick={() => updateNameOrderMutation.mutate(undefined)}
              style={{ marginBottom: 16 }}
            >
              {t("vault_settings.name_order_clear")}
            </Button>
          )}

          {nameOrderMode === "override" && (
            <div style={{ paddingLeft: 24, borderLeft: `2px solid ${token.colorBorder}`, marginLeft: 8 }}>
              <Radio.Group
                value={nameOrderTemplate}
                onChange={(e) => setNameOrderTemplate(e.target.value as string)}
                style={{ display: "flex", flexDirection: "column", gap: 8 }}
              >
                {NAME_ORDER_PRESETS.map((preset) => (
                  <Radio key={preset} value={preset}>
                    <span style={{ fontWeight: 500 }}>{presetLabels[preset]}</span>
                    <Text type="secondary" style={{ marginLeft: 8, fontSize: 13 }}>
                      — {presetExamples[preset]}
                    </Text>
                  </Radio>
                ))}
                <Radio value={CUSTOM_SENTINEL}>
                  <span style={{ fontWeight: 500 }}>
                    {t("settings.preferences.name_order_custom")}
                  </span>
                </Radio>
              </Radio.Group>

              {nameOrderTemplate === CUSTOM_SENTINEL && (
                <div style={{ marginTop: 12, paddingLeft: 24 }}>
                  <Input
                    value={customNameOrderTemplate}
                    onChange={(e) => setCustomNameOrderTemplate(e.target.value)}
                    placeholder="%first_name% %last_name%"
                    style={{ marginBottom: 8, maxWidth: 400 }}
                  />
                  <Text type="secondary" style={{ fontSize: 12, display: "block" }}>
                    {t("settings.preferences.name_order_custom_help")}
                  </Text>
                </div>
              )}

              <Text type="secondary" style={{ display: "block", fontSize: 12, marginTop: 12 }}>
                {t("vault_settings.name_order_condition_help")}
              </Text>

              <div style={{ marginTop: 16, padding: "8px 12px", background: token.colorFillQuaternary, borderRadius: token.borderRadius, display: "flex", alignItems: "center", gap: 8, width: "fit-content" }}>
                <Text type="secondary" style={{ fontSize: 13 }}>{t("settings.preferences.name_order_preview")}:</Text>
                <Text strong style={{ fontSize: 14 }}>{formatContactName(activeNameOrderTemplate, SAMPLE_CONTACT)}</Text>
              </div>

              <Space style={{ marginTop: 16 }}>
                <Button
                  type="primary"
                  icon={<SaveOutlined />}
                  loading={updateNameOrderMutation.isPending}
                  disabled={!canSaveNameOrder}
                  onClick={() => updateNameOrderMutation.mutate(activeNameOrderTemplate.trim())}
                >
                  {t("vault_settings.name_order_save")}
                </Button>
                {hasNameOrderOverride && (
                  <Button
                    loading={updateNameOrderMutation.isPending}
                    onClick={() => updateNameOrderMutation.mutate(undefined)}
                  >
                    {t("vault_settings.name_order_clear")}
                  </Button>
                )}
              </Space>
            </div>
          )}
        </Card>
      </Space>
    );
  };


  const TabsTab = () => {
    if (!vaultSettings) return null;

    const tabs = [
      { key: "show_group_tab", label: t("vault_settings.tab_group") },
      { key: "show_tasks_tab", label: t("vault_settings.tab_tasks") },
      { key: "show_files_tab", label: t("vault_settings.tab_files") },
      { key: "show_journal_tab", label: t("vault_settings.tab_journal") },
      { key: "show_companies_tab", label: t("vault_settings.tab_companies") },
      { key: "show_reports_tab", label: t("vault_settings.tab_reports") },
      { key: "show_calendar_tab", label: t("vault_settings.tab_calendar") },
    ];

    return (
      <Card title={t("vault_settings.tabs")}>
        <List
          dataSource={tabs}
          renderItem={(item) => (
            <List.Item
              actions={[
                <Switch
                  key="toggle"
                  checked={
                    vaultSettings[item.key as keyof typeof vaultSettings] as boolean
                  }
                  onChange={(checked) =>
                    updateTabVisibilityMutation.mutate({ [item.key]: checked })
                  }
                  loading={updateTabVisibilityMutation.isPending}
                />,
              ]}
            >
              <List.Item.Meta
                title={t("vault_settings.show_tab", { tab: item.label })}
              />
            </List.Item>
          )}
        />
      </Card>
    );
  };

  const UsersTab = () => {
    const { data: users = [], isLoading } = useQuery({
      queryKey: ["vault", vaultId, "users"],
      queryFn: async () => {
        const res = await api.vaultSettings.settingsUsersList(String(vaultId));
        return res.data ?? [];
      },
    });

    const inviteMutation = useMutation({
      mutationFn: (values: { email: string; permission: 100 | 200 | 300 }) =>
        api.vaultSettings.settingsUsersCreate(String(vaultId), values),
      onSuccess: () => {
        message.success(t("invitations.status.pending")); // Or specific success message
        queryClient.invalidateQueries({ queryKey: ["vault", vaultId, "users"] });
      },
      onError: (e: APIError) => message.error(e.message),
    });

    const removeUserMutation = useMutation({
      mutationFn: (userId: number) => api.vaultSettings.settingsUsersDelete(String(vaultId), userId),
      onSuccess: () => {
        message.success(t("common.deleted"));
        queryClient.invalidateQueries({ queryKey: ["vault", vaultId, "users"] });
      },
      onError: (e: APIError) => message.error(e.message),
    });

    const updateUserPermMutation = useMutation({
      mutationFn: ({ userId, permission }: { userId: number; permission: 100 | 200 | 300 }) =>
        api.vaultSettings.settingsUsersUpdate(String(vaultId), userId, { permission }),
      onSuccess: () => {
        message.success(t("common.updated"));
        queryClient.invalidateQueries({ queryKey: ["vault", vaultId, "users"] });
      },
      onError: (e: APIError) => message.error(e.message),
    });

    const [inviteForm] = Form.useForm();

    return (
      <Space direction="vertical" style={{ width: "100%" }}>
        <Card title={t("vault_settings.add_user")}>
          <Form
            form={inviteForm}
            layout="inline"
            onFinish={(values) => {
              inviteMutation.mutate(values);
              inviteForm.resetFields();
            }}
          >
            <Form.Item
              name="email"
              rules={[{ required: true, type: "email", message: t("common.required") }]}
            >
              <Input placeholder={t("vault_settings.user_email")} />
            </Form.Item>
            <Form.Item
              name="permission"
              initialValue={200}
              rules={[{ required: true }]}
            >
              <Select style={{ width: 120 }}>
                <Option value={100}>{t("invitations.permission.manager")}</Option>
                <Option value={200}>{t("invitations.permission.editor")}</Option>
                <Option value={300}>{t("invitations.permission.viewer")}</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Button
                type="primary"
                htmlType="submit"
                icon={<UserAddOutlined />}
                loading={inviteMutation.isPending}
              >
                {t("common.add")}
              </Button>
            </Form.Item>
          </Form>
        </Card>

        <Card title={t("vault_settings.users")}>
          <List
            loading={isLoading}
            dataSource={users}
            renderItem={(user: VaultUserResponse) => (
              <List.Item
                actions={[
                   <Select<100 | 200 | 300>
                    key="perm"
                    defaultValue={(user.permission ?? 300) as 100 | 200 | 300}
                    style={{ width: 120 }}
                    onChange={(val) =>
                      updateUserPermMutation.mutate({ userId: user.id!, permission: val })
                    }
                    disabled={updateUserPermMutation.isPending}
                  >
                    <Option value={100}>{t("invitations.permission.manager")}</Option>
                    <Option value={200}>{t("invitations.permission.editor")}</Option>
                    <Option value={300}>{t("invitations.permission.viewer")}</Option>
                  </Select>,
                  <Popconfirm
                    key="del"
                    title={t("common.delete_confirm")}
                    onConfirm={() => removeUserMutation.mutate(user.id!)}
                  >
                    <Button danger icon={<DeleteOutlined />} loading={removeUserMutation.isPending} />
                  </Popconfirm>,
                ]}
              >
                <List.Item.Meta
                  title={formatContactName(nameOrder, user)}
                  description={user.email}
                />
              </List.Item>
            )}
          />
        </Card>
      </Space>
    );
  };

  const LabelsTab = () => {
    const queryKey = ["vault", vaultId, "labels"];
    const { data: items = [], isLoading } = useQuery({
      queryKey,
      queryFn: async () => (await api.vaultSettings.settingsLabelsList(String(vaultId))).data ?? [],
    });

    const createMutation = useMutation({
      mutationFn: (data: { name: string; description?: string; bg_color: string; text_color: string }) => api.vaultSettings.settingsLabelsCreate(String(vaultId), data),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey });
        message.success(t("common.created"));
        form.resetFields();
      },
    });

    const updateMutation = useMutation({
      mutationFn: ({ id, data }: { id: number; data: { name: string; description?: string; bg_color: string; text_color: string } }) =>
        api.vaultSettings.settingsLabelsUpdate(String(vaultId), id, data),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey });
        message.success(t("common.updated"));
        setEditingId(null);
        form.resetFields();
      },
    });

    const deleteMutation = useMutation({
      mutationFn: (id: number) => api.vaultSettings.settingsLabelsDelete(String(vaultId), id),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey });
        message.success(t("common.deleted"));
      },
    });

    const [form] = Form.useForm();
    const [editingId, setEditingId] = useState<number | null>(null);

    const onFinish = (values: { name: string; description?: string; bg_color: string | { toHexString: () => string }; text_color: string | { toHexString: () => string } }) => {
      const data = {
        name: values.name,
        description: values.description,
        bg_color: typeof values.bg_color === 'string' ? values.bg_color : values.bg_color.toHexString(),
        text_color: typeof values.text_color === 'string' ? values.text_color : values.text_color.toHexString(),
      };
      
      if (editingId) {
        updateMutation.mutate({ id: editingId, data });
      } else {
        createMutation.mutate(data);
      }
    };

    const startEdit = (item: LabelResponse) => {
      setEditingId(item.id ?? null);
      form.setFieldsValue({
        name: item.name,
        description: item.description,
        bg_color: item.bg_color,
        text_color: item.text_color,
      });
    };

    const cancelEdit = () => {
      setEditingId(null);
      form.resetFields();
    };

    return (
      <Space direction="vertical" style={{ width: "100%" }}>
        <Card title={editingId ? t("common.edit") : t("common.add")}>
          <Form form={form} layout="inline" onFinish={onFinish}>
            <Form.Item name="name" rules={[{ required: true }]}>
              <Input placeholder={t("common.name")} />
            </Form.Item>
            <Form.Item name="bg_color" initialValue="#1677ff">
               <ColorPicker showText />
            </Form.Item>
             <Form.Item name="text_color" initialValue="#ffffff">
               <ColorPicker showText />
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" loading={createMutation.isPending || updateMutation.isPending}>
                {editingId ? t("common.update") : t("common.add")}
              </Button>
              {editingId && (
                <Button onClick={cancelEdit} style={{ marginLeft: 8 }}>
                  {t("common.cancel")}
                </Button>
              )}
            </Form.Item>
          </Form>
        </Card>

        <Card title={t("vault_settings.labels")}>
          <List<LabelResponse>
            loading={isLoading}
            dataSource={items as LabelResponse[]}
            renderItem={(item) => {
              const labelTagColors = getReadableLabelTagColors(item.bg_color, item.text_color);
              return (
                <List.Item
                  actions={[
                    <Button icon={<EditOutlined />} onClick={() => startEdit(item)} />,
                    <Popconfirm title={t("common.delete_confirm")} onConfirm={() => deleteMutation.mutate(item.id!)}>
                      <Button danger icon={<DeleteOutlined />} />
                    </Popconfirm>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={
                      <Tag color={labelTagColors.color} style={labelTagColors.style}>
                        {item.name}
                      </Tag>
                    }
                    description={item.description}
                  />
                </List.Item>
              );
            }}
          />
        </Card>
      </Space>
    );
  };

  // Generalized CRUD Component for simple lists (Tags, DateTypes, MoodParams, QuickFactTemplates)
  interface ExtraField {
    name: string;
    label?: string;
    type?: "color" | "text";
    initialValue?: string;
    rules?: { required?: boolean }[];
  }
  const SimpleCrudTab = <T extends { id: number; label?: string; name?: string; hex_color?: string; position?: number }>({
    queryKeySuffix,
    apiList,
    apiCreate,
    apiUpdate,
    apiDelete,
    title,
    itemNameKey = "label",
    extraFields = [],
    positionEntityType,
  }: {
    queryKeySuffix: string;
    apiList: (vid: string) => Promise<{ data?: T[] }>;
    apiCreate: (vid: string, data: Record<string, unknown>) => Promise<unknown>;
    apiUpdate: (vid: string, id: number, data: Record<string, unknown>) => Promise<unknown>;
    apiDelete: (vid: string, id: number) => Promise<unknown>;
    title: string;
    itemNameKey?: "label" | "name";
    extraFields?: ExtraField[];
    positionEntityType?: string;
  }) => {
    const queryKey = ["vault", vaultId, queryKeySuffix];
    const { data: items = [], isLoading } = useQuery({
      queryKey,
      queryFn: async () => (await apiList(vaultId)).data ?? [],
    });

    const createMutation = useMutation({
      mutationFn: (data: Record<string, unknown>) => apiCreate(vaultId, data),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey });
        message.success(t("common.created"));
        form.resetFields();
      },
    });

    const updateMutation = useMutation({
      mutationFn: ({ id, data }: { id: number; data: Record<string, unknown> }) =>
        apiUpdate(vaultId, id, data),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey });
        message.success(t("common.updated"));
        setEditingId(null);
        form.resetFields();
      },
    });

    const deleteMutation = useMutation({
      mutationFn: (id: number) => apiDelete(vaultId, id),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey });
        message.success(t("common.deleted"));
      },
    });

    const [form] = Form.useForm();
    const [editingId, setEditingId] = useState<number | null>(null);

    const onFinish = (values: Record<string, unknown>) => {
       // Handle ColorPicker value
       const processed = { ...values };
       if (processed.hex_color && typeof processed.hex_color !== 'string') {
          processed.hex_color = (processed.hex_color as { toHexString: () => string }).toHexString();
       }

      if (editingId) {
        updateMutation.mutate({ id: editingId, data: processed });
      } else {
        createMutation.mutate(processed);
      }
    };

    const startEdit = (item: T) => {
      setEditingId(item.id);
      form.setFieldsValue(item);
    };

    const cancelEdit = () => {
      setEditingId(null);
      form.resetFields();
    };

    return (
      <Space direction="vertical" style={{ width: "100%" }}>
        <Card title={editingId ? t("common.edit") : t("common.add")}>
          <Form form={form} layout="inline" onFinish={onFinish}>
            <Form.Item name={itemNameKey} rules={[{ required: true }]}>
              <Input placeholder={t("common.name")} />
            </Form.Item>
            {extraFields.map((field) => (
              <Form.Item
                key={field.name}
                name={field.name}
                initialValue={field.initialValue}
                rules={field.rules}
                label={field.label}
              >
                {field.type === 'color' ? <ColorPicker showText /> : <Input />}
              </Form.Item>
            ))}
            <Form.Item>
              <Button type="primary" htmlType="submit" loading={createMutation.isPending || updateMutation.isPending}>
                {editingId ? t("common.update") : t("common.add")}
              </Button>
              {editingId && (
                <Button onClick={cancelEdit} style={{ marginLeft: 8 }}>
                  {t("common.cancel")}
                </Button>
              )}
            </Form.Item>
          </Form>
        </Card>

        <Card title={title}>
          <List
            loading={isLoading}
            dataSource={items}
            renderItem={(item: T, index: number) => (
              <List.Item
                actions={[
                  ...(positionEntityType ? [
                    <Button
                      key="up"
                      size="small"
                      icon={<ArrowUpOutlined />}
                      title={t("vault_settings.move_up")}
                      disabled={index === 0}
                      onClick={() => positionMutation.mutate({ entityType: positionEntityType, id: item.id, position: index - 1 })}
                    />,
                    <Button
                      key="down"
                      size="small"
                      icon={<ArrowDownOutlined />}
                      title={t("vault_settings.move_down")}
                      disabled={index === items.length - 1}
                      onClick={() => positionMutation.mutate({ entityType: positionEntityType, id: item.id, position: index + 1 })}
                    />,
                  ] : []),
                  <Button key="edit" icon={<EditOutlined />} onClick={() => startEdit(item)} />,
                  <Popconfirm key="del" title={t("common.delete_confirm")} onConfirm={() => deleteMutation.mutate(item.id)}>
                    <Button danger icon={<DeleteOutlined />} />
                  </Popconfirm>,
                ]}
              >
                <List.Item.Meta
                  avatar={item.hex_color && <div style={{width: 20, height: 20, backgroundColor: item.hex_color, borderRadius: 4}} />}
                  title={item[itemNameKey]}
                />
              </List.Item>
            )}
          />
        </Card>
      </Space>
    );
  };
  
  // Life Event Categories - Nested CRUD
  const LifeEventsTab = () => {
    const queryKey = ["vault", vaultId, "lifeEventCategories"];
    const { data: categories = [] } = useQuery({
      queryKey,
      queryFn: async () => (await api.vaultSettings.settingsLifeEventCategoriesList(String(vaultId))).data ?? [],
    });

    const createCategory = useMutation({
        mutationFn: (data: { label: string }) => api.vaultSettings.settingsLifeEventCategoriesCreate(String(vaultId), data),
        onSuccess: () => { queryClient.invalidateQueries({ queryKey }); message.success(t("common.created")); },
    });
    const updateCategory = useMutation({
        mutationFn: ({ id, data }: { id: number; data: { label: string } }) =>
            api.vaultSettings.settingsLifeEventCategoriesUpdate(String(vaultId), id, data),
        onSuccess: () => { queryClient.invalidateQueries({ queryKey }); message.success(t("vault_settings.life_event_category_updated")); setEditingCatId(null); setEditingCatLabel(""); },
        onError: (e: APIError) => message.error(e.message),
    });
    const deleteCategory = useMutation({
        mutationFn: (id: number) => api.vaultSettings.settingsLifeEventCategoriesDelete(String(vaultId), id),
        onSuccess: () => { queryClient.invalidateQueries({ queryKey }); message.success(t("common.deleted")); },
    });

    const createType = useMutation({
        mutationFn: ({ catId, data }: { catId: number, data: { label: string } }) => 
            api.vaultSettings.settingsLifeEventCategoriesTypesCreate(String(vaultId), catId, data),
        onSuccess: () => { queryClient.invalidateQueries({ queryKey }); message.success(t("common.created")); },
    });

    const updateType = useMutation({
        mutationFn: ({ catId, typeId, data }: { catId: number; typeId: number; data: { label: string } }) =>
            api.vaultSettings.settingsLifeEventCategoriesTypesUpdate(String(vaultId), catId, typeId, data),
        onSuccess: () => { queryClient.invalidateQueries({ queryKey }); message.success(t("vault_settings.life_event_type_updated")); setEditingTypeId(null); setEditingTypeLabel(""); },
        onError: (e: APIError) => message.error(e.message),
    });
    
    const deleteType = useMutation({
        mutationFn: ({ catId, typeId }: { catId: number, typeId: number }) => 
            api.vaultSettings.settingsLifeEventCategoriesTypesDelete(String(vaultId), catId, typeId),
        onSuccess: () => { queryClient.invalidateQueries({ queryKey }); message.success(t("common.deleted")); },
    });

    const [newCatLabel, setNewCatLabel] = useState("");
    const [newTypeLabel, setNewTypeLabel] = useState<Record<number, string>>({});
    const [editingCatId, setEditingCatId] = useState<number | null>(null);
    const [editingCatLabel, setEditingCatLabel] = useState("");
    const [editingTypeId, setEditingTypeId] = useState<number | null>(null);
    const [editingTypeLabel, setEditingTypeLabel] = useState("");

    const handleAddType = (catId: number) => {
        if (!newTypeLabel[catId]) return;
        createType.mutate({ catId, data: { label: newTypeLabel[catId] } });
        setNewTypeLabel(prev => ({ ...prev, [catId]: "" }));
    }

    return (
        <Space direction="vertical" style={{ width: "100%" }}>
             <Card title={t("vault_settings.add_category")}>
                <Space>
                    <Input 
                        placeholder={t("common.name")} 
                        value={newCatLabel} 
                        onChange={e => setNewCatLabel(e.target.value)} 
                        onPressEnter={() => { if(newCatLabel) { createCategory.mutate({ label: newCatLabel }); setNewCatLabel(""); } }}
                    />
                    <Button type="primary" onClick={() => { if(newCatLabel) { createCategory.mutate({ label: newCatLabel }); setNewCatLabel(""); } }}>
                        {t("common.add")}
                    </Button>
                </Space>
             </Card>
             
             <Card title={t("vault_settings.life_events")}>
                <Collapse accordion>
                    {categories.map((cat: LifeEventCategoryResponse, catIndex: number) => (
                         <Collapse.Panel 
                            key={cat.id!} 
                            header={
                                editingCatId === cat.id ? (
                                    <Space onClick={(e) => e.stopPropagation()}>
                                        <Input
                                            size="small"
                                            value={editingCatLabel}
                                            onChange={(e) => setEditingCatLabel(e.target.value)}
                                            onPressEnter={() => { if (editingCatLabel.trim()) updateCategory.mutate({ id: cat.id!, data: { label: editingCatLabel.trim() } }); }}
                                            onClick={(e) => e.stopPropagation()}
                                        />
                                        <Button size="small" type="primary" loading={updateCategory.isPending} onClick={(e) => { e.stopPropagation(); if (editingCatLabel.trim()) updateCategory.mutate({ id: cat.id!, data: { label: editingCatLabel.trim() } }); }}>
                                            {t("common.save")}
                                        </Button>
                                        <Button size="small" onClick={(e) => { e.stopPropagation(); setEditingCatId(null); setEditingCatLabel(""); }}>
                                            {t("common.cancel")}
                                        </Button>
                                    </Space>
                                ) : cat.label
                            }
                            extra={
                                <Space onClick={(e) => e.stopPropagation()}>
                                    <Button
                                        size="small"
                                        icon={<ArrowUpOutlined />}
                                        title={t("vault_settings.move_up")}
                                        disabled={catIndex === 0}
                                        onClick={(e) => { e.stopPropagation(); positionMutation.mutate({ entityType: "lifeEventCategories", id: cat.id!, position: catIndex - 1 }); }}
                                    />
                                    <Button
                                        size="small"
                                        icon={<ArrowDownOutlined />}
                                        title={t("vault_settings.move_down")}
                                        disabled={catIndex === categories.length - 1}
                                        onClick={(e) => { e.stopPropagation(); positionMutation.mutate({ entityType: "lifeEventCategories", id: cat.id!, position: catIndex + 1 }); }}
                                    />
                                    <Button
                                        size="small"
                                        icon={<EditOutlined />}
                                        onClick={(e) => { e.stopPropagation(); setEditingCatId(cat.id!); setEditingCatLabel(cat.label ?? ""); }}
                                    />
                                    <Popconfirm title={t("common.delete_confirm")} onConfirm={(e) => { e?.stopPropagation(); deleteCategory.mutate(cat.id!); }}>
                                        <DeleteOutlined onClick={(e) => e.stopPropagation()} style={{ color: token.colorError }} />
                                    </Popconfirm>
                                </Space>
                            }
                        >
                             <List
                                dataSource={cat.types}
                                header={
                                    <Space style={{width: '100%'}}>
                                        <Input 
                                            placeholder={t("vault_settings.add_type")} 
                                            value={newTypeLabel[cat.id!] || ""}
                                            onChange={e => setNewTypeLabel(prev => ({ ...prev, [cat.id!]: e.target.value }))}
                                            onPressEnter={() => handleAddType(cat.id!)}
                                        />
                                        <Button type="dashed" onClick={() => handleAddType(cat.id!)}>{t("common.add")}</Button>
                                    </Space>
                                }
                                renderItem={(type: LifeEventCategoryTypeResponse, typeIndex: number) => (
                                    <List.Item
                                        actions={[
                                            <Button
                                                key="up"
                                                size="small"
                                                icon={<ArrowUpOutlined />}
                                                title={t("vault_settings.move_up")}
                                                type="text"
                                                disabled={typeIndex === 0}
                                                onClick={() => positionMutation.mutate({ entityType: "lifeEventTypes", id: type.id!, position: typeIndex - 1, categoryId: cat.id! })}
                                            />,
                                            <Button
                                                key="down"
                                                size="small"
                                                icon={<ArrowDownOutlined />}
                                                title={t("vault_settings.move_down")}
                                                type="text"
                                                disabled={typeIndex === (cat.types?.length ?? 1) - 1}
                                                onClick={() => positionMutation.mutate({ entityType: "lifeEventTypes", id: type.id!, position: typeIndex + 1, categoryId: cat.id! })}
                                            />,
                                            ...(editingTypeId === type.id ? [
                                                <Button
                                                    key="save"
                                                    size="small"
                                                    type="primary"
                                                    loading={updateType.isPending}
                                                    onClick={() => { if (editingTypeLabel.trim()) updateType.mutate({ catId: cat.id!, typeId: type.id!, data: { label: editingTypeLabel.trim() } }); }}
                                                >
                                                    {t("common.save")}
                                                </Button>,
                                                <Button
                                                    key="cancel-edit"
                                                    size="small"
                                                    type="text"
                                                    onClick={() => { setEditingTypeId(null); setEditingTypeLabel(""); }}
                                                >
                                                    {t("common.cancel")}
                                                </Button>,
                                            ] : [
                                                <Button
                                                    key="edit"
                                                    size="small"
                                                    icon={<EditOutlined />}
                                                    type="text"
                                                    onClick={() => { setEditingTypeId(type.id!); setEditingTypeLabel(type.label ?? ""); }}
                                                />,
                                            ]),
                                            <Popconfirm key="del" title={t("common.delete_confirm")} onConfirm={() => deleteType.mutate({ catId: cat.id!, typeId: type.id! })}>
                                                <Button danger size="small" icon={<DeleteOutlined />} type="text" />
                                            </Popconfirm>
                                        ]}
                                    >
                                        {editingTypeId === type.id ? (
                                            <Input
                                                size="small"
                                                value={editingTypeLabel}
                                                onChange={(e) => setEditingTypeLabel(e.target.value)}
                                                onPressEnter={() => { if (editingTypeLabel.trim()) updateType.mutate({ catId: cat.id!, typeId: type.id!, data: { label: editingTypeLabel.trim() } }); }}
                                            />
                                        ) : type.label}
                                    </List.Item>
                                )}
                             />
                        </Collapse.Panel>
                    ))}
                </Collapse>
             </Card>
        </Space>
    )
  };


  // ── CSV Import ──────────────────────────────────────────────────────────
  const CSV_FIELDS: { key: string; label: string }[] = [
    { key: "first_name",          label: t("vault_settings.csv_import.field_first_name") },
    { key: "last_name",           label: t("vault_settings.csv_import.field_last_name") },
    { key: "middle_name",         label: t("vault_settings.csv_import.field_middle_name") },
    { key: "nickname",            label: t("vault_settings.csv_import.field_nickname") },
    { key: "prefix",              label: t("vault_settings.csv_import.field_prefix") },
    { key: "suffix",              label: t("vault_settings.csv_import.field_suffix") },
    { key: "gender",              label: t("vault_settings.csv_import.field_gender") },
    { key: "birthday",            label: t("vault_settings.csv_import.field_birthday") },
    { key: "email",               label: t("vault_settings.csv_import.field_email") },
    { key: "phone",               label: t("vault_settings.csv_import.field_phone") },
    { key: "company",             label: t("vault_settings.csv_import.field_company") },
    { key: "job_title",           label: t("vault_settings.csv_import.field_job_title") },
    { key: "tags",                label: t("vault_settings.csv_import.field_tags") },
    { key: "groups",              label: t("vault_settings.csv_import.field_groups") },
    { key: "notes",               label: t("vault_settings.csv_import.field_notes") },
    { key: "address_street",      label: t("vault_settings.csv_import.field_address_street") },
    { key: "address_city",        label: t("vault_settings.csv_import.field_address_city") },
    { key: "address_state",       label: t("vault_settings.csv_import.field_address_state") },
    { key: "address_postal_code", label: t("vault_settings.csv_import.field_address_postal_code") },
    { key: "address_country",     label: t("vault_settings.csv_import.field_address_country") },
  ];

  // Parse CSV header row in the browser (handles basic quoting).
  function parseCSVHeaders(text: string): string[] {
    const firstLine = text.split(/\r?\n/)[0] ?? "";
    const headers: string[] = [];
    let cur = "";
    let inQuote = false;
    for (let i = 0; i < firstLine.length; i++) {
      const ch = firstLine[i];
      if (ch === '"') { inQuote = !inQuote; }
      else if (ch === "," && !inQuote) { headers.push(cur.trim()); cur = ""; }
      else { cur += ch; }
    }
    headers.push(cur.trim());
    return headers;
  }

  // Auto-map columns: lowercase-normalise both sides and pick the first match.
  function autoMap(headers: string[]): Record<string, string> {
    const norm = (s: string) => s.toLowerCase().replace(/[^a-z0-9]/g, "");
    const aliases: Record<string, string[]> = {
      first_name:          ["firstname", "first", "givenname", "prenom"],
      last_name:           ["lastname", "last", "surname", "familyname", "nom"],
      middle_name:         ["middlename", "middle"],
      nickname:            ["nickname", "alias", "pseudo"],
      prefix:              ["prefix", "title", "salutation"],
      suffix:              ["suffix"],
      gender:              ["gender", "sexe", "genre"],
      birthday:            ["birthday", "birthdate", "dob", "dateofbirth", "naissance"],
      email:               ["email", "emailaddress", "mail", "courriel"],
      phone:               ["phone", "phonenumber", "mobile", "telephone", "tel"],
      company:             ["company", "organization", "organisation", "employer", "societe"],
      job_title:           ["jobtitle", "job", "position", "title", "role", "fonction"],
      tags:                ["tags", "labels", "categories"],
      groups:              ["groups", "groupes"],
      notes:               ["notes", "note", "comment", "comments", "remarks"],
      address_street:      ["street", "address", "addressstreet", "line1", "rue"],
      address_city:        ["city", "ville"],
      address_state:       ["state", "province", "region"],
      address_postal_code: ["postalcode", "zip", "zipcode", "postcode", "codepostal"],
      address_country:     ["country", "pays"],
    };
    const mapping: Record<string, string> = {};
    for (const [field, aliasList] of Object.entries(aliases)) {
      const match = headers.find(h => aliasList.includes(norm(h)));
      mapping[field] = match ?? "";
    }
    return mapping;
  }

  const CSVImportTab = () => {
    const [step, setStep] = useState<"upload" | "map" | "done">("upload");
    const [csvFile, setCsvFile] = useState<File | null>(null);
    const [csvHeaders, setCsvHeaders] = useState<string[]>([]);
    const [mapping, setMapping] = useState<Record<string, string>>({});
    const [importing, setImporting] = useState(false);
    const [importResult, setImportResult] = useState<GithubComNaibaBondsInternalDtoCSVImportResponse | null>(null);
    const [importError, setImportError] = useState<string | null>(null);

    const handleBeforeUpload = (file: File): boolean => {
      const reader = new FileReader();
      reader.onload = (e) => {
        const text = (e.target?.result as string) ?? "";
        const headers = parseCSVHeaders(text);
        setCsvHeaders(headers);
        setMapping(autoMap(headers));
        setCsvFile(file);
        setStep("map");
      };
      reader.readAsText(file);
      return false;
    };

    const handleImport = async () => {
      if (!csvFile) return;
      setImporting(true);
      setImportError(null);
      try {
        const res = await api.vaultSettings.settingsImportCsvCreate(String(vaultId), {
          file: csvFile,
          mapping: JSON.stringify(mapping),
        });
        setImportResult(res.data ?? null);
        setStep("done");
      } catch (err: unknown) {
        const msg = (err instanceof Error) ? err.message : t("vault_settings.csv_import.error");
        setImportError(msg);
      } finally {
        setImporting(false);
      }
    };

    const reset = () => { setCsvFile(null); setCsvHeaders([]); setMapping({}); setImportResult(null); setImportError(null); setStep("upload"); };

    return (
      <Space direction="vertical" style={{ width: "100%" }} size="large">
        <Title level={4} style={{ margin: 0 }}>{t("vault_settings.csv_import.title")}</Title>
        <Text type="secondary">{t("vault_settings.csv_import.description")}</Text>

        {step === "upload" && (
          <Upload.Dragger accept=".csv" showUploadList={false} beforeUpload={handleBeforeUpload} multiple={false}>
            <p className="ant-upload-drag-icon"><InboxOutlined /></p>
            <p className="ant-upload-text">{t("vault_settings.csv_import.upload_hint")}</p>
            <p className="ant-upload-hint">{t("vault_settings.csv_import.upload_next_step")}</p>
            <p className="ant-upload-hint">{t("vault_settings.csv_import.csv_only")}</p>
          </Upload.Dragger>
        )}

        {step === "map" && (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <Text strong>{csvFile?.name}</Text>
            <Text type="secondary">{t("vault_settings.csv_import.company_note")}</Text>
            <Text type="secondary">{t("vault_settings.csv_import.groups_note")}</Text>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
              {CSV_FIELDS.map(({ key, label }) => (
                <div key={key} style={{ display: "contents" }}>
                  <Text style={{ alignSelf: "center" }}>{label}</Text>
                  <Select
                    style={{ width: "100%" }}
                    value={mapping[key] ?? ""}
                    onChange={(v) => setMapping((prev) => ({ ...prev, [key]: v }))}
                  >
                    <Option value="">{t("vault_settings.csv_import.not_mapped")}</Option>
                    {csvHeaders.map((h) => <Option key={h} value={h}>{h}</Option>)}
                  </Select>
                </div>
              ))}
            </div>
            {importError && <Alert type="error" message={importError} showIcon />}
            <Space>
              <Button onClick={reset}>← {t("vault_settings.csv_import.step_upload")}</Button>
              <Button type="primary" onClick={handleImport} loading={importing}>
                {t("vault_settings.csv_import.import_button")}
              </Button>
            </Space>
          </Space>
        )}

        {step === "done" && importResult && (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <Alert
              type="success"
              message={t("vault_settings.csv_import.success")}
              description={
                <Space direction="vertical" size="small">
                  <Text>{t("vault_settings.csv_import.contacts_imported")}: {importResult.imported_contacts}</Text>
                  {(importResult.skipped_count ?? 0) > 0 && (
                    <Text type="warning">{t("vault_settings.csv_import.skipped")}: {importResult.skipped_count}</Text>
                  )}
                  {importResult.errors && importResult.errors.length > 0 && (
                    <Text type="danger">{t("vault_settings.csv_import.errors")}: {importResult.errors.slice(0, 5).join("; ")}</Text>
                  )}
                </Space>
              }
              showIcon
            />
            <Button onClick={reset}>← {t("vault_settings.csv_import.step_upload")}</Button>
          </Space>
        )}
      </Space>
    );
  };

  const MonicaImportTab = () => {
    const [importing, setImporting] = useState(false);
    const [importResult, setImportResult] = useState<GithubComNaibaBondsInternalDtoMonicaImportResponse | null>(null);
    const [importError, setImportError] = useState<string | null>(null);

    const handleBeforeUpload = async (file: File): Promise<boolean> => {
      setImporting(true);
      setImportResult(null);
      setImportError(null);
      try {
        const res = await api.vaultSettings.settingsImportMonicaCreate(String(vaultId), { file });
        setImportResult(res.data ?? null);
      } catch (err: unknown) {
        const msg = (err instanceof Error) ? err.message : t("vault_settings.monica_import.error");
        setImportError(msg);
      } finally {
        setImporting(false);
      }
      return false;
    };

    return (
      <Space direction="vertical" style={{ width: "100%" }} size="large">
        <Title level={4} style={{ margin: 0 }}>
          {t("vault_settings.monica_import.title")}
        </Title>
        <Text type="secondary">
          {t("vault_settings.monica_import.description")}
        </Text>

        <Upload.Dragger
          accept=".json"
          showUploadList={false}
          beforeUpload={handleBeforeUpload}
          disabled={importing}
          multiple={false}
        >
          <p className="ant-upload-drag-icon">
            <InboxOutlined />
          </p>
          <p className="ant-upload-text">
            {t("vault_settings.monica_import.upload_hint")}
          </p>
          <p className="ant-upload-hint">JSON only</p>
        </Upload.Dragger>

        {importing && (
          <div style={{ textAlign: "center" }}>
            <Spin size="large" />
            <Text style={{ marginLeft: 8 }}>{t("vault_settings.monica_import.importing")}</Text>
          </div>
        )}

        {importError && (
          <Alert type="error" message={importError} showIcon />
        )}

        {importResult && (
          <Alert
            type="success"
            message={t("vault_settings.monica_import.success")}
            description={
              <Space direction="vertical" size="small">
                {([
                  ["contacts", importResult.imported_contacts],
                  ["notes", importResult.imported_notes],
                  ["calls", importResult.imported_calls],
                  ["tasks", importResult.imported_tasks],
                  ["reminders", importResult.imported_reminders],
                  ["relationships", importResult.imported_relationships],
                  ["addresses", importResult.imported_addresses],
                  ["life_events", importResult.imported_life_events],
                  ["documents", importResult.imported_documents],
                  ["photos", importResult.imported_photos],
                ] as [string, number | undefined][]).map(([key, val]) => (
                  <Text key={key}>
                    {t(`vault_settings.monica_import.${key}`)}: {val ?? 0}
                  </Text>
                ))}
                {(importResult.skipped_count ?? 0) > 0 && (
                  <Text type="warning">
                    {t("vault_settings.monica_import.skipped")}: {importResult.skipped_count}
                  </Text>
                )}
                {Array.isArray(importResult.errors) && importResult.errors.length > 0 && (
                  <Text type="danger">
                    {t("vault_settings.monica_import.errors")}: {importResult.errors.slice(0, 3).join("; ")}
                  </Text>
                )}
              </Space>
            }
            showIcon
          />
        )}
      </Space>
    );
  };

  const tabItems: TabsProps["items"] = [
    { key: "general", label: t("vault_settings.general"), children: <GeneralTab /> },
    { key: "tabs", label: t("vault_settings.tabs"), children: <TabsTab /> },
    { key: "users", label: t("vault_settings.users"), children: <UsersTab /> },
    { key: "labels", label: t("vault_settings.labels"), children: <LabelsTab /> },
    { key: "companies", label: t("vault.companies.title"), children: <VaultCompanies vaultId={vaultId} /> },
    { key: "tags", label: t("vault_settings.tags"), children: <SimpleCrudTab 
        queryKeySuffix="tags" 
        apiList={(vid) => api.vaultSettings.settingsTagsList(String(vid))}
        apiCreate={(vid, data) => api.vaultSettings.settingsTagsCreate(String(vid), data as unknown as import("@/api/generated/data-contracts").GithubComNaibaBondsInternalDtoCreateTagRequest)}
        apiUpdate={(vid, id, data) => api.vaultSettings.settingsTagsUpdate(String(vid), id, data as unknown as import("@/api/generated/data-contracts").GithubComNaibaBondsInternalDtoUpdateTagRequest)}
        apiDelete={(vid, id) => api.vaultSettings.settingsTagsDelete(String(vid), id)}
        title={t("vault_settings.tags")}
        itemNameKey="name"
    /> },
    { key: "dateTypes", label: t("vault_settings.date_types"), children: <SimpleCrudTab 
        queryKeySuffix="contactImportantDateTypes"
        apiList={(vid) => api.vaultSettings.settingsDateTypesList(String(vid))}
        apiCreate={(vid, data) => api.vaultSettings.settingsDateTypesCreate(String(vid), data as unknown as import("@/api/generated/data-contracts").GithubComNaibaBondsInternalDtoCreateImportantDateTypeRequest)}
        apiUpdate={(vid, id, data) => api.vaultSettings.settingsDateTypesUpdate(String(vid), id, data as unknown as import("@/api/generated/data-contracts").GithubComNaibaBondsInternalDtoUpdateImportantDateTypeRequest)}
        apiDelete={(vid, id) => api.vaultSettings.settingsDateTypesDelete(String(vid), id)}
        title={t("vault_settings.date_types")}
    /> },
    { key: "moodParams", label: t("vault_settings.mood_params"), children: <SimpleCrudTab 
        queryKeySuffix="moodTrackingParameters"
        apiList={(vid) => api.vaultSettings.settingsMoodParamsList(String(vid))}
        apiCreate={(vid, data) => api.vaultSettings.settingsMoodParamsCreate(String(vid), data as unknown as import("@/api/generated/data-contracts").GithubComNaibaBondsInternalDtoCreateMoodTrackingParameterRequest)}
        apiUpdate={(vid, id, data) => api.vaultSettings.settingsMoodParamsUpdate(String(vid), id, data as unknown as import("@/api/generated/data-contracts").GithubComNaibaBondsInternalDtoUpdateMoodTrackingParameterRequest)}
        apiDelete={(vid, id) => api.vaultSettings.settingsMoodParamsDelete(String(vid), id)}
        title={t("vault_settings.mood_params")}
        extraFields={[{name: 'hex_color', label: t("vault_settings.hex_color"), type: 'color', initialValue: '#1677ff'}]}
        positionEntityType="moodParams"
    /> },
    { key: "lifeEvents", label: t("vault_settings.life_events"), children: <LifeEventsTab /> },
    { key: "quickFacts", label: t("vault_settings.quick_facts"), children: <SimpleCrudTab 
        queryKeySuffix="quickFactTemplates"
        apiList={(vid) => api.vaultSettings.settingsQuickFactTemplatesList(String(vid))}
        apiCreate={(vid, data) => api.vaultSettings.settingsQuickFactTemplatesCreate(String(vid), data as unknown as import("@/api/generated/data-contracts").GithubComNaibaBondsInternalDtoCreateQuickFactTemplateRequest)}
        apiUpdate={(vid, id, data) => api.vaultSettings.settingsQuickFactTemplatesUpdate(String(vid), id, data as unknown as import("@/api/generated/data-contracts").GithubComNaibaBondsInternalDtoUpdateQuickFactTemplateRequest)}
        apiDelete={(vid, id) => api.vaultSettings.settingsQuickFactTemplatesDelete(String(vid), id)}
        title={t("vault_settings.quick_facts")}
        positionEntityType="quickFactTemplates"
    /> },
    { key: "csv_import", label: t("vault_settings.csv_import.tab_label"), children: <CSVImportTab /> },
    { key: "import", label: t("vault_settings.monica_import.tab_label"), children: <MonicaImportTab /> },
  ];

  return (
    <div>
      <div style={{ marginBottom: 24, display: "flex", alignItems: "center", gap: 16 }}>
        <Link to={`/vaults/${vaultId}`}>
          <Button icon={<ArrowLeftOutlined />} shape="circle" />
        </Link>
        <div>
          <Title level={3} style={{ margin: 0 }}>
            {t("vault_settings.title")}
          </Title>
          <Text type="secondary">{t("vault_settings.description")}</Text>
        </div>
      </div>

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={tabItems}
        tabPosition="left"
        style={{ background: token.colorBgContainer, padding: 16, borderRadius: 8 }}
      />
    </div>
  );
}
