import { useMemo, useState } from "react";
import { Card, Tag, Typography, Button, Grid, theme } from "antd";
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
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { VaultTask } from "@/api";
import { useDateFormat, formatShortDate } from "@/utils/dateFormat";
import { TASK_STATUSES, type TaskStatus, normalizeTaskStatus } from "@/utils/taskStatus";
import TaskEditModal from "./TaskEditModal";

const { Text } = Typography;
const { useBreakpoint } = Grid;

const TASK_QUERY_KEY = (vaultId: string) => ["vaults", vaultId, "all-tasks"];

interface TasksKanbanProps {
  vaultId: string;
  tasks: VaultTask[];
  // External hook so the parent (VaultTasks) can use the same modal instance
  // when a list-view row is clicked. When omitted, this component owns its
  // modal state internally.
  onTaskClick?: (task: VaultTask) => void;
}

export default function TasksKanban({ vaultId, tasks, onTaskClick }: TasksKanbanProps) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const queryClient = useQueryClient();
  const dateFormats = useDateFormat();
  const screens = useBreakpoint();
  // Side-by-side + drag at lg breakpoint and up (≥992px). Below that we
  // stack vertically and disable drag — drag-drop on touch fights with
  // scroll, and 3 columns can't fit narrowly without text mangling.
  const isWide = !!screens.lg;

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));

  // Internal modal state (used when no external onTaskClick provided)
  const [internalModalOpen, setInternalModalOpen] = useState(false);
  const [internalEditingTask, setInternalEditingTask] = useState<VaultTask | null>(null);
  const [internalDefaultStatus, setInternalDefaultStatus] = useState<TaskStatus>("todo");

  // Optimistic move via PATCH /position. setQueryData on onMutate so the UI
  // feels instant, rollback + invalidate on error.
  const moveMutation = useMutation({
    mutationFn: ({ id, position, status }: { id: number; position: number; status: TaskStatus }) =>
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
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: TASK_QUERY_KEY(vaultId) });
    },
  });

  const columns = useMemo(() => {
    const grouped: Record<TaskStatus, VaultTask[]> = { todo: [], in_progress: [], done: [] };
    for (const task of tasks) {
      grouped[normalizeTaskStatus(task.status)].push(task);
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
    let destStatus: TaskStatus;
    let destPosition: number;

    if (overId.startsWith("col:")) {
      destStatus = overId.slice(4) as TaskStatus;
      destPosition = columns[destStatus].length;
    } else {
      const overTask = tasks.find((t) => t.id === Number(overId));
      if (!overTask) return;
      destStatus = normalizeTaskStatus(overTask.status);
      destPosition = columns[destStatus].findIndex((t) => t.id === overTask.id);
      if (destPosition < 0) destPosition = columns[destStatus].length;
    }

    const srcStatus = normalizeTaskStatus(activeTask.status);
    const srcPosition = columns[srcStatus].findIndex((t) => t.id === activeId);
    if (srcStatus === destStatus && srcPosition === destPosition) return;

    moveMutation.mutate({ id: activeId, position: destPosition, status: destStatus });
  };

  const openInternalCreateModal = (status: TaskStatus) => {
    setInternalEditingTask(null);
    setInternalDefaultStatus(status);
    setInternalModalOpen(true);
  };

  const openInternalEditModal = (task: VaultTask) => {
    setInternalEditingTask(task);
    setInternalModalOpen(true);
  };

  // Card click handler — defer to parent if it's wired one up, else open
  // our own modal.
  const handleCardClick = onTaskClick ?? openInternalEditModal;

  const columnGrid = isWide
    ? { display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: 16 }
    : { display: "flex", flexDirection: "column" as const, gap: 16 };

  const renderColumns = () => (
    <div style={columnGrid}>
      {TASK_STATUSES.map((status) => (
        <KanbanColumn
          key={status}
          status={status}
          title={t(`vault.tasks.col_${status}`)}
          tasks={columns[status]}
          token={token}
          dateFormats={dateFormats}
          dueLabel={t("vault.tasks.due", { date: "" }).replace(/\s*$/, "")}
          emptyLabel={t("vault.tasks.empty_column")}
          onAdd={() => openInternalCreateModal(status)}
          onTaskClick={handleCardClick}
          addLabel={t("vault.tasks.new_task")}
          interactive={isWide}
        />
      ))}
    </div>
  );

  return (
    <div>
      {isWide ? (
        <DndContext sensors={sensors} collisionDetection={closestCorners} onDragEnd={handleDragEnd}>
          {renderColumns()}
        </DndContext>
      ) : (
        renderColumns()
      )}

      {/* Only mount our own modal if the parent isn't handling clicks */}
      {!onTaskClick && (
        <TaskEditModal
          vaultId={vaultId}
          open={internalModalOpen}
          task={internalEditingTask}
          defaultStatus={internalDefaultStatus}
          onClose={() => setInternalModalOpen(false)}
        />
      )}
    </div>
  );
}

// Pure helper: move a task to (status, position) inside a flat task array.
// Used by optimistic update in onMutate so the cache reflects the drop instantly.
function reorderInCache(tasks: VaultTask[], id: number, status: TaskStatus, position: number): VaultTask[] {
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
  status: TaskStatus;
  title: string;
  tasks: VaultTask[];
  token: ReturnType<typeof theme.useToken>["token"];
  dateFormats: ReturnType<typeof useDateFormat>;
  dueLabel: string;
  emptyLabel: string;
  addLabel: string;
  onAdd: () => void;
  onTaskClick: (task: VaultTask) => void;
  // When false, render plain task cards (no drag/sort hooks). Used on
  // narrow viewports where drag conflicts with touch scrolling.
  interactive: boolean;
}

function KanbanColumn(props: KanbanColumnProps) {
  const { status, title, tasks, token, dateFormats, dueLabel, emptyLabel, addLabel, onAdd, onTaskClick, interactive } = props;

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
      }}
    >
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <Text strong style={{ fontSize: 13, textTransform: "uppercase", letterSpacing: 0.5 }}>
          {title} <Text type="secondary" style={{ fontWeight: 400 }}>· {tasks.length}</Text>
        </Text>
        <Button type="text" size="small" icon={<PlusOutlined />} onClick={onAdd} aria-label={addLabel} />
      </div>

      {interactive ? (
        <DroppableColumnArea status={status} token={token} empty={tasks.length === 0} emptyLabel={emptyLabel}>
          <SortableContext items={tasks.map((t) => String(t.id))} strategy={verticalListSortingStrategy}>
            {tasks.map((task) => (
              <SortableTaskCard
                key={task.id}
                task={task}
                token={token}
                dateFormats={dateFormats}
                dueLabel={dueLabel}
                onClick={() => onTaskClick(task)}
              />
            ))}
          </SortableContext>
        </DroppableColumnArea>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 8, minHeight: 60 }}>
          {tasks.length === 0 ? (
            <EmptyColumnPlaceholder token={token} label={emptyLabel} />
          ) : (
            tasks.map((task) => (
              <PlainTaskCard
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
      )}
    </div>
  );
}

