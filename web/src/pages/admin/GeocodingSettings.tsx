import {
  Alert,
  App,
  Button,
  Card,
  Form,
  Input,
  Popconfirm,
  Select,
  Space,
  Spin,
  Tag,
  Typography,
} from "antd";
import { DeleteOutlined, SaveOutlined } from "@ant-design/icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type {
  APIError,
  GeocodingAdminSettings,
  GeocodingProvider,
} from "@/api";

const { Link, Paragraph, Text } = Typography;

function providerFormValues(data: GeocodingAdminSettings) {
  const providers: Record<string, Record<string, string>> = {};
  for (const provider of data.providers ?? []) {
    if (provider.id !== undefined) {
      providers[provider.id] = provider.config ?? {};
    }
  }
  return {
    active_provider: data.active_provider ?? "",
    precision: data.precision ?? "exact",
    providers,
  };
}

export default function GeocodingSettings() {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const form = Form.useFormInstance();
  const precision = Form.useWatch("precision", form);
  const queryKey = ["admin", "geocoding"];

  const { data, isLoading } = useQuery({
    queryKey,
    queryFn: async () => {
      const response = await api.admin.geocodingList();
      const settings: GeocodingAdminSettings = response.data ?? {};
      form.setFieldsValue(providerFormValues(settings));
      return settings;
    },
  });

  const settingsMutation = useMutation({
    mutationFn: (settings: {
      active_provider: string;
      precision: "exact" | "locality";
    }) => api.admin.geocodingUpdate(settings),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
      message.success(t("admin.settings.geocoding.settings_saved"));
    },
    onError: (error: APIError) => message.error(error.message),
  });

  const providerMutation = useMutation({
    mutationFn: ({
      provider,
      config,
    }: {
      provider: string;
      config: Record<string, string>;
    }) => api.admin.geocodingProvidersUpdate(provider, { config }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
      message.success(t("admin.settings.geocoding.provider_saved"));
    },
    onError: (error: APIError) => message.error(error.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (provider: string) =>
      api.admin.geocodingProvidersDelete(provider),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
      message.success(t("admin.settings.geocoding.provider_reset"));
    },
    onError: (error: APIError) => message.error(error.message),
  });

  async function saveSettings() {
    try {
      await form.validateFields(["active_provider", "precision"]);
    } catch {
      return;
    }
    const activeValue: unknown = form.getFieldValue("active_provider");
    const precisionValue: unknown = form.getFieldValue("precision");
    settingsMutation.mutate({
      active_provider: typeof activeValue === "string" ? activeValue : "",
      precision: precisionValue === "locality" ? "locality" : "exact",
    });
  }

  async function saveProvider(provider: GeocodingProvider) {
    if (provider.id === undefined) return;
    const providerID = provider.id;
    const config: Record<string, string> = {};
    const fieldPaths = (provider.fields ?? [])
      .filter((field) => field.key !== undefined)
      .map((field) => ["providers", providerID, field.key]);
    try {
      await form.validateFields(fieldPaths);
    } catch {
      return;
    }
    for (const field of provider.fields ?? []) {
      if (field.key === undefined) continue;
      const value: unknown = form.getFieldValue([
        "providers",
        providerID,
        field.key,
      ]);
      config[field.key] = typeof value === "string" ? value : "";
    }
    providerMutation.mutate({ provider: providerID, config });
  }

  if (isLoading) {
    return <Spin />;
  }

  const providers = data?.providers ?? [];

  return (
    <>
      <Alert
        type="info"
        showIcon
        title={t("admin.settings.geocoding.privacy_title")}
        description={
          precision === "locality"
            ? t("admin.settings.geocoding.privacy_locality")
            : t("admin.settings.geocoding.privacy_exact")
        }
        style={{ marginBottom: 16 }}
      />

      <Form.Item
        name="active_provider"
        label={t("admin.settings.geocoding.provider")}
      >
        <Select
          options={[
            { value: "", label: t("admin.settings.geocoding.none") },
            ...providers.map((provider) => ({
              value: provider.id ?? "",
              label: provider.name ?? provider.id ?? "",
              disabled: !provider.configured,
            })),
          ]}
        />
      </Form.Item>
      <Form.Item
        name="precision"
        label={t("admin.settings.geocoding.precision")}
      >
        <Select
          options={[
            {
              value: "exact",
              label: t("admin.settings.geocoding.precision_exact"),
            },
            {
              value: "locality",
              label: t("admin.settings.geocoding.precision_locality"),
            },
          ]}
        />
      </Form.Item>
      <Button
        type="primary"
        icon={<SaveOutlined />}
        onClick={saveSettings}
        loading={settingsMutation.isPending}
        style={{ marginBottom: 20 }}
      >
        {t("admin.settings.geocoding.save_settings")}
      </Button>

      <Space orientation="vertical" size="middle" style={{ display: "flex" }}>
        {providers.map((provider) => {
          const providerID = provider.id ?? "";
          const saving =
            providerMutation.isPending &&
            providerMutation.variables?.provider === providerID;
          const deleting =
            deleteMutation.isPending && deleteMutation.variables === providerID;
          return (
            <Card
              key={providerID}
              size="small"
              title={
                <Space wrap>
                  <Text strong>{provider.name ?? providerID}</Text>
                  <Tag color={provider.configured ? "green" : "default"}>
                    {provider.configured
                      ? t("admin.settings.geocoding.configured")
                      : t("admin.settings.geocoding.not_configured")}
                  </Tag>
                  <Tag
                    color={provider.supports_autocomplete ? "blue" : "orange"}
                  >
                    {provider.supports_autocomplete
                      ? t("admin.settings.geocoding.autocomplete_supported")
                      : t("admin.settings.geocoding.autocomplete_unavailable")}
                  </Tag>
                </Space>
              }
            >
              <Paragraph type="secondary">
                {t(
                  `admin.settings.geocoding.providers.${providerID}.description`,
                )}
              </Paragraph>

              {provider.notice === "public_nominatim" && (
                <Alert
                  type="warning"
                  showIcon
                  title={t("admin.settings.geocoding.nominatim_notice_title")}
                  description={t("admin.settings.geocoding.nominatim_notice")}
                  style={{ marginBottom: 16 }}
                />
              )}
              {provider.notice === "public_demo" && (
                <Alert
                  type="warning"
                  showIcon
                  title={t("admin.settings.geocoding.photon_notice_title")}
                  description={t("admin.settings.geocoding.photon_notice")}
                  style={{ marginBottom: 16 }}
                />
              )}

              {(provider.fields ?? []).map((field) => {
                const fieldKey = field.key ?? "";
                const extra =
                  field.secret && provider.config?.[fieldKey] === "***"
                    ? t("admin.settings.geocoding.secret_hint")
                    : undefined;
                return (
                  <Form.Item
                    key={fieldKey}
                    name={["providers", providerID, fieldKey]}
                    label={t(`admin.settings.geocoding.fields.${fieldKey}`)}
                    extra={extra}
                    rules={field.required ? [{ required: true }] : undefined}
                  >
                    {field.type === "password" ? (
                      <Input.Password />
                    ) : (
                      <Input type={field.type === "url" ? "url" : "text"} />
                    )}
                  </Form.Item>
                );
              })}

              {(provider.fields?.length ?? 0) > 0 && (
                <Space wrap style={{ marginBottom: 12 }}>
                  <Button
                    type="primary"
                    icon={<SaveOutlined />}
                    onClick={() => saveProvider(provider)}
                    loading={saving}
                  >
                    {t("admin.settings.geocoding.save_provider")}
                  </Button>
                  {provider.has_stored_config && (
                    <Popconfirm
                      title={t("admin.settings.geocoding.reset_confirm")}
                      onConfirm={() => deleteMutation.mutate(providerID)}
                    >
                      <Button
                        danger
                        icon={<DeleteOutlined />}
                        loading={deleting}
                      >
                        {t("admin.settings.geocoding.reset_provider")}
                      </Button>
                    </Popconfirm>
                  )}
                </Space>
              )}

              <div>
                <Text type="secondary">
                  {t("admin.settings.geocoding.attribution")}:{" "}
                </Text>
                <Space size="small" wrap>
                  {(provider.attribution ?? []).map((credit) => (
                    <Link
                      key={credit.url ?? credit.label}
                      href={credit.url}
                      target="_blank"
                      rel="noreferrer"
                    >
                      {credit.label}
                    </Link>
                  ))}
                </Space>
              </div>
            </Card>
          );
        })}
      </Space>
    </>
  );
}
