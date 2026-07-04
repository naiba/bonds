import type { VaultTask } from "@/api";

export const TASK_SORT_STORAGE_KEY = "bonds_vault_tasks_sort";

export type TaskSortMode = "custom" | "due_date";

function parseDueAt(value: string | undefined): number | null {
  if (!value) return null;

  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? null : timestamp;
}

function compareByDueDate(left: VaultTask, right: VaultTask): number {
  const leftDueAt = parseDueAt(left.due_at);
  const rightDueAt = parseDueAt(right.due_at);

  if (leftDueAt === null && rightDueAt === null) return 0;
  if (leftDueAt === null) return 1;
  if (rightDueAt === null) return -1;

  return leftDueAt - rightDueAt;
}

export function loadTaskSortMode(): TaskSortMode {
  try {
    const saved = localStorage.getItem(TASK_SORT_STORAGE_KEY);
    if (saved === "custom" || saved === "due_date") {
      return saved;
    }
  } catch (error) {
    if (error instanceof Error) {
      return "custom";
    }

    throw error;
  }

  return "custom";
}

export function persistTaskSortMode(next: TaskSortMode): void {
  try {
    localStorage.setItem(TASK_SORT_STORAGE_KEY, next);
  } catch (error) {
    if (!(error instanceof Error)) {
      throw error;
    }
  }
}

export function sortTasksForListView(tasks: readonly VaultTask[], mode: TaskSortMode): VaultTask[] {
  if (mode === "custom") {
    return [...tasks];
  }

  return [...tasks]
    .map((task, index) => ({ task, index }))
    .sort((left, right) => {
      const dueDateOrder = compareByDueDate(left.task, right.task);
      if (dueDateOrder !== 0) return dueDateOrder;
      return left.index - right.index;
    })
    .map(({ task }) => task);
}
