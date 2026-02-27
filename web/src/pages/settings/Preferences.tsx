import { useState } from "react";
import {
  Card,
  Typography,
  Form,
  Select,
  Switch,
  Button,
  App,
  Spin,
  Divider,
  Radio,
  Input,
  theme,
} from "antd";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { UserPreferences, APIError } from "@/api";
import { formatContactName } from "@/utils/nameFormat";
import type { ContactNameFields } from "@/utils/nameFormat";

const { Title, Text } = Typography;

const dateFormats = [
  { value: "YYYY-MM-DD", label: "2026-01-15" },
  { value: "MM/DD/YYYY", label: "01/15/2026" },
  { value: "DD/MM/YYYY", label: "15/01/2026" },
  { value: "MMM D, YYYY", label: "Jan 15, 2026" },
];

const timezones = [
  "UTC",
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "Europe/London",
  "Europe/Paris",
  "Europe/Berlin",
  "Asia/Tokyo",
  "Asia/Shanghai",
  "Asia/Kolkata",
  "Australia/Sydney",
].map((tz) => ({ value: tz, label: tz }));

const locales = [
  { value: "en", label: "English" },
  { value: "fr", label: "Français" },
  { value: "de", label: "Deutsch" },
  { value: "es", label: "Español" },
  { value: "pt", label: "Português" },
  { value: "zh", label: "中文" },
  { value: "ja", label: "日本語" },
];

const mapSites = [
  { value: "google_maps", label: "Google Maps" },
  { value: "open_street_maps", label: "OpenStreetMap" },
];

const distanceFormats = [
  { value: "km", label: "Kilometers (km)" },
  { value: "mi", label: "Miles (mi)" },
];

const numberFormats = [
  { value: "1,234.56", label: "1,234.56" },
  { value: "1.234,56", label: "1.234,56" },
  { value: "1 234,56", label: "1 234,56" },
];

const NAME_ORDER_PRESETS = [
  "%first_name% %last_name%",
  "%last_name% %first_name%",
  "%first_name% %last_name% (%nickname%)",
  "%nickname%",
] as const;

const CUSTOM_SENTINEL = "__custom__";

const SAMPLE_CONTACT: ContactNameFields = {
  first_name: "James",
  last_name: "Bond",
  middle_name: "Herbert",
  nickname: "007",
  maiden_name: "",
  prefix: "",
  suffix: "",
};

