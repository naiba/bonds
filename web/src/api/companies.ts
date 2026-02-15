import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Company, CreateCompanyRequest } from "@/types/modules";

export const companiesApi = {
  list: (vaultId: string) =>
    client.get<APIResponse<Company[]>>(`/vaults/${vaultId}/companies`),

  get: (vaultId: string, id: number) =>
    client.get<APIResponse<Company>>(`/vaults/${vaultId}/companies/${id}`),

  create: (vaultId: string, data: CreateCompanyRequest) =>
    client.post<APIResponse<Company>>(`/vaults/${vaultId}/companies`, data),

  update: (vaultId: string, id: number, data: CreateCompanyRequest) =>
    client.put<APIResponse<Company>>(`/vaults/${vaultId}/companies/${id}`, data),

  delete: (vaultId: string, id: number) =>
    client.delete<APIResponse<void>>(`/vaults/${vaultId}/companies/${id}`),
};
