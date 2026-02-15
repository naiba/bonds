import { useState, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Table, Button, Typography, Input, Tag, Space, App, Upload, theme } from "antd";
import {
  PlusOutlined,
  SearchOutlined,
  StarFilled,
  DownloadOutlined,
  UploadOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { contactsApi } from "@/api/contacts";
import { vcardApi } from "@/api/vcard";
import type { Contact } from "@/types/contact";
import type { ColumnsType } from "antd/es/table";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title, Text } = Typography;

const AVATAR_COLORS = [
  "#5b8c5a", "#6b9e7a", "#7eb09c", "#4a8c8c", "#5a7c9e",
  "#8c7a5b", "#9e8a6b", "#b09a7e", "#8c6b5a", "#7a8c5b",
  "#6b7a5a", "#5a6b8c", "#7a5a8c", "#8c5a7a", "#5a8c6b",
];

function getAvatarColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}

export default function ContactList() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const vaultId = id!;
  const [search, setSearch] = useState("");
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const { data, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "contacts"],
    queryFn: async () => {
      const res = await contactsApi.list(vaultId);
      return res.data.data ?? [];
    },
    enabled: !!vaultId,
    meta: {
      onError: () => message.error(t("contact.list.load_failed")),
    },
  });

  const contacts = useMemo(() => {
    const all = data ?? [];
    if (!search) return all;
    const q = search.toLowerCase();
    return all.filter((c) =>
      c.first_name.toLowerCase().includes(q) ||
      c.last_name.toLowerCase().includes(q) ||
      c.nickname?.toLowerCase().includes(q)
    );
  }, [data, search]);

  const columns: ColumnsType<Contact> = [
    {
      title: t("contact.list.col_name"),
      key: "name",
      render: (_, record) => {
        const initials = `${record.first_name.charAt(0)}${record.last_name?.charAt(0) ?? ""}`.toUpperCase();
        const bgColor = getAvatarColor(record.first_name + record.last_name);
        return (
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <div
              style={{
                width: 34,
                height: 34,
                borderRadius: "50%",
                backgroundColor: bgColor,
                color: "#fff",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 13,
                fontWeight: 600,
                flexShrink: 0,
                letterSpacing: 0.5,
              }}
            >
              {initials}
            </div>
            <span style={{ fontWeight: 500 }}>
              {record.first_name} {record.last_name}
            </span>
            {record.is_favorite && (
              <StarFilled style={{ color: token.colorWarning, fontSize: 13 }} />
            )}
          </div>
        );
      },
      sorter: (a, b) => a.first_name.localeCompare(b.first_name),
    },
    {
      title: t("contact.list.col_nickname"),
      dataIndex: "nickname",
      key: "nickname",
      responsive: ["md"],
      render: (val: string) =>
        val ? (
          <Text type="secondary" style={{ fontStyle: "italic" }}>
            {val}
          </Text>
        ) : (
          <Text type="secondary">—</Text>
        ),
    },
    {
      title: t("contact.list.col_status"),
      key: "status",
      responsive: ["lg"],
      render: (_, record) =>
        record.is_archived ? (
          <Tag color="default">{t("common.archived")}</Tag>
        ) : (
          <Tag color="green">{t("common.active")}</Tag>
        ),
    },
    {
      title: t("contact.list.col_updated"),
      dataIndex: "updated_at",
      key: "updated_at",
      responsive: ["md"],
      render: (val: string) => (
        <Text type="secondary">{dayjs(val).format("MMM D, YYYY")}</Text>
      ),
      sorter: (a, b) =>
        dayjs(a.updated_at).unix() - dayjs(b.updated_at).unix(),
    },
  ];

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <div
        style={{
          marginBottom: 24,
          paddingBottom: 20,
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            flexWrap: "wrap",
            gap: 12,
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <div
              style={{
                width: 40,
                height: 40,
                borderRadius: 10,
                backgroundColor: token.colorPrimaryBg,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <TeamOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
            </div>
            <div>
              <Title level={4} style={{ margin: 0 }}>
                {t("contact.list.title")}
              </Title>
              {!isLoading && (
                <Text type="secondary" style={{ fontSize: 13 }}>
                  {t("contact.list.total", { count: data?.length ?? 0 })}
                </Text>
              )}
            </div>
          </div>
          <Space size={8} wrap>
            <Upload
              accept=".vcf"
              showUploadList={false}
              customRequest={async ({ file }) => {
                try {
                  const res = await vcardApi.importVCard(vaultId, file as File);
                  const count = (res.data.data as { count?: number })?.count ?? 0;
                  message.success(t("vcard.importSuccess", { count }));
                } catch {
                  message.error(t("contact.list.load_failed"));
                }
              }}
            >
              <Button size="small" icon={<UploadOutlined />}>{t("vcard.import")}</Button>
            </Upload>
            <Button
              size="small"
              icon={<DownloadOutlined />}
              onClick={async () => {
                try {
                  const res = await vcardApi.exportVault(vaultId);
                  const blob = new Blob([res.data as BlobPart]);
                  const url = URL.createObjectURL(blob);
                  const a = document.createElement("a");
                  a.href = url;
                  a.download = "contacts.vcf";
                  a.click();
                  URL.revokeObjectURL(url);
                } catch {
                  message.error(t("contact.list.load_failed"));
                }
              }}
            >
              {t("vcard.exportAll")}
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => navigate(`/vaults/${vaultId}/contacts/create`)}
            >
              {t("contact.list.add_contact")}
            </Button>
          </Space>
        </div>
      </div>

      <div style={{ marginBottom: 16 }}>
        <Input
          prefix={<SearchOutlined style={{ color: token.colorTextQuaternary }} />}
          placeholder={t("contact.list.search_placeholder")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          allowClear
          style={{
            maxWidth: 400,
            borderRadius: 20,
          }}
        />
      </div>

      <Table<Contact>
        columns={columns}
        dataSource={contacts}
        rowKey="id"
        loading={isLoading}
        onRow={(record) => ({
          onClick: () =>
            navigate(`/vaults/${vaultId}/contacts/${record.id}`),
          style: { cursor: "pointer" },
        })}
        style={{ borderRadius: token.borderRadius }}
        pagination={{
          pageSize: 20,
          showSizeChanger: false,
          showTotal: (total) => t("contact.list.total", { count: total }),
        }}
        locale={{
          emptyText: search ? t("contact.list.no_match") : t("contact.list.no_contacts"),
        }}
      />
    </div>
  );
}
