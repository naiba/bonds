import { useQuery } from "@tanstack/react-query";
import { api } from "@/api";

// Server-side TaskStatus row, surfaced via the personalize endpoint.
// Created via Personalize > Task statuses; ordered by position; the row
// flagged is_default is the one new tasks fall back to when no status is
// supplied. Core rows have can_be_deleted=false (Todo/In Progress/Done).
export interface TaskStatus {
  id: number;
  slug: string;
  label: string;
  position: number;
  is_default: boolean;
  can_be_deleted: boolean;
}

const TASK_STATUSES_QUERY_KEY = ["settings", "personalize", "task-statuses"];

/** React Query hook returning the account's configured task statuses,
 *  pre-sorted by position. The same query key is shared by the kanban,
 *  the edit modal, and the Personalize page so all three see the same
 *  cache and stay in sync after add/delete/rename. */
export function useTaskStatuses() {
  return useQuery<TaskStatus[]>({
    queryKey: TASK_STATUSES_QUERY_KEY,
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail("task-statuses");
      // The endpoint returns PersonalizeEntityResponse — extras (slug,
      // is_default, can_be_deleted) are populated only for task-statuses.
      type Row = {
        id?: number;
        slug?: string;
        label?: string;
        name?: string;
        position?: number;
        is_default?: boolean;
        can_be_deleted?: boolean;
      };
      const rows = (res.data ?? []) as Row[];
      return rows
        .map<TaskStatus>((r) => ({
          id: r.id ?? 0,
          slug: r.slug ?? "",
          label: r.label || r.name || "",
          position: r.position ?? 0,
          is_default: !!r.is_default,
          can_be_deleted: r.can_be_deleted !== false,
        }))
        .sort((a, b) => a.position - b.position);
    },
    staleTime: 60_000,
  });
}

export function defaultStatusSlug(statuses: TaskStatus[]): string {
  return (
    statuses.find((s) => s.is_default)?.slug ??
    statuses[0]?.slug ??
    "todo"
  );
}
