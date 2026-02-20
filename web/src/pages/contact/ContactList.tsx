import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Table, Button, Typography, Input, Tag, Space, App, Upload, theme, Select } from "antd";
import {
  PlusOutlined,
  SearchOutlined,
  StarFilled,
  DownloadOutlined,
  UploadOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation } from "@tanstack/react-query";
import { api } from "@/api";
import type { Contact, PaginationMeta, LabelResponse } from "@/api";
import type { ColumnsType } from "antd/es/table";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import ContactAvatar from "@/components/ContactAvatar";

const { Title, Text } = Typography;
const { Option } = Select;

// Map frontend sort values to backend sort param values
const SORT_MAP: Record<string, string> = {
  name: "first_name",
  updated_at: "updated_at",
};

export default function ContactList() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const vaultId = id!;
  const [search, setSearch] = useState("");
  const [sortBy, setSortBy] = useState<string>("name");
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [labelFilter, setLabelFilter] = useState<number | null>(null);
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const { data: labels = [] } = useQuery({
    queryKey: ["vault", vaultId, "labels"],
    queryFn: async () => (await api.vaultSettings.settingsLabelsList(String(vaultId))).data ?? [],
  });

  const { data: contactsResponse, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", labelFilter, currentPage, pageSize, sortBy, search],
    queryFn: async () => {
      if (labelFilter) {
        const res = await api.contacts.contactsLabelsDetail(String(vaultId), labelFilter, {
          page: currentPage,
          per_page: pageSize,
        });
        return {
          contacts: (res.data as { contacts?: Contact[] })?.contacts ?? [],
          meta: res.meta as PaginationMeta | undefined,
        };
      }
      const res = await api.contacts.contactsList(String(vaultId), {
        page: currentPage,
        per_page: pageSize,
        sort: SORT_MAP[sortBy] ?? "updated_at",
        ...(search.length > 2 ? { search } : {}),
      });
      return {
        contacts: res.data ?? [],
        meta: res.meta as PaginationMeta | undefined,
      };
    },
    enabled: !!vaultId,
    meta: {
      onError: () => message.error(t("contact.list.load_failed")),
    },
  });

  const contacts = contactsResponse?.contacts ?? [];
  const paginationMeta = contactsResponse?.meta;

  const sortMutation = useMutation({
      mutationFn: (data: { sort_by: string; sort_order: "asc" | "desc" }) => 
        api.contacts.contactsSortUpdate(String(vaultId), data),
      onSuccess: () => {
          message.success(t("contact.list.sort_updated"));
      }
  });

  const handleSortChange = (value: string) => {
      setSortBy(value);
      setCurrentPage(1);
      // Save user preference as side effect
      sortMutation.mutate({ sort_by: value, sort_order: "asc" });
  };

  const handleSearch = (val: string) => {
      setSearch(val);
      setCurrentPage(1);
  };

  const columns: ColumnsType<Contact> = [
    {
      title: t("contact.list.col_name"),
      key: "name",
      render: (_, record) => (
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <ContactAvatar
              vaultId={String(id)}
              contactId={record.id ?? ""}
              firstName={record.first_name}
              lastName={record.last_name}
              size={34}
              updatedAt={record.updated_at}
            />
            <span style={{ fontWeight: 500 }}>
              {record.first_name} {record.last_name}
            </span>
            {record.is_favorite && (
              <StarFilled style={{ color: token.colorWarning, fontSize: 13 }} />
            )}
          </div>
        ),
      sorter: (a, b) => (a.first_name ?? '').localeCompare(b.first_name ?? ''),
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
                  {t("contact.list.total", { count: paginationMeta?.total ?? contacts.length })}
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
                  const res = await api.vcard.contactsImportCreate(String(vaultId), { file: file as File });
                  const count = (res.data as { count?: number })?.count ?? 0;
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
                  const res = await api.vcard.contactsExportList(String(vaultId));
                  const blob = new Blob([res as BlobPart]);
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

      <div style={{ marginBottom: 16, display: "flex", gap: 16, flexWrap: "wrap" }}>
        <Input
          prefix={<SearchOutlined style={{ color: token.colorTextQuaternary }} />}
          placeholder={t("contact.list.quick_search")}
          value={search}
          onChange={(e) => handleSearch(e.target.value)}
          allowClear
          style={{
            maxWidth: 300,
            borderRadius: 20,
          }}
        />
        <Select
            placeholder={t("contact.list.sort_by")}
            value={sortBy}
            onChange={handleSortChange}
            style={{ width: 160 }}
        >
            <Option value="name">{t("contact.list.sort_name")}</Option>
            <Option value="updated_at">{t("contact.list.sort_updated")}</Option>
        </Select>
        <Select
            placeholder={t("contact.list.filter_label")}
            value={labelFilter}
            onChange={(v) => { setLabelFilter(v); setCurrentPage(1); }}
            style={{ width: 200 }}
            allowClear
        >
            <Option value={null}>{t("contact.list.all_labels")}</Option>
            {labels.map((l: LabelResponse) => (
                <Option key={l.id} value={l.id}>{l.name}</Option>
            ))}
        </Select>
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
          current: currentPage,
          pageSize: pageSize,
          total: paginationMeta?.total ?? contacts.length,
          onChange: (page, size) => { setCurrentPage(page); setPageSize(size); },
          showSizeChanger: true,
          showTotal: (total) => t("contact.list.total", { count: total }),
        }}
        locale={{
          emptyText: search ? t("contact.list.no_match") : t("contact.list.no_contacts"),
        }}
      />
    </div>
  );
}
