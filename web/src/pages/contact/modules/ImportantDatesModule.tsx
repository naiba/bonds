import { useState } from "react";
import { Card, Button, Modal, Form, theme } from "antd";
import { PlusOutlined } from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api";
import type {
  ImportantDate,
  UserPreferences,
  ImportantDateTypeResponse,
} from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useDateFormat } from "@/utils/dateFormat";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";
import ImportantDatesModuleForm from "./ImportantDatesModuleForm";
import {
  buildImportantDateRequest,
  canScheduleImportantDateReminder,
} from "@/utils/importantDatePrecision";
import type { ImportantDateFormValues } from "@/utils/importantDatePrecision";
import {
  buildDefaultImportantDatePickerValue,
  buildImportantDatePickerValue,
} from "./importantDatesModuleHelpers";
import {
  queryKeyPrefixes,
  type ContactQueryScope,
  type QueryInvalidationScopes,
} from "@/utils/queryInvalidation";
import { createContactSaveMutationOperation } from "./contactSaveMutationOperation";
import ImportantDateList from "./ImportantDateList";
import { useImportantDateMutations } from "./useImportantDateMutations";

export default function ImportantDatesModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form] = Form.useForm<ImportantDateFormValues>();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const dateFormats = useDateFormat();
  const scope = {
    vaultId: String(vaultId),
    contactId: String(contactId),
  } as const satisfies ContactQueryScope;
  const affectedScopes = {
    vaultIds: [scope.vaultId],
    contacts: [scope],
  } as const satisfies QueryInvalidationScopes;
  const qk = queryKeyPrefixes.calendar.contact(scope);

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
      const res = await api.vaultSettings.settingsDateTypesList(
        String(vaultId),
      );
      return res.data ?? [];
    },
  });

  const selectedTypeId = Form.useWatch("contact_important_date_type_id", form);
  const selectedType = dateTypes.find((dt) => dt.id === selectedTypeId);
  const isLabelRequired = !selectedType?.internal_type;
  const selectedCalendarDate = Form.useWatch<
    CalendarDatePickerValue | undefined
  >("calendarDate", form);
  const canScheduleReminder =
    canScheduleImportantDateReminder(selectedCalendarDate);

  const { data: dates = [], isLoading } = useQuery<ImportantDate[]>({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.importantDates.contactsDatesList(
        String(vaultId),
        String(contactId),
      );
      return res.data ?? [];
    },
  });

  const selectableDateTypes = dateTypes.filter((dateType) => {
    if (
      dateType.internal_type !== "birthdate" &&
      dateType.internal_type !== "deceased_date"
    ) {
      return true;
    }
    const editingDate = dates.find((date) => date.id === editingId);
    if (editingDate?.contact_important_date_type_id === dateType.id) {
      return true;
    }
    return !dates.some(
      (date) => date.contact_important_date_type_id === dateType.id,
    );
  });

  function openEdit(d: ImportantDate) {
    setEditingId(d.id ?? null);
    const pickerVal = buildImportantDatePickerValue(d);
    form.setFieldsValue({
      label: d.label,
      calendarDate: pickerVal,
      contact_important_date_type_id: d.contact_important_date_type_id,
      remind_me: d.remind_me ?? false,
    });
    setOpen(true);
  }

  function openAdd() {
    const defaultCalendarDate = buildDefaultImportantDatePickerValue(dayjs());
    form.setFieldsValue({
      calendarDate: defaultCalendarDate,
      remind_me: false,
    });
    setOpen(true);
  }

  function closeModal() {
    setOpen(false);
    setEditingId(null);
    form.resetFields();
  }

  const { saveMutation, deleteMutation } = useImportantDateMutations({
    closeSaveForm: closeModal,
  });

  return (
    <Card
      title={
        <span style={{ fontWeight: 500 }}>
          {t("modules.important_dates.title")}
        </span>
      }
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: "16px 24px" },
      }}
      extra={
        <Button
          type="text"
          icon={<PlusOutlined />}
          onClick={openAdd}
          style={{ color: token.colorPrimary }}
        >
          {t("modules.important_dates.add")}
        </Button>
      }
    >
      <ImportantDateList
        dates={dates}
        dateTypes={dateTypes}
        isLoading={isLoading}
        alternativeCalendarEnabled={altCalendar}
        dateFormats={dateFormats}
        onEdit={openEdit}
        onDelete={(id) => {
          deleteMutation.mutate({
            kind: "delete",
            source: scope,
            listQueryKey: qk,
            affectedScopes,
            id,
          });
        }}
      />

      <Modal
        title={
          editingId
            ? t("modules.important_dates.modal_edit")
            : t("modules.important_dates.modal_add")
        }
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
      >
        <ImportantDatesModuleForm
          form={form}
          dateTypes={selectableDateTypes}
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
          onSubmit={(values) => {
            const matchedType = dateTypes.find(
              (dateType) =>
                dateType.id === values.contact_important_date_type_id,
            );
            saveMutation.mutate({
              ...createContactSaveMutationOperation(editingId, values),
              source: scope,
              listQueryKey: qk,
              affectedScopes,
              request: buildImportantDateRequest(
                values,
                matchedType?.label ?? "",
              ),
            });
          }}
        />
      </Modal>
    </Card>
  );
}
