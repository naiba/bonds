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
import { contactExtraApi } from "@/api/contactExtra";
import { settingsApi } from "@/api/settings";
import { useTranslation } from "react-i18next";
import type { Contact, UpdateContactReligionRequest } from "@/types/contact";
import type { APIError } from "@/types/api";
import { useState } from "react";

const { Title, Text } = Typography;

interface ExtraInfoModuleProps {
  vaultId: string;
  contactId: string;
  contact: Contact;
}

export default function ExtraInfoModule({ vaultId, contactId, contact }: ExtraInfoModuleProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [isReligionModalOpen, setIsReligionModalOpen] = useState(false);
  const [religionForm] = Form.useForm();

  const { data: religions = [] } = useQuery({
    queryKey: ["vaults", vaultId, "personalize", "religions"],
    queryFn: async () => {
      const res = await settingsApi.listPersonalizeItems("religions");
      return res.data.data ?? [];
    },
  });

  const religionMutation = useMutation({
    mutationFn: (values: UpdateContactReligionRequest) =>
      contactExtraApi.updateReligion(vaultId, contactId, values),
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
              religionForm.setFieldsValue({ religion_id: contact.religion_id });
              setIsReligionModalOpen(true);
            }}
          >
            {t("common.edit")}
          </Button>
        }
      >
        {contact.religion_id ? (
          <Text>{religions.find((r) => r.id === contact.religion_id)?.label || contact.religion_id}</Text>
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
              {contact.company_id ? `Company #${contact.company_id}` : "—"} 
            </Descriptions.Item>
            <Descriptions.Item label={t("contact.detail.job_position")}>
              {contact.job_position || "—"}
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
              options={religions.map((r) => ({ label: r.label, value: r.id }))}
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
