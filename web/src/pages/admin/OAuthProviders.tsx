import { useState } from "react";
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
  Modal,
  Form,
  Input,
  Select,
  Switch,
} from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  KeyOutlined,
  TeamOutlined,
  SettingOutlined,
  DatabaseOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { httpClient } from "@/api";
import type { ColumnsType } from "antd/es/table";

const { Title, Text } = Typography;

interface OAuthProvider {
  id: number;
  type: string;
  name: string;
  client_id: string;
  has_secret: boolean;
  enabled: boolean;
  display_name: string;
  discovery_url: string;
  scopes: string;
}

interface OAuthProviderForm {
  type: string;
  name: string;
  client_id: string;
  client_secret?: string;
  display_name?: string;
  discovery_url?: string;
  scopes?: string;
  enabled: boolean;
}

const PROVIDER_TYPES = ["github", "google", "gitlab", "discord", "oidc"];

export default function AdminOAuthProviders() {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [form] = Form.useForm<OAuthProviderForm>();
  const [modalOpen, setModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const selectedType = Form.useWatch("type", form);
  const qk = ["admin", "oauth-providers"];

  const { data: providers = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await httpClient.instance.get<{ data: OAuthProvider[] }>(
        "/admin/oauth-providers",
      );
      return res.data.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: (data: OAuthProviderForm) =>
      httpClient.instance.post("/admin/oauth-providers", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("admin.oauth_providers.created"));
      closeModal();
    },
    onError: () => message.error("Failed to create provider"),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: OAuthProviderForm }) =>
      httpClient.instance.put(`/admin/oauth-providers/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("admin.oauth_providers.updated"));
      closeModal();
    },
    onError: () => message.error("Failed to update provider"),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) =>
      httpClient.instance.delete(`/admin/oauth-providers/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("admin.oauth_providers.deleted"));
    },
    onError: () => message.error("Failed to delete provider"),
  });

  function openCreate() {
    setEditingId(null);
    form.resetFields();
    form.setFieldsValue({ enabled: true });
    setModalOpen(true);
  }

  function openEdit(record: OAuthProvider) {
    setEditingId(record.id);
    form.setFieldsValue({
      type: record.type,
      name: record.name,
      client_id: record.client_id,
      client_secret: undefined,
      display_name: record.display_name,
      discovery_url: record.discovery_url,
      scopes: record.scopes,
      enabled: record.enabled,
    });
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditingId(null);
    form.resetFields();
  }

  function handleSubmit() {
    form.validateFields().then((values) => {
      if (editingId !== null) {
        updateMutation.mutate({ id: editingId, data: values });
      } else {
        createMutation.mutate(values);
      }
    });
  }

  function handleTypeChange(type: string) {
    if (type !== "oidc") {
      form.setFieldsValue({ name: type });
    }
  }

  function getTypeLabel(type: string): string {
    const key = `admin.oauth_providers.type_${type}`;
    const result = t(key);
    return result === key ? type : result;
  }

  const columns: ColumnsType<OAuthProvider> = [
    {
      title: t("admin.oauth_providers.name"),
      dataIndex: "name",
      key: "name",
    },
    {
      title: t("admin.oauth_providers.type"),
      dataIndex: "type",
      key: "type",
      width: 140,
      render: (type: string) => <Tag>{getTypeLabel(type)}</Tag>,
    },
    {
      title: t("admin.oauth_providers.client_id"),
      dataIndex: "client_id",
      key: "client_id",
      ellipsis: true,
    },
    {
      title: t("admin.oauth_providers.status"),
      key: "status",
      width: 100,
      render: (_: unknown, record: OAuthProvider) =>
        record.enabled ? (
          <Tag color="success">{t("admin.oauth_providers.active")}</Tag>
        ) : (
          <Tag color="default">{t("admin.oauth_providers.inactive")}</Tag>
        ),
    },
    {
      title: t("admin.oauth_providers.actions"),
      key: "actions",
      width: 160,
      render: (_: unknown, record: OAuthProvider) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openEdit(record)}
          >
            {t("common.edit")}
          </Button>
          <Popconfirm
            title={t("admin.oauth_providers.delete_confirm")}
            onConfirm={() => deleteMutation.mutate(record.id)}
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
    <div style={{ maxWidth: 1000, margin: "0 auto" }}>
      <Segmented
        value="oauth-providers"
        onChange={(val) => {
          if (val === "users") navigate("/admin/users");
          if (val === "settings") navigate("/admin/settings");
          if (val === "backups") navigate("/admin/backups");
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
            <KeyOutlined style={{ marginRight: 8 }} />
            {t("admin.oauth_providers.title")}
          </Title>
          <Text type="secondary">{t("admin.oauth_providers.description")}</Text>
        </div>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={openCreate}
          style={{ flexShrink: 0, marginTop: 4 }}
        >
          {t("admin.oauth_providers.add")}
        </Button>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={providers}
          rowKey="id"
          pagination={false}
          size="small"
          scroll={{ x: 700 }}
        />
      </Card>

      <Modal
        title={editingId !== null ? t("admin.oauth_providers.edit") : t("admin.oauth_providers.add")}
        open={modalOpen}
        onCancel={closeModal}
        onOk={handleSubmit}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="type"
            label={t("admin.oauth_providers.type")}
            rules={[{ required: true }]}
          >
            <Select
              placeholder={t("admin.oauth_providers.type")}
              onChange={handleTypeChange}
              disabled={editingId !== null}
            >
              {PROVIDER_TYPES.map((pt) => (
                <Select.Option key={pt} value={pt}>
                  {getTypeLabel(pt)}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="name"
            label={t("admin.oauth_providers.name")}
            rules={[{ required: true }]}
            tooltip={selectedType !== "oidc" ? t("admin.oauth_providers.name_auto") : undefined}
          >
            <Input
              placeholder={t("admin.oauth_providers.name")}
              disabled={selectedType !== "oidc" && editingId === null}
            />
          </Form.Item>

          <Form.Item
            name="client_id"
            label={t("admin.oauth_providers.client_id")}
            rules={[{ required: true }]}
          >
            <Input placeholder={t("admin.oauth_providers.client_id")} />
          </Form.Item>

          <Form.Item
            name="client_secret"
            label={t("admin.oauth_providers.client_secret")}
            rules={editingId === null ? [{ required: true }] : undefined}
          >
            <Input.Password
              placeholder={
                editingId !== null
                  ? "••••••••"
                  : t("admin.oauth_providers.client_secret")
              }
            />
          </Form.Item>

          <Form.Item
            name="display_name"
            label={t("admin.oauth_providers.display_name")}
          >
            <Input placeholder={t("admin.oauth_providers.display_name")} />
          </Form.Item>

          {selectedType === "oidc" && (
            <Form.Item
              name="discovery_url"
              label={t("admin.oauth_providers.discovery_url")}
              rules={[{ required: true }]}
              tooltip={t("admin.oauth_providers.discovery_url_hint")}
            >
              <Input placeholder="https://accounts.google.com/.well-known/openid-configuration" />
            </Form.Item>
          )}

          <Form.Item
            name="scopes"
            label={t("admin.oauth_providers.scopes")}
            tooltip={t("admin.oauth_providers.scopes_hint")}
          >
            <Input placeholder={t("admin.oauth_providers.scopes_hint")} />
          </Form.Item>

          <Form.Item
            name="enabled"
            label={t("admin.oauth_providers.enabled")}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
