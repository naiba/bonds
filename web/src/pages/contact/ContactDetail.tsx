import React, { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Spin,
  Button,
  Tabs,
  Descriptions,
  Space,
  Popconfirm,
  App,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  Upload,
  theme,
  Dropdown,
} from "antd";
import {
  EditOutlined,
  DeleteOutlined,
  StarOutlined,
  StarFilled,
  InboxOutlined,
  ArrowLeftOutlined,
  DownloadOutlined,
  CameraOutlined,
  ExportOutlined,
  MoreOutlined,
  LayoutOutlined,
} from "@ant-design/icons";
import { useMutation, useQueryClient, useQuery } from "@tanstack/react-query";
import { api, httpClient } from "@/api";
import type { APIError, UpdateContactRequest, Vault, PersonalizeItem, ContactTabsResponse, ContactTabPage } from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

import NotesModule from "./modules/NotesModule";
import RemindersModule from "./modules/RemindersModule";
import ImportantDatesModule from "./modules/ImportantDatesModule";
import TasksModule from "./modules/TasksModule";
import CallsModule from "./modules/CallsModule";
import AddressesModule from "./modules/AddressesModule";
import ContactInfoModule from "./modules/ContactInfoModule";
import LoansModule from "./modules/LoansModule";
import PetsModule from "./modules/PetsModule";
import RelationshipsModule from "./modules/RelationshipsModule";
import GoalsModule from "./modules/GoalsModule";
import LifeEventsModule from "./modules/LifeEventsModule";
import MoodTrackingModule from "./modules/MoodTrackingModule";
import QuickFactsModule from "./modules/QuickFactsModule";
import PhotosModule from "./modules/PhotosModule";
import DocumentsModule from "./modules/DocumentsModule";
import LabelsModule from "./modules/LabelsModule";
import FeedModule from "./modules/FeedModule";
import ExtraInfoModule from "./modules/ExtraInfoModule";
import GroupsModule from "./modules/GroupsModule";

const { Title, Text } = Typography;

// Module type → component mapping for dynamic tab rendering.
// Modules like avatar, contact_names, family_summary, gender_pronoun, company,
// religions are handled by the contact header card and ExtraInfoModule, not here.
const MODULE_COMPONENT_MAP: Record<
  string,
  React.ComponentType<{ vaultId: string; contactId: string; [key: string]: unknown }>
> = {
  notes: NotesModule,
  labels: LabelsModule,
  quick_facts: QuickFactsModule,
  relationships: RelationshipsModule,
  contact_information: ContactInfoModule,
  addresses: AddressesModule,
  important_dates: ImportantDatesModule,
  pets: PetsModule,
  tasks: TasksModule,
  calls: CallsModule,
  reminders: RemindersModule,
  loans: LoansModule,
  goals: GoalsModule,
  life_events: LifeEventsModule,
  groups: GroupsModule,
  mood_tracking: MoodTrackingModule,
  photos: PhotosModule,
  documents: DocumentsModule,
  feed: FeedModule,
};

