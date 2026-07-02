import { useState } from "react";
import {
  Card,
  List,
  Button,
  Modal,
  Form,
  Popconfirm,
  App,
  Tag,
  Empty,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { ImportantDate, CreateImportantDateRequest, APIError, UserPreferences, ImportantDateTypeResponse } from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useDateFormat } from "@/utils/dateFormat";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";
import ImportantDatesModuleForm from "./ImportantDatesModuleForm";
import { computeAgeAtImportantDate, computeImportantDateAge, formatImportantDateDisplay } from "@/utils/importantDateDisplay";
import {
  buildImportantDateRequest,
  canScheduleImportantDateReminder,
} from "@/utils/importantDatePrecision";
import type { ImportantDateFormValues } from "@/utils/importantDatePrecision";
import {
  buildDefaultImportantDatePickerValue,
  buildImportantDatePickerValue,
} from "./importantDatesModuleHelpers";

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
  const dateFormats = useDateFormat();
  const qk = ["vaults", vaultId, "contacts", contactId, "important-dates"];

  const { data: prefs } = useQuery({
    queryKey: ["settings", "preferences"],
    queryFn: async () => {
      const res = await api.preferences.preferencesList();
      return res.data as UserPreferences | undefined;
    },
  });
  const altCalendar = prefs?.enable_alternative_calendar ?? false;

  const { data: dateTypes = [] } = useQuery<ImportantDateTypeResponse[]>({
    queryKey: ["vaults", vaultId, "settings", "date-types"],
    queryFn: async () => {
      const res = await api.vaultSettings.settingsDateTypesList(String(vaultId));
      return res.data ?? [];
    },
  });

  const selectedTypeId = Form.useWatch("contact_important_date_type_id", form);
  const selectedType = dateTypes.find((dt) => dt.id === selectedTypeId);
  const isLabelRequired = !selectedType?.internal_type;
  const selectedCalendarDate = Form.useWatch<CalendarDatePickerValue | undefined>(
    "calendarDate",
    form,
  );
  const canScheduleReminder = canScheduleImportantDateReminder(selectedCalendarDate);

  const { data: dates = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.importantDates.contactsDatesList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: (values: ImportantDateFormValues) => {
      const matchedType = dateTypes.find((dt) => dt.id === values.contact_important_date_type_id);
      const data: CreateImportantDateRequest = buildImportantDateRequest(values, matchedType?.label || "");

      if (editingId) {
        return api.importantDates.contactsDatesUpdate(String(vaultId), String(contactId), editingId, data);
      }
      return api.importantDates.contactsDatesCreate(String(vaultId), String(contactId), data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      closeModal();
      message.success(editingId ? t("modules.important_dates.updated") : t("modules.important_dates.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.importantDates.contactsDatesDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.important_dates.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function openEdit(d: ImportantDate) {
    setEditingId(d.id ?? null);
    const pickerVal = buildImportantDatePickerValue(d);
    form.setFieldsValue({ label: d.label, calendarDate: pickerVal, contact_important_date_type_id: d.contact_important_date_type_id, remind_me: d.remind_me ?? false });
    setOpen(true);
  }

  function openAdd() {
    const defaultCalendarDate = buildDefaultImportantDatePickerValue(dayjs());
    form.setFieldsValue({ calendarDate: defaultCalendarDate, remind_me: false });
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
        <Button type="text" icon={<PlusOutlined />} onClick={openAdd} style={{ color: token.colorPrimary }}>
          {t("modules.important_dates.add")}
        </Button>
      }
    >
      <List
        loading={isLoading}
        dataSource={dates}
        locale={{ emptyText: <Empty description={t("modules.important_dates.no_dates")} /> }}
        split={false}
        renderItem={(d: ImportantDate) => {
          const findByInternalType = (kind: string): ImportantDate | undefined =>
            (dates as ImportantDate[]).find((x: ImportantDate) => {
              const tp = dateTypes.find((dt) => dt.id === x.contact_important_date_type_id);
              return tp?.internal_type === kind;
            });
          const matchedType = dateTypes.find((dt) => dt.id === d.contact_important_date_type_id);
          const isBirthday = matchedType?.internal_type === "birthdate";
          const isDeceasedItem = matchedType?.internal_type === "deceased_date";
          const birthDate = findByInternalType("birthdate");
          const deceasedDate = findByInternalType("deceased_date");
          const isDeceased = !!deceasedDate;
          let age: number | null = null;
          if (isBirthday && !isDeceased) {
            age = computeImportantDateAge(d);
          } else if (isDeceasedItem) {
            age = computeAgeAtImportantDate(birthDate, deceasedDate);
          }
          return (
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
                <Popconfirm key="d" title={t("modules.important_dates.delete_confirm")} onConfirm={() => deleteMutation.mutate(d.id!)}>
                  <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>,
              ]}
            >
              <List.Item.Meta
                title={
                  <span style={{ fontWeight: 500 }}>
                    {d.label}
                    {age !== null && (
                      <Tag style={{ marginLeft: 8 }}>{t("modules.important_dates.age_years", { count: age })}</Tag>
                    )}
                  </span>
                }
                 description={
                   <>
                      <span style={{ color: token.colorTextSecondary }}>{formatImportantDateDisplay(d, dateFormats)}</span>{" "}
                     {altCalendar && d.calendar_type && d.calendar_type !== "gregorian" && (
                       <Tag color="volcano">{d.calendar_type}</Tag>
                     )}
                   </>
                 }
              />
            </List.Item>
          );
        }}
      />

      <Modal
        title={editingId ? t("modules.important_dates.modal_edit") : t("modules.important_dates.modal_add")}
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
      >
        <ImportantDatesModuleForm
          form={form}
          dateTypes={dateTypes}
          isLabelRequired={isLabelRequired}
          altCalendar={altCalendar}
          canScheduleReminder={canScheduleReminder}
          labels={{
            dateType: t("modules.important_dates.date_type"),
            selectType: t("modules.important_dates.select_type"),
            label: t("modules.important_dates.label"),
            date: t("modules.important_dates.date"),
            remindMe: t("modules.important_dates.remind_me"),
          }}
          onSubmit={(values) => saveMutation.mutate(values)}
        />
      </Modal>
    </Card>
  );
}
