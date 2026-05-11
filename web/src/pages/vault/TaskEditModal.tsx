import { Modal, Form, Input, Select, App, Button, Space, DatePicker, Popconfirm } from "antd";
import { DeleteOutlined } from "@ant-design/icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import dayjs, { type Dayjs } from "dayjs";
import { api } from "@/api";
import type { VaultTask, Contact } from "@/api";
import { useTaskStatuses, defaultStatusSlug, type TaskStatus } from "@/utils/taskStatus";

const TASK_QUERY_KEY = (vaultId: string) => ["vaults", vaultId, "all-tasks"];

interface TaskEditModalProps {
  vaultId: string;
  open: boolean;
  task: VaultTask | null;
  defaultStatus?: string;
  statuses?: TaskStatus[];
  onClose: () => void;
}

interface FormValues {
  label: string;
  description?: string;
  contact_ids?: string[];
  parent_task_id?: number | null;
  status?: string;
  due_at?: Dayjs | null;
}

export default function TaskEditModal({
  vaultId,
  open,
  task,
  defaultStatus,
  statuses,
  onClose,
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
          key={task?.id ?? "create"}
          vaultId={vaultId}
          task={task}
          defaultStatus={defaultStatus}
          statusesProp={statuses}
          onClose={onClose}
        />
      )}
    </Modal>
  );
}

interface ContentProps {
  vaultId: string;
  task: VaultTask | null;
  defaultStatus?: string;
  statusesProp?: TaskStatus[];
  onClose: () => void;
}

function TaskEditModalContent({
  vaultId,
  task,
  defaultStatus,
  statusesProp,
  onClose,
}: ContentProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [form] = Form.useForm<FormValues>();

  const { data: ownStatuses = [] } = useTaskStatuses();
  const statuses = statusesProp && statusesProp.length > 0 ? statusesProp : ownStatuses;

  const isEdit = task !== null;
  const fallbackSlug = defaultStatus ?? defaultStatusSlug(statuses);

  const initialValues: FormValues = isEdit && task
    ? {
        label: task.label ?? "",
        description: task.description ?? "",
        contact_ids: (task.contacts ?? []).map((c) => c.id!).filter(Boolean),
        parent_task_id: task.parent_task_id ?? null,
        status: task.status || fallbackSlug,
        due_at: task.due_at ? dayjs(task.due_at) : null,
      }
    : { label: "", contact_ids: [], parent_task_id: null, status: fallbackSlug, due_at: null };

  const { data: contacts = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", "for-task-modal"],
    queryFn: async () => {
      const res = await api.contacts.contactsList(String(vaultId), { per_page: 200 });
      return (res.data ?? []) as Contact[];
    },
  });

  // Parent-task candidates: every top-level task in this vault, excluding
  // this task itself (a task can't be its own parent).
  const { data: allTasks = [] } = useQuery({
    queryKey: TASK_QUERY_KEY(vaultId),
    queryFn: async () => {
      const res = await api.vaultTasks.tasksList(vaultId, {});
      return (res.data ?? []) as VaultTask[];
    },
  });
  const parentOptions = allTasks
    .filter((t) => !t.parent_task_id && t.id !== task?.id)
    .map((t) => ({ value: t.id, label: t.label }));

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
        due_at: values.due_at ? values.due_at.toISOString() : undefined,
      }),
    onSuccess,
    onError,
  });

  const updateMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.vaultTasks.tasksPartialUpdate(vaultId, task!.id!, {
        label: values.label,
        description: values.description ?? "",
        contact_ids: values.contact_ids ?? [],
        parent_task_id: values.parent_task_id ?? undefined,
        status: values.status ?? fallbackSlug,
        due_at: values.due_at ? values.due_at.toISOString() : undefined,
      }),
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
          optionFilterProp="label"
          options={contacts.map((c) => ({
            value: c.id,
            label: [c.first_name, c.last_name].filter(Boolean).join(" ") || c.id,
          }))}
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
      <Form.Item name="due_at" label={t("vault.tasks.due_label")}>
        <DatePicker
          style={{ width: "100%" }}
          showTime={{ format: "HH:mm" }}
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
