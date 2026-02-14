import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Reminder, CreateReminderRequest } from "@/types/modules";

export const remindersApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<Reminder[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/reminders`,
    );
  },

  create(vaultId: string | number, contactId: string | number, data: CreateReminderRequest) {
    return client.post<APIResponse<Reminder>>(
      `/vaults/${vaultId}/contacts/${contactId}/reminders`,
      data,
    );
  },

  update(
    vaultId: string | number,
    contactId: string | number,
    reminderId: number,
    data: CreateReminderRequest,
  ) {
    return client.put<APIResponse<Reminder>>(
      `/vaults/${vaultId}/contacts/${contactId}/reminders/${reminderId}`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number, reminderId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/reminders/${reminderId}`,
    );
  },
};
