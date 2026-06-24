import { useState, useCallback } from "react";
import {
  Card,
  Button,
  Modal,
  Form,
  Input,
  DatePicker,
  Popconfirm,
  App,
  Empty,
  Timeline,
  Typography,
  Collapse,
  Space,
  Tag,
  theme,
  Select,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  StarOutlined,
  StarFilled,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { TimelineEvent as TEvent, PaginationMeta, APIError, LifeEventCategoryResponse, UserPreferences, Contact } from "@/api";
import { useTranslation } from "react-i18next";
import { useDateFormat, formatDate, formatMonthYear } from "@/utils/dateFormat";
import { formatContactName, useVaultNameOrder } from "@/utils/nameFormat";
import dayjs from "dayjs";
import CalendarAwareDatePicker from "@/components/CalendarAwareDatePicker";
import { buildCalendarAwareValue } from "@/components/calendarAwareDateValue";
import type { CalendarAwareDateValue } from "@/components/calendarAwareDateValue";

const { Text } = Typography;

export default function LifeEventsModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [tlOpen, setTlOpen] = useState(false);
  const [leOpen, setLeOpen] = useState(false);
  const [selectedTimeline, setSelectedTimeline] = useState<number | null>(null);
  const [editingLeId, setEditingLeId] = useState<number | null>(null);
  const [page, setPage] = useState(1);
  const [allTimelines, setAllTimelines] = useState<TEvent[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const [tlForm] = Form.useForm();
  const [leForm] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const dateFormats = useDateFormat();
  const nameOrder = useVaultNameOrder(String(vaultId));
  const qk = ["vaults", vaultId, "contacts", contactId, "timelineEvents"];

  const [contactSearch, setContactSearch] = useState("");

  const { data: contactsData = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", "for-le-modal", contactSearch],
    queryFn: async () => {
      const params: Parameters<typeof api.contacts.contactsList>[1] = { per_page: 200 };
      if (contactSearch.length > 2) {
        params.search = contactSearch;
      }
      const res = await api.contacts.contactsList(String(vaultId), params);
      return (res.data ?? []) as Contact[];
    },
  });

  const getForcedParticipantIds = useCallback((timelineId?: number) => {
    const forced = new Set<string>();
    forced.add(String(contactId));
    if (timelineId !== undefined) {
      const tl = allTimelines.find(t => t.id === timelineId);
      if (tl?.participants) {
        tl.participants.forEach(p => {
          if (p.id) forced.add(String(p.id));
        });
      }
    }
    return forced;
  }, [contactId, allTimelines]);

  const contactOptions = (() => {
    const forcedIds = getForcedParticipantIds(selectedTimeline ?? undefined);

    const optionsMap = new Map<string, { value: string; label: string }>();

    contactsData.forEach((c) => {
      if (c.id && !forcedIds.has(String(c.id))) {
        optionsMap.set(String(c.id), {
          value: String(c.id),
          label: formatContactName(nameOrder, c),
        });
      }
    });

    if (editingLeId && selectedTimeline) {
      const tl = allTimelines.find(t => t.id === selectedTimeline);
      const le = tl?.life_events?.find(e => e.id === editingLeId);
      if (le?.participants) {
        le.participants.forEach(p => {
          const participantId = p.id ? String(p.id) : "";
          if (participantId && !forcedIds.has(participantId) && !optionsMap.has(participantId)) {
            optionsMap.set(participantId, {
              value: participantId,
              label: p.name || participantId,
            });
          }
        });
      }
    }

    return Array.from(optionsMap.values());
  })();

  // Fetch life event categories to get a valid type ID (instead of hardcoded 1)
  const { data: lifeEventCategories } = useQuery({
    queryKey: ["vault", vaultId, "lifeEventCategories"],
    queryFn: async () => {
      const res = await api.vaultSettings.settingsLifeEventCategoriesList(String(vaultId));
      return (res.data ?? []) as LifeEventCategoryResponse[];
    },
  });

  // Use the first available life event type ID from seed data
  const defaultLifeEventTypeId = lifeEventCategories
    ?.flatMap((cat) => cat.types ?? [])
    ?.find((t) => t.id)?.id ?? 0;

  const resetPagination = useCallback(() => {
    setPage(1);
    setAllTimelines([]);
  }, []);

  const { isLoading, isFetching } = useQuery({
    queryKey: [...qk, page],
    queryFn: async () => {
      const res = await api.lifeEvents.contactsTimelineEventsList(String(vaultId), String(contactId), { page, per_page: 15 });
      const newItems = (res.data ?? []) as TEvent[];
      const meta = res.meta as PaginationMeta | undefined;
      setAllTimelines(prev => page === 1 ? newItems : [...prev, ...newItems]);
      setHasMore(meta ? meta.page! < meta.total_pages! : newItems.length >= 15);
      return newItems;
    },
  });

  const createTimelineMutation = useMutation({
    mutationFn: (values: { label: string; started_at: dayjs.Dayjs; participants?: string[] }) =>
      api.lifeEvents.contactsTimelineEventsCreate(String(vaultId), String(contactId), {
        label: values.label,
        started_at: values.started_at.toISOString(),
        participants: values.participants,
      }),
    onSuccess: () => {
      resetPagination();
      queryClient.invalidateQueries({ queryKey: qk });
      setTlOpen(false);
      tlForm.resetFields();
      message.success(t("modules.life_events.timeline_created"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteTimelineMutation = useMutation({
    mutationFn: (id: number) => api.lifeEvents.contactsTimelineEventsDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      resetPagination();
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.life_events.timeline_deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const { data: prefs } = useQuery({
    queryKey: ["settings", "preferences"],
    queryFn: async () => {
      const res = await api.preferences.preferencesList();
      return res.data as UserPreferences | undefined;
    },
  });
  const altCalendar = prefs?.enable_alternative_calendar ?? false;

  const createLifeEventMutation = useMutation({
    mutationFn: (values: { label: string; happened_at: CalendarAwareDateValue; description?: string; participants?: string[] }) => {
      if (!selectedTimeline) throw new Error("No timeline");
      const data = {
        summary: values.label,
        happened_at: values.happened_at.date.toISOString(),
        description: values.description,
        life_event_type_id: defaultLifeEventTypeId,
        calendar_type: values.happened_at.calendarType,
        original_day: values.happened_at.originalDay ?? undefined,
        original_month: values.happened_at.originalMonth ?? undefined,
        original_year: values.happened_at.originalYear ?? undefined,
        participants: values.participants,
      };
      if (editingLeId) {
        return api.lifeEvents.contactsTimelineEventsLifeEventsUpdate(String(vaultId), String(contactId), selectedTimeline, editingLeId, data);
      }
      return api.lifeEvents.contactsTimelineEventsLifeEventsCreate(String(vaultId), String(contactId), selectedTimeline, data);
    },
    onSuccess: () => {
      resetPagination();
      queryClient.invalidateQueries({ queryKey: qk });
      setLeOpen(false);
      setEditingLeId(null);
      leForm.resetFields();
      message.success(editingLeId ? t("modules.life_events.event_updated") : t("modules.life_events.event_added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteLifeEventMutation = useMutation({
    mutationFn: ({ timelineId, lifeEventId }: { timelineId: number; lifeEventId: number }) =>
      api.lifeEvents.contactsTimelineEventsLifeEventsDelete(String(vaultId), String(contactId), timelineId, lifeEventId),
    onSuccess: () => {
      resetPagination();
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.life_events.event_deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const toggleTimelineMutation = useMutation({
    mutationFn: (timelineId: number) =>
      api.lifeEvents.contactsTimelineEventsToggleUpdate(String(vaultId), String(contactId), timelineId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.life_events.timeline_toggled"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const toggleLifeEventMutation = useMutation({
    mutationFn: ({ timelineId, lifeEventId }: { timelineId: number; lifeEventId: number }) =>
      api.lifeEvents.contactsTimelineEventsLifeEventsToggleUpdate(String(vaultId), String(contactId), timelineId, lifeEventId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.life_events.event_toggled"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  if (isLoading && page === 1) return <Card loading />;

  const collapseItems = allTimelines.map((tl: TEvent) => ({
    key: tl.id,
    label: (
       <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
         <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
           <span style={{ fontWeight: 500, opacity: tl.collapsed ? 0.5 : 1 }}>{tl.label} — <span style={{ color: token.colorTextSecondary, fontWeight: 400 }}>{formatMonthYear(tl.started_at, dateFormats)}</span></span>
           {tl.participants && tl.participants.length > 0 && (
             <div style={{ display: "flex", gap: 4, flexWrap: "wrap", marginTop: 2 }}>
               {tl.participants.map(p => (
                 <Tag key={p.id} bordered={false} style={{ margin: 0, fontSize: 12 }}>
                   {p.name}
                 </Tag>
               ))}
             </div>
           )}
         </div>
        <Space>
          <Button
            type="text"
            size="small"
            icon={tl.collapsed ? <EyeInvisibleOutlined /> : <EyeOutlined />}
            style={{ color: tl.collapsed ? token.colorTextDisabled : token.colorPrimary }}
            onClick={(e) => {
              e.stopPropagation();
              toggleTimelineMutation.mutate(tl.id!);
            }}
            title={t("modules.life_events.toggle_timeline")}
          />
          <Button
            type="text"
            size="small"
            icon={<PlusOutlined />}
            style={{ color: token.colorPrimary }}
            onClick={(e) => {
              e.stopPropagation();
               setSelectedTimeline(tl.id ?? null);
               setEditingLeId(null);
               leForm.resetFields();
               leForm.setFieldsValue({ happened_at: buildCalendarAwareValue(dayjs(), "gregorian", null, null, null) });
               setLeOpen(true);
            }}
          >
            {t("modules.life_events.event")}
          </Button>
          <Popconfirm
            title={t("modules.life_events.delete_timeline_confirm")}
            onConfirm={(e) => {
              e?.stopPropagation();
               deleteTimelineMutation.mutate(tl.id!);
            }}
            onCancel={(e) => e?.stopPropagation()}
          >
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={(e) => e.stopPropagation()}
            />
          </Popconfirm>
        </Space>
      </div>
    ),
    children: tl.life_events?.length ? (
      <Timeline
        items={tl.life_events.map((le) => ({
          color: token.colorPrimary,
          children: (
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
              <div style={{ opacity: le.collapsed ? 0.5 : 1 }}>
                <strong style={{ fontWeight: 500 }}>{le.summary ?? le.description}</strong>
                 <br />
                 <Text type="secondary" style={{ fontSize: 12 }}>
                   {formatDate(le.happened_at, dateFormats)}
                   {le.calendar_type && le.calendar_type !== "gregorian" && (
                     <Tag style={{ marginLeft: 6 }} color="processing">
                       {t(`calendar.${le.calendar_type}`)}
                     </Tag>
                   )}
                 </Text>
                 {le.description && (
                   <div style={{ marginTop: 4, color: token.colorTextSecondary }}>{le.description}</div>
                 )}
                 {le.participants && le.participants.length > 0 && (
                   <div style={{ display: "flex", gap: 4, flexWrap: "wrap", marginTop: 6 }}>
                     {le.participants.map(p => (
                       <Tag key={p.id} bordered={false} style={{ margin: 0, fontSize: 12 }}>
                         {p.name}
                       </Tag>
                     ))}
                   </div>
                 )}
               </div>
               <div>
                <Button
                  type="text"
                  size="small"
                  icon={le.collapsed ? <StarFilled style={{ color: token.colorWarning }} /> : <StarOutlined />}
                  onClick={(e) => {
                    e.stopPropagation();
                    toggleLifeEventMutation.mutate({
                      timelineId: tl.id!,
                      lifeEventId: le.id!,
                    });
                  }}
                  title={t("modules.life_events.toggle_event")}
                />
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={(e) => {
                    e.stopPropagation();
                    setSelectedTimeline(tl.id ?? null);
                    setEditingLeId(le.id!);
                    leForm.setFieldsValue({
                      label: le.summary,
                      happened_at: buildCalendarAwareValue(
                        le.happened_at,
                        le.calendar_type,
                        le.original_day,
                        le.original_month,
                        le.original_year,
                      ),
                      description: le.description,
                      participants: (() => {
                        const forcedIds = getForcedParticipantIds(tl.id);
                        return le.participants?.flatMap((p) => {
                          if (!p.id) return [];
                          const participantId = String(p.id);
                          return forcedIds.has(participantId) ? [] : [participantId];
                        }) || [];
                      })(),
                    });
                    setLeOpen(true);
                  }}
                />
                <Popconfirm
                  title={t("modules.life_events.delete_event_confirm")}
                  onConfirm={() =>
                    deleteLifeEventMutation.mutate({
                      timelineId: tl.id!,
                      lifeEventId: le.id!,
                    })
                  }
                >
                  <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>
              </div>
            </div>
          ),
        }))}
      />
    ) : (
      <Empty description={t("modules.life_events.no_events")} image={Empty.PRESENTED_IMAGE_SIMPLE} />
    ),
  }));

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.life_events.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        <Button
          type="text"
          icon={<PlusOutlined />}
          onClick={() => {
            setSelectedTimeline(null);
            setEditingLeId(null);
            setContactSearch("");
            tlForm.resetFields();
            setTlOpen(true);
          }}
          style={{ color: token.colorPrimary }}
        >
          {t("modules.life_events.new_timeline")}
        </Button>
      }
    >
      {allTimelines.length === 0 ? (
        <Empty description={t("modules.life_events.no_timelines")} />
      ) : (
        <>
          <Collapse items={collapseItems} />
          {hasMore && allTimelines.length > 0 && (
            <div style={{ textAlign: "center", marginTop: 12 }}>
              <Button onClick={() => setPage(p => p + 1)} loading={isFetching}>
                {t("common.load_more")}
              </Button>
            </div>
          )}
        </>
      )}

      <Modal
        title={t("modules.life_events.timeline_modal")}
        open={tlOpen}
        onCancel={() => { setTlOpen(false); tlForm.resetFields(); }}
        onOk={() => tlForm.submit()}
        confirmLoading={createTimelineMutation.isPending}
      >
        <Form form={tlForm} layout="vertical" onFinish={(v) => createTimelineMutation.mutate(v)}>
          <Form.Item name="label" label={t("modules.life_events.label")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="started_at" label={t("modules.life_events.started_at")} rules={[{ required: true }]}>
            <DatePicker style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="participants" label={t("modules.life_events.participants")}>
            <Select
              mode="multiple"
              allowClear
              placeholder={t("modules.life_events.participants_placeholder")}
              showSearch
              onSearch={setContactSearch}
              filterOption={false}
              options={contactOptions}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingLeId ? t("modules.life_events.edit_event") : t("modules.life_events.event_modal")}
        open={leOpen}
        onCancel={() => { setLeOpen(false); setEditingLeId(null); leForm.resetFields(); }}
        onOk={() => leForm.submit()}
        confirmLoading={createLifeEventMutation.isPending}
      >
        <Form form={leForm} layout="vertical" onFinish={(v) => createLifeEventMutation.mutate(v)}>
          <Form.Item name="label" label={t("modules.life_events.label")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="happened_at" label={t("modules.life_events.happened_at")} rules={[{ required: true }]}>
            <CalendarAwareDatePicker enableAlternativeCalendar={altCalendar} />
          </Form.Item>
          <Form.Item name="description" label={t("common.description")}>
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="participants" label={t("modules.life_events.participants")}>
            <Select
              mode="multiple"
              allowClear
              placeholder={t("modules.life_events.participants_placeholder")}
              showSearch
              onSearch={setContactSearch}
              filterOption={false}
              options={contactOptions}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
