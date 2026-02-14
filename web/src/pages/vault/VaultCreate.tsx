import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, Form, Input, Button, Typography, App } from "antd";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { vaultsApi } from "@/api/vaults";
import type { CreateVaultRequest } from "@/types/vault";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";

const { Title } = Typography;

export default function VaultCreate() {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();

  const mutation = useMutation({
    mutationFn: (data: CreateVaultRequest) => vaultsApi.create(data),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ["vaults"] });
      message.success(t("vault.create.success"));
      const vault = res.data.data;
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
    <div style={{ maxWidth: 560, margin: "0 auto" }}>
      <Title level={4} style={{ marginBottom: 24 }}>
        {t("vault.create.title")}
      </Title>
      <Card>
        <Form layout="vertical" onFinish={onFinish}>
          <Form.Item
            name="name"
            label={t("vault.create.name_label")}
            rules={[{ required: true, message: t("vault.create.name_required") }]}
          >
            <Input placeholder={t("vault.create.name_placeholder")} />
          </Form.Item>

          <Form.Item name="description" label={t("vault.create.description_label")}>
            <Input.TextArea
              rows={3}
              placeholder={t("vault.create.description_placeholder")}
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: "right" }}>
            <Button
              style={{ marginRight: 8 }}
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
