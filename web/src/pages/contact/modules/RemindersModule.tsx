import { useState } from "react";
import {
  Card,
  List,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  App,
  Tag,
  Empty,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Reminder, CreateReminderRequest, APIError, UserPreferences } from "@/api";
import { useTranslation } from "react-i18next";
import CalendarDatePicker from "@/components/CalendarDatePicker";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";
import { getCalendarSystem } from "@/utils/calendar";
import type { CalendarType } from "@/utils/calendar";

const freqColor: Record<string, string> = {
  one_time: "blue",
  recurring_week: "green",
  recurring_month: "orange",
  recurring_year: "purple",
};

function formatReminderDate(r: Reminder): string {
  if (r.calendar_type && r.calendar_type !== "gregorian" && r.original_month != null && r.original_day != null) {
    const sys = getCalendarSystem(r.calendar_type as CalendarType);
    const formatted = sys.formatDate({
      day: r.original_day,
      month: r.original_month,
      year: r.original_year ?? 0,
    });
    const gd = r.year && r.month && r.day ? `${r.year}-${String(r.month).padStart(2, "0")}-${String(r.day).padStart(2, "0")}` : "";
    return gd ? `${formatted} (${gd})` : formatted;
  }
  if (r.year && r.month && r.day) {
    return `${r.year}-${String(r.month).padStart(2, "0")}-${String(r.day).padStart(2, "0")}`;
  }
  return "";
}

export default function RemindersModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "reminders"];

  const { data: prefs } = useQuery({
    queryKey: ["settings", "preferences"],
    queryFn: async () => {
      const res = await api.preferences.preferencesList();
      return res.data as UserPreferences | undefined;
    },
  });
  const altCalendar = prefs?.enable_alternative_calendar ?? false;

  const frequencyOptions = [
    { value: "one_time", label: t("modules.reminders.freq_one_time") },
    { value: "recurring_week", label: t("modules.reminders.freq_weekly") },
    { value: "recurring_month", label: t("modules.reminders.freq_monthly") },
    { value: "recurring_year", label: t("modules.reminders.freq_yearly") },
  ];

  const { data: reminders = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.reminders.contactsRemindersList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: (values: { label: string; calendarDate: CalendarDatePickerValue; frequency: string }) => {
      const { calendarDate } = values;
      const sys = getCalendarSystem(calendarDate.calendarType);
      const gd = sys.toGregorian({ day: calendarDate.day, month: calendarDate.month, year: calendarDate.year });

      const data: CreateReminderRequest = {
        label: values.label,
        day: gd.day,
        month: gd.month,
        year: gd.year,
        type: values.frequency,
        calendar_type: calendarDate.calendarType,
      };

      if (calendarDate.calendarType !== "gregorian") {
        data.original_day = calendarDate.day;
        data.original_month = calendarDate.month;
        data.original_year = calendarDate.year;
      }

      if (editingId) {
        return api.reminders.contactsRemindersUpdate(String(vaultId), String(contactId), editingId, data);
      }
      return api.reminders.contactsRemindersCreate(String(vaultId), String(contactId), data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      closeModal();
      message.success(editingId ? t("modules.reminders.updated") : t("modules.reminders.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.reminders.contactsRemindersDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.reminders.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function openEdit(r: Reminder) {
    setEditingId(r.id ?? null);
    const ct = (r.calendar_type || "gregorian") as CalendarType;
    const pickerVal: CalendarDatePickerValue =
      ct !== "gregorian" && r.original_day != null && r.original_month != null
        ? { calendarType: ct, day: r.original_day, month: r.original_month, year: r.original_year ?? new Date().getFullYear() }
        : { calendarType: "gregorian", day: r.day ?? 1, month: r.month ?? 1, year: r.year ?? new Date().getFullYear() };
    form.setFieldsValue({ label: r.label, calendarDate: pickerVal, frequency: r.type });
    setOpen(true);
  }

  function closeModal() {
    setOpen(false);
    setEditingId(null);
    form.resetFields();
  }

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.reminders.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        <Button type="text" icon={<PlusOutlined />} onClick={() => setOpen(true)} style={{ color: token.colorPrimary }}>
          {t("modules.reminders.add")}
        </Button>
      }
    >
      <List
        loading={isLoading}
        dataSource={reminders}
        locale={{ emptyText: <Empty description={t("modules.reminders.no_reminders")} /> }}
        split={false}
        renderItem={(r: Reminder) => (
          <List.Item
            style={{
              borderRadius: token.borderRadius,
              padding: '10px 12px',
              marginBottom: 4,
              transition: 'background 0.2s',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.background = token.colorFillQuaternary; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; }}
            actions={[
              <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => openEdit(r)} />,
              <Popconfirm key="d" title={t("modules.reminders.delete_confirm")} onConfirm={() => deleteMutation.mutate(r.id!)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={<span style={{ fontWeight: 500 }}>{r.label}</span>}
              description={
                <>
                  <span style={{ color: token.colorTextSecondary }}>{formatReminderDate(r)}</span>{" "}
                  <Tag color={freqColor[r.type!] ?? "default"}>{r.type}</Tag>
                  {altCalendar && r.calendar_type && r.calendar_type !== "gregorian" && (
                    <Tag color="volcano">{r.calendar_type}</Tag>
                  )}
                </>
              }
            />
          </List.Item>
        )}
      />

      <Modal
        title={editingId ? t("modules.reminders.modal_edit") : t("modules.reminders.modal_add")}
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => saveMutation.mutate(v)}>
          <Form.Item name="label" label={t("modules.reminders.label")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="calendarDate" label={t("modules.reminders.date")} rules={[{ required: true }]}>
            <CalendarDatePicker enableAlternativeCalendar={altCalendar} />
          </Form.Item>
          <Form.Item name="frequency" label={t("modules.reminders.frequency")} rules={[{ required: true }]}>
            <Select options={frequencyOptions} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