export default function ContactDetail() {
  const { id, contactId } = useParams<{ id: string; contactId: string }>();
  const vaultId = id!;
  const cId = contactId!;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isMoveModalOpen, setIsMoveModalOpen] = useState(false);
  const [isTemplateModalOpen, setIsTemplateModalOpen] = useState(false);
  const [avatarKey, setAvatarKey] = useState(0);
  const [editForm] = Form.useForm();
  const [moveForm] = Form.useForm();
  const [templateForm] = Form.useForm();

  const { data: contact, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", cId],
    queryFn: async () => {
      const res = await api.contacts.contactsDetail(String(vaultId), String(cId));
      return res.data!;
    },
    enabled: !!vaultId && !!cId,
  });

  const { data: vaults = [] } = useQuery({
    queryKey: ["vaults"],
    queryFn: async () => {
      const res = await api.vaults.vaultsList();
      return res.data?.data || [];
    },
    enabled: isMoveModalOpen,
  });

  const { data: templates = [] } = useQuery<PersonalizeItem[]>({
    queryKey: ["settings", "personalize", "templates"],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail("templates");
      return res.data ?? [];
    },
    enabled: isTemplateModalOpen,
  });

  const { data: tabsData } = useQuery<ContactTabsResponse>({
    queryKey: ["vaults", vaultId, "contacts", cId, "tabs"],
    queryFn: async () => {
      const res = await api.contacts.contactsTabsList(String(vaultId), String(cId));
      return res.data!;
    },
    enabled: !!vaultId && !!cId && !!contact,
  });

  const updateContactMutation = useMutation({
    mutationFn: (values: UpdateContactRequest) =>
      api.contacts.contactsUpdate(String(vaultId), String(cId), values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
      message.success(t("contact.detail.edit_success"));
      setIsEditModalOpen(false);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const updateTemplateMutation = useMutation({
    mutationFn: (templateId: number) =>
      api.contacts.contactsTemplateUpdate(String(vaultId), String(cId), { template_id: templateId }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
      message.success(t("contact.detail.template_updated"));
      setIsTemplateModalOpen(false);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => api.contacts.contactsDelete(String(vaultId), String(cId)),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts"],
      });
      message.success(t("contact.detail.deleted_success"));
      navigate(`/vaults/${vaultId}/contacts`);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("contact.detail.delete_failed"));
    },
  });

  const toggleFavoriteMutation = useMutation({
    mutationFn: () => api.contacts.contactsFavoriteUpdate(String(vaultId), String(cId)),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
    },
    onError: (err: APIError) => {
      message.error(err.message || t("contact.detail.delete_failed"));
    },
  });

  const toggleArchiveMutation = useMutation({
    mutationFn: () => api.contacts.contactsArchiveUpdate(String(vaultId), String(cId)),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts"],
      });
    },
    onError: (err: APIError) => {
      message.error(err.message || t("contact.detail.delete_failed"));
    },
  });

  const avatarUploadMutation = useMutation({
    mutationFn: (file: File) =>
      api.contacts.contactsAvatarUpdate(String(vaultId), String(cId), {
        file: file as unknown as File,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
      setAvatarKey((k) => k + 1);
      message.success(t("contact.detail.avatar_updated"));
    },
    onError: (err: APIError) => {
      message.error(err.message || t("contact.detail.upload_failed"));
    },
  });

  const avatarDeleteMutation = useMutation({
    mutationFn: () => api.contacts.contactsAvatarDelete(String(vaultId), String(cId)),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
      setAvatarKey((k) => k + 1);
      message.success(t("contact.detail.avatar_deleted"));
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const moveContactMutation = useMutation({
    mutationFn: (targetVaultId: string) =>
      api.contacts.contactsMoveCreate(String(vaultId), String(cId), {
        target_vault_id: targetVaultId,
      }),
    onSuccess: (_, targetVaultId) => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts"],
      });
      message.success(t("contact.detail.move_success"));
      setIsMoveModalOpen(false);
      navigate(`/vaults/${targetVaultId}/contacts/${cId}`);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!contact) return null;

  const initials = `${contact.first_name?.charAt(0) ?? ""}${contact.last_name?.charAt(0) ?? ""}`.toUpperCase();
  const moduleProps = { vaultId, contactId: cId };

  const overviewCard = (
    <Card>
      <Descriptions column={{ xs: 1, sm: 2 }}>
        <Descriptions.Item label={t("contact.detail.first_name")}>
          {contact.first_name}
        </Descriptions.Item>
        <Descriptions.Item label={t("contact.detail.last_name")}>
          {contact.last_name || "\u2014"}
        </Descriptions.Item>
        <Descriptions.Item label={t("contact.detail.nickname")}>
          {contact.nickname || "\u2014"}
        </Descriptions.Item>
        <Descriptions.Item label={t("contact.detail.status")}>
          {contact.is_archived ? (
            <Tag color="default">{t("common.archived")}</Tag>
          ) : (
            <Tag color="green">{t("common.active")}</Tag>
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t("common.created")}>
          {dayjs(contact.created_at).format("MMMM D, YYYY")}
        </Descriptions.Item>
        <Descriptions.Item label={t("common.last_updated")}>
          {dayjs(contact.updated_at).format("MMMM D, YYYY")}
        </Descriptions.Item>
      </Descriptions>
    </Card>
  );

  function renderModulesForPage(page: ContactTabPage): React.ReactNode {
    const modules = page.modules ?? [];
    const isContactPage = page.type === "contact";

    const children: React.ReactNode[] = [];
    let extraInfoRendered = false;

    if (isContactPage) {
      children.push(<React.Fragment key="overview-card">{overviewCard}</React.Fragment>);
    }

    for (const mod of modules) {
      const moduleType = mod.type ?? "";

      if (isContactPage && moduleType === "labels") {
        children.push(<LabelsModule key={`mod-${mod.id}`} {...moduleProps} />);
        continue;
      }
      if (isContactPage && moduleType === "quick_facts") {
        children.push(
          <QuickFactsModule key={`mod-${mod.id}`} {...moduleProps} templateId={tabsData?.template_id ?? 1} />,
        );
        continue;
      }
      if (moduleType === "gender_pronoun" || moduleType === "religions" || moduleType === "company") {
        if (!extraInfoRendered) {
          extraInfoRendered = true;
          children.push(
            <ExtraInfoModule key="extra-info" {...moduleProps} contact={contact} />,
          );
        }
        continue;
      }

      const Component = MODULE_COMPONENT_MAP[moduleType];
      if (Component) {
        children.push(<Component key={`mod-${mod.id}`} {...moduleProps} />);
      }
    }

    if (children.length === 0) {
      return null;
    }
    if (children.length === 1) {
      return children[0];
    }
    return (
      <Space direction="vertical" style={{ width: "100%" }} size={16}>
        {children}
      </Space>
    );
  }

  function buildDynamicTabs(data: ContactTabsResponse) {
    return (data.pages ?? []).map((page) => ({
      key: page.slug ?? String(page.id),
      label: page.name ?? page.slug ?? "",
      children: renderModulesForPage(page),
    }));
  }

  const fallbackTabItems = [
    {
      key: "overview",
      label: t("contact.detail.tabs.overview"),
      children: (
        <Space direction="vertical" style={{ width: "100%" }} size={16}>
          {overviewCard}
          <LabelsModule {...moduleProps} />
          <QuickFactsModule {...moduleProps} templateId={1} />
          <NotesModule {...moduleProps} />
        </Space>
      ),
    },
    {
      key: "relationships",
      label: t("contact.detail.tabs.relationships"),
      children: <RelationshipsModule {...moduleProps} />,
    },
    {
      key: "information",
      label: t("contact.detail.tabs.information"),
      children: (
        <Space direction="vertical" style={{ width: "100%" }} size={16}>
          <ContactInfoModule {...moduleProps} />
          <AddressesModule {...moduleProps} />
          <ImportantDatesModule {...moduleProps} />
          <ExtraInfoModule {...moduleProps} contact={contact} />
          <PetsModule {...moduleProps} />
        </Space>
      ),
    },
    {
      key: "activities",
      label: t("contact.detail.tabs.activities"),
      children: (
        <Space direction="vertical" style={{ width: "100%" }} size={16}>
          <TasksModule {...moduleProps} />
          <CallsModule {...moduleProps} />
          <RemindersModule {...moduleProps} />
          <LoansModule {...moduleProps} />
          <GoalsModule {...moduleProps} />
        </Space>
      ),
    },
    {
      key: "life",
      label: t("contact.detail.tabs.life"),
      children: (
        <Space direction="vertical" style={{ width: "100%" }} size={16}>
          <LifeEventsModule {...moduleProps} />
          <MoodTrackingModule {...moduleProps} />
        </Space>
      ),
    },
    {
      key: "photos",
      label: t("contact.detail.tabs.photos_docs"),
      children: (
        <Space direction="vertical" style={{ width: "100%" }} size={16}>
          <PhotosModule {...moduleProps} />
          <DocumentsModule {...moduleProps} />
        </Space>
      ),
    },
    {
      key: "feed",
      label: t("contact.detail.feed.title"),
      children: <FeedModule {...moduleProps} />,
    },
  ];

  const tabItems = tabsData ? buildDynamicTabs(tabsData) : fallbackTabItems;

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}/contacts`)}
        style={{ marginBottom: 16 }}
      >
        {t("contact.detail.back")}
      </Button>

      <Card
        style={{ marginBottom: 24, overflow: "hidden" }}
        styles={{
          body: { padding: 0 },
        }}
      >
        <div
          style={{
            background: `linear-gradient(135deg, ${token.colorPrimaryBg} 0%, ${token.colorBgContainer} 100%)`,
            padding: "28px 24px 20px",
          }}
        >
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "flex-start",
              flexWrap: "wrap",
              gap: 20,
            }}
          >
            <div
              style={{
                position: "relative",
                width: 80,
                height: 80,
                borderRadius: "50%",
                flexShrink: 0,
                boxShadow: `0 4px 12px ${token.colorPrimaryBorder}`,
              }}
            >
              <AvatarImageLoader 
                url={`/vaults/${vaultId}/contacts/${cId}/avatar?k=${avatarKey}`} 
                updatedAt={contact.updated_at ?? ""}
                initials={initials}
                token={token}
                onUpload={(file) => avatarUploadMutation.mutate(file)}
                onDelete={() => avatarDeleteMutation.mutate()}
                isUploading={avatarUploadMutation.isPending}
              />
            </div>
            <div style={{ minWidth: 0 }}>
              <Title level={3} style={{ margin: 0 }}>
                {contact.first_name} {contact.last_name}
              </Title>
              {contact.nickname && (
                <Text type="secondary" style={{ fontSize: 15 }}>
                  &ldquo;{contact.nickname}&rdquo;
                </Text>
              )}
              <div style={{ marginTop: 6, display: "flex", gap: 6, flexWrap: "wrap" }}>
                {contact.is_favorite && (
                  <Tag color="gold" icon={<StarFilled />}>
                    {t("contact.detail.favorite")}
                  </Tag>
                )}
                {contact.is_archived && <Tag color="default">{t("common.archived")}</Tag>}
              </div>
            </div>
          </div>
        </div>

        <div
          style={{
            padding: "12px 24px",
            display: "flex",
            alignItems: "center",
            justifyContent: "flex-end",
            flexWrap: "wrap",
            gap: 8,
            borderTop: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          <Space size={4} wrap>
            <Button
              icon={<EditOutlined />}
              type="text"
              onClick={() => {
                editForm.setFieldsValue({
                  first_name: contact.first_name,
                  last_name: contact.last_name,
                  nickname: contact.nickname,
                });
                setIsEditModalOpen(true);
              }}
            >
              {t("common.edit")}
            </Button>
            <Button
              icon={<ExportOutlined />}
              type="text"
              onClick={() => {
                setIsMoveModalOpen(true);
              }}
            >
              {t("contact.detail.move")}
            </Button>
            <Button
              icon={contact.is_favorite ? <StarFilled /> : <StarOutlined />}
              type="text"
              onClick={() => toggleFavoriteMutation.mutate()}
            >
              {contact.is_favorite ? t("contact.detail.unfavorite") : t("contact.detail.favorite")}
            </Button>
          </Space>

          <span style={{ width: 1, height: 20, background: token.colorBorderSecondary, margin: "0 4px", flexShrink: 0 }} />

          <Space size={4} wrap>
            <Button
              icon={<DownloadOutlined />}
              type="text"
              onClick={async () => {
                try {
                  const res = await api.vcard.contactsVcardList(String(vaultId), String(cId));
                  const blob = new Blob([res as BlobPart]);
                  const url = URL.createObjectURL(blob);
                  const a = document.createElement("a");
                  a.href = url;
                  a.download = `${contact.first_name}_${contact.last_name}.vcf`;
                  a.click();
                  URL.revokeObjectURL(url);
                } catch {
                  message.error(t("contact.detail.delete_failed"));
                }
              }}
            >
              {t("vcard.export")}
            </Button>
            <Button
              icon={<InboxOutlined />}
              type="text"
              onClick={() => toggleArchiveMutation.mutate()}
              loading={toggleArchiveMutation.isPending}
            >
              {contact.is_archived ? t("contact.detail.unarchive") : t("contact.detail.archive")}
            </Button>
          </Space>

          <span style={{ width: 1, height: 20, background: token.colorBorderSecondary, margin: "0 4px", flexShrink: 0 }} />

          <Dropdown
            menu={{
              items: [
                {
                  key: "template",
                  label: t("contact.detail.change_template"),
                  icon: <LayoutOutlined />,
                  onClick: () => {
                    templateForm.setFieldValue("template_id", contact.template_id);
                    setIsTemplateModalOpen(true);
                  },
                },
                {
                  type: "divider",
                },
                {
                  key: "delete",
                  label: t("common.delete"),
                  icon: <DeleteOutlined />,
                  danger: true,
                  onClick: () => {
                    Modal.confirm({
                      title: t("contact.detail.delete_confirm"),
                      content: t("contact.detail.delete_warning"),
                      okText: t("contact.detail.delete_ok"),
                      okType: "danger",
                      cancelText: t("common.cancel"),
                      onOk: () => deleteMutation.mutate(),
                    });
                  },
                },
              ],
            }}
            trigger={["click"]}
          >
            <Button icon={<MoreOutlined />} type="text" />
          </Dropdown>
        </div>
      </Card>

      <Tabs
        items={tabItems}
        defaultActiveKey={tabItems[0]?.key ?? "overview"}
        style={{
          marginTop: 4,
        }}
        tabBarStyle={{
          marginBottom: 20,
          paddingLeft: 4,
        }}
      />

      <Modal
        title={t("contact.detail.edit_title")}
        open={isEditModalOpen}
        onCancel={() => setIsEditModalOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Form
          form={editForm}
          layout="vertical"
          onFinish={(values) => updateContactMutation.mutate(values)}
        >
          <Form.Item
            name="first_name"
            label={t("contact.detail.first_name")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="last_name"
            label={t("contact.detail.last_name")}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="nickname"
            label={t("contact.detail.nickname")}
          >
            <Input />
          </Form.Item>
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => setIsEditModalOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              type="primary"
              htmlType="submit"
              loading={updateContactMutation.isPending}
            >
              {t("common.save")}
            </Button>
          </div>
        </Form>
      </Modal>

      <Modal
        title={t("contact.detail.move_title")}
        open={isMoveModalOpen}
        onCancel={() => setIsMoveModalOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Form
          form={moveForm}
          layout="vertical"
          onFinish={(values) => moveContactMutation.mutate(values.target_vault_id)}
        >
          <Form.Item
            name="target_vault_id"
            label={t("contact.detail.select_vault")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Select
              loading={!vaults.length}
              options={vaults
                .filter((v: Vault) => v.id !== vaultId)
                .map((v: Vault) => ({ label: v.name, value: v.id }))}
            />
          </Form.Item>
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => setIsMoveModalOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              type="primary"
              htmlType="submit"
              loading={moveContactMutation.isPending}
            >
              {t("contact.detail.move")}
            </Button>
          </div>
        </Form>
      </Modal>

      <Modal
        title={t("contact.detail.change_template")}
        open={isTemplateModalOpen}
        onCancel={() => setIsTemplateModalOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Form
          form={templateForm}
          layout="vertical"
          onFinish={(values) => updateTemplateMutation.mutate(values.template_id)}
        >
          <Form.Item
            name="template_id"
            label={t("vault_settings.select_template")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Select
              loading={!templates.length}
              options={templates.map((tpl) => ({ label: tpl.label, value: tpl.id }))}
            />
          </Form.Item>
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => setIsTemplateModalOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              type="primary"
              htmlType="submit"
              loading={updateTemplateMutation.isPending}
            >
              {t("common.save")}
            </Button>
          </div>
        </Form>
      </Modal>
    </div>
  );
}

// Helper component to load authenticated image blob
function AvatarImageLoader({ 
  url, 
  updatedAt, 
  initials, 
  token,
  onUpload,
  onDelete,
  isUploading
}: { 
  url: string; 
  updatedAt: string;
  initials: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  token: any;
  onUpload: (file: File) => void;
  onDelete: () => void;
  isUploading: boolean;
}) {
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [hasAvatar, setHasAvatar] = useState(false);
  const { t } = useTranslation();

  // Fetch avatar image with auth header
  useEffect(() => {
    let revoke: string | null = null;
    let cancelled = false;

    httpClient.instance
      .get(url, { responseType: "blob", params: { t: dayjs(updatedAt).unix() } })
      .then((response) => {
        if (cancelled) return;
        const newUrl = URL.createObjectURL(response.data as Blob);
        revoke = newUrl;
        setBlobUrl(newUrl);
        setHasAvatar(true);
      })
      .catch(() => {
        if (cancelled) return;
        setBlobUrl(null);
        setHasAvatar(false);
      });

    return () => {
      cancelled = true;
      if (revoke) URL.revokeObjectURL(revoke);
    };
  }, [url, updatedAt]);

  return (
    <div
      style={{
        position: "absolute",
        inset: 0,
        borderRadius: "50%",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        overflow: "hidden",
        backgroundColor: token.colorPrimary,
      }}
    >
      {isUploading ? (
        <Spin />
      ) : hasAvatar && blobUrl ? (
        <img
          src={blobUrl}
          alt="Avatar"
          style={{ width: "100%", height: "100%", objectFit: "cover" }}
        />
      ) : (
        <span style={{ fontSize: 30, color: "#fff", fontWeight: 500 }}>
          {initials}
        </span>
      )}

      {/* Hover Overlay */}
      <div
        style={{
          position: "absolute",
          inset: 0,
          backgroundColor: "rgba(0,0,0,0.5)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          opacity: 0,
          transition: "opacity 0.2s",
          cursor: "pointer",
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.opacity = "1";
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.opacity = "0";
        }}
      >
         <Space>
            <Upload
              showUploadList={false}
              beforeUpload={(file) => {
                onUpload(file);
                return false;
              }}
            >
              <Button
                type="text"
                icon={<CameraOutlined style={{ color: "#fff", fontSize: 20 }} />}
                style={{ color: "#fff" }}
              />
            </Upload>
            {hasAvatar && (
              <Popconfirm
                title={t("contact.detail.delete_confirm")}
                onConfirm={onDelete}
              >
                 <Button
                  type="text"
                  icon={<DeleteOutlined style={{ color: "#fff", fontSize: 16 }} />}
                  danger
                />
              </Popconfirm>
            )}
          </Space>
      </div>
    </div>
  );
}
