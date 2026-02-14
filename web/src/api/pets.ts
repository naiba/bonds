import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Pet, CreatePetRequest } from "@/types/modules";

export const petsApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<Pet[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/pets`,
    );
  },

  create(vaultId: string | number, contactId: string | number, data: CreatePetRequest) {
    return client.post<APIResponse<Pet>>(
      `/vaults/${vaultId}/contacts/${contactId}/pets`,
      data,
    );
  },

  update(
    vaultId: string | number,
    contactId: string | number,
    petId: number,
    data: CreatePetRequest,
  ) {
    return client.put<APIResponse<Pet>>(
      `/vaults/${vaultId}/contacts/${contactId}/pets/${petId}`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number, petId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/pets/${petId}`,
    );
  },
};
