import { Modal, Form, Input, Select, App, Button, Space, Popconfirm, Tag, theme } from "antd";
import { DeleteOutlined, PlusOutlined } from "@ant-design/icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { VaultTask, Contact, UserPreferences } from "@/api";
import { useTaskStatuses, defaultStatusSlug, type TaskStatus } from "@/utils/taskStatus";
import CalendarAwareDatePicker from "@/components/CalendarAwareDatePicker";
import { buildCalendarAwareValue } from "@/components/calendarAwareDateValue";
import type { CalendarAwareDateValue } from "@/components/calendarAwareDateValue";
import { formatContactName, useVaultNameOrder } from "@/utils/nameFormat";

const TASK_QUERY_KEY = (vaultId: string) => ["vaults", vaultId, "all-tasks"];

interface TaskEditModalProps {
  vaultId: string;
  open: boolean;
  task: VaultTask | null;
  defaultStatus?: string;
  defaultParentTaskId?: number;
  statuses?: TaskStatus[];
  onClose: () => void;
  /** Called when the user clicks a sub-task row in the modal — the parent
   * component is expected to swap which task the modal is showing. */
  onSelectTask?: (task: VaultTask) => void;
  /** Called when the user clicks "+ Add sub-task" — the parent is expected
   * to re-open the modal in create mode with parent_task_id pre-filled. */
  onCreateSubTask?: (parentTaskId: number) => void;
}

interface FormValues {
  label: string;
  description?: string;
  contact_ids?: string[];
  parent_task_id?: number | null;
  status?: string;
  due_at?: CalendarAwareDateValue | null;
}

export default function TaskEditModal({
  vaultId,
  open,
  task,
  defaultStatus,
  defaultParentTaskId,
  statuses,
  onClose,
  onSelectTask,
  onCreateSubTask,
}: TaskEditModalProps) {
  const { t } = useTranslation();
  return (
    <Modal
      title={
        task !== null
          ? t("vault.tasks.edit_task_modal_title")
          : t("vault.tasks.new_task_modal_title")
      }
      open={open}
      onCancel={onClose}
      footer={null}
      destroyOnHidden
    >
      {open && (
        <TaskEditModalContent
          key={task?.id ?? `create-${defaultParentTaskId ?? "root"}`}
          vaultId={vaultId}
          task={task}
          defaultStatus={defaultStatus}
          defaultParentTaskId={defaultParentTaskId}
          statusesProp={statuses}
          onClose={onClose}
          onSelectTask={onSelectTask}
          onCreateSubTask={onCreateSubTask}
        />
      )}
    </Modal>
  );
}

interface ContentProps {
  vaultId: string;
  task: VaultTask | null;
  defaultStatus?: string;
  defaultParentTaskId?: number;
  statusesProp?: TaskStatus[];
  onClose: () => void;
  onSelectTask?: (task: VaultTask) => void;
  onCreateSubTask?: (parentTaskId: number) => void;
}

