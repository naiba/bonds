import { Card, Typography, Button, App, Spin, Form, Input, Select, Collapse, InputNumber } from "antd";
import { SettingOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { SystemSettingItem, APIError } from "@/api";

const { Title, Text } = Typography;

interface SettingDef {
  key: string;
  type?: "boolean" | "password" | "number" | "textarea" | "select";
  options?: { value: string; label: string }[];
  section: string;
}

const KNOWN_SETTINGS: SettingDef[] = [
  // Application
  { key: "app.name", section: "app" },
  { key: "app.url", section: "app" },
  { key: "announcement", type: "textarea", section: "app" },

  // Authentication
  { key: "password_auth_enabled", type: "boolean", section: "auth" },
  { key: "registration_enabled", type: "boolean", section: "auth" },

  // JWT
  { key: "jwt.expiry_hrs", type: "number", section: "jwt" },
  { key: "jwt.refresh_hrs", type: "number", section: "jwt" },

  // SMTP
  { key: "smtp.host", section: "smtp" },
  { key: "smtp.port", section: "smtp" },
  { key: "smtp.username", section: "smtp" },
  { key: "smtp.password", type: "password", section: "smtp" },
  { key: "smtp.from", section: "smtp" },

  // OAuth / OIDC
  { key: "oauth_github_key", section: "oauth" },
  { key: "oauth_github_secret", type: "password", section: "oauth" },
  { key: "oauth_google_key", section: "oauth" },
  { key: "oauth_google_secret", type: "password", section: "oauth" },
  { key: "oidc_client_id", section: "oauth" },
  { key: "oidc_client_secret", type: "password", section: "oauth" },
  { key: "oidc_discovery_url", section: "oauth" },
  { key: "oidc_name", section: "oauth" },

  // WebAuthn
  { key: "webauthn.rp_id", section: "webauthn" },
  { key: "webauthn.rp_display_name", section: "webauthn" },
  { key: "webauthn.rp_origins", section: "webauthn" },

  // Telegram
  { key: "telegram.bot_token", type: "password", section: "telegram" },

  // Geocoding
  {
    key: "geocoding.provider",
    type: "select",
    section: "geocoding",
    options: [
      { value: "", label: "admin.settings.geocoding.none" },
      { value: "nominatim", label: "admin.settings.geocoding.nominatim" },
      { value: "locationiq", label: "admin.settings.geocoding.locationiq" },
    ],
  },
  { key: "geocoding.api_key", type: "password", section: "geocoding" },

  // Storage
  { key: "storage.max_size", type: "number", section: "storage" },

  // Backup
  { key: "backup.cron", section: "backup" },
  { key: "backup.retention", type: "number", section: "backup" },
];

const SECTIONS = [
  "app",
  "auth",
  "jwt",
  "smtp",
  "oauth",
  "webauthn",
  "telegram",
  "geocoding",
  "storage",
  "backup",
] as const;

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
      value: values[ks.key] != null ? String(values[ks.key]) : "",
    }));
    saveMutation.mutate(settings);
  }

  function renderField(def: SettingDef) {
    const label = t(`admin.settings.${def.key}`);
    switch (def.type) {
      case "boolean":
        return (
          <Form.Item key={def.key} name={def.key} label={label}>
            <Select>
              <Select.Option value="true">
                {t("admin.settings.true")}
              </Select.Option>
              <Select.Option value="false">
                {t("admin.settings.false")}
              </Select.Option>
            </Select>
          </Form.Item>
        );
      case "password":
        return (
          <Form.Item key={def.key} name={def.key} label={label}>
            <Input.Password placeholder={label} />
          </Form.Item>
        );
      case "number":
        return (
          <Form.Item key={def.key} name={def.key} label={label}>
            <InputNumber style={{ width: "100%" }} min={0} placeholder={label} />
          </Form.Item>
        );
      case "textarea":
        return (
          <Form.Item key={def.key} name={def.key} label={label}>
            <Input.TextArea rows={3} placeholder={label} />
          </Form.Item>
        );
      case "select":
        return (
          <Form.Item key={def.key} name={def.key} label={label}>
            <Select>
              {def.options?.map((opt) => (
                <Select.Option key={opt.value} value={opt.value}>
                  {t(opt.label)}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        );
      default:
        return (
          <Form.Item key={def.key} name={def.key} label={label}>
            <Input placeholder={label} />
          </Form.Item>
        );
    }
  }

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  const collapseItems = SECTIONS.map((section) => ({
    key: section,
    label: t(`admin.settings.section_${section}`),
    children: (
      <>
        {KNOWN_SETTINGS.filter((s) => s.section === section).map(renderField)}
      </>
    ),
  }));

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
          <Collapse
            defaultActiveKey={["app", "auth"]}
            items={collapseItems}
            style={{ marginBottom: 24 }}
          />

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
