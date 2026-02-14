import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Address, CreateAddressRequest } from "@/types/modules";

export const addressesApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<Address[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/addresses`,
    );
  },

  create(vaultId: string | number, contactId: string | number, data: CreateAddressRequest) {
    return client.post<APIResponse<Address>>(
      `/vaults/${vaultId}/contacts/${contactId}/addresses`,
      data,
    );
  },

  update(
    vaultId: string | number,
    contactId: string | number,
    addressId: number,
    data: CreateAddressRequest,
  ) {
    return client.put<APIResponse<Address>>(
      `/vaults/${vaultId}/contacts/${contactId}/addresses/${addressId}`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number, addressId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/addresses/${addressId}`,
    );
  },
};
