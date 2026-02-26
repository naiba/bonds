import {
  Card,
  Button,
  Typography,
  Space,
  App,
  Form,
  Modal,
  Select,
  Descriptions,
  Input,
  Popconfirm,
  Tag,
  Empty,
} from "antd";
import { EditOutlined, ShopOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { api } from "@/api";
import { useTranslation } from "react-i18next";
import type { Contact, Company, UpdateContactReligionRequest, APIError } from "@/api";
import { useState } from "react";

const { Title, Text } = Typography;

// The backend returns religion_id, company_id, job_position in JSON but swagger
// annotations don't include them in ContactResponse. Use an extended interface.
interface ContactExtra {
  religion_id?: number;
  company_id?: number;
  job_position?: string;
}

interface ExtraInfoModuleProps {
  vaultId: string;
  contactId: string;
  contact: Contact;
}

export default function ExtraInfoModule({ vaultId, contactId, contact }: ExtraInfoModuleProps) {
  const extra = contact as Contact & ContactExtra;
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [isReligionModalOpen, setIsReligionModalOpen] = useState(false);
  const [religionForm] = Form.useForm();

  const { data: religions = [] } = useQuery({
    queryKey: ["vaults", vaultId, "personalize", "religions"],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail("religions");
      return res.data ?? [];
    },
  });

  const religionMutation = useMutation({
    mutationFn: (values: UpdateContactReligionRequest) =>
      api.contacts.contactsReligionUpdate(String(vaultId), String(contactId), values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId],
      });
      message.success(t("common.updated"));
      setIsReligionModalOpen(false);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });
  
  const [isJobModalOpen, setIsJobModalOpen] = useState(false);
  const [jobForm] = Form.useForm();

  const { data: companies = [] } = useQuery({
    queryKey: ["vaults", vaultId, "companies"],
    queryFn: async () => {
      const res = await api.companies.companiesList(String(vaultId));
      return res.data ?? [];
    },
    enabled: isJobModalOpen,
  });

  const jobMutation = useMutation({
    mutationFn: (values: { company_id?: number; job_position?: string }) =>
      api.contacts.contactsJobInformationUpdate(String(vaultId), String(contactId), values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId],
      });
      message.success(t("contact.detail.job_updated"));
      setIsJobModalOpen(false);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const clearJobMutation = useMutation({
    mutationFn: () => api.contacts.contactsJobInformationDelete(String(vaultId), String(contactId)),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId],
      });
      message.success(t("contact.detail.job_updated"));
      setIsJobModalOpen(false);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const { data: contactCompanies = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "companies"],
    queryFn: async () => {
      const res = await api.companies.contactsCompaniesListList(String(vaultId), String(contactId));
      return (res.data ?? []) as Company[];
    },
    enabled: !!vaultId && !!contactId,
  });

  return (
    <Space orientation="vertical" style={{ width: "100%" }} size={16}>
      <Card
        title={
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <Title level={5} style={{ margin: 0 }}>
              {t("contact.detail.religion")}
            </Title>
          </div>
        }
        extra={
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => {
              religionForm.setFieldsValue({ religion_id: extra.religion_id });
              setIsReligionModalOpen(true);
            }}
          >
            {t("common.edit")}
          </Button>
        }
      >
        {extra.religion_id ? (
          <Text>{/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
          {religions.find((r: any) => r.id === extra.religion_id)?.label || extra.religion_id}</Text>
        ) : (
          <Text type="secondary">{t("contact.detail.no_religion")}</Text>
        )}
      </Card>

      <Card
        title={
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <Title level={5} style={{ margin: 0 }}>
              {t("contact.detail.job_info")}
            </Title>
          </div>
        }
        extra={
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => {
              jobForm.setFieldsValue({
                company_id: extra.company_id,
                job_position: extra.job_position,
              });
              setIsJobModalOpen(true);
            }}
          >
            {t("common.edit")}
          </Button>
        }
      >
         <Descriptions column={1}>
             <Descriptions.Item label={t("contact.detail.company")}>
               {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
               {extra.company_id && companies.length ? companies.find((c: any) => c.id === extra.company_id)?.name : extra.company_id ? `Company #${extra.company_id}` : "—"} 
             </Descriptions.Item>
             <Descriptions.Item label={t("contact.detail.job_position")}>
               {extra.job_position || "—"}
             </Descriptions.Item>
         </Descriptions>
      </Card>

      <Card
        title={
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <ShopOutlined />
            <Title level={5} style={{ margin: 0 }}>
              {t("modules.extra_info.companies")}
            </Title>
          </div>
        }
        extra={
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => navigate(`/vaults/${vaultId}/companies`)}
          >
            {t("common.edit")}
          </Button>
        }
      >
        {contactCompanies.length > 0 ? (
          <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
            {contactCompanies.map((company: Company) => (
              <Tag
                key={company.id}
                style={{ cursor: "pointer", fontSize: 14, padding: "4px 10px" }}
                onClick={() => navigate(`/vaults/${vaultId}/companies`)}
              >
                {company.name}
              </Tag>
            ))}
          </div>
        ) : (
          <Empty
            description={t("modules.extra_info.no_companies")}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        )}
      </Card>

      <Modal
        title={t("contact.detail.religion")}
        open={isReligionModalOpen}
        onCancel={() => setIsReligionModalOpen(false)}
        footer={null}
        destroyOnClose={true}
      >
        <Form
          form={religionForm}
          layout="vertical"
          onFinish={(values) => religionMutation.mutate(values)}
        >
          <Form.Item name="religion_id" label={t("contact.detail.religion")}>
            <Select
              allowClear
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              options={religions.map((r: any) => ({ label: r.label, value: r.id }))}
              placeholder={t("contact.detail.labels.select_placeholder")}
            />
          </Form.Item>
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => setIsReligionModalOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              type="primary"
              htmlType="submit"
              loading={religionMutation.isPending}
            >
              {t("common.save")}
            </Button>
          </div>
        </Form>
      </Modal>

      <Modal
        title={t("contact.detail.edit_job")}
        open={isJobModalOpen}
        onCancel={() => setIsJobModalOpen(false)}
        footer={null}
        destroyOnClose={true}
      >
        <Form
          form={jobForm}
          layout="vertical"
          onFinish={(values) => jobMutation.mutate(values)}
        >
          <Form.Item name="company_id" label={t("contact.detail.company")}>
            <Select
              allowClear
              showSearch
              filterOption={(input, option) =>
                String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())
              }
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              options={companies.map((c: any) => ({ label: c.name, value: c.id }))}
              placeholder={t("contact.detail.labels.select_placeholder")}
            />
          </Form.Item>
          <Form.Item name="job_position" label={t("contact.detail.job_position")}>
            <Input />
          </Form.Item>
            <div style={{ display: "flex", justifyContent: "space-between", gap: 8 }}>
             {(extra.company_id || extra.job_position) && (
              <Popconfirm
                title={t("contact.detail.delete_confirm")}
                onConfirm={() => clearJobMutation.mutate()}
              >
                <Button danger type="text">
                  {t("contact.detail.clear_job")}
                </Button>
              </Popconfirm>
            )}
            <div style={{ display: "flex", gap: 8, marginLeft: "auto" }}>
              <Button onClick={() => setIsJobModalOpen(false)}>
                {t("common.cancel")}
              </Button>
              <Button
                type="primary"
                htmlType="submit"
                loading={jobMutation.isPending}
              >
                {t("common.save")}
              </Button>
            </div>
          </div>
        </Form>
      </Modal>
    </Space>
  );
}
