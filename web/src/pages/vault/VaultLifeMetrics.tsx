import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Typography,
  Button,
  Table,
  Space,
  Modal,
  Form,
  Input,
  message,
  theme,
} from "antd";
import {
  HeartOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import { api } from "@/api";
import type { LifeMetric } from "@/api";

const { Title, Text } = Typography;

export default function VaultLifeMetrics() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { token } = theme.useToken();
  const [form] = Form.useForm();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingMetric, setEditingMetric] = useState<LifeMetric | null>(null);

  const { data: metrics = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "lifeMetrics"],
    queryFn: async () => {
      const res = await api.lifeMetrics.lifeMetricsList(String(vaultId));
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  const createMutation = useMutation({
    mutationFn: (values: { label: string }) =>
      api.lifeMetrics.lifeMetricsCreate(String(vaultId), values),
    onSuccess: () => {
      message.success(t("common.saved"));
      setIsModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "lifeMetrics"] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: (values: { id: number; label: string }) =>
      api.lifeMetrics.lifeMetricsUpdate(String(vaultId), values.id, { label: values.label }),
    onSuccess: () => {
      message.success(t("common.saved"));
      setIsModalOpen(false);
      setEditingMetric(null);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "lifeMetrics"] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.lifeMetrics.lifeMetricsDelete(String(vaultId), id),
    onSuccess: () => {
      message.success(t("common.deleted"));
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "lifeMetrics"] });
    },
  });

  const handleEdit = (metric: LifeMetric) => {
    setEditingMetric(metric);
    form.setFieldsValue({ label: metric.label });
    setIsModalOpen(true);
  };

  const handleDelete = (id: number) => {
    Modal.confirm({
      title: t("common.confirmDelete"),
      content: t("common.confirmDeleteDescription"),
      okText: t("common.delete"),
      cancelText: t("common.cancel"),
      okType: "danger",
      onOk: () => deleteMutation.mutate(id),
    });
  };

  return (
    <div style={{ maxWidth: 1000, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 24 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate(`/vaults/${vaultId}`)}
            style={{ color: token.colorTextSecondary }}
          />
          <HeartOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
          <Title level={4} style={{ margin: 0 }}>{t("vault.lifeMetrics.title")}</Title>
        </div>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingMetric(null);
            form.resetFields();
            setIsModalOpen(true);
          }}
        >
          {t("vault.lifeMetrics.create")}
        </Button>
      </div>

      {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
      <Table<any>
        dataSource={metrics}
        rowKey="id"
        loading={isLoading}
        pagination={false}
        columns={[
          {
            title: t("vault.lifeMetrics.label"),
            dataIndex: "label",
            key: "label",
            render: (text) => <Text strong>{text}</Text>,
          },
          {
            title: t("common.actions"),
            key: "actions",
            width: 120,
            render: (_, record) => (
              <Space>
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => handleEdit(record)}
                />
                <Button
                  type="text"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={() => handleDelete(record.id)}
                />
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title={editingMetric ? t("vault.lifeMetrics.edit") : t("vault.lifeMetrics.create")}
        open={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => {
            if (editingMetric) {
              updateMutation.mutate({ id: editingMetric.id!, label: values.label });
            } else {
              createMutation.mutate(values);
            }
          }}
        >
          <Form.Item
            name="label"
            label={t("vault.lifeMetrics.label")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
