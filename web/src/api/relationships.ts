import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Relationship, CreateRelationshipRequest } from "@/types/modules";

export const relationshipsApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<Relationship[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/relationships`,
    );
  },

  create(
    vaultId: string | number,
    contactId: string | number,
    data: CreateRelationshipRequest,
  ) {
    return client.post<APIResponse<Relationship>>(
      `/vaults/${vaultId}/contacts/${contactId}/relationships`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number, relationshipId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/relationships/${relationshipId}`,
    );
  },
};
