// Task statuses surfaced in the kanban + edit modal.
// Mirrors models.TaskStatus* constants on the Go side. Keep in sync.
//
// "blocked" and "cancelled" exist as valid statuses on the backend
// (NormalizeTaskStatus accepts them) but the kanban currently doesn't
// render columns for them — adding columns is a future enhancement,
// likely tied to making the status set configurable via Personalize.
export const TASK_STATUSES = ["todo", "in_progress", "done"] as const;
export type TaskStatus = (typeof TASK_STATUSES)[number];

export function isTaskStatus(s: string | undefined | null): s is TaskStatus {
  return !!s && (TASK_STATUSES as readonly string[]).includes(s);
}

export function normalizeTaskStatus(s: string | undefined | null): TaskStatus {
  return isTaskStatus(s) ? s : "todo";
}
