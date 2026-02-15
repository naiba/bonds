import client from "./client";
import type { APIResponse } from "@/types/api";
import type { ImportantDate, CreateImportantDateRequest } from "@/types/modules";

export const importantDatesApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<ImportantDate[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/dates`,
    );
  },

  create(
    vaultId: string | number,
    contactId: string | number,
    data: CreateImportantDateRequest,
  ) {
    return client.post<APIResponse<ImportantDate>>(
      `/vaults/${vaultId}/contacts/${contactId}/dates`,
      data,
    );
  },

  update(
    vaultId: string | number,
    contactId: string | number,
    dateId: number,
    data: CreateImportantDateRequest,
  ) {
    return client.put<APIResponse<ImportantDate>>(
      `/vaults/${vaultId}/contacts/${contactId}/dates/${dateId}`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number, dateId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/dates/${dateId}`,
    );
  },
};
