import {
  Card,
  Typography,
  Form,
  Select,
  Button,
  App,
  Spin,
  Divider,
  theme,
} from "antd";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { UserPreferences, APIError } from "@/api";

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

export default function Preferences() {
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const nameOrders = [
    { value: "first_last", label: t("settings.preferences.name_first_last") },
    { value: "last_first", label: t("settings.preferences.name_last_first") },
  ];

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
          onFinish={(v) => updateMutation.mutate(v)}
        >
          <Form.Item
            name="name_order"
            label={<span style={labelStyle}>{t("settings.preferences.name_order")}</span>}
          >
            <Select options={nameOrders} />
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
