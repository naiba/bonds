import { Card, Typography, Button, App, Spin, Form, Input, Select, Collapse, InputNumber, Segmented } from "antd";
import { SettingOutlined, TeamOutlined, DatabaseOutlined, KeyOutlined, SearchOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { api } from "@/api";
import type { SystemSettingItem, APIError } from "@/api";

const { Title, Text } = Typography;

interface SettingDef {
  key: string;
  type?: "boolean" | "password" | "number" | "textarea" | "select";
  options?: { value: string; label: string }[];
  section: string;
  /** Explicit placeholder shown in the input; falls back to the label if omitted. */
  placeholder?: string;
}

const KNOWN_SETTINGS: SettingDef[] = [
  // Application
  { key: "app.name", section: "app" },
  { key: "app.url", section: "app" },
  { key: "announcement", type: "textarea", section: "app" },
  { key: "swagger.enabled", type: "boolean", section: "app" },

  // Authentication
  { key: "auth.password.enabled", type: "boolean", section: "auth" },
  { key: "registration.enabled", type: "boolean", section: "auth" },
  { key: "auth.require_email_verification", type: "boolean", section: "auth" },

  // JWT
  { key: "jwt.expiry_hrs", type: "number", section: "jwt" },
  { key: "jwt.refresh_hrs", type: "number", section: "jwt" },

  // SMTP
  { key: "smtp.host", section: "smtp" },
  { key: "smtp.port", section: "smtp" },
  { key: "smtp.username", section: "smtp" },
  { key: "smtp.password", type: "password", section: "smtp" },
  { key: "smtp.from", section: "smtp" },

  // WebAuthn
  { key: "webauthn.rp_id", section: "webauthn", placeholder: "e.g. bonds.example.com" },
  { key: "webauthn.rp_display_name", section: "webauthn", placeholder: "e.g. Bonds" },
  { key: "webauthn.rp_origins", section: "webauthn", placeholder: "e.g. https://bonds.example.com" },

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
  { key: "storage.default_limit_mb", type: "number", section: "storage", placeholder: "0 = unlimited" },

  // Backup
  { key: "backup.cron", section: "backup" },
  { key: "backup.retention", type: "number", section: "backup" },
];

const SECTIONS = [
  "app",
  "auth",
  "jwt",
  "smtp",
  "webauthn",
  "geocoding",
  "storage",
  "backup",
] as const;

export default function AdminSettings() {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
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
  const rebuildSearchMutation = useMutation({
    mutationFn: () => api.admin.searchRebuildCreate(),
    onSuccess: (res) => {
      message.success(
        t("admin.settings.rebuild_index_success", {
          contacts: res.data?.contacts_indexed ?? 0,
          notes: res.data?.notes_indexed ?? 0,
        })
      );
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
    const label = t(`admin.settings.${def.key}`);    const ph = def.placeholder ?? label;
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
            <Input.Password placeholder={ph} />
          </Form.Item>
        );
      case "number":
        return (
          <Form.Item key={def.key} name={def.key} label={label}>
            <InputNumber style={{ width: "100%" }} min={0} placeholder={ph} />
          </Form.Item>
        );
      case "textarea":
        return (
          <Form.Item key={def.key} name={def.key} label={label}>
            <Input.TextArea rows={3} placeholder={ph} />
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
            <Input placeholder={ph} />
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
      <Segmented
        value="settings"
        onChange={(val) => {
          if (val === "users") navigate("/admin/users");
          if (val === "backups") navigate("/admin/backups");
          if (val === "oauth-providers") navigate("/admin/oauth-providers");
        }}
        options={[
          { label: t("admin.tab_users"), value: "users", icon: <TeamOutlined /> },
          { label: t("admin.tab_settings"), value: "settings", icon: <SettingOutlined /> },
          { label: t("admin.tab_backups"), value: "backups", icon: <DatabaseOutlined /> },
          { label: t("admin.tab_oauth"), value: "oauth-providers", icon: <KeyOutlined /> },
        ]}
        style={{ marginBottom: 24 }}
      />

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

      <Card
        title={
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <SearchOutlined />
            {t("admin.settings.section_search")}
          </div>
        }
        style={{ marginTop: 24 }}
      >
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary">
            {t("admin.settings.rebuild_index_description")}
          </Text>
        </div>
        <Button
          type="primary"
          icon={<SearchOutlined />}
          onClick={() => rebuildSearchMutation.mutate()}
          loading={rebuildSearchMutation.isPending}
        >
          {t("admin.settings.rebuild_index")}
        </Button>
      </Card>
    </div>
  );
}
