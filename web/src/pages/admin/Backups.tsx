import {
  Card,
  Typography,
  Button,
  Table,
  App,
  Popconfirm,
  Descriptions,
  Tag,
  Space,
  Spin,
  Segmented,
} from "antd";
import {
  PlusOutlined,
  DownloadOutlined,
  DeleteOutlined,
  UndoOutlined,
  DatabaseOutlined,
  TeamOutlined,
  SettingOutlined,
  KeyOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { httpClient } from "@/api";
import { filesize } from "filesize";
import type { ColumnsType } from "antd/es/table";

const { Title, Text } = Typography;

interface BackupItem {
  filename: string;
  size: number;
  created_at: string;
}

interface BackupConfig {
  cron_enabled: boolean;
  cron_spec: string;
  retention_days: number;
  backup_dir: string;
  db_driver: string;
}

export default function AdminBackups() {
  const { t } = useTranslation();
  const { message, modal } = App.useApp();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const qk = ["settings", "backups"];

  const { data: backups = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await httpClient.instance.get<{ data: BackupItem[] }>(
        "/settings/backups",
      );
      return res.data.data ?? [];
    },
  });

  const { data: backupConfig } = useQuery({
    queryKey: ["settings", "backups", "config"],
    queryFn: async () => {
      const res = await httpClient.instance.get<{ data: BackupConfig }>(
        "/settings/backups/config",
      );
      return res.data.data;
    },
  });

  const createMutation = useMutation({
    mutationFn: () => httpClient.instance.post("/settings/backups"),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("backups.created"));
    },
    onError: () => message.error(t("backups.creating")),
  });

  const deleteMutation = useMutation({
    mutationFn: (filename: string) =>
      httpClient.instance.delete(`/settings/backups/${filename}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("backups.deleted"));
    },
    onError: () => message.error("Failed to delete backup"),
  });

  const restoreMutation = useMutation({
    mutationFn: (filename: string) =>
      httpClient.instance.post(`/settings/backups/${filename}/restore`),
    onSuccess: () => {
      message.success(t("backups.restored"));
    },
    onError: () => message.error("Failed to restore backup"),
  });

  function handleDownload(filename: string) {
    const token = localStorage.getItem("token");
    const url = `/api/settings/backups/${filename}/download?token=${token}`;
    window.open(url, "_blank");
  }

  function handleRestore(filename: string) {
    modal.confirm({
      title: t("backups.restore"),
      content: (
        <div>
          <p>{t("backups.restore_confirm")}</p>
          <p style={{ color: "red", fontWeight: "bold" }}>
            {t("backups.restore_warning")}
          </p>
        </div>
      ),
      okButtonProps: { danger: true },
      onOk: () => restoreMutation.mutateAsync(filename),
    });
  }

  const columns: ColumnsType<BackupItem> = [
    {
      title: t("backups.filename"),
      dataIndex: "filename",
      key: "filename",
      ellipsis: true,
    },
    {
      title: t("backups.size"),
      dataIndex: "size",
      key: "size",
      width: 120,
      render: (size: number) => filesize(size) as string,
    },
    {
      title: t("backups.created_at"),
      dataIndex: "created_at",
      key: "created_at",
      width: 200,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: t("backups.actions"),
      key: "actions",
      width: 200,
      render: (_: unknown, record: BackupItem) => (
        <Space>
          <Button
            type="text"
            size="small"
            icon={<DownloadOutlined />}
            onClick={() => handleDownload(record.filename)}
          >
            {t("backups.download")}
          </Button>
          <Button
            type="text"
            size="small"
            icon={<UndoOutlined />}
            onClick={() => handleRestore(record.filename)}
            loading={
              restoreMutation.isPending &&
              restoreMutation.variables === record.filename
            }
          >
            {t("backups.restore")}
          </Button>
          <Popconfirm
            title={t("backups.delete_confirm")}
            onConfirm={() => deleteMutation.mutate(record.filename)}
          >
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
            />
          </Popconfirm>
        </Space>
      ),
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
    <div style={{ maxWidth: 800, margin: "0 auto" }}>
      <Segmented
        value="backups"
        onChange={(val) => {
          if (val === "users") navigate("/admin/users");
          if (val === "settings") navigate("/admin/settings");
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

      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          marginBottom: 24,
        }}
      >
        <div>
          <Title level={4} style={{ marginBottom: 4 }}>
            {t("backups.title")}
          </Title>
          <Text type="secondary">{t("backups.description")}</Text>
        </div>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => createMutation.mutate()}
          loading={createMutation.isPending}
          style={{ flexShrink: 0, marginTop: 4 }}
        >
          {createMutation.isPending
            ? t("backups.creating")
            : t("backups.create")}
        </Button>
      </div>

      {backupConfig && (
        <Card
          title={t("backups.config")}
          style={{ marginBottom: 24 }}
          size="small"
        >
          <Descriptions column={2} size="small">
            <Descriptions.Item label={t("backups.cron_enabled")}>
              {backupConfig.cron_enabled ? (
                <Tag color="success">{t("backups.cron_enabled_label")}</Tag>
              ) : (
                <Tag>{t("backups.cron_disabled")}</Tag>
              )}
            </Descriptions.Item>
            {backupConfig.cron_enabled && (
              <Descriptions.Item label={t("backups.cron_schedule")}>
                <code>{backupConfig.cron_spec}</code>
              </Descriptions.Item>
            )}
            <Descriptions.Item label={t("backups.retention")}>
              {t("backups.retention_days", {
                days: backupConfig.retention_days,
              })}
            </Descriptions.Item>
            <Descriptions.Item label={t("backups.db_driver")}>
              <Tag color="blue">{backupConfig.db_driver}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t("backups.backup_dir")}>
              <code>{backupConfig.backup_dir}</code>
            </Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      <Card>
        <Table
          columns={columns}
          dataSource={backups}
          rowKey="filename"
          pagination={false}
          locale={{ emptyText: t("backups.no_backups") }}
          size="small"
        />
      </Card>
    </div>
  );
}
