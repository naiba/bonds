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
import { importantDatesApi } from "@/api/importantDates";
import type { ImportantDate, CreateImportantDateRequest } from "@/types/modules";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";
import CalendarDatePicker from "@/components/CalendarDatePicker";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";
import { getCalendarSystem } from "@/utils/calendar";
import type { CalendarType } from "@/utils/calendar";

function formatDateDisplay(d: ImportantDate): string {
  if (d.calendar_type && d.calendar_type !== "gregorian" && d.original_month != null && d.original_day != null) {
    const sys = getCalendarSystem(d.calendar_type as CalendarType);
    const formatted = sys.formatDate({
      day: d.original_day,
      month: d.original_month,
      year: d.original_year ?? 0,
    });
    const gd = d.year && d.month && d.day ? `${d.year}-${String(d.month).padStart(2, "0")}-${String(d.day).padStart(2, "0")}` : "";
    return gd ? `${formatted} (${gd})` : formatted;
  }
  if (d.year && d.month && d.day) {
    return `${d.year}-${String(d.month).padStart(2, "0")}-${String(d.day).padStart(2, "0")}`;
  }
  return "";
}

export default function ImportantDatesModule({
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
  const qk = ["vaults", vaultId, "contacts", contactId, "important-dates"];

  const dateTypes = [
    { value: "birthday", label: t("modules.important_dates.type_birthday") },
    { value: "anniversary", label: t("modules.important_dates.type_anniversary") },
    { value: "death", label: t("modules.important_dates.type_death") },
    { value: "first_met", label: t("modules.important_dates.type_first_met") },
    { value: "other", label: t("modules.important_dates.type_other") },
  ];

  const { data: dates = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await importantDatesApi.list(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: (values: { label: string; calendarDate: CalendarDatePickerValue; type: string }) => {
      const { calendarDate } = values;
      const sys = getCalendarSystem(calendarDate.calendarType);
      const gd = sys.toGregorian({ day: calendarDate.day, month: calendarDate.month, year: calendarDate.year });

      const data: CreateImportantDateRequest = {
        label: values.label,
        day: gd.day,
        month: gd.month,
        year: gd.year,
        calendar_type: calendarDate.calendarType,
      };

      if (calendarDate.calendarType !== "gregorian") {
        data.original_day = calendarDate.day;
        data.original_month = calendarDate.month;
        data.original_year = calendarDate.year;
      }

      if (editingId) {
        return importantDatesApi.update(vaultId, contactId, editingId, data);
      }
      return importantDatesApi.create(vaultId, contactId, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      closeModal();
      message.success(editingId ? t("modules.important_dates.updated") : t("modules.important_dates.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => importantDatesApi.delete(vaultId, contactId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.important_dates.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function openEdit(d: ImportantDate) {
    setEditingId(d.id);
    const ct = (d.calendar_type || "gregorian") as CalendarType;
    const pickerVal: CalendarDatePickerValue =
      ct !== "gregorian" && d.original_day != null && d.original_month != null
        ? { calendarType: ct, day: d.original_day, month: d.original_month, year: d.original_year ?? new Date().getFullYear() }
        : { calendarType: "gregorian", day: d.day ?? 1, month: d.month ?? 1, year: d.year ?? new Date().getFullYear() };
    form.setFieldsValue({ label: d.label, calendarDate: pickerVal, type: "other" });
    setOpen(true);
  }

  function closeModal() {
    setOpen(false);
    setEditingId(null);
    form.resetFields();
  }

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.important_dates.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        <Button type="text" icon={<PlusOutlined />} onClick={() => setOpen(true)} style={{ color: token.colorPrimary }}>
          {t("modules.important_dates.add")}
        </Button>
      }
    >
      <List
        loading={isLoading}
        dataSource={dates}
        locale={{ emptyText: <Empty description={t("modules.important_dates.no_dates")} /> }}
        split={false}
        renderItem={(d: ImportantDate) => (
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
              <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => openEdit(d)} />,
              <Popconfirm key="d" title={t("modules.important_dates.delete_confirm")} onConfirm={() => deleteMutation.mutate(d.id)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={<span style={{ fontWeight: 500 }}>{d.label}</span>}
              description={
                <>
                  <span style={{ color: token.colorTextSecondary }}>{formatDateDisplay(d)}</span>{" "}
                  {d.calendar_type && d.calendar_type !== "gregorian" && (
                    <Tag color="volcano">{d.calendar_type}</Tag>
                  )}
                </>
              }
            />
          </List.Item>
        )}
      />

      <Modal
        title={editingId ? t("modules.important_dates.modal_edit") : t("modules.important_dates.modal_add")}
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => saveMutation.mutate(v)}>
          <Form.Item name="label" label={t("modules.important_dates.label")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="calendarDate" label={t("modules.important_dates.date")} rules={[{ required: true }]}>
            <CalendarDatePicker />
          </Form.Item>
          <Form.Item name="type" label={t("modules.important_dates.type")} rules={[{ required: true }]}>
            <Select options={dateTypes} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
