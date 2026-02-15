import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Table, Button, Typography, Input, Tag, Space, App, Upload, theme } from "antd";
import {
  PlusOutlined,
  SearchOutlined,
  StarFilled,
  DownloadOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { contactsApi } from "@/api/contacts";
import { vcardApi } from "@/api/vcard";
import type { Contact } from "@/types/contact";
import type { ColumnsType } from "antd/es/table";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title } = Typography;

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

  const contacts = (data ?? []).filter((c) => {
    if (!search) return true;
    const q = search.toLowerCase();
    return (
      c.first_name.toLowerCase().includes(q) ||
      c.last_name.toLowerCase().includes(q) ||
      c.nickname?.toLowerCase().includes(q)
    );
  });

  const columns: ColumnsType<Contact> = [
    {
      title: t("contact.list.col_name"),
      key: "name",
      render: (_, record) => (
        <Space>
          <span>
            {record.first_name} {record.last_name}
          </span>
          {record.is_favorite && <StarFilled style={{ color: token.colorWarning }} />}
        </Space>
      ),
      sorter: (a, b) => a.first_name.localeCompare(b.first_name),
    },
    {
      title: t("contact.list.col_nickname"),
      dataIndex: "nickname",
      key: "nickname",
      responsive: ["md"],
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
      render: (val: string) => dayjs(val).format("MMM D, YYYY"),
      sorter: (a, b) =>
        dayjs(a.updated_at).unix() - dayjs(b.updated_at).unix(),
    },
  ];

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 16,
        }}
      >
        <Title level={4} style={{ margin: 0 }}>
          {t("contact.list.title")}
        </Title>
        <Space>
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
            <Button icon={<UploadOutlined />}>{t("vcard.import")}</Button>
          </Upload>
          <Button
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

      <div style={{ marginBottom: 16 }}>
        <Input
          prefix={<SearchOutlined />}
          placeholder={t("contact.list.search_placeholder")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          allowClear
          style={{ maxWidth: 320 }}
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
