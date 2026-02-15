import client from "./client";
import type { APIResponse } from "@/types/api";
import type { ContactLabel, AddContactLabelRequest } from "@/types/contact";

export const contactLabelsApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<ContactLabel[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/labels`
    );
  },

  add(vaultId: string | number, contactId: string | number, data: AddContactLabelRequest) {
    return client.post<APIResponse<ContactLabel>>(
      `/vaults/${vaultId}/contacts/${contactId}/labels`,
      data
    );
  },

  remove(vaultId: string | number, contactId: string | number, labelId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/labels/${labelId}`
    );
  },
};
