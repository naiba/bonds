import { useState } from "react";
import { formatContactName, useNameOrder } from "@/utils/nameFormat";
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
  Tag,
  Select,
} from "antd";
import {
  HeartOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
  UserAddOutlined,
} from "@ant-design/icons";
import { api } from "@/api";
import type { LifeMetric, SearchResult } from "@/api";

const { Title, Text } = Typography;

export default function VaultLifeMetrics() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { token } = theme.useToken();
  const nameOrder = useNameOrder();
  const [form] = Form.useForm();
  const [contactForm] = Form.useForm();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isContactModalOpen, setIsContactModalOpen] = useState(false);
  const [editingMetric, setEditingMetric] = useState<LifeMetric | null>(null);
  const [selectedMetricId, setSelectedMetricId] = useState<number | null>(null);
  const [contactSearchResults, setContactSearchResults] = useState<SearchResult[]>([]);

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

  const addContactMutation = useMutation({
    mutationFn: (values: { metricId: number; contactId: number }) =>
      api.lifeMetrics.lifeMetricsContactsCreate(String(vaultId), values.metricId, { contact_id: String(values.contactId) }),
    onSuccess: () => {
      message.success(t("common.saved"));
      setIsContactModalOpen(false);
      contactForm.resetFields();
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "lifeMetrics"] });
    },
  });

  const removeContactMutation = useMutation({
    mutationFn: (values: { metricId: number; contactId: string }) =>
      api.lifeMetrics.lifeMetricsContactsDelete(String(vaultId), values.metricId, values.contactId),
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

  const handleAddContact = (metricId: number) => {
    setSelectedMetricId(metricId);
    setContactSearchResults([]);
    setIsContactModalOpen(true);
  };

  const handleContactSearch = async (value: string) => {
    if (!value) {
      setContactSearchResults([]);
      return;
    }
    try {
      const res = await api.search.searchList(String(vaultId), { q: value });
      const data = res.data as { contacts?: SearchResult[] };
      setContactSearchResults(data?.contacts ?? []);
    } catch {
      setContactSearchResults([]);
    }
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
            title: t("vault.lifeMetrics.contacts"),
            key: "contacts",
            render: (_, record) => (
              <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {record.contacts?.map((contact: any) => (
                  <Tag
                    key={contact.id}
                    closable
                    onClose={(e) => {
                      e.preventDefault();
                      removeContactMutation.mutate({ metricId: record.id, contactId: contact.id });
                    }}
                    style={{ margin: 0 }}
                  >
                    <a
                      onClick={(e) => {
                        e.preventDefault();
                        navigate(`/vaults/${vaultId}/contacts/${contact.id}`);
                      }}
                    >
                      {formatContactName(nameOrder, contact)}
                    </a>
                  </Tag>
                ))}
                <Button
                  type="dashed"
                  size="small"
                  icon={<UserAddOutlined />}
                  onClick={() => handleAddContact(record.id)}
                  style={{ borderRadius: 12, fontSize: 12 }}
                />
              </div>
            ),
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

      <Modal
        title={t("vault.lifeMetrics.addContact")}
        open={isContactModalOpen}
        onCancel={() => setIsContactModalOpen(false)}
        onOk={() => contactForm.submit()}
        confirmLoading={addContactMutation.isPending}
      >
        <Form
          form={contactForm}
          layout="vertical"
          onFinish={(values) => {
            if (selectedMetricId) {
              addContactMutation.mutate({ metricId: selectedMetricId, contactId: values.contactId });
            }
          }}
        >
          <Form.Item
            name="contactId"
            label={t("common.contact")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Select
              showSearch
              placeholder={t("search.placeholder")}
              defaultActiveFirstOption={false}
              showArrow={false}
              filterOption={false}
              onSearch={handleContactSearch}
              notFoundContent={null}
              options={contactSearchResults.map((c) => ({
                value: c.id,
                label: c.name,
              }))}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
