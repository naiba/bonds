import { useState } from "react";
import {
  Card,
  Typography,
  Button,
  Table,
  Modal,
  Form,
  Input,
  DatePicker,
  Popconfirm,
  Spin,
  App,
  Alert,
  Select,
  Tag,
} from "antd";
import { DeleteOutlined, PlusOutlined, CopyOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { httpClient } from "@/api";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { useDateFormat, formatDate, formatDateTime } from "@/utils/dateFormat";

const { Title, Text } = Typography;

interface ApiToken {
  id: number;
  name: string;
  token_hint: string;
  scopes: string[];
  expires_at: string | null;
  last_used_at: string | null;
  created_at: string;
}

const SCOPE_FULL = "full";

export default function ApiTokens() {
  const [open, setOpen] = useState(false);
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const dateFormats = useDateFormat();
  const qk = ["settings", "tokens"];

  const { data: tokens = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await httpClient.instance.get<{ data: ApiToken[] }>(
        "/settings/tokens",
      );
      return res.data.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: async (values: {
      name: string;
      expires_at?: string;
      scopes: string[];
    }) => {
      const payload: { name: string; expires_at?: string; scopes: string[] } = {
        name: values.name,
        scopes: values.scopes,
      };
      if (values.expires_at) {
        payload.expires_at = values.expires_at;
      }
      const res = await httpClient.instance.post<{
        data: { token: string };
        error?: { message: string };
      }>("/settings/tokens", payload);
      return res.data;
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      form.resetFields();
      setCreatedToken(res.data.token);
    },
    onError: (e: { message: string }) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      await httpClient.instance.delete(`/settings/tokens/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
    },
    onError: (e: { message: string }) => message.error(e.message),
  });

  const handleCopy = async (token: string) => {
    await navigator.clipboard.writeText(token);
    message.success(t("api_tokens.copied"));
  };

  const columns: ColumnsType<ApiToken> = [
    {
      title: t("api_tokens.name"),
      dataIndex: "name",
      key: "name",
      render: (name: string) => <Text strong>{name}</Text>,
    },
    {
      title: t("api_tokens.token_hint"),
      dataIndex: "token_hint",
      key: "token_hint",
      render: (hint: string) => (
        <Text code style={{ fontSize: 12 }}>
          {hint}
        </Text>
      ),
    },
    {
      title: t("api_tokens.scopes"),
      dataIndex: "scopes",
      key: "scopes",
      render: (scopes: string[]) =>
        scopes && scopes.length > 0 ? (
          scopes.map((s) => (
            <Tag key={s} color="blue">
              {s}
            </Tag>
          ))
        ) : (
          <Tag>{t("api_tokens.scope_full")}</Tag>
        ),
    },
    {
      title: t("api_tokens.expires_at"),
      dataIndex: "expires_at",
      key: "expires_at",
      render: (val: string | null) =>
        val ? (
          <Text type="secondary">{formatDate(val, dateFormats)}</Text>
        ) : (
          <Text type="secondary">{t("api_tokens.no_expiry")}</Text>
        ),
    },
    {
      title: t("api_tokens.last_used"),
      dataIndex: "last_used_at",
      key: "last_used_at",
      render: (val: string | null) =>
        val ? (
          <Text type="secondary">{formatDateTime(val, dateFormats)}</Text>
        ) : (
          <Text type="secondary">{t("api_tokens.never_used")}</Text>
        ),
    },
    {
      title: t("common.created"),
      dataIndex: "created_at",
      key: "created_at",
      render: (val: string) => (
        <Text type="secondary">{formatDate(val, dateFormats)}</Text>
      ),
    },
    {
      title: "",
      key: "actions",
      render: (_, record) => (
        <Popconfirm
          title={t("api_tokens.delete_confirm")}
          onConfirm={() => deleteMutation.mutate(record.id)}
        >
          <Button
            type="text"
            size="small"
            danger
            icon={<DeleteOutlined />}
          />
        </Popconfirm>
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
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
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
            {t("api_tokens.title")}
          </Title>
          <Text type="secondary">{t("api_tokens.description")}</Text>
        </div>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setOpen(true)}
          style={{ flexShrink: 0, marginTop: 4 }}
        >
          {t("api_tokens.create")}
        </Button>
      </div>

      <Card>
        <Table<ApiToken>
          columns={columns}
          dataSource={tokens}
          rowKey="id"
          pagination={false}
        />
      </Card>

      <Card
        title={t("api_tokens.calendar_feed_title")}
        size="small"
        style={{ marginTop: 24 }}
      >
        <Text type="secondary" style={{ display: "block", marginBottom: 8 }}>
          {t("api_tokens.calendar_feed_help")}
        </Text>
        <Typography.Paragraph
          copyable={{ tooltips: [t("common.copy"), t("api_tokens.copied")] }}
          style={{
            margin: 0,
            background: "rgba(0,0,0,0.04)",
            padding: "6px 12px",
            borderRadius: 6,
            fontFamily: "monospace",
            fontSize: 13,
          }}
        >
          {`${window.location.origin}/api/vaults/{vault_id}/calendar.ics?token={token}`}
        </Typography.Paragraph>
      </Card>

      <Modal
        title={t("api_tokens.create")}
        open={open}
        onCancel={() => {
          setOpen(false);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ scope: SCOPE_FULL }}
          onFinish={(v) => {
            const scope = (v.scope as string) ?? SCOPE_FULL;
            const values = {
              name: v.name as string,
              scopes: scope === SCOPE_FULL ? [] : [scope],
              expires_at: v.expires_at
                ? (v.expires_at as dayjs.Dayjs).toISOString()
                : undefined,
            };
            createMutation.mutate(values);
          }}
        >
          <Form.Item
            name="name"
            label={t("api_tokens.name")}
            rules={[{ required: true }]}
          >
            <Input placeholder={t("api_tokens.name_placeholder")} />
          </Form.Item>
          <Form.Item
            name="scope"
            label={t("api_tokens.scopes")}
            extra={t("api_tokens.scope_help")}
          >
            <Select
              options={[
                { value: SCOPE_FULL, label: t("api_tokens.scope_full") },
                {
                  value: "calendar:read",
                  label: t("api_tokens.scope_calendar_read"),
                },
              ]}
            />
          </Form.Item>
          <Form.Item name="expires_at" label={t("api_tokens.expires_at")}>
            <DatePicker
              style={{ width: "100%" }}
              disabledDate={(current) =>
                current && current < dayjs().startOf("day")
              }
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t("api_tokens.created_title")}
        open={!!createdToken}
        onCancel={() => setCreatedToken(null)}
        footer={[
          <Button key="close" onClick={() => setCreatedToken(null)}>
            {t("common.cancel")}
          </Button>,
          <Button
            key="copy"
            type="primary"
            icon={<CopyOutlined />}
            onClick={() => createdToken && handleCopy(createdToken)}
          >
            {t("common.copy")}
          </Button>,
        ]}
      >
        <Alert
          type="warning"
          message={t("api_tokens.created_warning")}
          style={{ marginBottom: 16 }}
        />
        <Input.TextArea
          value={createdToken ?? ""}
          readOnly
          autoSize
          style={{ fontFamily: "monospace", fontSize: 13 }}
        />
      </Modal>
    </div>
  );
}
