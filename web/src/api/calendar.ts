import client from "./client";
import type { APIResponse } from "@/types/api";
import type { CalendarResponse, VaultReminderItem } from "@/types/modules";

export const calendarApi = {
  get: (vaultId: string) =>
    client.get<APIResponse<CalendarResponse>>(`/vaults/${vaultId}/calendar`),

  getMonth: (vaultId: string, year: number, month: number) =>
    client.get<APIResponse<CalendarResponse>>(
      `/vaults/${vaultId}/calendar/years/${year}/months/${month}`
    ),

  getDay: (vaultId: string, year: number, month: number, day: number) =>
    client.get<APIResponse<CalendarResponse>>(
      `/vaults/${vaultId}/calendar/years/${year}/months/${month}/days/${day}`
    ),

  getReminders: (vaultId: string) =>
    client.get<APIResponse<VaultReminderItem[]>>(`/vaults/${vaultId}/reminders`),
};
