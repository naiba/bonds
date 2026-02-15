import client from "./client";
import type { APIResponse } from "@/types/api";
import type {
  AddressReportItem,
  ImportantDateReportItem,
  MoodReportItem,
} from "@/types/modules";

export const reportsApi = {
  index: (vaultId: string) =>
    client.get<APIResponse<{ key: string; name: string }[]>>(`/vaults/${vaultId}/reports`),

  addresses: (vaultId: string) =>
    client.get<APIResponse<AddressReportItem[]>>(`/vaults/${vaultId}/reports/addresses`),

  importantDates: (vaultId: string) =>
    client.get<APIResponse<ImportantDateReportItem[]>>(`/vaults/${vaultId}/reports/importantDates`),

  moodTracking: (vaultId: string) =>
    client.get<APIResponse<MoodReportItem[]>>(`/vaults/${vaultId}/reports/moodTrackingEvents`),
};
