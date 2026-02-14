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
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { groupsApi } from "@/api/groups";
import type { Group } from "@/types/modules";
import type { APIError } from "@/types/api";
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
  const qk = ["vaults", vaultId, "groups"];

  const { data: groups = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await groupsApi.list(vaultId);
      return res.data.data ?? [];
    },
    enabled: !!vaultId,
  });

  const createMutation = useMutation({
    mutationFn: (values: { name: string }) => groupsApi.create(vaultId, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      form.resetFields();
      message.success(t("vault.group_list.created_success"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (groupId: number) => groupsApi.delete(vaultId, groupId),
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
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}`)}
        style={{ marginBottom: 16 }}
      >
        {t("vault.group_list.back")}
      </Button>

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>{t("vault.group_list.title")}</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          {t("vault.group_list.new_group")}
        </Button>
      </div>

      <List
        dataSource={groups}
        locale={{ emptyText: <Empty description={t("vault.group_list.no_groups")} /> }}
        renderItem={(group: Group) => (
          <List.Item
            actions={[
              <Popconfirm
                key="d"
                title={t("vault.group_list.delete_confirm")}
                onConfirm={() => deleteMutation.mutate(group.id)}
              >
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={<TeamOutlined style={{ fontSize: 20 }} />}
              title={
                <a onClick={() => navigate(`/vaults/${vaultId}/groups/${group.id}`)}>
                  {group.name}
                </a>
              }
              description={
                <Tag>{t("vault.group_list.members_count", { count: group.contacts?.length ?? 0 })}</Tag>
              }
            />
          </List.Item>
        )}
      />

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
