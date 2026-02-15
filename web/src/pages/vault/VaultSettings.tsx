import { useState } from "react";
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
} from "antd";
import {
  SaveOutlined,
  DeleteOutlined,
  EditOutlined,
  UserAddOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import type { TabsProps } from "antd";
import type { AxiosResponse } from "axios";
import { vaultSettingsApi } from "@/api/vaultSettings";
import { settingsApi } from "@/api/settings";
import type { APIError, APIResponse } from "@/types/api";
import type { PersonalizeItem } from "@/types/modules";
import type {
  UpdateVaultSettingsRequest,
  VaultUserResponse,
  LabelResponse,
  LifeEventCategoryResponse,
  LifeEventCategoryTypeResponse,
} from "@/types/vaultSettings";

const { Title, Text } = Typography;
const { Option } = Select;

export default function VaultSettings() {
  const { id } = useParams<{ id: string }>();
  const vaultId = parseInt(id!, 10);
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  const [activeTab, setActiveTab] = useState("general");

  const { data: vaultSettings } = useQuery({
    queryKey: ["vault", vaultId, "settings"],
    queryFn: async () => {
      const res = await vaultSettingsApi.getSettings(vaultId);
      return res.data.data;
    },
    enabled: !!vaultId,
  });

  const { data: personalizeTemplates } = useQuery<PersonalizeItem[]>({
    queryKey: ["settings", "personalize", "templates"],
    queryFn: async () => {
      const res = await settingsApi.listPersonalizeItems("templates");
      return res.data.data ?? [];
    },
  });

  const updateSettingsMutation = useMutation({
    mutationFn: (data: UpdateVaultSettingsRequest) =>
      vaultSettingsApi.updateSettings(vaultId, data),
    onSuccess: () => {
      message.success(t("common.saved"));
      queryClient.invalidateQueries({ queryKey: ["vault", vaultId] });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateTabVisibilityMutation = useMutation({
    mutationFn: (data: Record<string, boolean>) =>
      vaultSettingsApi.updateTabVisibility(vaultId, data),
    onSuccess: () => {
      message.success(t("common.saved"));
      queryClient.invalidateQueries({ queryKey: ["vault", vaultId] });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  // --- Components for each tab ---

  const GeneralTab = () => {
    const [form] = Form.useForm();

    if (!vaultSettings) return null;

    return (
      <Card title={t("vault_settings.general")}>
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            name: vaultSettings.name,
            description: vaultSettings.description,
            default_template_id: vaultSettings.default_template_id,
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
          <Form.Item
            name="default_template_id"
            label={t("vault_settings.default_template")}
          >
            <Select>
              {personalizeTemplates?.map((tpl) => (
                <Option key={tpl.id} value={tpl.id}>
                  {tpl.label}
                </Option>
              ))}
            </Select>
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
        const res = await vaultSettingsApi.listUsers(vaultId);
        return res.data.data ?? [];
      },
    });

    const inviteMutation = useMutation({
      mutationFn: (values: { email: string; permission: number }) =>
        vaultSettingsApi.inviteUser(vaultId, values),
      onSuccess: () => {
        message.success(t("invitations.status.pending")); // Or specific success message
        queryClient.invalidateQueries({ queryKey: ["vault", vaultId, "users"] });
      },
      onError: (e: APIError) => message.error(e.message),
    });

    const removeUserMutation = useMutation({
      mutationFn: (userId: number) => vaultSettingsApi.removeUser(vaultId, userId),
      onSuccess: () => {
        message.success(t("common.deleted"));
        queryClient.invalidateQueries({ queryKey: ["vault", vaultId, "users"] });
      },
      onError: (e: APIError) => message.error(e.message),
    });

    const updateUserPermMutation = useMutation({
      mutationFn: ({ userId, permission }: { userId: number; permission: number }) =>
        vaultSettingsApi.updateUserPermission(vaultId, userId, { permission }),
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
                  <Select
                    key="perm"
                    defaultValue={user.permission}
                    style={{ width: 120 }}
                    onChange={(val) =>
                      updateUserPermMutation.mutate({ userId: user.user_id, permission: val })
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
                    onConfirm={() => removeUserMutation.mutate(user.user_id)}
                  >
                    <Button danger icon={<DeleteOutlined />} loading={removeUserMutation.isPending} />
                  </Popconfirm>,
                ]}
              >
                <List.Item.Meta
                  title={`${user.first_name} ${user.last_name}`}
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
      queryFn: async () => (await vaultSettingsApi.listLabels(vaultId)).data.data ?? [],
    });

    const createMutation = useMutation({
      mutationFn: (data: { name: string; description?: string; bg_color: string; text_color: string }) => vaultSettingsApi.createLabel(vaultId, data),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey });
        message.success(t("common.created"));
        form.resetFields();
      },
    });

    const updateMutation = useMutation({
      mutationFn: ({ id, data }: { id: number; data: { name?: string; description?: string; bg_color?: string; text_color?: string } }) =>
        vaultSettingsApi.updateLabel(vaultId, id, data),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey });
        message.success(t("common.updated"));
        setEditingId(null);
        form.resetFields();
      },
    });

    const deleteMutation = useMutation({
      mutationFn: (id: number) => vaultSettingsApi.deleteLabel(vaultId, id),
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
      setEditingId(item.id);
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
          <List
            loading={isLoading}
            dataSource={items}
            renderItem={(item) => (
              <List.Item
                actions={[
                  <Button icon={<EditOutlined />} onClick={() => startEdit(item)} />,
                  <Popconfirm title={t("common.delete_confirm")} onConfirm={() => deleteMutation.mutate(item.id)}>
                    <Button danger icon={<DeleteOutlined />} />
                  </Popconfirm>,
                ]}
              >
                <List.Item.Meta
                  avatar={
                    <Tag color={item.bg_color} style={{ color: item.text_color }}>
                      {item.name}
                    </Tag>
                  }
                  description={item.description}
                />
              </List.Item>
            )}
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
  const SimpleCrudTab = <T extends { id: number; label?: string; name?: string; hex_color?: string }>({
    queryKeySuffix,
    apiList,
    apiCreate,
    apiUpdate,
    apiDelete,
    title,
    itemNameKey = "label",
    extraFields = [],
  }: {
    queryKeySuffix: string;
    apiList: (vid: number) => Promise<AxiosResponse<APIResponse<T[]>>>;
    apiCreate: (vid: number, data: Record<string, unknown>) => Promise<unknown>;
    apiUpdate: (vid: number, id: number, data: Record<string, unknown>) => Promise<unknown>;
    apiDelete: (vid: number, id: number) => Promise<unknown>;
    title: string;
    itemNameKey?: "label" | "name";
    extraFields?: ExtraField[];
  }) => {
    const queryKey = ["vault", vaultId, queryKeySuffix];
    const { data: items = [], isLoading } = useQuery({
      queryKey,
      queryFn: async () => (await apiList(vaultId)).data.data ?? [],
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
            renderItem={(item: T) => (
              <List.Item
                actions={[
                  <Button icon={<EditOutlined />} onClick={() => startEdit(item)} />,
                  <Popconfirm title={t("common.delete_confirm")} onConfirm={() => deleteMutation.mutate(item.id)}>
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
      queryFn: async () => (await vaultSettingsApi.listLifeEventCategories(vaultId)).data.data ?? [],
    });

    const createCategory = useMutation({
        mutationFn: (data: { label: string }) => vaultSettingsApi.createLifeEventCategory(vaultId, data),
        onSuccess: () => { queryClient.invalidateQueries({ queryKey }); message.success(t("common.created")); },
    });
    const deleteCategory = useMutation({
        mutationFn: (id: number) => vaultSettingsApi.deleteLifeEventCategory(vaultId, id),
        onSuccess: () => { queryClient.invalidateQueries({ queryKey }); message.success(t("common.deleted")); },
    });

    const createType = useMutation({
        mutationFn: ({ catId, data }: { catId: number, data: { label: string } }) => 
            vaultSettingsApi.createLifeEventCategoryType(vaultId, catId, data),
        onSuccess: () => { queryClient.invalidateQueries({ queryKey }); message.success(t("common.created")); },
    });
    
    const deleteType = useMutation({
        mutationFn: ({ catId, typeId }: { catId: number, typeId: number }) => 
            vaultSettingsApi.deleteLifeEventCategoryType(vaultId, catId, typeId),
        onSuccess: () => { queryClient.invalidateQueries({ queryKey }); message.success(t("common.deleted")); },
    });

    const [newCatLabel, setNewCatLabel] = useState("");
    const [newTypeLabel, setNewTypeLabel] = useState<Record<number, string>>({});

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
                    {categories.map((cat: LifeEventCategoryResponse) => (
                        <Collapse.Panel 
                            key={cat.id} 
                            header={cat.label}
                            extra={
                                <Popconfirm title={t("common.delete_confirm")} onConfirm={(e) => { e?.stopPropagation(); deleteCategory.mutate(cat.id); }}>
                                    <DeleteOutlined onClick={(e) => e.stopPropagation()} style={{color: 'red'}} />
                                </Popconfirm>
                            }
                        >
                             <List
                                dataSource={cat.types}
                                header={
                                    <Space style={{width: '100%'}}>
                                        <Input 
                                            placeholder={t("vault_settings.add_type")} 
                                            value={newTypeLabel[cat.id] || ""}
                                            onChange={e => setNewTypeLabel(prev => ({ ...prev, [cat.id]: e.target.value }))}
                                            onPressEnter={() => handleAddType(cat.id)}
                                        />
                                        <Button type="dashed" onClick={() => handleAddType(cat.id)}>{t("common.add")}</Button>
                                    </Space>
                                }
                                renderItem={(type: LifeEventCategoryTypeResponse) => (
                                    <List.Item
                                        actions={[
                                            <Popconfirm title={t("common.delete_confirm")} onConfirm={() => deleteType.mutate({ catId: cat.id, typeId: type.id })}>
                                                <Button danger size="small" icon={<DeleteOutlined />} type="text" />
                                            </Popconfirm>
                                        ]}
                                    >
                                        {type.label}
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


  const tabItems: TabsProps["items"] = [
    { key: "general", label: t("vault_settings.general"), children: <GeneralTab /> },
    { key: "tabs", label: t("vault_settings.tabs"), children: <TabsTab /> },
    { key: "users", label: t("vault_settings.users"), children: <UsersTab /> },
    { key: "labels", label: t("vault_settings.labels"), children: <LabelsTab /> },
    { key: "tags", label: t("vault_settings.tags"), children: <SimpleCrudTab 
        queryKeySuffix="tags" 
        apiList={(vid) => vaultSettingsApi.listTags(vid)}
        apiCreate={(vid, data) => vaultSettingsApi.createTag(vid, data as unknown as import("@/types/vaultSettings").CreateTagRequest)}
        apiUpdate={(vid, id, data) => vaultSettingsApi.updateTag(vid, id, data as unknown as import("@/types/vaultSettings").UpdateTagRequest)}
        apiDelete={(vid, id) => vaultSettingsApi.deleteTag(vid, id)}
        title={t("vault_settings.tags")}
        itemNameKey="name"
    /> },
    { key: "dateTypes", label: t("vault_settings.date_types"), children: <SimpleCrudTab 
        queryKeySuffix="contactImportantDateTypes"
        apiList={(vid) => vaultSettingsApi.listImportantDateTypes(vid)}
        apiCreate={(vid, data) => vaultSettingsApi.createImportantDateType(vid, data as unknown as import("@/types/vaultSettings").CreateImportantDateTypeRequest)}
        apiUpdate={(vid, id, data) => vaultSettingsApi.updateImportantDateType(vid, id, data as unknown as import("@/types/vaultSettings").UpdateImportantDateTypeRequest)}
        apiDelete={(vid, id) => vaultSettingsApi.deleteImportantDateType(vid, id)}
        title={t("vault_settings.date_types")}
    /> },
    { key: "moodParams", label: t("vault_settings.mood_params"), children: <SimpleCrudTab 
        queryKeySuffix="moodTrackingParameters"
        apiList={(vid) => vaultSettingsApi.listMoodTrackingParameters(vid)}
        apiCreate={(vid, data) => vaultSettingsApi.createMoodTrackingParameter(vid, data as unknown as import("@/types/vaultSettings").CreateMoodTrackingParameterRequest)}
        apiUpdate={(vid, id, data) => vaultSettingsApi.updateMoodTrackingParameter(vid, id, data as unknown as import("@/types/vaultSettings").UpdateMoodTrackingParameterRequest)}
        apiDelete={(vid, id) => vaultSettingsApi.deleteMoodTrackingParameter(vid, id)}
        title={t("vault_settings.mood_params")}
        extraFields={[{name: 'hex_color', label: t("vault_settings.hex_color"), type: 'color', initialValue: '#1677ff'}]}
    /> },
    { key: "lifeEvents", label: t("vault_settings.life_events"), children: <LifeEventsTab /> },
    { key: "quickFacts", label: t("vault_settings.quick_facts"), children: <SimpleCrudTab 
        queryKeySuffix="quickFactTemplates"
        apiList={(vid) => vaultSettingsApi.listQuickFactTemplates(vid)}
        apiCreate={(vid, data) => vaultSettingsApi.createQuickFactTemplate(vid, data as unknown as import("@/types/vaultSettings").CreateQuickFactTemplateRequest)}
        apiUpdate={(vid, id, data) => vaultSettingsApi.updateQuickFactTemplate(vid, id, data as unknown as import("@/types/vaultSettings").UpdateQuickFactTemplateRequest)}
        apiDelete={(vid, id) => vaultSettingsApi.deleteQuickFactTemplate(vid, id)}
        title={t("vault_settings.quick_facts")}
    /> },
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
        style={{ background: "#fff", padding: 16, borderRadius: 8 }}
      />
    </div>
  );
}
