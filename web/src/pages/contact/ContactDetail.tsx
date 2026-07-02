import React, { useState, useEffect } from "react";
import { formatContactName, formatContactInitials, useVaultNameOrder } from "@/utils/nameFormat";
import { useDateFormat, formatDate } from "@/utils/dateFormat";
import { dateInputToTimestamp, formatDateOnly, timestampToDateInput } from "@/utils/dateOnlyInput";
import CalendarDatePicker from "@/components/CalendarDatePicker";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";
import { buildContactFirstMetRequest, contactFirstMetToCalendarDate, formatContactFirstMetDisplay } from "@/utils/contactFirstMet";
import { Link, useParams, useNavigate, useLocation } from "react-router-dom";
import {
  Card,
  Typography,
  Spin,
  Button,
  Tabs,
  Space,
  Popconfirm,
  App,
  Tag,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Upload,
  theme,
  Dropdown,
  Checkbox,
  Segmented,
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
  CheckCircleOutlined,
} from "@ant-design/icons";
import { useMutation, useQueryClient, useQuery } from "@tanstack/react-query";
import { api, httpClient } from "@/api";
import type { APIError, Contact, UpdateContactRequest, Vault, PersonalizeItem, ContactTabsResponse, ContactTabPage } from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

import NotesModule from "./modules/NotesModule";
import RemindersModule from "./modules/RemindersModule";
import ImportantDatesModule from "./modules/ImportantDatesModule";
import TasksModule from "./modules/TasksModule";
import CallsModule from "./modules/CallsModule";
import AddressesModule from "./modules/AddressesModule";
import ContactInfoModule from "./modules/ContactInfoModule";
import GiftsModule from "./modules/GiftsModule";
import LoansModule from "./modules/LoansModule";
import PetsModule from "./modules/PetsModule";
import RelationshipsModule from "./modules/RelationshipsModule";
import GoalsModule from "./modules/GoalsModule";
import LifeEventsModule from "./modules/LifeEventsModule";
import QuickFactsModule from "./modules/QuickFactsModule";
import PhotosModule from "./modules/PhotosModule";
import DocumentsModule from "./modules/DocumentsModule";
import LabelsModule from "./modules/LabelsModule";
import FeedModule from "./modules/FeedModule";
import ExtraInfoModule from "./modules/ExtraInfoModule";
import GroupsModule from "./modules/GroupsModule";
import ContactSummaryCard from "./modules/ContactSummaryCard";

const { Title, Text } = Typography;

function buildContactListUrl(vaultId: string, search: string): string {
  const incomingParams = new URLSearchParams(search);
  const listParams = new URLSearchParams();
  const page = incomingParams.get("page");
  const perPage = incomingParams.get("per_page");

  if (page) listParams.set("page", page);
  if (perPage) listParams.set("per_page", perPage);

  const query = listParams.toString();
  return `/vaults/${vaultId}/contacts${query ? `?${query}` : ""}`;
}

type ContactEditFormValues = Omit<UpdateContactRequest, "last_talked_to" | "first_met_at" | "first_met_date_precision" | "first_met_year" | "first_met_month" | "first_met_day"> & {
  last_talked_to?: string;
  first_met?: CalendarDatePickerValue;
};

function buildUpdateContactRequest(values: ContactEditFormValues): UpdateContactRequest {
  const request: UpdateContactRequest = {
    ...values,
    last_talked_to: dateInputToTimestamp(values.last_talked_to),
    ...buildContactFirstMetRequest(values.first_met),
  };
  if (!request.last_talked_to) delete request.last_talked_to;
  if (!request.first_met_at) delete request.first_met_at;
  if (!request.first_met_date_precision) delete request.first_met_date_precision;
  if (request.first_met_year == null) delete request.first_met_year;
  if (request.first_met_month == null) delete request.first_met_month;
  if (request.first_met_day == null) delete request.first_met_day;
  if (!request.first_met_through_contact_id) delete request.first_met_through_contact_id;
  if (request.stay_in_touch_frequency_days == null) delete request.stay_in_touch_frequency_days;
  return request;
}

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
  gifts: GiftsModule,
  goals: GoalsModule,
  life_events: LifeEventsModule,
  groups: GroupsModule,
  photos: PhotosModule,
  documents: DocumentsModule,
  feed: FeedModule,
};

