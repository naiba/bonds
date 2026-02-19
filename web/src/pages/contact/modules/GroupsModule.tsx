import { useState } from "react";
import {
  Card,
  Button,
  Typography,
  Space,
  Tag,
  App,
  Select,
  Form,
  Modal,
  Spin,
} from "antd";
import { PlusOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import type { APIError } from "@/api";

const { Title, Text } = Typography;

interface GroupsModuleProps {
  vaultId: string;
  contactId: string;
}

export default function GroupsModule({ vaultId, contactId }: GroupsModuleProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [form] = Form.useForm();

  const { data: contactGroups = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "groups"],
    queryFn: async () => {
      const res = await api.groups.contactsGroupsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const { data: allGroups = [] } = useQuery({
    queryKey: ["vaults", vaultId, "groups"],
    queryFn: async () => {
      const res = await api.groups.groupsList(String(vaultId));
      return res.data ?? [];
    },
    enabled: isModalOpen,
  });

  const addMutation = useMutation({
    mutationFn: (values: { group_id: number }) =>
      api.groups.contactsGroupsCreate(String(vaultId), String(contactId), values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId, "groups"],
      });
      message.success(t("contact.detail.groups.added"));
      setIsModalOpen(false);
      form.resetFields();
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const removeMutation = useMutation({
    mutationFn: (groupId: number) =>
      api.groups.contactsGroupsDelete(String(vaultId), String(contactId), groupId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId, "groups"],
      });
      message.success(t("contact.detail.groups.removed"));
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const availableGroups = allGroups.filter(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (g: any) => !contactGroups.some((cg: any) => cg.id === g.id)
  );

  return (
    <Card
      title={
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <Title level={5} style={{ margin: 0 }}>
            {t("contact.detail.groups.title")}
          </Title>
        </div>
      }
      extra={
        <Button
          type="primary"
          size="small"
          icon={<PlusOutlined />}
          onClick={() => setIsModalOpen(true)}
        >
          {t("common.add")}
        </Button>
      }
    >
      {isLoading ? (
        <div style={{ textAlign: "center", padding: 20 }}>
          <Spin />
        </div>
      ) : contactGroups.length === 0 ? (
        <Text type="secondary">{t("contact.detail.groups.no_groups")}</Text>
      ) : (
        <Space size={[8, 8]} wrap>
          {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
          {contactGroups.map((group: any) => (
            <Tag
              key={group.id}
              color="blue"
              closable
              onClose={(e) => {
                e.preventDefault();
                removeMutation.mutate(group.id);
              }}
              style={{
                margin: 0,
                fontSize: 14,
                padding: "4px 10px",
                borderRadius: 16,
                cursor: "pointer",
                display: "inline-flex",
                alignItems: "center",
                gap: 6,
              }}
              onClick={() => navigate(`/vaults/${vaultId}/groups/${group.id}`)}
            >
              {group.name}
            </Tag>
          ))}
        </Space>
      )}

      <Modal
        title={t("contact.detail.groups.add")}
        open={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        footer={null}
        destroyOnClose={true}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => addMutation.mutate(values)}
        >
          <Form.Item
            name="group_id"
            label={t("contact.detail.groups.select_group")}
            rules={[{ required: true }]}
          >
            <Select
              placeholder={t("contact.detail.groups.select_group")}
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              options={availableGroups.map((g: any) => ({
                label: g.name,
                value: g.id,
              }))}
              disabled={availableGroups.length === 0}
            />
          </Form.Item>
          {availableGroups.length === 0 && (
            <Text type="secondary" style={{ display: "block", marginBottom: 16 }}>
              {t("contact.detail.groups.no_groups")}
            </Text>
          )}
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => setIsModalOpen(false)}>{t("common.cancel")}</Button>
            <Button
              type="primary"
              htmlType="submit"
              loading={addMutation.isPending}
              disabled={availableGroups.length === 0}
            >
              {t("common.save")}
            </Button>
          </div>
        </Form>
      </Modal>
    </Card>
  );
}
