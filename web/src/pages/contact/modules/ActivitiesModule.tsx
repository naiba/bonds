import { useMemo, useState } from "react";
import {
  App,
  Button,
  Card,
  DatePicker,
  Empty,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Timeline,
  Typography,
} from "antd";
import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import dayjs, { type Dayjs } from "dayjs";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router-dom";
import { api } from "@/api";
import type {
  APIError,
  ContactSearchItem,
  Activity,
  ActivityCategoryResponse,
  PaginationMeta,
  UserPreferences,
} from "@/api";
import ContactMentionEditor from "@/components/journal/ContactMentionEditor";
import ContactMentionText from "@/components/journal/ContactMentionText";
import CalendarDatePicker from "@/components/CalendarDatePicker";
import { dateInputToTimestamp } from "@/utils/dateOnlyInput";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";
import { getCalendarSystem } from "@/utils/calendar";
import { formatDate, formatMonthYear, useDateFormat } from "@/utils/dateFormat";

const { Text } = Typography;

type Precision = "day" | "month" | "year";
type EndStatus = "none" | "known" | "ongoing" | "unknown";
type FormValues = {
  activity_type_id: number;
  title: string;
  description: string;
  start_calendar: CalendarDatePickerValue;
  end_status: EndStatus;
  end_precision?: Precision;
  end_date?: Dayjs;
  parent_id?: number;
  participant_ids: string[];
  duration_in_minutes?: number;
  place?: string;
};

