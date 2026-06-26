import { useState } from "react";
import { useParams, useNavigate, useSearchParams } from "react-router-dom";
import { Table, Button, Typography, Input, Tag, Space, App, Upload, theme, Select, Checkbox, Popover, Modal, Form } from "antd";
import {
  PlusOutlined,
  SearchOutlined,
  StarFilled,
  DownloadOutlined,
  UploadOutlined,
  TeamOutlined,
  SettingOutlined,
  ExportOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { APIError, Contact, Group, PaginationMeta, LabelResponse, Vault } from "@/api";
import { formatContactName, useVaultNameOrder } from "@/utils/nameFormat";
import { useDateFormat, formatDate } from "@/utils/dateFormat";
import { formatDateOnly } from "@/utils/dateOnlyInput";
import type { ColumnsType } from "antd/es/table";
import type { Breakpoint } from "antd";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import ContactAvatar from "@/components/ContactAvatar";

const { Title, Text } = Typography;
const { Option } = Select;

const SORT_MAP: Record<string, string> = {
  name: "first_name",
  first_met_at: "first_met_at",
  updated_at: "updated_at",
};

const COLUMNS_STORAGE_KEY = "bonds_contact_list_columns";
const DEFAULT_VISIBLE_COLUMNS = ["name", "nickname", "first_met_at", "status", "updated_at"];
const DEFAULT_PAGE = 1;
const DEFAULT_PAGE_SIZE = 20;
const PAGE_SIZE_OPTIONS = ["10", "20", "50", "100"];

type ContactGroupSummary = {
  id: number;
  name: string;
};

function parsePositiveInteger(value: string | null): number | null {
  if (!value) return null;
  const parsed = Number(value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null;
}

function parsePage(value: string | null): number {
  return parsePositiveInteger(value) ?? DEFAULT_PAGE;
}

function parsePageSize(value: string | null): number {
  const parsed = parsePositiveInteger(value);
  return parsed && PAGE_SIZE_OPTIONS.includes(String(parsed)) ? parsed : DEFAULT_PAGE_SIZE;
}

function buildPaginationSearch(params: URLSearchParams, page: number, pageSize: number): string {
  const next = new URLSearchParams(params);
  next.set("page", String(page));
  next.set("per_page", String(pageSize));
  return `?${next.toString()}`;
}

function loadVisibleColumns(): string[] {
  try {
    const saved = localStorage.getItem(COLUMNS_STORAGE_KEY);
    if (saved) {
      const parsed = JSON.parse(saved) as string[];
      if (Array.isArray(parsed) && parsed.length > 0) return parsed;
    }
  } catch { /* fallback */ }
  return DEFAULT_VISIBLE_COLUMNS;
}

export default function ContactList() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const vaultId = id!;
  const [search, setSearch] = useState("");
  const [sortBy, setSortBy] = useState<string>("name");
  const [labelFilter, setLabelFilter] = useState<number | null>(parsePositiveInteger(searchParams.get("label")));
  const [groupFilter, setGroupFilter] = useState<number | null>(parsePositiveInteger(searchParams.get("group")));
  const [statusFilter, setStatusFilter] = useState<string>("active");
  const [visibleColumns, setVisibleColumns] = useState<string[]>(loadVisibleColumns);
  const [selectedContactIds, setSelectedContactIds] = useState<string[]>([]);
  const [bulkMoveOpen, setBulkMoveOpen] = useState(false);
  const [bulkMoveForm] = Form.useForm<{ target_vault_id: string }>();
  const currentPage = parsePage(searchParams.get("page"));
  const pageSize = parsePageSize(searchParams.get("per_page"));
  const contactListSearch = buildPaginationSearch(searchParams, currentPage, pageSize);
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const nameOrder = useVaultNameOrder(vaultId);
  const dateFormats = useDateFormat();
  const { data: labels = [] } = useQuery({
    queryKey: ["vault", vaultId, "labels"],
    queryFn: async () => (await api.vaultSettings.settingsLabelsList(String(vaultId))).data ?? [],
  });

  const { data: groups = [] } = useQuery<Group[]>({
    queryKey: ["vault", vaultId, "groups"],
    queryFn: async () => (await api.groups.groupsList(String(vaultId))).data ?? [],
  });

  const { data: vaults = [] } = useQuery<Vault[]>({
    queryKey: ["vaults", "bulkMoveTargets"],
    queryFn: async () => (await api.vaults.vaultsList()).data ?? [],
    enabled: bulkMoveOpen,
  });

  const { data: contactsResponse, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", labelFilter, groupFilter, currentPage, pageSize, sortBy, search, statusFilter],
    queryFn: async () => {
      if (labelFilter) {
        const res = await api.contacts.contactsLabelsDetail(String(vaultId), labelFilter, {
          page: currentPage,
          per_page: pageSize,
          sort: SORT_MAP[sortBy] ?? "updated_at",
          filter: statusFilter,
        });
        return {
          contacts: res.data ?? [],
          meta: res.meta as PaginationMeta | undefined,
        };
      }
      if (groupFilter) {
        const res = await api.contacts.contactsList(String(vaultId), {
          per_page: 9999,
          sort: SORT_MAP[sortBy] ?? "updated_at",
          filter: statusFilter,
          ...(search.length > 2 ? { search } : {}),
        });
        const filtered = ((res.data ?? []) as (Contact & { groups?: ContactGroupSummary[] })[])
          .filter((c) => c.groups?.some((g) => g.id === groupFilter));
        const start = (currentPage - 1) * pageSize;
        return {
          contacts: filtered.slice(start, start + pageSize),
          meta: { ...(res.meta as PaginationMeta), total: filtered.length } as PaginationMeta,
        };
      }
      const res = await api.contacts.contactsList(String(vaultId), {
        page: currentPage,
        per_page: pageSize,
        sort: SORT_MAP[sortBy] ?? "updated_at",
        filter: statusFilter,
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
  const targetVaultOptions = (vaults as Vault[])
    .filter((vault) => vault.id && String(vault.id) !== String(vaultId))
    .map((vault) => ({ label: vault.name ?? vault.id!, value: String(vault.id) }));

  const updatePaginationParams = (page: number, size: number, replace = false) => {
    const nextPage = Number.isInteger(page) && page > 0 ? page : DEFAULT_PAGE;
    const nextPageSize = PAGE_SIZE_OPTIONS.includes(String(size)) ? size : DEFAULT_PAGE_SIZE;
    const nextParams = new URLSearchParams(searchParams);
    nextParams.set("page", String(nextPage));
    nextParams.set("per_page", String(nextPageSize));
    setSearchParams(nextParams, { replace });
  };

  const resetToFirstPage = () => updatePaginationParams(DEFAULT_PAGE, pageSize, true);

  const applyTagFilter = (kind: "label" | "group", value: number | null) => {
    const nextParams = new URLSearchParams(searchParams);
    nextParams.delete("label");
    nextParams.delete("group");
    if (kind === "label") {
      setLabelFilter(value);
      setGroupFilter(null);
      if (value) nextParams.set("label", String(value));
    } else {
      setGroupFilter(value);
      setLabelFilter(null);
      if (value) nextParams.set("group", String(value));
    }
    nextParams.set("page", String(DEFAULT_PAGE));
    nextParams.set("per_page", String(pageSize));
    setSearchParams(nextParams, { replace: true });
  };

  const sortMutation = useMutation({
      mutationFn: (data: { sort_by: string; sort_order: "asc" | "desc" }) => 
        api.contacts.contactsSortUpdate(String(vaultId), data),
      onSuccess: () => {
          message.success(t("contact.list.sort_updated"));
      }
  });

  const bulkMoveMutation = useMutation({
    mutationFn: (targetVaultId: string) => api.contacts.contactsBulkMoveCreate(String(vaultId), {
      contact_ids: selectedContactIds,
      target_vault_id: targetVaultId,
    }),
    onSuccess: (res) => {
      const result = res.data as { moved_count?: number } | undefined;
      const movedCount = result?.moved_count ?? selectedContactIds.length;
      setSelectedContactIds([]);
      setBulkMoveOpen(false);
      bulkMoveForm.resetFields();
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "contacts"] });
      message.success(t("contact.list.bulk_move_success", { count: movedCount }));
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const handleSortChange = (value: string) => {
      setSortBy(value);
      resetToFirstPage();
      // Save user preference as side effect
      sortMutation.mutate({ sort_by: value, sort_order: "asc" });
  };

  const handleSearch = (val: string) => {
      setSearch(val);
      resetToFirstPage();
  };

  const handleColumnToggle = (key: string, checked: boolean) => {
    const next = checked
      ? [...visibleColumns, key]
      : visibleColumns.filter((k) => k !== key);
    setVisibleColumns(next);
    localStorage.setItem(COLUMNS_STORAGE_KEY, JSON.stringify(next));
  };

  const allColumns: (ColumnsType<Contact>[number] & { key: string; alwaysVisible?: boolean; responsive?: Breakpoint[] })[] = [
    {
      title: t("contact.list.col_name"),
      key: "name",
      alwaysVisible: true,
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
              {formatContactName(nameOrder, record)}
            </span>
            {record.is_favorite && (
              <StarFilled style={{ color: token.colorWarning, fontSize: 13 }} />
            )}
            {record.needs_verification && (
              <Tag color="warning" style={{ marginInlineEnd: 0 }}>
                {t("contact.needs_verification.badge")}
              </Tag>
            )}
          </div>
        ),
      sorter: (a, b) => (a.first_name ?? '').localeCompare(b.first_name ?? ''),
    },
    {
      title: t("contact.list.col_nickname"),
      dataIndex: "nickname",
      key: "nickname",
      responsive: ["md"] as Breakpoint[],
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
      title: t("contact.list.col_birthday"),
      key: "birthday",
      responsive: ["md"] as Breakpoint[],
      render: (_, record) => {
        const bday = (record as Contact & { birthday?: string }).birthday;
        // 使用用户日期格式偏好格式化生日（fix #65）
        return bday ? (
          <Text type="secondary">{formatDate(bday, dateFormats)}</Text>
        ) : (
          <Text type="secondary">—</Text>
        );
      },
    },
    {
      title: t("contact.list.col_age"),
      key: "age",
      responsive: ["lg"] as Breakpoint[],
      render: (_, record) => {
        const age = (record as Contact & { age?: number }).age;
        return age != null ? (
          <Text type="secondary">{age}</Text>
        ) : (
          <Text type="secondary">—</Text>
        );
      },
    },
    {
      title: t("contact.list.col_groups"),
      key: "groups",
      responsive: ["lg"] as Breakpoint[],
      render: (_, record) => {
        const groups = (record as Contact & { groups?: { id: number; name: string }[] }).groups;
        return groups && groups.length > 0 ? (
          <Space size={4} wrap>
            {groups.map((g) => (
              <Tag key={g.id} color="blue">{g.name}</Tag>
            ))}
          </Space>
        ) : (
          <Text type="secondary">—</Text>
        );
      },
    },
    {
      title: t("contact.list.col_status"),
      key: "status",
      responsive: ["lg"] as Breakpoint[],
      render: (_, record) =>
        record.is_archived ? (
          <Tag color="default">{t("common.archived")}</Tag>
        ) : (
          <Tag color="green">{t("common.active")}</Tag>
        ),
    },
    {
      title: t("contact.list.col_first_met"),
      dataIndex: "first_met_at",
      key: "first_met_at",
      responsive: ["md"] as Breakpoint[],
      render: (val: string | undefined) => (
        val ? <Text type="secondary">{formatDateOnly(val, dateFormats)}</Text> : <Text type="secondary">—</Text>
      ),
      sorter: (a, b) =>
        dayjs(a.first_met_at ?? 0).unix() - dayjs(b.first_met_at ?? 0).unix(),
    },
    {
      title: t("contact.list.col_updated"),
      dataIndex: "updated_at",
      key: "updated_at",
      responsive: ["md"] as Breakpoint[],
      render: (val: string) => (
        <Text type="secondary">{formatDate(val, dateFormats)}</Text>
      ),
      sorter: (a, b) =>
        dayjs(a.updated_at).unix() - dayjs(b.updated_at).unix(),
    },
  ];

  const filteredColumns = allColumns.filter(
    (col) => col.alwaysVisible || visibleColumns.includes(col.key),
  );

  const toggleableColumns = allColumns.filter((col) => !col.alwaysVisible);

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
                borderRadius: 12,
                backgroundColor: token.colorPrimaryBg, boxShadow: `inset 0 0 0 1px ${token.colorPrimary}20`,
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
            {selectedContactIds.length > 0 && (
              <Button
                icon={<ExportOutlined />}
                onClick={() => setBulkMoveOpen(true)}
              >
                {t("contact.list.bulk_move_selected", { count: selectedContactIds.length })}
              </Button>
            )}
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
              <Button icon={<UploadOutlined />}>{t("vcard.import")}</Button>
            </Upload>
            <Button
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
          style={{ maxWidth: 300 }}
        />
        <Select
            data-testid="contact-sort-select"
            placeholder={t("contact.list.sort_by")}
            value={sortBy}
            onChange={handleSortChange}
            style={{ width: 160 }}
        >
            <Option value="name">{t("contact.list.sort_name")}</Option>
            <Option value="first_met_at">{t("contact.list.sort_first_met")}</Option>
            <Option value="updated_at">{t("contact.list.sort_updated")}</Option>
        </Select>
        <Popover
          trigger="click"
          placement="bottomRight"
          content={
            <div style={{ display: "flex", flexDirection: "column", gap: 8, minWidth: 160 }}>
              {toggleableColumns.map((col) => (
                <Checkbox
                  key={col.key}
                  checked={visibleColumns.includes(col.key)}
                  onChange={(e) => handleColumnToggle(col.key, e.target.checked)}
                >
                  {String(col.title)}
                </Checkbox>
              ))}
            </div>
          }
        >
          <Button icon={<SettingOutlined />}>{t("contact.list.columns")}</Button>
        </Popover>
        <Select
            data-testid="contact-label-filter"
            placeholder={t("contact.list.filter_label")}
            value={labelFilter}
            onChange={(v) => applyTagFilter("label", v ?? null)}
            style={{ width: 200 }}
            allowClear
        >
            <Option value={null}>{t("contact.list.all_labels")}</Option>
            {labels.map((l: LabelResponse) => (
                <Option key={l.id} value={l.id}>{l.name}</Option>
            ))}
        </Select>
        <Select
            data-testid="contact-group-filter"
            placeholder={t("contact.list.filter_group")}
            value={groupFilter}
            onChange={(v) => applyTagFilter("group", v ?? null)}
            style={{ width: 200 }}
            allowClear
        >
            <Option value={null}>{t("contact.list.all_groups")}</Option>
            {groups.map((g) => (
                <Option key={g.id} value={g.id}>{g.name}</Option>
            ))}
        </Select>
        <Select
            data-testid="status-filter"
            placeholder={t("contact.list.filter_status")}
            value={statusFilter}
            onChange={(v) => { setStatusFilter(v); resetToFirstPage(); }}
            style={{ width: 160 }}
        >
            <Option value="active">{t("contact.list.filter_active")}</Option>
            <Option value="favorites">{t("contact.list.filter_favorites")}</Option>
            <Option value="needs_verification">{t("contact.list.filter_needs_verification")}</Option>
            <Option value="archived">{t("contact.list.filter_archived")}</Option>
            <Option value="all">{t("contact.list.filter_all")}</Option>
        </Select>
      </div>

      <Table<Contact>
        columns={filteredColumns}
        dataSource={contacts}
        rowKey="id"
        rowSelection={{
          selectedRowKeys: selectedContactIds,
          preserveSelectedRowKeys: true,
          onChange: (keys) => setSelectedContactIds(keys.map(String)),
        }}
        loading={isLoading}
        onRow={(record) => ({
          onClick: (event) => {
            const target = event.target as HTMLElement;
            if (target.closest(".ant-checkbox")) return;
            navigate(`/vaults/${vaultId}/contacts/${record.id}${contactListSearch}`);
          },
          style: { cursor: "pointer" },
        })}
        style={{ borderRadius: token.borderRadius }}
        pagination={{
          current: currentPage,
          pageSize: pageSize,
          total: paginationMeta?.total ?? contacts.length,
          pageSizeOptions: PAGE_SIZE_OPTIONS,
          onChange: (page, size) => updatePaginationParams(page, size),
          showSizeChanger: true,
          showTotal: (total) => t("contact.list.total", { count: total }),
        }}
        locale={{
          emptyText: search ? t("contact.list.no_match") : t("contact.list.no_contacts"),
        }}
      />

      <Modal
        title={t("contact.list.bulk_move_title")}
        open={bulkMoveOpen}
        onCancel={() => {
          setBulkMoveOpen(false);
          bulkMoveForm.resetFields();
        }}
        footer={null}
        destroyOnHidden
      >
        <Form
          form={bulkMoveForm}
          layout="vertical"
          onFinish={(values) => bulkMoveMutation.mutate(values.target_vault_id)}
        >
          <Text type="secondary" style={{ display: "block", marginBottom: 16 }}>
            {t("contact.list.bulk_move_count", { count: selectedContactIds.length })}
          </Text>
          <Form.Item
            name="target_vault_id"
            label={t("contact.list.bulk_move_select_vault")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Select
              data-testid="bulk-move-vault-select"
              loading={bulkMoveOpen && !vaults.length}
              options={targetVaultOptions}
              placeholder={t("contact.list.bulk_move_select_vault")}
              notFoundContent={t("contact.list.bulk_move_no_targets")}
            />
          </Form.Item>
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => setBulkMoveOpen(false)}>{t("common.cancel")}</Button>
            <Button type="primary" htmlType="submit" loading={bulkMoveMutation.isPending} disabled={selectedContactIds.length === 0}>
              {t("contact.list.bulk_move")}
            </Button>
          </div>
        </Form>
      </Modal>
    </div>
  );
}
