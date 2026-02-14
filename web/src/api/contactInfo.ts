import client from "./client";
import type { APIResponse } from "@/types/api";
import type { ContactInfo, CreateContactInfoRequest } from "@/types/modules";

export const contactInfoApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<ContactInfo[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/contact-info`,
    );
  },

  create(vaultId: string | number, contactId: string | number, data: CreateContactInfoRequest) {
    return client.post<APIResponse<ContactInfo>>(
      `/vaults/${vaultId}/contacts/${contactId}/contact-info`,
      data,
    );
  },

  update(
    vaultId: string | number,
    contactId: string | number,
    infoId: number,
    data: CreateContactInfoRequest,
  ) {
    return client.put<APIResponse<ContactInfo>>(
      `/vaults/${vaultId}/contacts/${contactId}/contact-info/${infoId}`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number, infoId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/contact-info/${infoId}`,
    );
  },
};
