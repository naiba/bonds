import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Call, CreateCallRequest } from "@/types/modules";

export const callsApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<Call[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/calls`,
    );
  },

  create(vaultId: string | number, contactId: string | number, data: CreateCallRequest) {
    return client.post<APIResponse<Call>>(
      `/vaults/${vaultId}/contacts/${contactId}/calls`,
      data,
    );
  },

  update(
    vaultId: string | number,
    contactId: string | number,
    callId: number,
    data: CreateCallRequest,
  ) {
    return client.put<APIResponse<Call>>(
      `/vaults/${vaultId}/contacts/${contactId}/calls/${callId}`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number, callId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/calls/${callId}`,
    );
  },
};
