import {
  Card,
  Typography,
  Form,
  Select,
  Button,
  App,
  Spin,
} from "antd";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { settingsApi } from "@/api/settings";
import type { UserPreferences } from "@/types/modules";
import type { APIError } from "@/types/api";

const { Title } = Typography;

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

  const nameOrders = [
    { value: "first_last", label: t("settings.preferences.name_first_last") },
    { value: "last_first", label: t("settings.preferences.name_last_first") },
  ];

  const { data: prefs, isLoading } = useQuery({
    queryKey: ["settings", "preferences"],
    queryFn: async () => {
      const res = await settingsApi.getPreferences();
      return res.data.data!;
    },
  });

  const updateMutation = useMutation({
    mutationFn: (values: Partial<UserPreferences>) =>
      settingsApi.updatePreferences(values),
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

  return (
    <div style={{ maxWidth: 640, margin: "0 auto" }}>
      <Title level={4} style={{ marginBottom: 24 }}>
        {t("settings.preferences.title")}
      </Title>

      <Card>
        <Form
          form={form}
          layout="vertical"
          initialValues={prefs}
          onFinish={(v) => updateMutation.mutate(v)}
        >
          <Form.Item name="name_order" label={t("settings.preferences.name_order")}>
            <Select options={nameOrders} />
          </Form.Item>
          <Form.Item name="date_format" label={t("settings.preferences.date_format")}>
            <Select options={dateFormats} />
          </Form.Item>
          <Form.Item name="timezone" label={t("settings.preferences.timezone")}>
            <Select
              showSearch
              options={timezones}
              filterOption={(input, option) =>
                (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item name="locale" label={t("settings.preferences.language")}>
            <Select options={locales} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={updateMutation.isPending}>
              {t("settings.preferences.save")}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
