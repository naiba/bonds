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
  // null = create mode; set = edit mode
  task: VaultTask | null;
  // Default status slug when creating (column the "+" came from). Ignored
  // in edit mode (the form's Status select is the source of truth there).
  defaultStatus?: string;
  // Optional: pass through the parent's already-fetched status list to
  // avoid a duplicate query. The hook below falls back to its own fetch.
  statuses?: TaskStatus[];
  onClose: () => void;
}

interface FormValues {
  label: string;
  description?: string;
  contact_id?: string;
  status?: string;
  // AntD DatePicker hands back a Dayjs (or null when cleared). The
  // submit handler converts to ISO string for the API.
  due_at?: Dayjs | null;
}

/**
 * Outer wrapper. Owns the Modal chrome and a key on the inner content so
 * each (task, open) transition mounts a brand-new TaskEditModalContent
 * with its own Form.useForm() instance. Without this, switching from
 * editing task A to task B caused B's initialValues to be ignored
 * because the shared form instance still held A's values.
 */
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

  // initialValues are applied on first (and only) mount of this component;
  // the parent ensures a fresh mount per task via its `key` prop.
  const initialValues: FormValues = isEdit && task
    ? {
        label: task.label ?? "",
        description: task.description ?? "",
        contact_id: task.contact_id || undefined,
        status: task.status || fallbackSlug,
        due_at: task.due_at ? dayjs(task.due_at) : null,
      }
    : { label: "", status: fallbackSlug, due_at: null };

  const { data: contacts = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", "for-task-modal"],
    queryFn: async () => {
      const res = await api.contacts.contactsList(String(vaultId), { per_page: 200 });
      return (res.data ?? []) as Contact[];
    },
  });

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
        contact_id: values.contact_id ?? "",
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
        contact_id: values.contact_id ?? "",
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
      <Form.Item name="contact_id" label={t("vault.tasks.new_task_contact_optional")}>
        <Select
          allowClear
          placeholder={t("vault.tasks.new_task_no_contact")}
          showSearch
          optionFilterProp="label"
          options={contacts.map((c) => ({
            value: c.id,
            label: [c.first_name, c.last_name].filter(Boolean).join(" ") || c.id,
          }))}
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
