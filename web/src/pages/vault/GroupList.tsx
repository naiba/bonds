import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Typography,
  Button,
  List,
  Modal,
  Form,
  Input,
  Popconfirm,
  App,
  Empty,
  Spin,
  Tag,
  theme,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Group, APIError } from "@/api";
import { useTranslation } from "react-i18next";

const { Title } = Typography;

export default function GroupList() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "groups"];

  const { data: groups = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.groups.groupsList(String(vaultId));
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  const createMutation = useMutation({
    mutationFn: (values: { name: string }) => api.groups.groupsCreate(vaultId, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      form.resetFields();
      message.success(t("vault.group_list.created_success"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (groupId: number) => api.groups.groupsDelete(String(vaultId), Number(groupId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("vault.group_list.deleted_success"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 24 }}>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <div
          style={{
            width: 32,
            height: 32,
            borderRadius: "50%",
            background: token.colorPrimaryBg,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          <TeamOutlined style={{ fontSize: 16, color: token.colorPrimary }} />
        </div>
        <Title level={4} style={{ margin: 0, flex: 1 }}>{t("vault.group_list.title")}</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          {t("vault.group_list.new_group")}
        </Button>
      </div>

      <div
        style={{
          background: token.colorBgContainer,
          borderRadius: token.borderRadiusLG,
          boxShadow: token.boxShadowTertiary,
          padding: "8px 0",
        }}
      >
        <List
          dataSource={groups}
          locale={{ emptyText: <Empty description={t("vault.group_list.no_groups")} style={{ padding: 32 }} /> }}
          renderItem={(group: Group) => (
            <List.Item
              style={{
                margin: "4px 16px",
                paddingLeft: 16,
                borderRadius: token.borderRadius,
                cursor: "pointer",
              }}
              actions={[
                <Popconfirm
                  key="d"
                  title={t("vault.group_list.delete_confirm")}
                  onConfirm={(e) => { e?.stopPropagation(); deleteMutation.mutate(group.id!); }}
                >
                  <Button
                    type="text"
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={(e) => e.stopPropagation()}
                  />
                </Popconfirm>,
              ]}
              onClick={() => navigate(`/vaults/${vaultId}/groups/${group.id}`)}
            >
              <List.Item.Meta
                avatar={
                  <div
                    style={{
                      width: 40,
                      height: 40,
                      borderRadius: "50%",
                      background: token.colorPrimaryBg,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                    }}
                  >
                    <TeamOutlined style={{ fontSize: 18, color: token.colorPrimary }} />
                  </div>
                }
                title={
                  <span style={{ fontWeight: 600 }}>
                    {group.name}
                  </span>
                }
                description={
                  <Tag
                    style={{
                      background: token.colorFillSecondary,
                      border: "none",
                      borderRadius: 12,
                      fontSize: 12,
                    }}
                  >
                    {t("vault.group_list.members_count", { count: group.contacts?.length ?? 0 })}
                  </Tag>
                }
              />
            </List.Item>
          )}
        />
      </div>

      <Modal
        title={t("vault.group_list.modal_title")}
        open={open}
        onCancel={() => { setOpen(false); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => createMutation.mutate(v)}>
          <Form.Item name="name" label={t("common.name")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
