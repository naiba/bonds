import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Spin,
  Avatar,
  Button,
  Tabs,
  Descriptions,
  Space,
  Popconfirm,
  App,
  Tag,

  theme,
} from "antd";
import {
  EditOutlined,
  DeleteOutlined,
  StarOutlined,
  StarFilled,
  InboxOutlined,
  ArrowLeftOutlined,
  UserOutlined,
  DownloadOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { APIError } from "@/api";
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

const { Title, Text } = Typography;

export default function ContactDetail() {
  const { id, contactId } = useParams<{ id: string; contactId: string }>();
  const vaultId = id!;
  const cId = contactId!;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const { data: contact, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", cId],
    queryFn: async () => {
      const res = await api.contacts.contactsDetail(String(vaultId), String(cId));
      return res.data!;
    },
    enabled: !!vaultId && !!cId,
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

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!contact) return null;

  const initials = `${contact.first_name.charAt(0)}${contact.last_name?.charAt(0) ?? ""}`.toUpperCase();
  const moduleProps = { vaultId, contactId: cId };

  const tabItems = [
    {
      key: "overview",
      label: t("contact.detail.tabs.overview"),
      children: (
        <Space orientation="vertical" style={{ width: "100%" }} size={16}>
          <Card>
            <Descriptions column={{ xs: 1, sm: 2 }}>
              <Descriptions.Item label={t("contact.detail.first_name")}>
                {contact.first_name}
              </Descriptions.Item>
              <Descriptions.Item label={t("contact.detail.last_name")}>
                {contact.last_name || "—"}
              </Descriptions.Item>
              <Descriptions.Item label={t("contact.detail.nickname")}>
                {contact.nickname || "—"}
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
        <Space orientation="vertical" style={{ width: "100%" }} size={16}>
          <ContactInfoModule {...moduleProps} />
          <AddressesModule {...moduleProps} />
          <ImportantDatesModule {...moduleProps} />
          <PetsModule {...moduleProps} />
        </Space>
      ),
    },
    {
      key: "activities",
      label: t("contact.detail.tabs.activities"),
      children: (
        <Space orientation="vertical" style={{ width: "100%" }} size={16}>
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
        <Space orientation="vertical" style={{ width: "100%" }} size={16}>
          <LifeEventsModule {...moduleProps} />
          <MoodTrackingModule {...moduleProps} />
        </Space>
      ),
    },
    {
      key: "photos",
      label: t("contact.detail.tabs.photos_docs"),
      children: (
        <Space orientation="vertical" style={{ width: "100%" }} size={16}>
          <PhotosModule {...moduleProps} />
          <DocumentsModule {...moduleProps} />
        </Space>
      ),
    },
  ];

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
                display: "flex",
                gap: 20,
                alignItems: "center",
                flex: 1,
                minWidth: 0,
              }}
            >
              <Avatar
                size={80}
                icon={<UserOutlined />}
                style={{
                  fontSize: 30,
                  flexShrink: 0,
                  backgroundColor: token.colorPrimary,
                  boxShadow: `0 4px 12px ${token.colorPrimaryBorder}`,
                }}
              >
                {initials}
              </Avatar>
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
            >
              {t("common.edit")}
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

          <Popconfirm
            title={t("contact.detail.delete_confirm")}
            description={t("contact.detail.delete_warning")}
            onConfirm={() => deleteMutation.mutate()}
            okText={t("contact.detail.delete_ok")}
            okButtonProps={{ danger: true }}
          >
            <Button danger type="text" icon={<DeleteOutlined />}>
              {t("common.delete")}
            </Button>
          </Popconfirm>
        </div>
      </Card>

      <Tabs
        items={tabItems}
        defaultActiveKey="overview"
        style={{
          marginTop: 4,
        }}
        tabBarStyle={{
          marginBottom: 20,
          paddingLeft: 4,
        }}
      />
    </div>
  );
}
