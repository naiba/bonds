import { Card, Typography, Button, App, Spin, Form, Input, Select, Divider } from "antd";
import { SettingOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { SystemSettingItem, APIError } from "@/api";

const { Title, Text } = Typography;

const KNOWN_SETTINGS = [
  { key: "oauth_github_key", secret: false },
  { key: "oauth_github_secret", secret: true },
  { key: "oauth_google_key", secret: false },
  { key: "oauth_google_secret", secret: true },
  { key: "oidc_client_id", secret: false },
  { key: "oidc_client_secret", secret: true },
  { key: "oidc_discovery_url", secret: false },
  { key: "oidc_name", secret: false },
  { key: "password_auth_enabled", type: "boolean" as const },
  { key: "registration_enabled", type: "boolean" as const },
];

export default function AdminSettings() {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();
  const qk = ["admin", "settings"];

  const { isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.admin.settingsList();
      const settings = (res.data?.settings ?? []) as SystemSettingItem[];
      const values: Record<string, string> = {};
      for (const s of settings) {
        if (s.key) values[s.key] = s.value ?? "";
      }
      form.setFieldsValue(values);
      return settings;
    },
  });

  const saveMutation = useMutation({
    mutationFn: (settings: SystemSettingItem[]) =>
      api.admin.settingsUpdate({ settings }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("admin.settings.saved"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function handleSave() {
    const values = form.getFieldsValue();
    const settings: SystemSettingItem[] = KNOWN_SETTINGS.map((ks) => ({
      key: ks.key,
      value: (values[ks.key] as string) ?? "",
    }));
    saveMutation.mutate(settings);
  }

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 700, margin: "0 auto" }}>
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ marginBottom: 4 }}>
          <SettingOutlined style={{ marginRight: 8 }} />
          {t("admin.settings.title")}
        </Title>
        <Text type="secondary">{t("admin.settings.description")}</Text>
      </div>

      <Card>
        <Form form={form} layout="vertical">
          <Title level={5}>OAuth / OIDC</Title>
          {KNOWN_SETTINGS.filter((ks) => !ks.type).map((ks) => (
            <Form.Item
              key={ks.key}
              name={ks.key}
              label={t(`admin.settings.${ks.key}`)}
            >
              {ks.secret ? (
                <Input.Password placeholder={t(`admin.settings.${ks.key}`)} />
              ) : (
                <Input placeholder={t(`admin.settings.${ks.key}`)} />
              )}
            </Form.Item>
          ))}

          <Divider />
          <Title level={5}>{t("auth.login.title")}</Title>
          {KNOWN_SETTINGS.filter((ks) => ks.type === "boolean").map((ks) => (
            <Form.Item
              key={ks.key}
              name={ks.key}
              label={t(`admin.settings.${ks.key}`)}
            >
              <Select>
                <Select.Option value="true">
                  {t("admin.settings.true")}
                </Select.Option>
                <Select.Option value="false">
                  {t("admin.settings.false")}
                </Select.Option>
              </Select>
            </Form.Item>
          ))}

          <Form.Item>
            <Button
              type="primary"
              onClick={handleSave}
              loading={saveMutation.isPending}
            >
              {t("admin.settings.save")}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
