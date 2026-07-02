import { Form, Input, Select, Switch } from "antd";
import type { FormInstance } from "antd/es/form";
import CalendarDatePicker from "@/components/CalendarDatePicker";
import type { ImportantDateTypeResponse } from "@/api";
import { canScheduleImportantDateReminder } from "@/utils/importantDatePrecision";
import type { ImportantDateFormValues } from "@/utils/importantDatePrecision";

interface ImportantDatesModuleFormProps {
  readonly form: FormInstance<ImportantDateFormValues>;
  readonly dateTypes: readonly ImportantDateTypeResponse[];
  readonly isLabelRequired: boolean;
  readonly altCalendar: boolean;
  readonly canScheduleReminder: boolean;
  readonly labels: {
    readonly dateType: string;
    readonly selectType: string;
    readonly label: string;
    readonly date: string;
    readonly remindMe: string;
  };
  readonly onSubmit: (values: ImportantDateFormValues) => void;
}

export default function ImportantDatesModuleForm({
  form,
  dateTypes,
  isLabelRequired,
  altCalendar,
  canScheduleReminder,
  labels,
  onSubmit,
}: ImportantDatesModuleFormProps) {
  return (
    <Form
      form={form}
      layout="vertical"
      onFinish={onSubmit}
      onValuesChange={(changedValues: Partial<ImportantDateFormValues>) => {
        if (!changedValues.calendarDate) {
          return;
        }

        if (!canScheduleImportantDateReminder(changedValues.calendarDate)) {
          form.setFieldValue("remind_me", false);
        }
      }}
    >
      <Form.Item name="contact_important_date_type_id" label={labels.dateType}>
        <Select
          allowClear
          placeholder={labels.selectType}
          options={dateTypes.map((dateType) => ({
            label: dateType.label,
            value: dateType.id,
          }))}
          onChange={(value: number | undefined) => {
            if (!value) {
              return;
            }

            const matched = dateTypes.find((dateType) => dateType.id === value);
            if (matched?.internal_type) {
              form.setFieldValue("label", matched.label);
            }
            if (
              matched?.internal_type === "birthdate"
              && canScheduleImportantDateReminder(form.getFieldValue("calendarDate"))
            ) {
              form.setFieldValue("remind_me", true);
            }
          }}
        />
      </Form.Item>
      <Form.Item
        name="label"
        label={labels.label}
        rules={[{ required: isLabelRequired }]}
      >
        <Input />
      </Form.Item>
      <Form.Item name="calendarDate" label={labels.date} rules={[{ required: true }]}> 
        <CalendarDatePicker
          enableAlternativeCalendar={altCalendar}
          enableNoYear
          enableDatePrecision
        />
      </Form.Item>
      <Form.Item name="remind_me" label={labels.remindMe} valuePropName="checked">
        <Switch disabled={!canScheduleReminder} />
      </Form.Item>
    </Form>
  );
}