function TaskEditModalContent({
  vaultId,
  task,
  defaultStatus,
  defaultParentTaskId,
  statusesProp,
  onClose,
  onSelectTask,
  onCreateSubTask,
}: ContentProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const { token } = theme.useToken();
  const queryClient = useQueryClient();
  const [form] = Form.useForm<FormValues>();
  const nameOrder = useVaultNameOrder(vaultId);

  const { data: ownStatuses = [] } = useTaskStatuses();
  const statuses = statusesProp && statusesProp.length > 0 ? statusesProp : ownStatuses;

  const { data: prefs } = useQuery<UserPreferences>({
    queryKey: ["preferences"],
    queryFn: async () => {
      const res = await api.preferences.preferencesList();
      return res.data!;
    },
  });
  const altCalendar = !!prefs?.enable_alternative_calendar;

  const isEdit = task !== null;
  const fallbackSlug = defaultStatus ?? defaultStatusSlug(statuses);

  const initialValues: FormValues = isEdit && task
    ? {
        label: task.label ?? "",
        description: task.description ?? "",
        contact_ids: (task.contacts ?? []).map((c) => c.id!).filter(Boolean),
        parent_task_id: task.parent_task_id ?? null,
        status: task.status || fallbackSlug,
        due_at: task.due_at
          ? buildCalendarAwareValue(
              task.due_at,
              task.calendar_type,
              task.original_day ?? null,
              task.original_month ?? null,
              task.original_year ?? null,
            )
          : null,
      }
    : {
        label: "",
        contact_ids: [],
        parent_task_id: defaultParentTaskId ?? null,
        status: fallbackSlug,
        due_at: null,
      };

  const [contactSearch, setContactSearch] = useState("");

  const { data: contactsData = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", "for-task-modal", contactSearch],
    queryFn: async () => {
      // The dropdown only preloads 200 contacts, so when users search for a contact
      // by prefix we must call the API instead of relying on local option filtering.
      const params: Parameters<typeof api.contacts.contactsList>[1] = { per_page: 200 };
      if (contactSearch.length > 2) {
        params.search = contactSearch;
      }
      const res = await api.contacts.contactsList(String(vaultId), params);
      return (res.data ?? []) as Contact[];
    },
  });

  const contactOptions = contactsData.flatMap((c) => {
    if (!c.id) return [];
    return [{
      value: c.id,
      label: formatContactName(nameOrder, c),
    }];
  });
  const contactOptionIds = new Set(contactOptions.map((option) => option.value));
  for (const selectedContact of task?.contacts ?? []) {
    if (selectedContact.id && !contactOptionIds.has(selectedContact.id)) {
      contactOptions.push({
        value: selectedContact.id,
        label: selectedContact.name || selectedContact.id,
      });
      contactOptionIds.add(selectedContact.id);
    }
  }

  const { data: allTasks = [] } = useQuery({
    queryKey: TASK_QUERY_KEY(vaultId),
    queryFn: async () => {
      const res = await api.vaultTasks.tasksList(vaultId, {});
      return (res.data ?? []) as VaultTask[];
    },
  });

  // A task may not become its own descendant's child, so the parent picker
  // excludes itself AND every transitive descendant. The walk is bounded
  // by the number of tasks to defend against any pre-existing cycles in
  // the data.
  const descendantIds = (() => {
    if (!task?.id) return new Set<number>();
    const childrenByParent = new Map<number, number[]>();
    for (const t of allTasks) {
      if (t.parent_task_id != null && t.id != null) {
        const arr = childrenByParent.get(t.parent_task_id) ?? [];
        arr.push(t.id);
        childrenByParent.set(t.parent_task_id, arr);
      }
    }
    const out = new Set<number>();
    const queue: number[] = [task.id];
    let guard = allTasks.length + 1;
    while (queue.length > 0 && guard-- > 0) {
      const current = queue.shift()!;
      for (const child of childrenByParent.get(current) ?? []) {
        if (!out.has(child)) {
          out.add(child);
          queue.push(child);
        }
      }
    }
    return out;
  })();
  const parentOptions = allTasks
    .filter((t) =>
      t.id != null &&
      t.id !== task?.id &&
      !descendantIds.has(t.id),
    )
    .map((t) => ({ value: t.id, label: t.label }));

  // Sub-tasks: children of the currently-edited task. Read from the same
  // task list we already loaded — no extra round-trip.
  const subTasks: VaultTask[] = isEdit && task?.id != null
    ? allTasks.filter((t) => t.parent_task_id === task.id)
    : [];

  const statusLabel = (slug?: string) =>
    statuses.find((s) => s.slug === slug)?.label || slug || "";

  const onSuccess = () => {
    queryClient.invalidateQueries({ queryKey: TASK_QUERY_KEY(vaultId) });
    onClose();
  };
  const onError = () => message.error(t("vault.tasks.save_failed"));

  const createMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.vaultTasks.tasksCreate(vaultId, {
        label: values.label,
        description: values.description ?? "",
        contact_ids: values.contact_ids ?? [],
        parent_task_id: values.parent_task_id ?? undefined,
        status: values.status ?? fallbackSlug,
        due_at: values.due_at ? values.due_at.date.toISOString() : undefined,
        calendar_type: values.due_at?.calendarType,
        original_day: values.due_at?.originalDay ?? undefined,
        original_month: values.due_at?.originalMonth ?? undefined,
        original_year: values.due_at?.originalYear ?? undefined,
      }),
    onSuccess,
    onError,
  });

  const updateMutation = useMutation({
    mutationFn: (values: FormValues) => {
      // parent_task_id is tri-state on the server (NullableUint): preserve
      // the distinction between cleared (null) and a real number. AntD
      // Select with allowClear emits `null` for clear and `undefined` for
      // "field never touched", which maps cleanly to the server contract
      // once we forward both literally. The generated TS type narrows the
      // field to `number | undefined`, so we cast through unknown to keep
      // the explicit null on the wire.
      const body = {
        label: values.label,
        description: values.description ?? "",
        contact_ids: values.contact_ids ?? [],
        parent_task_id: values.parent_task_id,
        status: values.status ?? fallbackSlug,
        due_at: values.due_at ? values.due_at.date.toISOString() : undefined,
        calendar_type: values.due_at?.calendarType,
        original_day: values.due_at?.originalDay ?? undefined,
        original_month: values.due_at?.originalMonth ?? undefined,
        original_year: values.due_at?.originalYear ?? undefined,
      } as unknown as Parameters<typeof api.vaultTasks.tasksPartialUpdate>[2];
      return api.vaultTasks.tasksPartialUpdate(vaultId, task!.id!, body);
    },
    onSuccess,
    onError,
  });

  const deleteMutation = useMutation({
    mutationFn: () => api.vaultTasks.tasksDelete(vaultId, task!.id!),
    onSuccess,
    onError,
  });

  const submitting = createMutation.isPending || updateMutation.isPending || deleteMutation.isPending;

  return (
    <Form
      form={form}
      layout="vertical"
      initialValues={initialValues}
      onFinish={(values) => {
        if (isEdit) updateMutation.mutate(values);
        else createMutation.mutate(values);
      }}
      onFinishFailed={({ errorFields }) => {
        const first = errorFields[0]?.errors?.[0];
        if (first) message.error(first);
      }}
    >
      <Form.Item name="label" rules={[{ required: true }]}>
        <Input placeholder={t("vault.tasks.new_task_label_placeholder")} autoFocus />
      </Form.Item>
      <Form.Item name="description">
        <Input.TextArea
          placeholder={t("vault.tasks.new_task_description_placeholder")}
          autoSize={{ minRows: 2, maxRows: 6 }}
        />
      </Form.Item>
      {isEdit && (
        <Form.Item name="status" label={t("vault.tasks.status_label")}>
          <Select
            options={statuses.map((s) => ({
              value: s.slug,
              label: s.label,
            }))}
          />
        </Form.Item>
      )}
      <Form.Item name="contact_ids" label={t("vault.tasks.contacts_label")}>
        <Select
          mode="multiple"
          allowClear
          placeholder={t("vault.tasks.no_contacts_placeholder")}
          showSearch
          onSearch={setContactSearch}
          filterOption={false}
          options={contactOptions}
        />
      </Form.Item>
      <Form.Item name="parent_task_id" label={t("vault.tasks.parent_label")}>
        <Select
          allowClear
          placeholder={t("vault.tasks.no_parent_placeholder")}
          showSearch
          optionFilterProp="label"
          options={parentOptions}
        />
      </Form.Item>
      {isEdit && (
        <Form.Item
          label={t("vault.tasks.sub_tasks_label", { count: subTasks.length })}
          style={{ marginBottom: 16 }}
        >
          {subTasks.length > 0 && (
            <div
              style={{
                border: `1px solid ${token.colorBorderSecondary}`,
                borderRadius: token.borderRadius,
                padding: 4,
                marginBottom: 8,
              }}
            >
              {subTasks.map((st) => (
                <div
                  key={st.id}
                  onClick={() => onSelectTask && st.id != null && onSelectTask(st)}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 8,
                    padding: "6px 8px",
                    cursor: onSelectTask ? "pointer" : "default",
                    borderRadius: token.borderRadius,
                    transition: "background 0.15s",
                  }}
                  onMouseEnter={(e) => {
                    if (onSelectTask) e.currentTarget.style.background = token.colorFillQuaternary;
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = "transparent";
                  }}
                >
                  <Tag style={{ marginRight: 0 }}>{statusLabel(st.status)}</Tag>
                  <span
                    style={{
                      flex: 1,
                      minWidth: 0,
                      overflowWrap: "anywhere",
                      wordBreak: "break-word",
                      textDecoration: st.completed ? "line-through" : undefined,
                      color: st.completed ? token.colorTextSecondary : undefined,
                    }}
                  >
                    {st.label}
                  </span>
                </div>
              ))}
            </div>
          )}
          {onCreateSubTask && task?.id != null && (
            <Button
              type="dashed"
              icon={<PlusOutlined />}
              onClick={() => onCreateSubTask(task.id!)}
              style={{ width: "100%" }}
            >
              {t("vault.tasks.add_sub_task")}
            </Button>
          )}
        </Form.Item>
      )}
      <Form.Item name="due_at" label={t("vault.tasks.due_label")}>
        <CalendarAwareDatePicker
          enableAlternativeCalendar={altCalendar}
          showTime
          format="YYYY-MM-DD HH:mm"
          allowClear
        />
      </Form.Item>
      <Form.Item style={{ marginBottom: 0 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          {isEdit ? (
            <Popconfirm
              title={t("vault.tasks.delete_confirm")}
              onConfirm={() => deleteMutation.mutate()}
              okButtonProps={{ danger: true }}
            >
              <Button danger icon={<DeleteOutlined />} loading={deleteMutation.isPending}>
                {t("vault.tasks.delete")}
              </Button>
            </Popconfirm>
          ) : (
            <span />
          )}
          <Space>
            <Button onClick={onClose}>{t("vault.tasks.cancel")}</Button>
            <Button type="primary" htmlType="submit" loading={submitting}>
              {isEdit ? t("vault.tasks.save") : t("vault.tasks.create")}
            </Button>
          </Space>
        </div>
      </Form.Item>
    </Form>
  );
}
