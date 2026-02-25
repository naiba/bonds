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
  Drawer,
  Descriptions,
  List,
  Empty,
} from "antd";
import {
  BankOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import ContactAvatar from "@/components/ContactAvatar";
import { api } from "@/api";
import type { Company } from "@/api";

const { Title, Text } = Typography;

export default function VaultCompanies() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { token } = theme.useToken();
  const nameOrder = useNameOrder();
  const [form] = Form.useForm();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingCompany, setEditingCompany] = useState<Company | null>(null);
  const [selectedCompany, setSelectedCompany] = useState<Company | null>(null);

  const { data: companies = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "companies"],
    queryFn: async () => {
      const res = await api.companies.companiesList(String(vaultId));
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  const { data: companyDetails } = useQuery({
    queryKey: ["vaults", vaultId, "companies", selectedCompany?.id],
    queryFn: async () => {
      if (!selectedCompany?.id) return null;
      try {
        const res = await api.companies.companiesDetail(String(vaultId), selectedCompany.id);
        return res.data;
      } catch {
        return selectedCompany;
      }
    },
    enabled: !!selectedCompany?.id,
  });

  const employees = companyDetails?.contacts ?? selectedCompany?.contacts ?? [];

  const createMutation = useMutation({
    mutationFn: (values: { name: string; type: string }) =>
      api.companies.companiesCreate(String(vaultId), values),
    onSuccess: () => {
      message.success(t("common.saved"));
      setIsModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "companies"] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: (values: { id: number; name: string; type: string }) =>
      api.companies.companiesUpdate(String(vaultId), values.id, { name: values.name, type: values.type }),
    onSuccess: () => {
      message.success(t("common.saved"));
      setIsModalOpen(false);
      setEditingCompany(null);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "companies"] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (companyId: number) => api.companies.companiesDelete(String(vaultId), companyId),
    onSuccess: () => {
      message.success(t("common.deleted"));
      setSelectedCompany(null);
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "companies"] });
    },
  });

  const handleEdit = (company: Company) => {
    setEditingCompany(company);
    form.setFieldsValue({ name: company.name, type: company.type });
    setIsModalOpen(true);
  };

  const handleDelete = (companyId: number) => {
    Modal.confirm({
      title: t("common.confirmDelete"),
      content: t("vault.companies.delete_confirm"),
      okText: t("common.delete"),
      cancelText: t("common.cancel"),
      okType: "danger",
      onOk: () => deleteMutation.mutate(companyId),
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

      {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
      <Table<any>
        dataSource={companies}
        rowKey="id"
        loading={isLoading}
        pagination={false}
        locale={{ emptyText: (
          <div className="bonds-empty-hero">
            <div className="bonds-empty-hero-icon" style={{ background: token.colorPrimaryBg }}>
              <BankOutlined style={{ fontSize: 32, color: token.colorPrimary }} />
            </div>
            <div className="bonds-empty-hero-title">{t("vault.companies.title")}</div>
            <div className="bonds-empty-hero-desc" style={{ color: token.colorTextSecondary }}>{t("empty.companies")}</div>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => { setEditingCompany(null); form.resetFields(); setIsModalOpen(true); }}>
              {t("vault.companies.create")}
            </Button>
          </div>
        ) }}
        onRow={(record) => ({
          onClick: () => setSelectedCompany(record),
          style: { cursor: "pointer" },
        })}
        columns={[
          {
            title: t("vault.companies.name"),
            dataIndex: "name",
            key: "name",
            render: (text) => <Text strong>{text}</Text>,
          },
          {
            title: t("vault.companies.type"),
            dataIndex: "type",
            key: "type",
          },
          {
            title: t("vault.companies.employees"),
            key: "contacts",
            render: (_, record) => (
              <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {record.contacts?.map((contact: any) => (
                  <Tag
                    key={contact.id}
                    style={{ margin: 0 }}
                  >
                    <span
                      onClick={(e) => {
                        e.stopPropagation();
                        navigate(`/vaults/${vaultId}/contacts/${contact.id}`);
                      }}
                    >
                      {formatContactName(nameOrder, contact)}
                    </span>
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
                  onClick={(e) => {
                    e.stopPropagation();
                    handleEdit(record);
                  }}
                />
                <Button
                  type="text"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDelete(record.id);
                  }}
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
              updateMutation.mutate({ id: editingCompany.id!, name: values.name, type: values.type ?? "" });
            } else {
              createMutation.mutate({ name: values.name, type: values.type ?? "" });
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
          <Form.Item
            name="type"
            label={t("vault.companies.type")}
          >
            <Input />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={companyDetails?.name || selectedCompany?.name}
        placement="right"
        onClose={() => setSelectedCompany(null)}
        open={!!selectedCompany}
        width={500}
      >
        {selectedCompany && (
            <>
            <Descriptions column={1} bordered>
                <Descriptions.Item label={t("vault.companies.name")}>
                {companyDetails?.name || selectedCompany.name}
                </Descriptions.Item>
                <Descriptions.Item label={t("vault.companies.type")}>
                {companyDetails?.type || selectedCompany.type || "—"}
                </Descriptions.Item>
            </Descriptions>

            <Title level={5} style={{ marginTop: 24, marginBottom: 16 }}>
                {t("vault.companies.employees")}
            </Title>
            
            <List
                itemLayout="horizontal"
                dataSource={employees}
                locale={{ emptyText: <Empty description={t("vault.companies.no_employees")} /> }}
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                renderItem={(item: any) => (
                <List.Item
                    actions={[
                    <Button 
                        type="link" 
                        size="small"
                        onClick={() => navigate(`/vaults/${vaultId}/contacts/${item.id}`)}
                    >
                        {t("common.view")}
                    </Button>
                    ]}
                >
                    <List.Item.Meta
                    avatar={
                      <ContactAvatar
                        vaultId={String(id)}
                        contactId={item.id}
                        firstName={item.first_name}
                        lastName={item.last_name}
                        size={32}
                        updatedAt={item.updated_at}
                      />
                    }
                    title={formatContactName(nameOrder, item)}
                    description={item.job_title}
                    />
                </List.Item>
                )}
            />
            </>
        )}
      </Drawer>
    </div>
  );
}
