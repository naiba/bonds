import { useMemo, useState } from "react";
import { Card, Tag, Typography, Button, Modal, Form, Input, Select, App, theme } from "antd";
import { PlusOutlined } from "@ant-design/icons";
import {
  DndContext,
  closestCorners,
  PointerSensor,
  useSensor,
  useSensors,
  useDroppable,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { VaultTask, Contact } from "@/api";
import { useDateFormat, formatShortDate } from "@/utils/dateFormat";

const { Text } = Typography;

const STATUSES = ["todo", "in_progress", "done"] as const;
type ColumnStatus = (typeof STATUSES)[number];

interface TasksKanbanProps {
  vaultId: string;
  tasks: VaultTask[];
}

const TASK_QUERY_KEY = (vaultId: string) => ["vaults", vaultId, "all-tasks"];

export default function TasksKanban({ vaultId, tasks }: TasksKanbanProps) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const dateFormats = useDateFormat();

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));

  // Single modal serves both Create (editingTask = null) and Edit (editingTask set).
  // modalStatus is the column the create came from; ignored in edit mode (status
  // is taken from the form's Status select instead).
  const [modalOpen, setModalOpen] = useState(false);
  const [modalStatus, setModalStatus] = useState<ColumnStatus>("todo");
  const [editingTask, setEditingTask] = useState<VaultTask | null>(null);
  const [form] = Form.useForm();

  const { data: contacts = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", "for-task-create"],
    queryFn: async () => {
      const res = await api.contacts.contactsList(String(vaultId), { per_page: 200 });
      return (res.data ?? []) as Contact[];
    },
    enabled: modalOpen,
  });

  // Mutation for moving a task across columns OR within a column.
  // Optimistic via TanStack setQueryData so the UI feels instant; on error we
  // invalidate to refetch from the server.
  const moveMutation = useMutation({
    mutationFn: ({ id, position, status }: { id: number; position: number; status: ColumnStatus }) =>
      api.vaultTasks.tasksPositionPartialUpdate(vaultId, id, { position, status }),
    onMutate: async ({ id, position, status }) => {
      await queryClient.cancelQueries({ queryKey: TASK_QUERY_KEY(vaultId) });
      const previous = queryClient.getQueryData<VaultTask[]>(TASK_QUERY_KEY(vaultId));
      if (previous) {
        queryClient.setQueryData<VaultTask[]>(TASK_QUERY_KEY(vaultId), (cur) => {
          if (!cur) return cur;
          return reorderInCache(cur, id, status, position);
        });
      }
      return { previous };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.previous) queryClient.setQueryData(TASK_QUERY_KEY(vaultId), ctx.previous);
      message.error(t("vault.tasks.save_failed"));
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: TASK_QUERY_KEY(vaultId) });
    },
  });

  const closeModal = () => {
    setModalOpen(false);
    setEditingTask(null);
    form.resetFields();
  };

  const createMutation = useMutation({
    mutationFn: (values: { label: string; description?: string; contact_id?: string; status: ColumnStatus }) =>
      api.vaultTasks.tasksCreate(vaultId, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TASK_QUERY_KEY(vaultId) });
      closeModal();
    },
    onError: () => message.error(t("vault.tasks.save_failed")),
  });

  const updateMutation = useMutation({
    mutationFn: ({
      id,
      values,
    }: {
      id: number;
      values: { label: string; description?: string; contact_id?: string; status: ColumnStatus };
    }) =>
      api.vaultTasks.tasksPartialUpdate(vaultId, id, {
        label: values.label,
        description: values.description ?? "",
        contact_id: values.contact_id ?? "",
        status: values.status,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TASK_QUERY_KEY(vaultId) });
      closeModal();
    },
    onError: () => message.error(t("vault.tasks.save_failed")),
  });

  const columns = useMemo(() => {
    const grouped: Record<ColumnStatus, VaultTask[]> = { todo: [], in_progress: [], done: [] };
    for (const task of tasks) {
      const status = (STATUSES as readonly string[]).includes(task.status ?? "")
        ? (task.status as ColumnStatus)
        : "todo";
      grouped[status].push(task);
    }
    return grouped;
  }, [tasks]);

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over) return;
    const activeId = Number(active.id);
    const activeTask = tasks.find((t) => t.id === activeId);
    if (!activeTask) return;

    const overId = String(over.id);
    let destStatus: ColumnStatus;
    let destPosition: number;

    if (overId.startsWith("col:")) {
      destStatus = overId.slice(4) as ColumnStatus;
      destPosition = columns[destStatus].length;
    } else {
      const overTask = tasks.find((t) => t.id === Number(overId));
      if (!overTask) return;
      destStatus = ((STATUSES as readonly string[]).includes(overTask.status ?? "")
        ? overTask.status
        : "todo") as ColumnStatus;
      destPosition = columns[destStatus].findIndex((t) => t.id === overTask.id);
      if (destPosition < 0) destPosition = columns[destStatus].length;
    }

    const srcStatus = ((STATUSES as readonly string[]).includes(activeTask.status ?? "")
      ? activeTask.status
      : "todo") as ColumnStatus;
    const srcPosition = columns[srcStatus].findIndex((t) => t.id === activeId);
    if (srcStatus === destStatus && srcPosition === destPosition) return;

    moveMutation.mutate({ id: activeId, position: destPosition, status: destStatus });
  };

  const openCreateModal = (status: ColumnStatus) => {
    setEditingTask(null);
    setModalStatus(status);
    form.resetFields();
    form.setFieldsValue({ status });
    setModalOpen(true);
  };

  const openEditModal = (task: VaultTask) => {
    setEditingTask(task);
    form.resetFields();
    form.setFieldsValue({
      label: task.label,
      description: task.description ?? "",
      contact_id: task.contact_id || undefined,
      status:
        (STATUSES as readonly string[]).includes(task.status ?? "")
          ? (task.status as ColumnStatus)
          : "todo",
    });
    setModalOpen(true);
  };

  return (
    <div>
      <DndContext sensors={sensors} collisionDetection={closestCorners} onDragEnd={handleDragEnd}>
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(3, minmax(0, 1fr))",
            gap: 16,
          }}
        >
          {STATUSES.map((status) => (
            <KanbanColumn
              key={status}
              status={status}
              title={t(`vault.tasks.col_${status}`)}
              tasks={columns[status]}
              token={token}
              dateFormats={dateFormats}
              dueLabel={t("vault.tasks.due", { date: "" }).replace(/\s*$/, "")}
              emptyLabel={t("vault.tasks.empty_column")}
              onAdd={() => openCreateModal(status)}
              onTaskClick={openEditModal}
              addLabel={t("vault.tasks.new_task")}
            />
          ))}
        </div>
      </DndContext>

      <Modal
        title={
          editingTask
            ? t("vault.tasks.edit_task_modal_title")
            : t("vault.tasks.new_task_modal_title")
        }
        open={modalOpen}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        okText={editingTask ? t("vault.tasks.save") : t("vault.tasks.create")}
        cancelText={t("vault.tasks.cancel")}
        destroyOnHidden
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values: {
            label: string;
            description?: string;
            contact_id?: string;
            status?: ColumnStatus;
          }) => {
            const status = values.status ?? modalStatus;
            if (editingTask) {
              updateMutation.mutate({
                id: editingTask.id!,
                values: {
                  label: values.label,
                  description: values.description || undefined,
                  contact_id: values.contact_id || undefined,
                  status,
                },
              });
            } else {
              createMutation.mutate({
                label: values.label,
                description: values.description || undefined,
                contact_id: values.contact_id || undefined,
                status,
              });
            }
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
          {editingTask && (
            <Form.Item name="status" label={t("vault.tasks.status_label")}>
              <Select
                options={STATUSES.map((s) => ({
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
    </div>
  );
}

// Pure helper: move a task to (status, position) inside a flat task array.
// Used by optimistic update in onMutate so the cache reflects the drop instantly.
function reorderInCache(tasks: VaultTask[], id: number, status: ColumnStatus, position: number): VaultTask[] {
  const moved = tasks.find((t) => t.id === id);
  if (!moved) return tasks;
  const filtered = tasks.filter((t) => t.id !== id);
  const sameStatus = filtered.filter((t) => t.status === status);
  const others = filtered.filter((t) => t.status !== status);
  const insertAt = Math.min(position, sameStatus.length);
  const updatedMoved: VaultTask = { ...moved, status };
  const newSameStatus = [
    ...sameStatus.slice(0, insertAt),
    updatedMoved,
    ...sameStatus.slice(insertAt),
  ];
  return [...others, ...newSameStatus];
}

interface KanbanColumnProps {
  status: ColumnStatus;
  title: string;
  tasks: VaultTask[];
  token: ReturnType<typeof theme.useToken>["token"];
  dateFormats: ReturnType<typeof useDateFormat>;
  dueLabel: string;
  emptyLabel: string;
  addLabel: string;
  onAdd: () => void;
  onTaskClick: (task: VaultTask) => void;
}

function KanbanColumn(props: KanbanColumnProps) {
  const { status, title, tasks, token, dateFormats, dueLabel, emptyLabel, addLabel, onAdd, onTaskClick } = props;
  const taskIds = tasks.map((t) => String(t.id));
  const { setNodeRef, isOver } = useDroppable({ id: `col:${status}` });

  return (
    <div
      data-status={status}
      style={{
        background: token.colorFillQuaternary,
        borderRadius: token.borderRadiusLG,
        padding: 12,
        minHeight: 200,
        display: "flex",
        flexDirection: "column",
        gap: 8,
        outline: isOver ? `2px solid ${token.colorPrimary}` : undefined,
        outlineOffset: -2,
        transition: "outline 120ms ease",
      }}
      ref={setNodeRef}
    >
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <Text strong style={{ fontSize: 13, textTransform: "uppercase", letterSpacing: 0.5 }}>
          {title} <Text type="secondary" style={{ fontWeight: 400 }}>· {tasks.length}</Text>
        </Text>
        <Button type="text" size="small" icon={<PlusOutlined />} onClick={onAdd} aria-label={addLabel} />
      </div>

      <SortableContext items={taskIds} strategy={verticalListSortingStrategy}>
        <div style={{ display: "flex", flexDirection: "column", gap: 8, minHeight: 60 }}>
          {tasks.length === 0 ? (
            <div
              style={{
                border: `1px dashed ${token.colorBorder}`,
                borderRadius: token.borderRadius,
                padding: "24px 12px",
                textAlign: "center",
                color: token.colorTextTertiary,
                fontSize: 12,
              }}
            >
              {emptyLabel}
            </div>
          ) : (
            tasks.map((task) => (
              <SortableTaskCard
                key={task.id}
                task={task}
                token={token}
                dateFormats={dateFormats}
                dueLabel={dueLabel}
                onClick={() => onTaskClick(task)}
              />
            ))
          )}
        </div>
      </SortableContext>
    </div>
  );
}

interface SortableTaskCardProps {
  task: VaultTask;
  token: ReturnType<typeof theme.useToken>["token"];
  dateFormats: ReturnType<typeof useDateFormat>;
  dueLabel: string;
  onClick: () => void;
}

function SortableTaskCard({ task, token, dateFormats, dueLabel, onClick }: SortableTaskCardProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: String(task.id),
  });
  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
    cursor: "grab",
  };
  // PointerSensor is configured with activationConstraint.distance:6 in the
  // parent — drags only start after moving 6px. So a stationary mousedown+up
  // fires onClick normally without conflicting with the drag listeners.
  return (
    <div ref={setNodeRef} style={style} {...attributes} {...listeners} onClick={onClick}>
      <Card
        size="small"
        styles={{ body: { padding: 12 } }}
        style={{ borderRadius: token.borderRadius }}
      >
        <div style={{ fontWeight: 500, marginBottom: task.contact_name || task.due_at ? 6 : 0 }}>
          {task.label}
        </div>
        {(task.contact_name || task.due_at) && (
          <div style={{ display: "flex", flexWrap: "wrap", gap: 6, alignItems: "center" }}>
            {task.contact_id && task.contact_name && (
              <Tag color="blue" style={{ marginRight: 0 }}>
                {task.contact_name}
              </Tag>
            )}
            {task.due_at && (
              <Tag color="orange" style={{ marginRight: 0 }}>
                {dueLabel} {formatShortDate(task.due_at, dateFormats)}
              </Tag>
            )}
          </div>
        )}
      </Card>
    </div>
  );
}
