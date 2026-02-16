import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, Form, Input, Button, Typography, App, theme } from "antd";
import { ArrowLeftOutlined, SafetyCertificateOutlined } from "@ant-design/icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { CreateVaultRequest, APIError } from "@/api";
import { useTranslation } from "react-i18next";

const { Title, Text } = Typography;

export default function VaultCreate() {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const mutation = useMutation({
    mutationFn: (data: CreateVaultRequest) => api.vaults.vaultsCreate(data),
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    onSuccess: (res: any) => {
      queryClient.invalidateQueries({ queryKey: ["vaults"] });
      message.success(t("vault.create.success"));
      const vault = res.data;
      navigate(vault ? `/vaults/${vault.id}` : "/vaults");
    },
    onError: (err: APIError) => {
      message.error(err.message || t("vault.create.failed"));
      setLoading(false);
    },
  });

  function onFinish(values: CreateVaultRequest) {
    setLoading(true);
    mutation.mutate(values);
  }

  return (
    <div style={{ maxWidth: 600, margin: "0 auto" }}>
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate("/vaults")}
        style={{ marginBottom: 16 }}
      >
        {t("vault.list.title")}
      </Button>

      <div style={{ marginBottom: 28 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 8 }}>
          <div
            style={{
              width: 42,
              height: 42,
              borderRadius: "50%",
              background: token.colorPrimaryBg,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              flexShrink: 0,
            }}
          >
            <SafetyCertificateOutlined
              style={{ fontSize: 20, color: token.colorPrimary }}
            />
          </div>
          <Title level={3} style={{ margin: 0 }}>
            {t("vault.create.title")}
          </Title>
        </div>
        <Text type="secondary" style={{ fontSize: 14, marginLeft: 54 }}>
          {t("vault.list.no_vaults_desc")}
        </Text>
      </div>

      <Card
        style={{
          borderTop: `3px solid ${token.colorPrimary}`,
        }}
        styles={{
          body: { padding: "32px 32px 24px" },
        }}
      >
        <Form layout="vertical" onFinish={onFinish} size="large">
          <Form.Item
            name="name"
            label={t("vault.create.name_label")}
            rules={[{ required: true, message: t("vault.create.name_required") }]}
          >
            <Input
              prefix={<SafetyCertificateOutlined style={{ color: token.colorTextQuaternary }} />}
              placeholder={t("vault.create.name_placeholder")}
            />
          </Form.Item>

          <Form.Item name="description" label={t("vault.create.description_label")}>
            <Input.TextArea
              rows={3}
              placeholder={t("vault.create.description_placeholder")}
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: "right", marginTop: 8 }}>
            <Button
              style={{ marginRight: 12 }}
              onClick={() => navigate("/vaults")}
            >
              {t("common.cancel")}
            </Button>
            <Button type="primary" htmlType="submit" loading={loading}>
              {t("vault.create.submit")}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
