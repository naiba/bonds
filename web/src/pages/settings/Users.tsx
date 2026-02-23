import { useState } from "react";
import {
  Card,
  Typography,
  Table,
  Tag,
  Spin,
  Empty,
  Button,
  Modal,
  Form,
  Input,
  Switch,
  Popconfirm,
  Space,
  App,
  theme,
} from "antd";
import {
  CrownOutlined,
  EditOutlined,
  DeleteOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { User, APIError } from "@/api";
import type { ColumnsType } from "antd/es/table";
import { useAuth } from "@/stores/auth";
import dayjs from "dayjs";
import { formatContactName, formatContactInitials, useNameOrder } from "@/utils/nameFormat";

const { Title, Text } = Typography;

const avatarColors = [
  "#5b8c5a", "#e8864b", "#5b7fb5", "#c75d8a",
  "#7b6bb5", "#d4a853", "#4ba8b5", "#b55b5b",
];
function getAvatarColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return avatarColors[Math.abs(hash) % avatarColors.length];
}

export default function Users() {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const nameOrder = useNameOrder();
  const { user: currentUser } = useAuth();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const qk = ["settings", "users"];

  const { data: users = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.users.usersList();
      return res.data ?? [];
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...values }: {
      id: string;
      first_name?: string;
      last_name?: string;
      is_admin?: boolean;
    }) => api.users.usersUpdate(id, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("settings.users.updated"));
      setOpen(false);
      setEditing(null);
      form.resetFields();
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.users.usersDelete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("settings.users.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const isAdmin = currentUser?.is_admin === true;

  const openEdit = (record: User) => {
    setEditing(record);
    form.setFieldsValue({
      first_name: record.first_name,
      last_name: record.last_name,
      is_admin: record.is_admin ?? false,
    });
    setOpen(true);
  };

  const handleSubmit = () => {
    form.validateFields().then((values) => {
      if (editing) {
        updateMutation.mutate({ id: editing.id!, ...values });
      }
    });
  };

  const columns: ColumnsType<User> = [
    {
      title: t("settings.users.col_name"),
      key: "name",
      render: (_, record) => {
        const fullName = formatContactName(nameOrder, record);
        const initials = formatContactInitials(nameOrder, record);
        const color = getAvatarColor(fullName || "U");
        return (
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <div
              style={{
                width: 32,
                height: 32,
                borderRadius: "50%",
                background: color,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                flexShrink: 0,
                color: "#fff",
                fontSize: 13,
                fontWeight: 600,
              }}
            >
              {initials || "U"}
            </div>
            <span style={{ fontWeight: 500 }}>{fullName || record.email}</span>
          </div>
        );
      },
    },
    {
      title: t("settings.users.col_email"),
      dataIndex: "email",
      key: "email",
      render: (email: string) => (
        <Text type="secondary">{email}</Text>
      ),
    },
    {
      title: t("settings.users.col_role"),
      key: "role",
      render: (_, record) =>
        record.is_admin ? (
          <Tag color="blue" icon={<CrownOutlined />}>
            {t("settings.users.role_admin")}
          </Tag>
        ) : (
          <Tag color={token.colorBgLayout}>
            {t("settings.users.role_user")}
          </Tag>
        ),
    },
    {
      title: t("settings.users.col_joined"),
      dataIndex: "created_at",
      key: "created_at",
      render: (date: string) => (
        <Text type="secondary">{dayjs(date).format("MMM D, YYYY")}</Text>
      ),
    },
    ...(isAdmin
      ? [
          {
            title: "",
            key: "actions",
            width: 100,
            render: (_: unknown, record: User) => {
              const isSelf = record.id === currentUser?.id;
              return (
                <Space size="small">
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    onClick={() => openEdit(record)}
                  />
                  {!isSelf && (
                    <Popconfirm
                      title={t("settings.users.delete_confirm")}
                      onConfirm={() => deleteMutation.mutate(record.id!)}
                      okButtonProps={{ danger: true }}
                    >
                      <Button
                        type="text"
                        size="small"
                        danger
                        icon={<DeleteOutlined />}
                      />
                    </Popconfirm>
                  )}
                </Space>
              );
            },
          } as ColumnsType<User>[number],
        ]
      : []),
  ];

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ marginBottom: 4 }}>
          {t("settings.users.title")}
        </Title>
        <Text type="secondary">{t("settings.users.description")}</Text>
      </div>

      <Card>
        {isLoading ? (
          <Spin />
        ) : users.length === 0 ? (
          <Empty description={t("settings.users.no_users")} />
        ) : (
          <Table
            dataSource={users}
            columns={columns}
            rowKey="id"
            pagination={false}
          />
        )}
      </Card>

      <Modal
        title={t("settings.users.modal_title_edit")}
        open={open}
        onCancel={() => {
          setOpen(false);
          setEditing(null);
          form.resetFields();
        }}
        onOk={handleSubmit}
        confirmLoading={updateMutation.isPending}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="first_name"
            label={t("settings.users.first_name")}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="last_name"
            label={t("settings.users.last_name")}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="is_admin"
            label={t("settings.users.is_admin")}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
