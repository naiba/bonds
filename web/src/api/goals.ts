import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Goal, Streak, CreateGoalRequest } from "@/types/modules";

export const goalsApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<Goal[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/goals`,
    );
  },

  create(vaultId: string | number, contactId: string | number, data: CreateGoalRequest) {
    return client.post<APIResponse<Goal>>(
      `/vaults/${vaultId}/contacts/${contactId}/goals`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number, goalId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/goals/${goalId}`,
    );
  },

  toggleStreak(
    vaultId: string | number,
    contactId: string | number,
    goalId: number,
    date: string,
  ) {
    return client.post<APIResponse<Streak>>(
      `/vaults/${vaultId}/contacts/${contactId}/goals/${goalId}/streaks`,
      { happened_at: date },
    );
  },
};