function DroppableColumnArea(props: {
  status: TaskStatus;
  token: ReturnType<typeof theme.useToken>["token"];
  empty: boolean;
  emptyLabel: string;
  children: React.ReactNode;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: `col:${props.status}` });
  return (
    <div
      ref={setNodeRef}
      style={{
        display: "flex",
        flexDirection: "column",
        gap: 8,
        minHeight: 60,
        outline: isOver ? `2px solid ${props.token.colorPrimary}` : undefined,
        outlineOffset: -2,
        transition: "outline 120ms ease",
        borderRadius: props.token.borderRadius,
      }}
    >
      {props.empty ? <EmptyColumnPlaceholder token={props.token} label={props.emptyLabel} /> : props.children}
    </div>
  );
}

function EmptyColumnPlaceholder(props: { token: ReturnType<typeof theme.useToken>["token"]; label: string }) {
  return (
    <div
      style={{
        border: `1px dashed ${props.token.colorBorder}`,
        borderRadius: props.token.borderRadius,
        padding: "24px 12px",
        textAlign: "center",
        color: props.token.colorTextTertiary,
        fontSize: 12,
      }}
    >
      {props.label}
    </div>
  );
}

interface TaskCardCommonProps {
  task: VaultTask;
  token: ReturnType<typeof theme.useToken>["token"];
  dateFormats: ReturnType<typeof useDateFormat>;
  dueLabel: string;
  onClick: () => void;
}

function TaskCardBody({ task, token, dateFormats, dueLabel }: Omit<TaskCardCommonProps, "onClick">) {
  return (
    <Card size="small" styles={{ body: { padding: 12 } }} style={{ borderRadius: token.borderRadius }}>
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
  );
}

function SortableTaskCard(props: TaskCardCommonProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: String(props.task.id),
  });
  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
    cursor: "grab",
  };
  // PointerSensor is configured with activationConstraint.distance:6 above —
  // a stationary mousedown+up fires onClick normally without triggering drag.
  return (
    <div ref={setNodeRef} style={style} {...attributes} {...listeners} onClick={props.onClick}>
      <TaskCardBody {...props} />
    </div>
  );
}

function PlainTaskCard(props: TaskCardCommonProps) {
  return (
    <div onClick={props.onClick} style={{ cursor: "pointer" }}>
      <TaskCardBody {...props} />
    </div>
  );
}
