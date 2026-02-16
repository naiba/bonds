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
} from "antd";
import { EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import { useTranslation } from "react-i18next";
import type { Contact, UpdateContactReligionRequest, APIError } from "@/api";
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
      >
         <Descriptions column={1}>
             <Descriptions.Item label={t("contact.detail.company")}>
               {extra.company_id ? `Company #${extra.company_id}` : "—"} 
             </Descriptions.Item>
             <Descriptions.Item label={t("contact.detail.job_position")}>
               {extra.job_position || "—"}
             </Descriptions.Item>
         </Descriptions>
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
    </Space>
  );
}
