import {
  Card,
  Typography,
  Table,
  Tag,
  Button,
  App,
  Popconfirm,
  Space,
  Spin,
  Segmented,
  InputNumber,
  Modal,
} from "antd";
import {
  CrownOutlined,
  DeleteOutlined,
  StopOutlined,
  CheckCircleOutlined,
  SettingOutlined,
  TeamOutlined,
  DatabaseOutlined,
  KeyOutlined,
  CloudOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { api } from "@/api";
import type { AdminUser, APIError } from "@/api";
import { useAuth } from "@/stores/auth";
import { filesize } from "filesize";
import type { ColumnsType } from "antd/es/table";
import { formatContactName, useNameOrder } from "@/utils/nameFormat";
import { useState } from "react";

const { Title, Text } = Typography;

export default function AdminUsers() {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const { user: currentUser } = useAuth();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const nameOrder = useNameOrder();
  const qk = ["admin", "users"];
  const [storageLimitModalUser, setStorageLimitModalUser] = useState<AdminUser | null>(null);
  const [storageLimitValue, setStorageLimitValue] = useState<number>(0);

  const { data: users = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.admin.usersList();
      return (res.data ?? []) as AdminUser[];
    },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, disabled }: { id: string; disabled: boolean }) =>
      api.admin.usersToggleUpdate(id, { disabled }),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(
        variables.disabled
          ? t("admin.users.disabled_success")
          : t("admin.users.enabled"),
      );
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const adminMutation = useMutation({
    mutationFn: ({
      id,
      is_instance_administrator,
    }: {
      id: string;
      is_instance_administrator: boolean;
    }) => api.admin.usersAdminUpdate(id, { is_instance_administrator }),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(
        variables.is_instance_administrator
          ? t("admin.users.admin_set")
          : t("admin.users.admin_removed"),
      );
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.admin.usersDelete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("admin.users.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const storageLimitMutation = useMutation({
    mutationFn: ({ id, storage_limit_in_mb }: { id: string; storage_limit_in_mb: number }) =>
      api.admin.usersStorageLimitUpdate(id, { storage_limit_in_mb }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("admin.users.storage_limit_updated"));
      setStorageLimitModalUser(null);
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const columns: ColumnsType<AdminUser> = [
    {
      title: t("admin.users.name"),
      key: "name",
      render: (_: unknown, record: AdminUser) =>
        formatContactName(nameOrder, record),
    },
    {
      title: t("admin.users.email"),
      dataIndex: "email",
      key: "email",
    },
    {
      title: t("admin.users.contacts"),
      dataIndex: "contact_count",
      key: "contact_count",
      width: 100,
      render: (v: number) => v ?? 0,
    },
    {
      title: t("admin.users.vaults"),
      dataIndex: "vault_count",
      key: "vault_count",
      width: 80,
      render: (v: number) => v ?? 0,
    },
    {
      title: t("admin.users.storage"),
      key: "storage",
      width: 160,
      render: (_: unknown, record: AdminUser) => {
        const used = record.storage_used ? (filesize(record.storage_used) as string) : "0 B";
        const limitMB = record.storage_limit_in_mb ?? 0;
        // storage_limit_in_mb=0 表示无限制
        const limitStr = limitMB > 0 ? (filesize(limitMB * 1024 * 1024) as string) : t("admin.users.unlimited");
        return (
          <Space direction="vertical" size={0}>
            <span>{used} / {limitStr}</span>
            <Button
              type="link"
              size="small"
              style={{ padding: 0, height: "auto" }}
              icon={<CloudOutlined />}
              onClick={() => {
                setStorageLimitModalUser(record);
                setStorageLimitValue(limitMB);
              }}
            >
              {t("admin.users.set_storage_limit")}
            </Button>
          </Space>
        );
      },
    },
    {
      title: t("admin.users.role"),
      key: "role",
      width: 100,
      render: (_: unknown, record: AdminUser) =>
        record.is_instance_administrator ? (
          <Tag color="gold" icon={<CrownOutlined />}>
            {t("admin.users.admin")}
          </Tag>
        ) : (
          <Tag>{t("admin.users.user")}</Tag>
        ),
    },
    {
      title: t("admin.users.status"),
      key: "status",
      width: 100,
      render: (_: unknown, record: AdminUser) =>
        record.disabled ? (
          <Tag color="error">{t("admin.users.disabled")}</Tag>
        ) : (
          <Tag color="success">{t("admin.users.active")}</Tag>
        ),
    },
    {
      title: t("admin.users.joined"),
      dataIndex: "created_at",
      key: "created_at",
      width: 160,
      render: (v: string) => (v ? new Date(v).toLocaleDateString() : "-"),
    },
    {
      title: t("admin.users.actions"),
      key: "actions",
      width: 240,
      render: (_: unknown, record: AdminUser) => {
        const isSelf = record.id === currentUser?.id;
        return (
          <Space size="small">
            {!isSelf && (
              <>
                <Button
                  type="text"
                  size="small"
                  icon={
                    record.disabled ? (
                      <CheckCircleOutlined />
                    ) : (
                      <StopOutlined />
                    )
                  }
                  onClick={() =>
                    toggleMutation.mutate({
                      id: record.id!,
                      disabled: !record.disabled,
                    })
                  }
                >
                  {record.disabled
                    ? t("admin.users.enable")
                    : t("admin.users.disable")}
                </Button>
                <Button
                  type="text"
                  size="small"
                  icon={<CrownOutlined />}
                  onClick={() =>
                    adminMutation.mutate({
                      id: record.id!,
                      is_instance_administrator:
                        !record.is_instance_administrator,
                    })
                  }
                >
                  {record.is_instance_administrator
                    ? t("admin.users.remove_admin")
                    : t("admin.users.set_admin")}
                </Button>
                <Popconfirm
                  title={t("admin.users.delete_confirm")}
                  onConfirm={() => deleteMutation.mutate(record.id!)}
                >
                  <Button
                    type="text"
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                  />
                </Popconfirm>
              </>
            )}
          </Space>
        );
      },
    },
  ];

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 1100, margin: "0 auto" }}>
      <Segmented
        value="users"
        onChange={(val) => {
          if (val === "settings") navigate("/admin/settings");
          if (val === "backups") navigate("/admin/backups");
          if (val === "oauth-providers") navigate("/admin/oauth-providers");
        }}
        options={[
          { label: t("admin.tab_users"), value: "users", icon: <TeamOutlined /> },
          { label: t("admin.tab_settings"), value: "settings", icon: <SettingOutlined /> },
          { label: t("admin.tab_backups"), value: "backups", icon: <DatabaseOutlined /> },
          { label: t("admin.tab_oauth"), value: "oauth-providers", icon: <KeyOutlined /> },
        ]}
        style={{ marginBottom: 24 }}
      />

      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ marginBottom: 4 }}>
          <CrownOutlined style={{ marginRight: 8 }} />
          {t("admin.users.title")}
        </Title>
        <Text type="secondary">{t("admin.users.description")}</Text>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={users}
          rowKey="id"
          pagination={false}
          size="small"
          scroll={{ x: 900 }}
        />
      </Card>

      <Modal
        title={t("admin.users.set_storage_limit")}
        open={!!storageLimitModalUser}
        onCancel={() => setStorageLimitModalUser(null)}
        onOk={() => {
          if (storageLimitModalUser?.id) {
            storageLimitMutation.mutate({
              id: storageLimitModalUser.id,
              storage_limit_in_mb: storageLimitValue,
            });
          }
        }}
        confirmLoading={storageLimitMutation.isPending}
      >
        <Typography.Paragraph type="secondary">
          {t("admin.users.storage_limit_hint")}
        </Typography.Paragraph>
        <InputNumber
          value={storageLimitValue}
          onChange={(v) => setStorageLimitValue(v ?? 0)}
          min={0}
          addonAfter="MB"
          style={{ width: "100%" }}
          placeholder={t("admin.users.storage_limit_placeholder")}
        />
      </Modal>
    </div>
  );
}