export default function Preferences() {
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const { data: prefs, isLoading } = useQuery({
    queryKey: ["settings", "preferences"],
    queryFn: async () => {
      const res = await api.preferences.preferencesList();
      return res.data!;
    },
  });

  const updateMutation = useMutation({
    mutationFn: (values: Partial<UserPreferences>) =>
      api.preferences.preferencesUpdate(values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "preferences"] });
      message.success(t("settings.preferences.saved"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const initialNameOrder = prefs?.name_order || "%first_name% %last_name%";
  const isPreset = (NAME_ORDER_PRESETS as readonly string[]).includes(initialNameOrder);

  const [radioValue, setRadioValue] = useState<string>(
    isPreset ? initialNameOrder : CUSTOM_SENTINEL,
  );
  const [customTemplate, setCustomTemplate] = useState<string>(
    isPreset ? "" : initialNameOrder,
  );
  const activeTemplate = radioValue === CUSTOM_SENTINEL ? customTemplate : radioValue;

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  const labelStyle: React.CSSProperties = {
    fontWeight: 500,
    color: token.colorText,
  };

  const presetLabels: Record<string, string> = {
    [NAME_ORDER_PRESETS[0]]: t("settings.preferences.name_order_first_last"),
    [NAME_ORDER_PRESETS[1]]: t("settings.preferences.name_order_last_first"),
    [NAME_ORDER_PRESETS[2]]: t("settings.preferences.name_order_first_last_nickname"),
    [NAME_ORDER_PRESETS[3]]: t("settings.preferences.name_order_nickname"),
  };

  const presetExamples: Record<string, string> = {
    [NAME_ORDER_PRESETS[0]]: "James Bond",
    [NAME_ORDER_PRESETS[1]]: "Bond James",
    [NAME_ORDER_PRESETS[2]]: "James Bond (007)",
    [NAME_ORDER_PRESETS[3]]: "007",
  };
  const handleFinish = (values: Record<string, unknown>) => {
    updateMutation.mutate({ ...values, name_order: activeTemplate } as Partial<UserPreferences>);
  };

  return (
    <div style={{ maxWidth: 640, margin: "0 auto" }}>
      <Title level={4} style={{ marginBottom: 4 }}>
        {t("settings.preferences.title")}
      </Title>
      <Text type="secondary" style={{ display: "block", marginBottom: 24 }}>
        {t("settings.preferences.description")}
      </Text>

      <Card>
        <Form
          form={form}
          layout="vertical"
          initialValues={prefs}
          onFinish={handleFinish}
        >
          <Form.Item
            label={<span style={labelStyle}>{t("settings.preferences.name_order")}</span>}
          >
            <Radio.Group
              value={radioValue}
              onChange={(e) => {
                setRadioValue(e.target.value as string);
              }}
              style={{ display: "flex", flexDirection: "column", gap: 8 }}
            >
              {NAME_ORDER_PRESETS.map((preset) => (
                <Radio key={preset} value={preset}>
                  <span style={{ fontWeight: 500 }}>{presetLabels[preset]}</span>
                  <Text type="secondary" style={{ marginLeft: 8, fontSize: 13 }}>
                    — {presetExamples[preset]}
                  </Text>
                </Radio>
              ))}
              <Radio value={CUSTOM_SENTINEL}>
                <span style={{ fontWeight: 500 }}>
                  {t("settings.preferences.name_order_custom")}
                </span>
              </Radio>
            </Radio.Group>

            {radioValue === CUSTOM_SENTINEL && (
              <div style={{ marginTop: 12, paddingLeft: 24 }}>
                <Input
                  value={customTemplate}
                  onChange={(e) => setCustomTemplate(e.target.value)}
                  placeholder="%first_name% %last_name%"
                  style={{ marginBottom: 8 }}
                />
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {t("settings.preferences.name_order_custom_help")}
                </Text>
              </div>
            )}

            {/* Live preview */}
            {activeTemplate && (
              <div
                style={{
                  marginTop: 12,
                  padding: "8px 12px",
                  background: token.colorFillQuaternary,
                  borderRadius: token.borderRadius,
                  display: "flex",
                  alignItems: "center",
                  gap: 8,
                }}
              >
                <Text type="secondary" style={{ fontSize: 13, flexShrink: 0 }}>
                  {t("settings.preferences.name_order_preview")}:
                </Text>
                <Text strong style={{ fontSize: 14 }}>
                  {formatContactName(activeTemplate, SAMPLE_CONTACT)}
                </Text>
              </div>
            )}
          </Form.Item>

          <Form.Item
            name="date_format"
            label={<span style={labelStyle}>{t("settings.preferences.date_format")}</span>}
          >
            <Select options={dateFormats} />
          </Form.Item>

          <Divider style={{ margin: "8px 0 24px" }} />

          <Form.Item
            name="timezone"
            label={<span style={labelStyle}>{t("settings.preferences.timezone")}</span>}
          >
            <Select
              showSearch
              options={timezones}
              filterOption={(input, option) =>
                (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item
            name="locale"
            label={<span style={labelStyle}>{t("settings.preferences.language")}</span>}
          >
            <Select options={locales} />
          </Form.Item>

          <Divider style={{ margin: "8px 0 24px" }} />

          <Form.Item
            name="default_map_site"
            label={<span style={labelStyle}>{t("settings.preferences.default_map_site")}</span>}
          >
            <Select options={mapSites} />
          </Form.Item>
          <Form.Item
            name="distance_format"
            label={<span style={labelStyle}>{t("settings.preferences.distance_format")}</span>}
          >
            <Select options={distanceFormats} />
          </Form.Item>
          <Form.Item
            name="number_format"
            label={<span style={labelStyle}>{t("settings.preferences.number_format")}</span>}
          >
            <Select options={numberFormats} />
          </Form.Item>

          <Form.Item
            name="enable_alternative_calendar"
            label={<span style={labelStyle}>{t("settings.preferences.enable_alternative_calendar")}</span>}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>

          <Divider style={{ margin: "8px 0 24px" }} />

          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" loading={updateMutation.isPending}>
              {t("settings.preferences.save")}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
