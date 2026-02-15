import client from "./client";
import type { APIResponse } from "@/types/api";
import type { LifeMetric, CreateLifeMetricRequest, AddLifeMetricContactRequest } from "@/types/modules";

export const lifeMetricsApi = {
  list: (vaultId: string) =>
    client.get<APIResponse<LifeMetric[]>>(`/vaults/${vaultId}/lifeMetrics`),

  create: (vaultId: string, data: CreateLifeMetricRequest) =>
    client.post<APIResponse<LifeMetric>>(`/vaults/${vaultId}/lifeMetrics`, data),

  update: (vaultId: string, id: number, data: CreateLifeMetricRequest) =>
    client.put<APIResponse<LifeMetric>>(`/vaults/${vaultId}/lifeMetrics/${id}`, data),

  delete: (vaultId: string, id: number) =>
    client.delete<APIResponse<void>>(`/vaults/${vaultId}/lifeMetrics/${id}`),

  addContact: (vaultId: string, id: number, data: AddLifeMetricContactRequest) =>
    client.post<APIResponse<void>>(`/vaults/${vaultId}/lifeMetrics/${id}/contacts`, data),

  removeContact: (vaultId: string, id: number, contactId: number) =>
    client.delete<APIResponse<void>>(`/vaults/${vaultId}/lifeMetrics/${id}/contacts/${contactId}`),
};
