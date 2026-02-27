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
  Spin
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import { useTranslation } from "react-i18next";
import type { APIError } from "@/api";
import { Link } from "react-router-dom";

const { Title, Text } = Typography;

interface LabelsModuleProps {
  vaultId: string;
  contactId: string;
}

export default function LabelsModule({ vaultId, contactId }: LabelsModuleProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingLabel, setEditingLabel] = useState<{ id: number; label_id: number } | null>(null);
  const [form] = Form.useForm();

  const { data: labels = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "labels"],
    queryFn: async () => {
      const res = await api.contactLabels.contactsLabelsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const { data: allLabels = [] } = useQuery({
    queryKey: ["vaults", vaultId, "labels"],
    queryFn: async () => {
      const res = await api.vaultSettings.settingsLabelsList(String(vaultId));
      return res.data ?? [];
    },
    enabled: isModalOpen || editingLabel !== null,
  });

  const addMutation = useMutation({
    mutationFn: (values: { label_id: number }) =>
      api.contactLabels.contactsLabelsCreate(String(vaultId), String(contactId), values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId, "labels"],
      });
      message.success(t("contact.detail.labels.added"));
      setIsModalOpen(false);
      form.resetFields();
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const updateMutation = useMutation({
    mutationFn: (values: { id: number; label_id: number }) =>
      api.contactLabels.contactsLabelsUpdate(String(vaultId), String(contactId), values.id, { label_id: values.label_id }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId, "labels"],
      });
      message.success(t("contact.detail.labels.updated"));
      setEditingLabel(null);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const removeMutation = useMutation({
    mutationFn: (labelId: number) =>
      api.contactLabels.contactsLabelsDelete(String(vaultId), String(contactId), labelId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId, "labels"],
      });
      message.success(t("contact.detail.labels.removed"));
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const availableLabels = allLabels.filter(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (l: any) => !labels.some((cl: any) => cl.label_id === l.id)
  );

  const editAvailableLabels = editingLabel
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ? allLabels.filter((l: any) => l.id !== editingLabel.label_id && !labels.some((cl: any) => cl.label_id === l.id))
    : [];

  return (
    <Card
      title={
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <Title level={5} style={{ margin: 0 }}>
            {t("contact.detail.labels.title")}
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
      ) : labels.length === 0 ? (
        <div>
          <Text type="secondary">{t("contact.detail.labels.no_labels")}</Text>
          <div style={{ marginTop: 8 }}>
            <Link to={`/vaults/${vaultId}/settings`} style={{ fontSize: 12 }}>
              {t("contact.detail.manage_labels_hint")}
            </Link>
          </div>
        </div>
      ) : (
        <Space size={[8, 8]} wrap>
          {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
          {labels.map((label: any) => (
            <Tag
              key={label.id}
              color={label.bg_color || "default"}
              style={{
                margin: 0,
                color: label.text_color,
                fontSize: 14,
                padding: "4px 10px",
                borderRadius: 16,
                display: "inline-flex",
                alignItems: "center",
                gap: 6,
              }}
            >
              {label.name}
              <EditOutlined
                style={{ cursor: "pointer", opacity: 0.6 }}
                onClick={() => setEditingLabel({ id: label.id, label_id: label.label_id })}
              />
              <DeleteOutlined
                style={{ cursor: "pointer", opacity: 0.6 }}
                onClick={() => removeMutation.mutate(label.id)}
              />
            </Tag>
          ))}
        </Space>
      )}

      <Modal
        title={t("contact.detail.labels.add")}
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
            name="label_id"
            label={t("contact.detail.labels.select_placeholder")}
            rules={[{ required: true }]}
          >
            <Select
              placeholder={t("contact.detail.labels.select_placeholder")}
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              options={availableLabels.map((l: any) => ({
                label: l.name,
                value: l.id,
              }))}
              disabled={availableLabels.length === 0}
            />
          </Form.Item>
          {availableLabels.length === 0 && (
            <div style={{ marginBottom: 16 }}>
              <Text type="secondary" style={{ display: "block" }}>
                {t("contact.detail.labels.no_labels_available")}
              </Text>
              <Link to={`/vaults/${vaultId}/settings`} style={{ fontSize: 12 }}>
                {t("contact.detail.manage_labels_hint")}
              </Link>
            </div>
          )}
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => setIsModalOpen(false)}>{t("common.cancel")}</Button>
            <Button 
              type="primary" 
              htmlType="submit" 
              loading={addMutation.isPending}
              disabled={availableLabels.length === 0}
            >
              {t("common.save")}
            </Button>
          </div>
        </Form>
      </Modal>
      <Modal
        title={t("contact.detail.labels.edit")}
        open={editingLabel !== null}
        onCancel={() => setEditingLabel(null)}
        footer={null}
        destroyOnClose={true}
      >
        <Form
          layout="vertical"
          onFinish={(values: { label_id: number }) => {
            if (editingLabel) {
              updateMutation.mutate({ id: editingLabel.id, label_id: values.label_id });
            }
          }}
        >
          <Form.Item
            name="label_id"
            label={t("contact.detail.labels.select_placeholder")}
            rules={[{ required: true }]}
          >
            <Select
              placeholder={t("contact.detail.labels.select_placeholder")}
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              options={editAvailableLabels.map((l: any) => ({
                label: l.name,
                value: l.id,
              }))}
              disabled={editAvailableLabels.length === 0}
            />
          </Form.Item>
          {editAvailableLabels.length === 0 && (
            <div style={{ marginBottom: 16 }}>
              <Text type="secondary" style={{ display: "block" }}>
                {t("contact.detail.labels.no_labels_available")}
              </Text>
              <Link to={`/vaults/${vaultId}/settings`} style={{ fontSize: 12 }}>
                {t("contact.detail.manage_labels_hint")}
              </Link>
            </div>
          )}
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => setEditingLabel(null)}>{t("common.cancel")}</Button>
            <Button
              type="primary"
              htmlType="submit"
              loading={updateMutation.isPending}
              disabled={editAvailableLabels.length === 0}
            >
              {t("common.save")}
            </Button>
          </div>
        </Form>
      </Modal>
    </Card>
  );
}
