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
import { PlusOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { contactLabelsApi } from "@/api/contactLabels";
import { vaultSettingsApi } from "@/api/vaultSettings";
import { useTranslation } from "react-i18next";
import type { APIError } from "@/types/api";

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
  const [form] = Form.useForm();

  const { data: labels = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "labels"],
    queryFn: async () => {
      const res = await contactLabelsApi.list(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const { data: allLabels = [] } = useQuery({
    queryKey: ["vaults", vaultId, "labels"],
    queryFn: async () => {
      const res = await vaultSettingsApi.listLabels(parseInt(vaultId, 10));
      return res.data.data ?? [];
    },
    enabled: isModalOpen,
  });

  const addMutation = useMutation({
    mutationFn: (values: { label_id: number }) =>
      contactLabelsApi.add(vaultId, contactId, values),
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

  const removeMutation = useMutation({
    mutationFn: (labelId: number) =>
      contactLabelsApi.remove(vaultId, contactId, labelId),
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
    (l) => !labels.some((cl) => cl.label_id === l.id)
  );

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
        <Text type="secondary">{t("contact.detail.labels.no_labels")}</Text>
      ) : (
        <Space size={[8, 8]} wrap>
          {labels.map((label) => (
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
              options={availableLabels.map((l) => ({
                label: l.name,
                value: l.id,
              }))}
              disabled={availableLabels.length === 0}
            />
          </Form.Item>
          {availableLabels.length === 0 && (
            <Text type="secondary" style={{ display: "block", marginBottom: 16 }}>
              {t("contact.detail.labels.no_labels_available")}
            </Text>
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
    </Card>
  );
}