export default function ContactDetail() {
  const { id, contactId } = useParams<{ id: string; contactId: string }>();
  const vaultId = id!;
  const cId = contactId!;
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const nameOrder = useVaultNameOrder(vaultId);
  const dateFormats = useDateFormat();
  const [viewMode, setViewMode] = useState<"read" | "edit">("read");
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isMoveModalOpen, setIsMoveModalOpen] = useState(false);
  const [isTemplateModalOpen, setIsTemplateModalOpen] = useState(false);
  const [avatarKey, setAvatarKey] = useState(0);
  const [editForm] = Form.useForm();
  const [moveForm] = Form.useForm();
  const [templateForm] = Form.useForm();
  const contactListUrl = buildContactListUrl(vaultId, location.search);

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
      // Fix #58: vaultsList() returns { data: VaultResponse[] } — don't double-unwrap with .data.data
      return res.data ?? [];
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

  const { data: metThroughContacts = [], isLoading: isMetThroughContactsLoading } = useQuery<Contact[]>({
    queryKey: ["vaults", vaultId, "contacts", "meeting-select"],
    queryFn: async () => {
      const res = await api.contacts.contactsList(String(vaultId), { per_page: 9999, filter: "all" });
      return res.data ?? [];
    },
    enabled: isEditModalOpen,
  });

  const updateContactMutation = useMutation({
    mutationFn: (values: UpdateContactRequest) =>
      api.contacts.contactsUpdate(String(vaultId), String(cId), values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts"],
      });
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "catchUp"],
      });
      message.success(t("contact.detail.edit_success"));
      setIsEditModalOpen(false);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const promoteRelationshipContactMutation = useMutation({
    mutationFn: (request: UpdateContactRequest) => api.contacts.contactsUpdate(String(vaultId), String(cId), request),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts"],
      });
      message.success(t("contact.needs_verification.promoted"));
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const markCaughtUpMutation = useMutation({
    mutationFn: () => api.contacts.contactsCatchUpCreate(String(vaultId), String(cId)),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts"],
      });
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "catchUp"],
      });
      message.success(t("contact.catch_up.marked_caught_up"));
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
      navigate(contactListUrl);
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

  const initials = formatContactInitials(nameOrder, contact);
  const moduleProps = { vaultId, contactId: cId };

  // Compact overview card — only shows fields that have values,
  // timestamps rendered as subtle footer text to save vertical space.
  const overviewFields = [
    contact.prefix && { label: t("contact.detail.prefix"), value: contact.prefix },
    { label: t("contact.detail.first_name"), value: contact.first_name },
    contact.middle_name && { label: t("contact.detail.middle_name"), value: contact.middle_name },
    contact.last_name && { label: t("contact.detail.last_name"), value: contact.last_name },
    contact.suffix && { label: t("contact.detail.suffix"), value: contact.suffix },
    contact.nickname && { label: t("contact.detail.nickname"), value: `\u201C${contact.nickname}\u201D` },
    contact.maiden_name && { label: t("contact.detail.maiden_name"), value: contact.maiden_name },
    (() => {
      const firstMetLabel = formatContactFirstMetDisplay(contact, dateFormats);
      return firstMetLabel ? { label: t("contact.meeting.first_met_at"), value: firstMetLabel } : null;
    })(),
  ].filter(Boolean) as { label: string; value: string }[];

  const metThroughContact = contact.first_met_through_contact;

  const stayInTouchSummary = [
    contact.last_talked_to && t("contact.catch_up.last_contact_summary", {
      date: formatDateOnly(contact.last_talked_to, dateFormats),
    }),
    contact.stay_in_touch_frequency_days && t("contact.catch_up.frequency_summary", {
      days: contact.stay_in_touch_frequency_days,
    }),
    contact.stay_in_touch_trigger_date && t("contact.catch_up.next_due_summary", {
      date: formatDateOnly(contact.stay_in_touch_trigger_date, dateFormats),
    }),
  ].filter(Boolean).join(" · ");

  const stayInTouchPanel = stayInTouchSummary ? (
    <div
      style={{
        padding: "10px 12px",
        borderRadius: token.borderRadius,
        background: token.colorFillQuaternary,
        display: "flex",
        justifyContent: "space-between",
        gap: 12,
        alignItems: "center",
      }}
    >
      <div style={{ minWidth: 0 }}>
        <Text strong style={{ fontSize: 13, display: "block" }}>
          {t("contact.catch_up.title")}
        </Text>
        <Text type="secondary" style={{ fontSize: 12 }}>
          {stayInTouchSummary}
        </Text>
      </div>
      <Button
        size="small"
        icon={<CheckCircleOutlined />}
        loading={markCaughtUpMutation.isPending}
        onClick={() => markCaughtUpMutation.mutate()}
      >
        {t("contact.catch_up.mark_caught_up")}
      </Button>
    </div>
  ) : null;

  const overviewCard = (
    <Card size="small" styles={{ body: { padding: "12px 16px" } }}>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))", gap: "6px 24px" }}>
        {overviewFields.map((f) => (
          <div key={f.label} style={{ display: "flex", gap: 8, alignItems: "baseline" }}>
            <Text type="secondary" style={{ fontSize: 13, flexShrink: 0 }}>{f.label}:</Text>
            <Text style={{ fontSize: 13 }}>{f.value}</Text>
          </div>
        ))}
      </div>
      <div style={{ marginTop: 8, display: "flex", gap: 16, flexWrap: "wrap", alignItems: "center" }}>
        {contact.is_archived ? (
          <Tag color="default" style={{ margin: 0 }}>{t("common.archived")}</Tag>
        ) : (
          <Tag color="green" style={{ margin: 0 }}>{t("common.active")}</Tag>
        )}
        {contact.needs_verification && (
          <Tag color="warning" style={{ margin: 0 }}>
            {t("contact.needs_verification.badge")}
          </Tag>
        )}
        {metThroughContact?.id && metThroughContact.name && (
          <Text type="secondary" style={{ fontSize: 12 }}>
            {t("contact.meeting.first_met_through")}{" "}
            <Link to={`/vaults/${vaultId}/contacts/${metThroughContact.id}`}>
              {metThroughContact.name}
            </Link>
          </Text>
        )}
        <Text type="secondary" style={{ fontSize: 12 }}>
          {t("common.created")} {formatDate(contact.created_at, dateFormats)}
          {" · "}
          {t("common.last_updated")} {formatDate(contact.updated_at, dateFormats)}
        </Text>
      </div>
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
          <QuickFactsModule key={`mod-${mod.id}`} {...moduleProps} readOnly={viewMode === "read"} />,
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
          <QuickFactsModule {...moduleProps} readOnly={viewMode === "read"} />
          <NotesModule {...moduleProps} readOnly={viewMode === "read"} />
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
  const isRelationshipOnlyHiddenContact = contact.needs_verification && !contact.listed;

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(contactListUrl)}
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
                borderRadius: 24,
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
            <div style={{ minWidth: 0, paddingTop: 4 }}>
              <Title level={2} style={{ margin: 0, fontFamily: "\x27Playfair Display\x27, serif" }}>
                {formatContactName(nameOrder, contact)}
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
                {isRelationshipOnlyHiddenContact && (
                  <Button
                    size="small"
                    onClick={() => promoteRelationshipContactMutation.mutate({
                      first_name: contact.first_name ?? "",
                      last_name: contact.last_name,
                      middle_name: contact.middle_name,
                      nickname: contact.nickname,
                      maiden_name: contact.maiden_name,
                      prefix: contact.prefix,
                      suffix: contact.suffix,
                      gender_id: contact.gender_id,
                      pronoun_id: contact.pronoun_id,
                      template_id: contact.template_id,
                      listed: true,
                      needs_verification: false,
                      first_met_through_contact_id: contact.first_met_through_contact_id,
                      ...buildContactFirstMetRequest(contactFirstMetToCalendarDate(contact)),
                      last_talked_to: dateInputToTimestamp(timestampToDateInput(contact.last_talked_to)),
                      stay_in_touch_frequency_days: contact.stay_in_touch_frequency_days,
                    })}
                    loading={promoteRelationshipContactMutation.isPending}
                  >
                    {t("contact.needs_verification.promote_action")}
                  </Button>
                )}
              </div>
            </div>
          </div>
        </div>

        <div
          style={{
            padding: "8px 24px",
            display: "flex",
            alignItems: "center",
            justifyContent: "flex-end",
            gap: 4,
            borderTop: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          <Button
            icon={<EditOutlined />}
            type="text"
            size="small"
            onClick={() => {
              editForm.setFieldsValue({
                prefix: contact.prefix,
                first_name: contact.first_name,
                middle_name: contact.middle_name,
                last_name: contact.last_name,
                suffix: contact.suffix,
                nickname: contact.nickname,
                maiden_name: contact.maiden_name,
                gender_id: contact.gender_id,
                pronoun_id: contact.pronoun_id,
                first_met: contactFirstMetToCalendarDate(contact),
                first_met_through_contact_id: contact.first_met_through_contact_id,
                last_talked_to: timestampToDateInput(contact.last_talked_to),
                stay_in_touch_frequency_days: contact.stay_in_touch_frequency_days,
                needs_verification: contact.needs_verification,
              });
              setIsEditModalOpen(true);
            }}
          >
            {t("common.edit")}
          </Button>
          <Button
            icon={contact.is_favorite ? <StarFilled /> : <StarOutlined />}
            type="text"
            size="small"
            onClick={() => toggleFavoriteMutation.mutate()}
          >
            {contact.is_favorite ? t("contact.detail.unfavorite") : t("contact.detail.favorite")}
          </Button>

          <Dropdown
            menu={{
              items: [
                {
                  key: "move",
                  label: t("contact.detail.move"),
                  icon: <ExportOutlined />,
                  onClick: () => setIsMoveModalOpen(true),
                },
                {
                  key: "export",
                  label: t("vcard.export"),
                  icon: <DownloadOutlined />,
                  onClick: async () => {
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
                  },
                },
                {
                  key: "archive",
                  label: contact.is_archived ? t("contact.detail.unarchive") : t("contact.detail.archive"),
                  icon: <InboxOutlined />,
                  onClick: () => toggleArchiveMutation.mutate(),
                },
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
            <Button icon={<MoreOutlined />} type="text" size="small" />
          </Dropdown>
        </div>
      </Card>

      {stayInTouchPanel && (
        <div style={{ marginBottom: 16 }}>
          {stayInTouchPanel}
        </div>
      )}

      <div style={{ marginBottom: 16 }}>
        <ContactSummaryCard vaultId={vaultId} contactId={cId} contact={contact} readOnly={viewMode === "read"} />
      </div>

      <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: 16 }}>
        <Segmented
          options={[
            { label: t("contact.detail.view_mode"), value: "read" },
            { label: t("contact.detail.edit_mode"), value: "edit" },
          ]}
          value={viewMode}
          onChange={(val) => setViewMode(val as "read" | "edit")}
        />
      </div>

      {viewMode === "edit" ? (
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
      ) : (
        <Space direction="vertical" style={{ width: "100%" }} size={16}>
          <QuickFactsModule {...moduleProps} readOnly={true} />
          <NotesModule {...moduleProps} readOnly={true} />
        </Space>
      )}

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
          onFinish={(values: ContactEditFormValues) => updateContactMutation.mutate(buildUpdateContactRequest(values))}
        >
          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item
              name="prefix"
              label={t("contact.detail.prefix")}
              style={{ flex: 1 }}
            >
              <Input placeholder={t("contact.create.prefix_placeholder")} />
            </Form.Item>
            <Form.Item
              name="first_name"
              label={t("contact.detail.first_name")}
              style={{ flex: 2 }}
              dependencies={["nickname"]}
              rules={[{
                validator: (_, value) => {
                  const nickname = editForm.getFieldValue("nickname");
                  if (!value?.trim() && !nickname?.trim()) {
                    return Promise.reject(new Error(t("contact.form.name_or_nickname_required")));
                  }
                  return Promise.resolve();
                },
              }]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="middle_name"
              label={t("contact.detail.middle_name")}
              style={{ flex: 2 }}
            >
              <Input />
            </Form.Item>
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item
              name="last_name"
              label={t("contact.detail.last_name")}
              style={{ flex: 2 }}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="suffix"
              label={t("contact.detail.suffix")}
              style={{ flex: 1 }}
            >
              <Input placeholder={t("contact.create.suffix_placeholder")} />
            </Form.Item>
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item
              name="nickname"
              label={t("contact.detail.nickname")}
              style={{ flex: 1 }}
              dependencies={["first_name"]}
              rules={[{
                validator: (_, value) => {
                  const firstName = editForm.getFieldValue("first_name");
                  if (!value?.trim() && !firstName?.trim()) {
                    return Promise.reject(new Error(t("contact.form.name_or_nickname_required")));
                  }
                  return Promise.resolve();
                },
              }]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="maiden_name"
              label={t("contact.detail.maiden_name")}
              style={{ flex: 1 }}
            >
              <Input />
            </Form.Item>
          </div>
          {/* Fix #62: gender and pronoun fields — fetched from personalize API */}
          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item name="gender_id" label={t("contact.detail.summary.gender")} style={{ flex: 1 }}>
              <GenderPronounSelect entity="genders" vaultId={vaultId} placeholder={t("contact.form.select_gender")} />
            </Form.Item>
            <Form.Item name="pronoun_id" label={t("contact.detail.summary.pronoun")} style={{ flex: 1 }}>
              <GenderPronounSelect entity="pronouns" vaultId={vaultId} placeholder={t("contact.form.select_pronoun")} />
            </Form.Item>
          </div>
          <Form.Item name="needs_verification" valuePropName="checked" style={{ marginBottom: 16 }}>
            <Checkbox>{t("contact.needs_verification.field_label")}</Checkbox>
          </Form.Item>
          <div
            style={{
              marginBottom: 16,
              padding: 16,
              border: `1px solid ${token.colorBorderSecondary}`,
              borderRadius: token.borderRadiusLG,
              background: token.colorFillQuaternary,
            }}
          >
            <Text strong style={{ display: "block", marginBottom: 4 }}>
              {t("contact.meeting.title")}
            </Text>
            <Text type="secondary" style={{ display: "block", fontSize: 13, marginBottom: 12 }}>
              {t("contact.meeting.description")}
            </Text>
            <div style={{ display: "flex", gap: 16 }}>
              <Form.Item
                name="first_met"
                label={t("contact.meeting.first_met_at")}
                extra={t("contact.meeting.first_met_at_help")}
                style={{ flex: 1 }}
              >
                <CalendarDatePicker enableDatePrecision allowedDatePrecisions={["full", "month", "year"]} />
              </Form.Item>
              <Form.Item
                name="first_met_through_contact_id"
                label={t("contact.meeting.first_met_through")}
                style={{ flex: 1 }}
              >
                <Select
                  loading={isMetThroughContactsLoading}
                  allowClear
                  showSearch
                  optionFilterProp="label"
                  placeholder={t("contact.meeting.first_met_through_placeholder")}
                  options={metThroughContacts
                    .filter((option) => option.id && option.id !== cId)
                    .map((option) => ({
                      label: formatContactName(nameOrder, option),
                      value: option.id,
                    }))}
                />
              </Form.Item>
            </div>
          </div>
          <div
            style={{
              marginBottom: 16,
              padding: 16,
              border: `1px solid ${token.colorBorderSecondary}`,
              borderRadius: token.borderRadiusLG,
              background: token.colorFillQuaternary,
            }}
          >
            <Text strong style={{ display: "block", marginBottom: 4 }}>
              {t("contact.catch_up.title")}
            </Text>
            <Text type="secondary" style={{ display: "block", fontSize: 13, marginBottom: 12 }}>
              {t("contact.catch_up.description")}
            </Text>
            <div style={{ display: "flex", gap: 16 }}>
              <Form.Item
                name="last_talked_to"
                label={t("contact.catch_up.last_talked_to")}
                extra={t("contact.catch_up.last_talked_to_help")}
                style={{ flex: 1 }}
              >
                <Input type="date" />
              </Form.Item>
              <Form.Item
                name="stay_in_touch_frequency_days"
                label={t("contact.catch_up.frequency_days")}
                extra={t("contact.catch_up.frequency_days_help")}
                style={{ flex: 1 }}
              >
                <InputNumber min={1} precision={0} style={{ width: "100%" }} />
              </Form.Item>
            </div>
          </div>
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
  token: ReturnType<typeof theme.useToken>["token"];
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
        borderRadius: 24,
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
        <span style={{ fontSize: 30, color: token.colorTextLightSolid, fontWeight: 500 }}>
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
                icon={<CameraOutlined style={{ color: token.colorTextLightSolid, fontSize: 20 }} />}
                style={{ color: token.colorTextLightSolid }}
              />
            </Upload>
            {hasAvatar && (
              <Popconfirm
                title={t("contact.detail.delete_confirm")}
                onConfirm={onDelete}
              >
                 <Button
                  type="text"
                  icon={<DeleteOutlined style={{ color: token.colorTextLightSolid, fontSize: 16 }} />}
                  danger
                />
              </Popconfirm>
            )}
          </Space>
      </div>
    </div>
  );
}

// Shared Select component for gender/pronoun fetched from personalize API
function GenderPronounSelect({ entity, vaultId, placeholder, ...props }: {
  entity: string;
  vaultId: string;
  placeholder: string;
  value?: number;
  onChange?: (value: number | undefined) => void;
}) {
  const { data: items = [], isLoading } = useQuery<PersonalizeItem[]>({
    queryKey: ["vaults", vaultId, "personalize", entity],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail(entity);
      return res.data ?? [];
    },
  });

  return (
    <Select
      {...props}
      loading={isLoading}
      allowClear
      placeholder={placeholder}
      options={items.map((item) => ({ label: item.label, value: item.id }))}
    />
  );
}
