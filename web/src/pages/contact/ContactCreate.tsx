import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, Form, Input, Button, Typography, App, theme } from "antd";
import { ArrowLeftOutlined, UserOutlined } from "@ant-design/icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { contactsApi } from "@/api/contacts";
import type { CreateContactRequest } from "@/types/contact";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";

const { Title, Text } = Typography;

export default function ContactCreate() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const mutation = useMutation({
    mutationFn: (data: CreateContactRequest) =>
      contactsApi.create(vaultId, data),
    onSuccess: (res) => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts"],
      });
      message.success(t("contact.create.success"));
      const contact = res.data.data;
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

  function onFinish(values: CreateContactRequest) {
    setLoading(true);
    mutation.mutate(values);
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
        <Form layout="vertical" onFinish={onFinish} requiredMark="optional">
          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item
              name="first_name"
              label={t("contact.create.first_name_label")}
              style={{ flex: 1 }}
              rules={[{ required: true, message: t("common.required") }]}
            >
              <Input placeholder={t("contact.create.first_name_placeholder")} />
            </Form.Item>
            <Form.Item
              name="last_name"
              label={t("contact.create.last_name_label")}
              style={{ flex: 1 }}
            >
              <Input placeholder={t("contact.create.last_name_placeholder")} />
            </Form.Item>
          </div>

          <Form.Item name="nickname" label={t("contact.create.nickname_label")}>
            <Input placeholder={t("contact.create.nickname_placeholder")} />
          </Form.Item>

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
