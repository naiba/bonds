import { useEffect } from "react";
import { Modal, Form, Input, Select, App } from "antd";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { VaultTask, Contact } from "@/api";
import { TASK_STATUSES, type TaskStatus, normalizeTaskStatus } from "@/utils/taskStatus";

const TASK_QUERY_KEY = (vaultId: string) => ["vaults", vaultId, "all-tasks"];

interface TaskEditModalProps {
  vaultId: string;
  open: boolean;
  // null = create mode; set = edit mode
  task: VaultTask | null;
  // Default status when creating (column the "+" came from). Ignored in edit
  // mode (the form's Status select is the source of truth there).
  defaultStatus?: TaskStatus;
  onClose: () => void;
}

interface FormValues {
  label: string;
  description?: string;
  contact_id?: string;
  status?: TaskStatus;
}

/**
 * Shared modal for both creating and editing vault tasks.
 *
 * Mounted by both VaultTasks (list view) and TasksKanban (kanban view).
 * Fully self-contained: owns its form, contacts query, and mutations.
 * Parent only manages open/close state and which task (or null) is being
 * edited.
 */
export default function TaskEditModal({
  vaultId,
  open,
  task,
  defaultStatus = "todo",
  onClose,
}: TaskEditModalProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [form] = Form.useForm<FormValues>();

  const isEdit = task !== null;

  // Reseed form whenever the modal opens or the task changes
  useEffect(() => {
    if (!open) return;
    if (isEdit && task) {
      form.setFieldsValue({
        label: task.label ?? "",
        description: task.description ?? "",
        contact_id: task.contact_id || undefined,
        status: normalizeTaskStatus(task.status),
      });
    } else {
      form.resetFields();
      form.setFieldsValue({ status: defaultStatus });
    }
  }, [open, task, isEdit, defaultStatus, form]);

  // Only fetch contacts when modal is open — no point pre-fetching
  const { data: contacts = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", "for-task-modal"],
    queryFn: async () => {
      const res = await api.contacts.contactsList(String(vaultId), { per_page: 200 });
      return (res.data ?? []) as Contact[];
    },
    enabled: open,
  });

  const onSuccess = () => {
    queryClient.invalidateQueries({ queryKey: TASK_QUERY_KEY(vaultId) });
    onClose();
    form.resetFields();
  };
  const onError = () => message.error(t("vault.tasks.save_failed"));

  const createMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.vaultTasks.tasksCreate(vaultId, {
        label: values.label,
        description: values.description ?? "",
        contact_id: values.contact_id ?? "",
        status: values.status ?? defaultStatus,
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
        status: values.status ?? defaultStatus,
      }),
    onSuccess,
    onError,
  });

  return (
    <Modal
      title={
        isEdit
          ? t("vault.tasks.edit_task_modal_title")
          : t("vault.tasks.new_task_modal_title")
      }
      open={open}
      onCancel={onClose}
      onOk={() => form.submit()}
      confirmLoading={createMutation.isPending || updateMutation.isPending}
      okText={isEdit ? t("vault.tasks.save") : t("vault.tasks.create")}
      cancelText={t("vault.tasks.cancel")}
      destroyOnHidden
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={(values) => {
          if (isEdit) updateMutation.mutate(values);
          else createMutation.mutate(values);
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
              options={TASK_STATUSES.map((s) => ({
                value: s,
                label: t(`vault.tasks.col_${s}`),
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
      </Form>
    </Modal>
  );
}
