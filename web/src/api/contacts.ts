import client from "./client";
import type { APIResponse } from "@/types/api";
import type {
  Contact,
  CreateContactRequest,
  UpdateContactRequest,
} from "@/types/contact";

export const contactsApi = {
  list(vaultId: string | number, params?: { page?: number; per_page?: number }) {
    return client.get<APIResponse<Contact[]>>(`/vaults/${vaultId}/contacts`, {
      params,
    });
  },

  get(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<Contact>>(
      `/vaults/${vaultId}/contacts/${contactId}`,
    );
  },

  create(vaultId: string | number, data: CreateContactRequest) {
    return client.post<APIResponse<Contact>>(
      `/vaults/${vaultId}/contacts`,
      data,
    );
  },

  update(vaultId: string | number, contactId: string | number, data: UpdateContactRequest) {
    return client.put<APIResponse<Contact>>(
      `/vaults/${vaultId}/contacts/${contactId}`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number) {
    return client.delete(`/vaults/${vaultId}/contacts/${contactId}`);
  },
};
