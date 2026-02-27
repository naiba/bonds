import {
  Card,
  Button,
  Typography,
  Space,
  App,
  Form,
  Modal,
  Select,
  Input,
  Popconfirm,
  List,
  Empty,
} from "antd";
import { EditOutlined, PlusOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import { Link } from "react-router-dom";
import { api } from "@/api";
import { useTranslation } from "react-i18next";
import type { Contact, UpdateContactReligionRequest, APIError, ContactJob } from "@/api";
import { useState } from "react";

const { Title, Text } = Typography;

interface ContactExtra {
  religion_id?: number;
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
  const [isReligionModalOpen, setIsReligionModalOpen] = useState(false);
  const [religionForm] = Form.useForm();

  // --- Job state ---
  const [isJobModalOpen, setIsJobModalOpen] = useState(false);
  const [editingJob, setEditingJob] = useState<ContactJob | null>(null);
  const [jobForm] = Form.useForm();

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

  // --- Jobs queries & mutations ---
  // Companies list always enabled — needed to display company names in the job list
  const { data: companies = [] } = useQuery({
    queryKey: ["vaults", vaultId, "companies"],
    queryFn: async () => {
      const res = await api.companies.companiesList(String(vaultId));
      return res.data ?? [];
    },
  });

  const { data: jobs = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "jobs"],
    queryFn: async () => {
      const res = await api.contacts.contactsJobsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const createJobMutation = useMutation({
    mutationFn: (values: { company_id: number; job_position?: string }) =>
      api.contacts.contactsJobsCreate(String(vaultId), String(contactId), values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId, "jobs"],
      });
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId],
      });
      message.success(t("contact.detail.job_added"));
      setIsJobModalOpen(false);
      setEditingJob(null);
      jobForm.resetFields();
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const updateJobMutation = useMutation({
    mutationFn: ({ jobId, values }: { jobId: number; values: { company_id: number; job_position?: string } }) =>
      api.contacts.contactsJobsUpdate(String(vaultId), String(contactId), jobId, values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId, "jobs"],
      });
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId],
      });
      message.success(t("contact.detail.job_updated"));
      setIsJobModalOpen(false);
      setEditingJob(null);
      jobForm.resetFields();
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const deleteJobMutation = useMutation({
    mutationFn: (jobId: number) =>
      api.contacts.contactsJobsDelete(String(vaultId), String(contactId), jobId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId, "jobs"],
      });
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", contactId],
      });
      message.success(t("contact.detail.job_deleted"));
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const openAddJob = () => {
    setEditingJob(null);
    jobForm.resetFields();
    setIsJobModalOpen(true);
  };

  const openEditJob = (job: ContactJob) => {
    setEditingJob(job);
    jobForm.setFieldsValue({
      company_id: job.company_id,
      job_position: job.job_position,
    });
    setIsJobModalOpen(true);
  };

  const handleJobSubmit = (values: { company_id: number; job_position?: string }) => {
    if (editingJob?.id) {
      updateJobMutation.mutate({ jobId: editingJob.id, values });
    } else {
      createJobMutation.mutate(values);
    }
  };

  return (
    <Space direction="vertical" style={{ width: "100%" }} size={16}>
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
            icon={<PlusOutlined />}
            onClick={openAddJob}
          >
            {t("contact.detail.add_job")}
          </Button>
        }
      >
        {jobs.length > 0 ? (
          <List
            itemLayout="horizontal"
            dataSource={jobs}
            renderItem={(job: ContactJob) => (
              <List.Item
                actions={[
                  <Button
                    key="edit"
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    onClick={() => openEditJob(job)}
                  />,
                  <Popconfirm
                    key="delete"
                    title={t("common.delete_confirm")}
                    onConfirm={() => job.id && deleteJobMutation.mutate(job.id)}
                  >
                    <Button
                      type="text"
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                    />
                  </Popconfirm>,
                ]}
              >
                <List.Item.Meta
                  title={job.company_name || `Company #${job.company_id}`}
                  description={job.job_position || "—"}
                />
              </List.Item>
            )}
          />
        ) : (
          <Empty description={t("contact.detail.no_jobs")} image={Empty.PRESENTED_IMAGE_SIMPLE}>
            <Link to={`/vaults/${vaultId}/settings`} style={{ fontSize: 12 }}>
              {t("contact.detail.manage_companies_hint")}
            </Link>
          </Empty>
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
        title={editingJob ? t("contact.detail.edit_job") : t("contact.detail.add_job")}
        open={isJobModalOpen}
        onCancel={() => {
          setIsJobModalOpen(false);
          setEditingJob(null);
          jobForm.resetFields();
        }}
        footer={null}
        destroyOnClose={true}
      >
        <Form
          form={jobForm}
          layout="vertical"
          onFinish={handleJobSubmit}
        >
          <Form.Item
            name="company_id"
            label={t("contact.detail.company")}
            rules={[{ required: true, message: t("common.required") }]}
          >
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
          {companies.length === 0 && (
            <div style={{ marginTop: -12, marginBottom: 16, fontSize: 12 }}>
              <Link to={`/vaults/${vaultId}/settings`}>
                {t("contact.detail.manage_companies_hint")}
              </Link>
            </div>
          )}
          <Form.Item name="job_position" label={t("contact.detail.job_position")}>
            <Input />
          </Form.Item>
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => {
              setIsJobModalOpen(false);
              setEditingJob(null);
              jobForm.resetFields();
            }}>
              {t("common.cancel")}
            </Button>
            <Button
              type="primary"
              htmlType="submit"
              loading={createJobMutation.isPending || updateJobMutation.isPending}
            >
              {t("common.save")}
            </Button>
          </div>
        </Form>
      </Modal>
    </Space>
  );
}
