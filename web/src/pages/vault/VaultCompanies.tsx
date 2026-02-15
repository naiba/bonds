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
  Tag,
} from "antd";
import {
  BankOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import { companiesApi } from "@/api/companies";
import type { Company } from "@/types/modules";

const { Title, Text } = Typography;

export default function VaultCompanies() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { token } = theme.useToken();
  const [form] = Form.useForm();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingCompany, setEditingCompany] = useState<Company | null>(null);

  const { data: companies = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "companies"],
    queryFn: async () => {
      const res = await companiesApi.list(vaultId);
      return res.data.data ?? [];
    },
    enabled: !!vaultId,
  });

  const createMutation = useMutation({
    mutationFn: (values: { name: string }) =>
      companiesApi.create(vaultId, values),
    onSuccess: () => {
      message.success(t("common.saved"));
      setIsModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "companies"] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: (values: { id: number; name: string }) =>
      companiesApi.update(vaultId, values.id, { name: values.name }),
    onSuccess: () => {
      message.success(t("common.saved"));
      setIsModalOpen(false);
      setEditingCompany(null);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "companies"] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => companiesApi.delete(vaultId, id),
    onSuccess: () => {
      message.success(t("common.deleted"));
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "companies"] });
    },
  });

  const handleEdit = (company: Company) => {
    setEditingCompany(company);
    form.setFieldsValue({ name: company.name });
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
          <BankOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
          <Title level={4} style={{ margin: 0 }}>{t("vault.companies.title")}</Title>
        </div>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingCompany(null);
            form.resetFields();
            setIsModalOpen(true);
          }}
        >
          {t("vault.companies.create")}
        </Button>
      </div>

      <Table
        dataSource={companies}
        rowKey="id"
        loading={isLoading}
        pagination={false}
        columns={[
          {
            title: t("vault.companies.name"),
            dataIndex: "name",
            key: "name",
            render: (text) => <Text strong>{text}</Text>,
          },
          {
            title: t("vault.companies.employees"),
            key: "contacts",
            render: (_, record) => (
              <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
                {record.contacts?.map((contact) => (
                  <Tag
                    key={contact.id}
                    style={{ margin: 0 }}
                  >
                    <a
                      onClick={(e) => {
                        e.preventDefault();
                        navigate(`/vaults/${vaultId}/contacts/${contact.id}`);
                      }}
                    >
                      {contact.first_name} {contact.last_name}
                    </a>
                  </Tag>
                ))}
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
        title={editingCompany ? t("vault.companies.edit") : t("vault.companies.create")}
        open={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => {
            if (editingCompany) {
              updateMutation.mutate({ id: editingCompany.id, name: values.name });
            } else {
              createMutation.mutate(values);
            }
          }}
        >
          <Form.Item
            name="name"
            label={t("vault.companies.name")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
