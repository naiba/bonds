import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Group, CreateGroupRequest } from "@/types/modules";

export const groupsApi = {
  list(vaultId: string | number) {
    return client.get<APIResponse<Group[]>>(`/vaults/${vaultId}/groups`);
  },

  get(vaultId: string | number, groupId: string | number) {
    return client.get<APIResponse<Group>>(
      `/vaults/${vaultId}/groups/${groupId}`,
    );
  },

  create(vaultId: string | number, data: CreateGroupRequest) {
    return client.post<APIResponse<Group>>(
      `/vaults/${vaultId}/groups`,
      data,
    );
  },

  update(vaultId: string | number, groupId: string | number, data: CreateGroupRequest) {
    return client.put<APIResponse<Group>>(
      `/vaults/${vaultId}/groups/${groupId}`,
      data,
    );
  },

  delete(vaultId: string | number, groupId: string | number) {
    return client.delete(`/vaults/${vaultId}/groups/${groupId}`);
  },

  addContact(vaultId: string | number, groupId: string | number, contactId: string | number) {
    return client.post(
      `/vaults/${vaultId}/groups/${groupId}/contacts`,
      { contact_id: contactId },
    );
  },

  removeContact(vaultId: string | number, groupId: string | number, contactId: string | number) {
    return client.delete(
      `/vaults/${vaultId}/groups/${groupId}/contacts/${contactId}`,
    );
  },
};