export default function ActivitiesModule({
  vaultId,
  contactId,
  initiallyOpen = false,
  onModalClose,
}: {
  vaultId: string | number;
  contactId?: string | number;
  initiallyOpen?: boolean;
  onModalClose?: () => void;
  target?: unknown;
}) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const dateFormats = useDateFormat();
  const location = useLocation();
  const [form] = Form.useForm<FormValues>();
  const [open, setOpen] = useState(initiallyOpen);
  const [editing, setEditing] = useState<Activity | null>(null);
  const [description, setDescription] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<Activity[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const queryKey = ["vaults", vaultId, "activities", contactId] as const;

  const { isLoading, isFetching } = useQuery({
    queryKey: [...queryKey, page],
    queryFn: async () => {
      const response = await api.activities.activitiesList(String(vaultId), {
        contact_id: contactId == null ? undefined : String(contactId),
        page,
        per_page: 15,
      });
      const next = response.data ?? [];
      setItems((current) => (page === 1 ? next : [...current, ...next]));
      const meta = response.meta as PaginationMeta | undefined;
      setHasMore((meta?.page ?? page) < (meta?.total_pages ?? page));
      return next;
    },
  });

  const { data: contacts = [] } = useQuery<ContactSearchItem[]>({
    queryKey: ["vaults", vaultId, "contacts", "activity-picker"],
    queryFn: async () =>
      (await api.contacts.contactsSelectableList(String(vaultId), {})).data ??
      [],
  });
  const contactOptions = contacts
    .filter(
      (contact) =>
        contact.id != null && String(contact.id) !== String(contactId ?? ""),
    )
    .map((contact) => ({
      value: String(contact.id),
      label: contact.name || String(contact.id),
    }));

  const { data: categories = [] } = useQuery({
    queryKey: ["vault", vaultId, "activityCategories"],
    queryFn: async () => {
      const response = await api.vaultSettings.settingsActivityCategoriesList(
        String(vaultId),
      );
      return (response.data ?? []) as ActivityCategoryResponse[];
    },
  });

  const { data: preferences } = useQuery<UserPreferences | undefined>({
    queryKey: ["settings", "preferences"],
    queryFn: async () =>
      (await api.preferences.preferencesList()).data ?? undefined,
  });
  const enableAlternativeCalendar =
    preferences?.enable_alternative_calendar ?? false;

  const typeOptions = useMemo(
    () =>
      categories.map((category) => ({
        label: category.label,
        options: (category.types ?? []).map((type) => ({
          value: type.id,
          label: type.label,
        })),
      })),
    [categories],
  );
  const parentOptions = useMemo(
    () =>
      items
        .filter((item) => !item.parent_id && item.id !== editing?.id)
        .map((item) => ({ value: item.id, label: item.title })),
    [editing?.id, items],
  );

  const resetList = async () => {
    setPage(1);
    setItems([]);
    await queryClient.invalidateQueries({ queryKey });
  };

  const saveMutation = useMutation({
    mutationFn: async (values: FormValues) => {
      const start = values.start_calendar;
      const startPrecision =
        start.datePrecision === "year"
          ? "year"
          : start.datePrecision === "month"
            ? "month"
            : "day";
      const gregorianStart =
        start.year == null
          ? undefined
          : getCalendarSystem(start.calendarType).toGregorian({
              year: start.year,
              month: start.month ?? 1,
              day: start.day ?? 1,
            });
      const payload = {
        primary_contact_id: contactId == null ? "" : String(contactId),
        participant_ids: values.participant_ids,
        parent_id: values.parent_id,
        activity_type_id: values.activity_type_id,
        title: values.title,
        description,
        start_date: gregorianStart
          ? dateInputToTimestamp(
              `${gregorianStart.year}-${String(gregorianStart.month).padStart(2, "0")}-${String(gregorianStart.day).padStart(2, "0")}`,
            )
          : undefined,
        start_precision: startPrecision as Precision,
        calendar_type: start.calendarType,
        original_day:
          start.calendarType === "gregorian"
            ? undefined
            : (start.day ?? undefined),
        original_month:
          start.calendarType === "gregorian"
            ? undefined
            : (start.month ?? undefined),
        original_year:
          start.calendarType === "gregorian"
            ? undefined
            : (start.year ?? undefined),
        end_status: values.end_status,
        end_date:
          values.end_status === "known" && values.end_date
            ? dateInputToTimestamp(values.end_date.format("YYYY-MM-DD"))
            : undefined,
        end_precision:
          values.end_status === "known" ? values.end_precision : undefined,
        duration_in_minutes: values.duration_in_minutes,
        place: values.place,
      };
      return editing?.id
        ? api.activities.activitiesUpdate(String(vaultId), editing.id, payload)
        : api.activities.activitiesCreate(String(vaultId), payload);
    },
    onSuccess: async () => {
      await resetList();
      setOpen(false);
      onModalClose?.();
      setEditing(null);
      setDescription("");
      form.resetFields();
      message.success(t("modules.activities.saved"));
    },
    onError: (error: APIError) => message.error(error.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) =>
      api.activities.activitiesDelete(String(vaultId), id),
    onSuccess: async () => {
      await resetList();
      message.success(t("modules.activities.deleted"));
    },
    onError: (error: APIError) => message.error(error.message),
  });

  const startCreate = () => {
    setEditing(null);
    setDescription("");
    form.setFieldsValue({
      start_calendar: {
        calendarType: "gregorian",
        year: dayjs().year(),
        month: dayjs().month() + 1,
        day: dayjs().date(),
        datePrecision: "full",
      },
      end_status: "none",
      participant_ids: [],
    });
    setOpen(true);
  };
  const startEdit = (activity: Activity) => {
    setEditing(activity);
    setDescription(activity.description ?? "");
    form.setFieldsValue({
      activity_type_id: activity.activity_type_id,
      title: activity.title,
      description: activity.description,
      parent_id: activity.parent_id,
      start_calendar: {
        calendarType:
          activity.calendar_type === "lunar" ? "lunar" : "gregorian",
        year:
          activity.calendar_type !== "gregorian" && activity.original_year
            ? activity.original_year
            : activity.start_date
              ? dayjs(activity.start_date).year()
              : null,
        month:
          activity.calendar_type !== "gregorian" && activity.original_month
            ? activity.original_month
            : activity.start_date
              ? dayjs(activity.start_date).month() + 1
              : null,
        day:
          activity.calendar_type !== "gregorian" && activity.original_day
            ? activity.original_day
            : activity.start_date
              ? dayjs(activity.start_date).date()
              : null,
        datePrecision:
          activity.start_precision === "year"
            ? "year"
            : activity.start_precision === "month"
              ? "month"
              : "full",
      },
      end_status: (activity.end_status as EndStatus) || "none",
      end_precision: (activity.end_precision as Precision) || "day",
      end_date: activity.end_date ? dayjs(activity.end_date) : undefined,
      participant_ids: (activity.participants ?? []).flatMap((contact) =>
        contact.id && String(contact.id) !== String(contactId ?? "")
          ? [contact.id]
          : [],
      ),
      duration_in_minutes: activity.duration_in_minutes,
      place: activity.place,
    });
    setOpen(true);
  };

  const endStatus = Form.useWatch("end_status", form) ?? "none";
  const endPrecision = Form.useWatch("end_precision", form) ?? "day";
  const formatActivityTime = (activity: Activity) => {
    if (!activity.start_date) return t("modules.activities.date_unknown");
    const start =
      activity.start_precision === "year"
        ? dayjs(activity.start_date).year().toString()
        : activity.start_precision === "month"
          ? formatMonthYear(activity.start_date, dateFormats)
          : formatDate(activity.start_date, dateFormats);
    if (activity.end_status === "ongoing")
      return `${start} – ${t("modules.activities.present")}`;
    if (activity.end_status === "unknown")
      return `${start} – ${t("modules.activities.end_unknown")}`;
    if (activity.end_status !== "known" || !activity.end_date) return start;
    const end =
      activity.end_precision === "year"
        ? dayjs(activity.end_date).year().toString()
        : activity.end_precision === "month"
          ? formatMonthYear(activity.end_date, dateFormats)
          : formatDate(activity.end_date, dateFormats);
    return `${start} – ${end}`;
  };
  const activityPath = (activity: Activity) =>
    `/vaults/${vaultId}/activities/${activity.id}`;
  const activityRouteState = {
    activityReturnTo: `${location.pathname}${location.search}`,
  };
  return (
    <Card
      title={t("modules.activities.title")}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={startCreate}>
          {t("modules.activities.add")}
        </Button>
      }
      loading={isLoading && items.length === 0}
    >
      {items.length === 0 ? (
        <Empty
          description={t(
            contactId == null
              ? "modules.activities.empty_description_vault"
              : "modules.activities.empty_description",
          )}
        >
          <Button type="primary" onClick={startCreate}>
            {t("modules.activities.add_first")}
          </Button>
        </Empty>
      ) : (
        <Timeline
          items={items.map((activity) => ({
            content: (
              <div>
                <Space
                  style={{ width: "100%", justifyContent: "space-between" }}
                >
                  <div>
                    <Link
                      to={activityPath(activity)}
                      state={activityRouteState}
                      style={{ fontWeight: 600 }}
                    >
                      {activity.title}
                    </Link>
                    <Text type="secondary" style={{ display: "block" }}>
                      {formatActivityTime(activity)}
                    </Text>
                    {activity.activity_type?.label && (
                      <Text type="secondary">
                        {activity.activity_type.label}
                      </Text>
                    )}
                    {activity.subject_user_id && (
                      <Text type="secondary" style={{ display: "block" }}>
                        {activity.subject_is_current_user
                          ? t("modules.activities.self_participant")
                          : activity.subject_user_name}
                      </Text>
                    )}
                  </div>
                  <Space size={0}>
                    <Button
                      type="text"
                      icon={<EditOutlined />}
                      aria-label={t("modules.activities.edit_named", {
                        title: activity.title,
                      })}
                      onClick={(event) => {
                        event.stopPropagation();
                        startEdit(activity);
                      }}
                    />
                    <Popconfirm
                      title={t("modules.activities.delete_confirm")}
                      onConfirm={() =>
                        activity.id && deleteMutation.mutate(activity.id)
                      }
                    >
                      <Button
                        type="text"
                        danger
                        icon={<DeleteOutlined />}
                        aria-label={t("modules.activities.delete_named", {
                          title: activity.title,
                        })}
                        onClick={(event) => event.stopPropagation()}
                      />
                    </Popconfirm>
                  </Space>
                </Space>
                {activity.description && (
                  <div style={{ marginTop: 6, whiteSpace: "pre-wrap" }}>
                    <ContactMentionText
                      vaultId={String(vaultId)}
                      contacts={activity.mentioned_contacts ?? []}
                      appendUnmentionedContacts={false}
                    >
                      {activity.description}
                    </ContactMentionText>
                  </div>
                )}
              </div>
            ),
          }))}
        />
      )}
      {hasMore && (
        <div style={{ textAlign: "center" }}>
          <Button
            loading={isFetching}
            onClick={() => setPage((value) => value + 1)}
          >
            {t("common.load_more")}
          </Button>
        </div>
      )}

      <Modal
        title={
          editing ? t("modules.activities.edit") : t("modules.activities.add")
        }
        open={open}
        onCancel={() => {
          setOpen(false);
          onModalClose?.();
        }}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
        destroyOnHidden
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            start_calendar: {
              calendarType: "gregorian",
              year: dayjs().year(),
              month: dayjs().month() + 1,
              day: dayjs().date(),
              datePrecision: "full",
            },
            end_status: "none",
            participant_ids: [],
          }}
          onFinish={(values) => saveMutation.mutate(values)}
        >
          <Form.Item
            name="activity_type_id"
            label={t("modules.activities.type")}
            rules={[{ required: true }]}
          >
            <Select showSearch options={typeOptions} optionFilterProp="label" />
          </Form.Item>
          <Form.Item
            name="participant_ids"
            label={t("modules.activities.participants")}
          >
            <Select
              mode="multiple"
              showSearch
              options={contactOptions}
              placeholder={t("modules.activities.participants_placeholder")}
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item
            name="title"
            label={t("modules.activities.title_label")}
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="start_calendar"
            label={t("modules.activities.happened_at")}
            rules={[{ required: true }]}
          >
            <CalendarDatePicker
              enableAlternativeCalendar={enableAlternativeCalendar}
              enableDatePrecision
              allowedDatePrecisions={["full", "month", "year"]}
            />
          </Form.Item>
          <Form.Item name="end_status" label={t("modules.activities.until")}>
            <Select
              options={(
                ["none", "ongoing", "known", "unknown"] as EndStatus[]
              ).map((value) => ({
                value,
                label: t(`modules.activities.end_${value}`),
              }))}
            />
          </Form.Item>
          {endStatus === "known" && (
            <Form.Item label={t("modules.activities.end_date")} required>
              <Space.Compact style={{ width: "100%" }}>
                <Form.Item name="end_precision" noStyle initialValue="day">
                  <Select
                    style={{ width: 110 }}
                    options={(["day", "month", "year"] as Precision[]).map(
                      (value) => ({
                        value,
                        label: t(`modules.activities.precision_${value}`),
                      }),
                    )}
                  />
                </Form.Item>
                <Form.Item name="end_date" noStyle rules={[{ required: true }]}>
                  <DatePicker
                    picker={endPrecision === "day" ? "date" : endPrecision}
                    style={{ width: "100%" }}
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          )}
          <Form.Item label={t("modules.activities.description")}>
            <ContactMentionEditor
              vaultId={String(vaultId)}
              value={description}
              onChange={setDescription}
              ariaLabel={t("modules.activities.description")}
              placeholder={t("modules.activities.description_placeholder")}
              rows={4}
              showHint
            />
          </Form.Item>
          <Form.Item name="place" label={t("modules.activities.place")}>
            <Input />
          </Form.Item>
          <Form.Item
            name="duration_in_minutes"
            label={t("modules.activities.duration_minutes")}
          >
            <Input type="number" min={0} />
          </Form.Item>
          <Form.Item
            name="parent_id"
            label={t("modules.activities.related_experience")}
          >
            <Select allowClear options={parentOptions} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
