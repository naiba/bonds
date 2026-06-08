import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, Form, Input, Button, Typography, App, theme, Select, Checkbox, InputNumber } from "antd";
import { ArrowLeftOutlined, UserOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { CreateContactRequest, APIError, Contact } from "@/api";
import { useTranslation } from "react-i18next";
import { dateInputToTimestamp } from "@/utils/dateOnlyInput";
import { formatContactName, useNameOrder } from "@/utils/nameFormat";

const { Title, Text } = Typography;

type ContactCreateFormValues = Omit<CreateContactRequest, "last_talked_to" | "first_met_at"> & {
  last_talked_to?: string;
  first_met_at?: string;
};

function buildCreateContactRequest(values: ContactCreateFormValues): CreateContactRequest {
  const request: CreateContactRequest = {
    ...values,
    last_talked_to: dateInputToTimestamp(values.last_talked_to),
    first_met_at: dateInputToTimestamp(values.first_met_at),
  };
  if (!request.last_talked_to) delete request.last_talked_to;
  if (!request.first_met_at) delete request.first_met_at;
  if (!request.first_met_through_contact_id) delete request.first_met_through_contact_id;
  if (request.stay_in_touch_frequency_days == null) delete request.stay_in_touch_frequency_days;
  return request;
}

export default function ContactCreate() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const nameOrder = useNameOrder();
  const [form] = Form.useForm();

  const { data: contactOptions = [], isLoading: isContactOptionsLoading } = useQuery<Contact[]>({
    queryKey: ["vaults", vaultId, "contacts", "meeting-options"],
    queryFn: async () => {
      const res = await api.contacts.contactsList(String(vaultId), { per_page: 9999, filter: "all" });
      return res.data ?? [];
    },
  });

  const meetingContactOptions = contactOptions.flatMap((contact) => (
    contact.id ? [{ label: formatContactName(nameOrder, contact), value: contact.id }] : []
  ));

  const mutation = useMutation({
    mutationFn: (data: CreateContactRequest) =>
      api.contacts.contactsCreate(String(vaultId), data),
    onSuccess: (res) => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts"],
      });
      message.success(t("contact.create.success"));
      const contact = res.data;
      navigate(
        contact
          ? `/vaults/${vaultId}/contacts/${contact.id}`
          : `/vaults/${vaultId}/contacts`,
      );
    },
    onError: (err: APIError) => {
      message.error(err.message || t("contact.create.failed"));
      setLoading(false);
    },
  });

  function onFinish(values: ContactCreateFormValues) {
    setLoading(true);
    mutation.mutate(buildCreateContactRequest(values));
  }

  return (
    <div style={{ maxWidth: 560, margin: "0 auto" }}>
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}/contacts`)}
        style={{ marginBottom: 16 }}
      >
        {t("contact.detail.back")}
      </Button>

      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 14,
          marginBottom: 24,
        }}
      >
        <div
          style={{
            width: 44,
            height: 44,
            borderRadius: 12,
            backgroundColor: token.colorPrimaryBg,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            flexShrink: 0,
          }}
        >
          <UserOutlined style={{ fontSize: 22, color: token.colorPrimary }} />
        </div>
        <div>
          <Title level={4} style={{ margin: 0 }}>
            {t("contact.create.title")}
          </Title>
          <Text type="secondary" style={{ fontSize: 13 }}>
            {t("contact.create.description", { defaultValue: t("contact.create.title") })}
          </Text>
        </div>
      </div>

      <Card>
        <Form form={form} layout="vertical" onFinish={onFinish} requiredMark="optional">
          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item
              name="prefix"
              label={t("contact.create.prefix_label")}
              style={{ flex: 1 }}
            >
              <Input placeholder={t("contact.create.prefix_placeholder")} />
            </Form.Item>
            <Form.Item
              name="first_name"
              dependencies={["nickname"]}
              label={t("contact.create.first_name_label")}
              style={{ flex: 2 }}
              rules={[{
                validator: (_, value) => {
                  const nickname = form.getFieldValue("nickname");
                  if (!value?.trim() && !nickname?.trim()) {
                    return Promise.reject(new Error(t("contact.form.name_or_nickname_required")));
                  }
                  return Promise.resolve();
                },
              }]}
            >
              <Input placeholder={t("contact.create.first_name_placeholder")} />
            </Form.Item>
            <Form.Item
              name="middle_name"
              label={t("contact.create.middle_name_label")}
              style={{ flex: 2 }}
            >
              <Input placeholder={t("contact.create.middle_name_placeholder")} />
            </Form.Item>
          </div>

          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item
              name="last_name"
              label={t("contact.create.last_name_label")}
              style={{ flex: 2 }}
            >
              <Input placeholder={t("contact.create.last_name_placeholder")} />
            </Form.Item>
            <Form.Item
              name="suffix"
              label={t("contact.create.suffix_label")}
              style={{ flex: 1 }}
            >
              <Input placeholder={t("contact.create.suffix_placeholder")} />
            </Form.Item>
          </div>

          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item
              name="nickname"
              dependencies={["first_name"]}
              label={t("contact.create.nickname_label")}
              style={{ flex: 1 }}
              rules={[{
                validator: (_, value) => {
                  const firstName = form.getFieldValue("first_name");
                  if (!value?.trim() && !firstName?.trim()) {
                    return Promise.reject(new Error(t("contact.form.name_or_nickname_required")));
                  }
                  return Promise.resolve();
                },
              }]}
            >
              <Input placeholder={t("contact.create.nickname_placeholder")} />
            </Form.Item>
            <Form.Item name="maiden_name" label={t("contact.create.maiden_name_label")} style={{ flex: 1 }}>
              <Input placeholder={t("contact.create.maiden_name_placeholder")} />
            </Form.Item>
          </div>

          {/* Fix #62: gender and pronoun fields — fetched from personalize API */}
          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item name="gender_id" label={t("contact.detail.summary.gender")} style={{ flex: 1 }}>
              <GenderPronounSelect entity="genders" vaultId={vaultId} placeholder={t("contact.form.select_gender")} />
            </Form.Item>
            <Form.Item name="pronoun_id" label={t("contact.detail.summary.pronoun")} style={{ flex: 1 }}>
              <GenderPronounSelect entity="pronouns" vaultId={vaultId} placeholder={t("contact.form.select_pronoun")} />
            </Form.Item>
          </div>

          <Form.Item name="needs_verification" valuePropName="checked" style={{ marginBottom: 8 }}>
            <Checkbox>{t("contact.needs_verification.field_label")}</Checkbox>
          </Form.Item>

          <div
            style={{
              margin: "16px 0",
              padding: 16,
              border: `1px solid ${token.colorBorderSecondary}`,
              borderRadius: token.borderRadiusLG,
              background: token.colorFillQuaternary,
            }}
          >
            <Text strong style={{ display: "block", marginBottom: 4 }}>
              {t("contact.meeting.title")}
            </Text>
            <Text type="secondary" style={{ display: "block", fontSize: 13, marginBottom: 12 }}>
              {t("contact.meeting.description")}
            </Text>
            <div style={{ display: "flex", gap: 16 }}>
              <Form.Item
                name="first_met_at"
                label={t("contact.meeting.first_met_at")}
                extra={t("contact.meeting.first_met_at_help")}
                style={{ flex: 1 }}
              >
                <Input type="date" />
              </Form.Item>
              <Form.Item
                name="first_met_through_contact_id"
                label={t("contact.meeting.first_met_through")}
                extra={t("contact.meeting.first_met_through_help")}
                style={{ flex: 1 }}
              >
                <Select
                  allowClear
                  showSearch
                  loading={isContactOptionsLoading}
                  optionFilterProp="label"
                  placeholder={t("contact.meeting.first_met_through_placeholder")}
                  options={meetingContactOptions}
                />
              </Form.Item>
            </div>
          </div>

          <div
            style={{
              margin: "16px 0",
              padding: 16,
              border: `1px solid ${token.colorBorderSecondary}`,
              borderRadius: token.borderRadiusLG,
              background: token.colorFillQuaternary,
            }}
          >
            <Text strong style={{ display: "block", marginBottom: 4 }}>
              {t("contact.catch_up.title")}
            </Text>
            <Text type="secondary" style={{ display: "block", fontSize: 13, marginBottom: 12 }}>
              {t("contact.catch_up.description")}
            </Text>
            <div style={{ display: "flex", gap: 16 }}>
              <Form.Item
                name="last_talked_to"
                label={t("contact.catch_up.last_talked_to")}
                extra={t("contact.catch_up.last_talked_to_help")}
                style={{ flex: 1 }}
              >
                <Input type="date" />
              </Form.Item>
              <Form.Item
                name="stay_in_touch_frequency_days"
                label={t("contact.catch_up.frequency_days")}
                extra={t("contact.catch_up.frequency_days_help")}
                style={{ flex: 1 }}
              >
                <InputNumber min={1} precision={0} style={{ width: "100%" }} />
              </Form.Item>
            </div>
          </div>

          <Form.Item style={{ marginBottom: 0, marginTop: 8, textAlign: "right" }}>
            <Button
              style={{ marginRight: 8 }}
              onClick={() => navigate(`/vaults/${vaultId}/contacts`)}
            >
              {t("common.cancel")}
            </Button>
            <Button type="primary" htmlType="submit" loading={loading}>
              {t("contact.create.submit")}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}

// Shared Select component for gender/pronoun fetched from personalize API
function GenderPronounSelect({ entity, vaultId, placeholder, ...props }: {
  entity: string;
  vaultId: string;
  placeholder: string;
  value?: number;
  onChange?: (value: number | undefined) => void;
}) {
  const { data: items = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "personalize", entity],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail(entity);
      return res.data ?? [];
    },
  });

  return (
    <Select
      {...props}
      loading={isLoading}
      allowClear
      placeholder={placeholder}
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      options={(items as any[]).map((item) => ({ label: item.label, value: item.id }))}
    />
  );
}
