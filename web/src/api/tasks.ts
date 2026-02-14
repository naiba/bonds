import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Task, CreateTaskRequest } from "@/types/modules";

export const tasksApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<Task[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/tasks`,
    );
  },

  listAll(vaultId: string | number) {
    return client.get<APIResponse<Task[]>>(`/vaults/${vaultId}/tasks`);
  },

  create(vaultId: string | number, contactId: string | number, data: CreateTaskRequest) {
    return client.post<APIResponse<Task>>(
      `/vaults/${vaultId}/contacts/${contactId}/tasks`,
      data,
    );
  },

  update(
    vaultId: string | number,
    contactId: string | number,
    taskId: number,
    data: Partial<CreateTaskRequest> & { is_completed?: boolean },
  ) {
    return client.put<APIResponse<Task>>(
      `/vaults/${vaultId}/contacts/${contactId}/tasks/${taskId}`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number, taskId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/tasks/${taskId}`,
    );
  },
};
