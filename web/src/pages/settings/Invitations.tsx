import { useState } from "react";
import {
  Card,
  Typography,
  Button,
  Table,
  Modal,
  Form,
  Input,
  Select,
  Tag,
  Popconfirm,
  Spin,
  App,
} from "antd";
import { DeleteOutlined, SendOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { invitationsApi } from "@/api/invitations";
import type { InvitationType } from "@/types/invitation";
import type { APIError } from "@/types/api";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";

const { Title, Text } = Typography;

export default function Invitations() {
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["settings", "invitations"];

  const { data: invitations = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await invitationsApi.list();
      return (res.data.data ?? []) as InvitationType[];
    },
  });

  const createMutation = useMutation({
    mutationFn: (values: { email: string; permission: number }) =>
      invitationsApi.create(values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      form.resetFields();
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => invitationsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const permissionLabel = (perm: number) => {
    switch (perm) {
      case 100:
        return t("invitations.permission.manager");
      case 200:
        return t("invitations.permission.editor");
      case 300:
        return t("invitations.permission.viewer");
      default:
        return String(perm);
    }
  };

  const permissionColor = (perm: number) => {
    switch (perm) {
      case 100: return "blue";
      case 200: return "cyan";
      case 300: return "default";
      default: return "default";
    }
  };

  const columns: ColumnsType<InvitationType> = [
    {
      title: t("invitations.email"),
      dataIndex: "email",
      key: "email",
      render: (email: string) => (
        <Text strong>{email}</Text>
      ),
    },
    {
      title: t("invitations.permission"),
      dataIndex: "permission",
      key: "permission",
      render: (val: number) => (
        <Tag color={permissionColor(val)}>
          {permissionLabel(val)}
        </Tag>
      ),
    },
    {
      title: t("common.type"),
      key: "status",
      render: (_, record) =>
        record.status === "accepted" ? (
          <Tag color="green">{t("invitations.status.accepted")}</Tag>
        ) : (
          <Tag color="orange">{t("invitations.status.pending")}</Tag>
        ),
    },
    {
      title: t("common.created"),
      dataIndex: "created_at",
      key: "created_at",
      render: (val: string) => (
        <Text type="secondary">{dayjs(val).format("MMM D, YYYY")}</Text>
      ),
    },
    {
      title: "",
      key: "actions",
      render: (_, record) =>
        record.status === "pending" ? (
          <Popconfirm
            title={t("invitations.deleteConfirm")}
            onConfirm={() => deleteMutation.mutate(record.id)}
          >
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
            />
          </Popconfirm>
        ) : null,
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
            {t("invitations.title")}
          </Title>
          <Text type="secondary">
            {t("invitations.description")}
          </Text>
        </div>
        <Button
          type="primary"
          icon={<SendOutlined />}
          onClick={() => setOpen(true)}
          style={{ flexShrink: 0, marginTop: 4 }}
        >
          {t("invitations.invite")}
        </Button>
      </div>

      <Card>
        <Table<InvitationType>
          columns={columns}
          dataSource={invitations}
          rowKey="id"
          pagination={false}
        />
      </Card>

      <Modal
        title={t("invitations.invite")}
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
          onFinish={(v) => createMutation.mutate(v)}
        >
          <Form.Item
            name="email"
            label={t("invitations.email")}
            rules={[{ required: true, type: "email" }]}
          >
            <Input placeholder={t("invitations.email")} />
          </Form.Item>
          <Form.Item
            name="permission"
            label={t("invitations.permission")}
            rules={[{ required: true }]}
          >
            <Select
              options={[
                {
                  value: 100,
                  label: t("invitations.permission.manager"),
                },
                {
                  value: 200,
                  label: t("invitations.permission.editor"),
                },
                {
                  value: 300,
                  label: t("invitations.permission.viewer"),
                },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
