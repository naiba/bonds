import { useQuery } from "@tanstack/react-query";
import {
  Button,
  Collapse,
  Form,
  Input,
  Select,
  Space,
  Spin,
  Switch,
  Typography,
  theme,
} from "antd";
import { DeleteOutlined, PlusOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { api } from "@/api";
import type { ImportantDateTypeResponse, UserPreferences } from "@/api";
import CalendarDatePicker from "@/components/CalendarDatePicker";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";
import { canScheduleImportantDateReminder } from "@/utils/importantDatePrecision";
import {
  createEmptyImportantDateDraft,
  type ContactImportantDateDraft,
} from "./contactImportantDates";

const { Text } = Typography;

type ContactImportantDatesEditorProps = {
  readonly vaultId: string;
  readonly value?: readonly ContactImportantDateDraft[];
  readonly onChange?: (value: ContactImportantDateDraft[]) => void;
};

function isSingletonType(type: ImportantDateTypeResponse | undefined): boolean {
  return (
    type?.internal_type === "birthdate" ||
    type?.internal_type === "deceased_date"
  );
}

export default function ContactImportantDatesEditor({
  vaultId,
  value = [],
  onChange,
}: ContactImportantDatesEditorProps) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const { data: dateTypes = [], isLoading: dateTypesLoading } = useQuery<
    ImportantDateTypeResponse[]
  >({
    queryKey: ["vaults", vaultId, "settings", "date-types"],
    queryFn: async () => {
      const response = await api.vaultSettings.settingsDateTypesList(vaultId);
      return response.data ?? [];
    },
  });
  const { data: preferences } = useQuery<UserPreferences>({
    queryKey: ["settings", "preferences"],
    queryFn: async () => (await api.preferences.preferencesList()).data ?? {},
  });
  const alternativeCalendarEnabled =
    preferences?.enable_alternative_calendar ?? false;
  const birthdateType = dateTypes.find(
    (dateType) => dateType.internal_type === "birthdate",
  );
  const birthdateDrafts = value.filter(
    (draft) =>
      dateTypes.find(
        (dateType) => dateType.id === draft.contact_important_date_type_id,
      )?.internal_type === "birthdate",
  );
  const birthdateDraft = birthdateDrafts[0];
  const otherDrafts = value.filter(
    (draft) => draft.key !== birthdateDraft?.key,
  );

  const replaceDraft = (
    key: string,
    update: (draft: ContactImportantDateDraft) => ContactImportantDateDraft,
  ) =>
    onChange?.(
      value.map((draft) => (draft.key === key ? update(draft) : draft)),
    );

  const removeDraft = (key: string) =>
    onChange?.(value.filter((draft) => draft.key !== key));

  const changeCalendarDate = (
    draft: ContactImportantDateDraft,
    calendarDate: CalendarDatePickerValue | null,
    options: {
      readonly defaultReminder?: boolean;
      readonly removeWhenEmpty?: boolean;
    } = {},
  ) => {
    if (!calendarDate) {
      if (options.removeWhenEmpty) {
        removeDraft(draft.key);
      } else {
        replaceDraft(draft.key, (current) => ({
          ...current,
          calendarDate: undefined,
          remind_me: false,
        }));
      }
      return;
    }
    const wasSchedulable = canScheduleImportantDateReminder(draft.calendarDate);
    const isSchedulable = canScheduleImportantDateReminder(calendarDate);
    replaceDraft(draft.key, (current) => ({
      ...current,
      calendarDate,
      remind_me: isSchedulable
        ? current.remind_me ||
          (!wasSchedulable && (options.defaultReminder ?? false))
        : false,
    }));
  };

  const birthdayControl = dateTypesLoading ? (
    <Spin size="small" />
  ) : birthdateType ? (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "minmax(0, 1fr) auto",
        gap: 12,
        alignItems: "start",
      }}
    >
      <CalendarDatePicker
        value={birthdateDraft?.calendarDate}
        onChange={(calendarDate) => {
          if (birthdateDraft) {
            changeCalendarDate(birthdateDraft, calendarDate, {
              defaultReminder: true,
              removeWhenEmpty: true,
            });
            return;
          }
          if (!calendarDate || birthdateType.id === undefined) return;
          const draft = createEmptyImportantDateDraft();
          onChange?.([
            ...value,
            {
              ...draft,
              label:
                birthdateType.label ||
                t("modules.important_dates.type_birthday"),
              contact_important_date_type_id: birthdateType.id,
              calendarDate,
              remind_me: canScheduleImportantDateReminder(calendarDate),
            },
          ]);
        }}
        enableAlternativeCalendar={alternativeCalendarEnabled}
        enableNoYear
        enableDatePrecision
        maxDate={dayjs()}
      />
      {birthdateDraft && (
        <Button
          type="text"
          danger
          aria-label={t("common.delete")}
          icon={<DeleteOutlined />}
          onClick={() => removeDraft(birthdateDraft.key)}
        />
      )}
    </div>
  ) : (
    <Text type="danger">
      {t("modules.important_dates.birthdate_unavailable")}
    </Text>
  );

  const renderOtherDate = (draft: ContactImportantDateDraft) => {
    const selectedType = dateTypes.find(
      (dateType) => dateType.id === draft.contact_important_date_type_id,
    );
    const usedSingletonTypes = new Set(
      value.flatMap((candidate) => {
        if (candidate.key === draft.key) return [];
        const candidateType = dateTypes.find(
          (dateType) =>
            dateType.id === candidate.contact_important_date_type_id,
        );
        return isSingletonType(candidateType) && candidateType?.internal_type
          ? [candidateType.internal_type]
          : [];
      }),
    );
    const selectableTypes = dateTypes.filter(
      (dateType) =>
        (dateType.internal_type !== "birthdate" ||
          dateType.id === selectedType?.id) &&
        (!isSingletonType(dateType) ||
          dateType.id === selectedType?.id ||
          !usedSingletonTypes.has(dateType.internal_type ?? "")),
    );

    return (
      <div
        key={draft.key}
        style={{
          border: `1px solid ${token.colorBorderSecondary}`,
          borderRadius: token.borderRadiusLG,
          padding: 12,
          marginBottom: 12,
        }}
      >
        <div style={{ display: "flex", gap: 8, alignItems: "start" }}>
          <Form.Item
            label={t("modules.important_dates.date_type")}
            style={{ flex: 1 }}
          >
            <Select
              allowClear
              value={draft.contact_important_date_type_id}
              placeholder={t("modules.important_dates.select_type")}
              options={selectableTypes.map((dateType) => ({
                label: dateType.label,
                value: dateType.id,
              }))}
              onChange={(typeID: number | undefined) => {
                const nextType = dateTypes.find(
                  (dateType) => dateType.id === typeID,
                );
                replaceDraft(draft.key, (current) => ({
                  ...current,
                  contact_important_date_type_id: typeID,
                  label: nextType?.label ?? current.label,
                }));
              }}
            />
          </Form.Item>
          <Button
            type="text"
            danger
            aria-label={t("common.delete")}
            icon={<DeleteOutlined />}
            onClick={() => removeDraft(draft.key)}
            style={{ marginTop: 30 }}
          />
        </div>
        <Form.Item label={t("modules.important_dates.label")}>
          <Input
            value={draft.label}
            onChange={(event) =>
              replaceDraft(draft.key, (current) => ({
                ...current,
                label: event.target.value,
              }))
            }
          />
        </Form.Item>
        <Form.Item label={t("modules.important_dates.date")}>
          <CalendarDatePicker
            value={draft.calendarDate}
            onChange={(calendarDate) => changeCalendarDate(draft, calendarDate)}
            enableAlternativeCalendar={alternativeCalendarEnabled}
            enableNoYear
            enableDatePrecision
            maxDate={
              selectedType?.internal_type === "deceased_date"
                ? dayjs()
                : undefined
            }
          />
        </Form.Item>
        <Space>
          <Switch
            checked={draft.remind_me}
            disabled={!canScheduleImportantDateReminder(draft.calendarDate)}
            onChange={(checked) =>
              replaceDraft(draft.key, (current) => ({
                ...current,
                remind_me: checked,
              }))
            }
          />
          <Text>{t("modules.important_dates.remind_me")}</Text>
        </Space>
      </div>
    );
  };

  return (
    <div
      style={{
        marginBottom: 16,
        padding: 16,
        border: `1px solid ${token.colorBorderSecondary}`,
        borderRadius: token.borderRadiusLG,
        background: token.colorFillQuaternary,
      }}
    >
      <Text strong style={{ display: "block", marginBottom: 4 }}>
        {t("modules.important_dates.title")}
      </Text>
      <Text
        type="secondary"
        style={{ display: "block", fontSize: 13, marginBottom: 12 }}
      >
        {t("modules.important_dates.profile_help")}
      </Text>
      <Form.Item
        label={
          birthdateType?.label || t("modules.important_dates.type_birthday")
        }
      >
        {birthdayControl}
      </Form.Item>
      {birthdateDraft && (
        <Space style={{ marginTop: -12, marginBottom: 12 }}>
          <Switch
            checked={birthdateDraft.remind_me}
            disabled={
              !canScheduleImportantDateReminder(birthdateDraft.calendarDate)
            }
            onChange={(checked) =>
              replaceDraft(birthdateDraft.key, (current) => ({
                ...current,
                remind_me: checked,
              }))
            }
          />
          <Text>{t("modules.important_dates.remind_me")}</Text>
        </Space>
      )}
      <Collapse
        ghost
        items={[
          {
            key: "other-important-dates",
            label: t("modules.important_dates.other_dates", {
              count: otherDrafts.length,
            }),
            children: (
              <>
                {otherDrafts.map(renderOtherDate)}
                <Button
                  type="dashed"
                  block
                  icon={<PlusOutlined />}
                  onClick={() =>
                    onChange?.([...value, createEmptyImportantDateDraft()])
                  }
                >
                  {t("modules.important_dates.add")}
                </Button>
              </>
            ),
          },
        ]}
      />
    </div>
  );
}
