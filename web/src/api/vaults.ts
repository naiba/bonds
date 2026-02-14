import client from "./client";
import type { APIResponse } from "@/types/api";
import type {
  Vault,
  CreateVaultRequest,
  UpdateVaultRequest,
} from "@/types/vault";

export const vaultsApi = {
  list() {
    return client.get<APIResponse<Vault[]>>("/vaults");
  },

  get(id: string | number) {
    return client.get<APIResponse<Vault>>(`/vaults/${id}`);
  },

  create(data: CreateVaultRequest) {
    return client.post<APIResponse<Vault>>("/vaults", data);
  },

  update(id: string | number, data: UpdateVaultRequest) {
    return client.put<APIResponse<Vault>>(`/vaults/${id}`, data);
  },

  delete(id: string | number) {
    return client.delete(`/vaults/${id}`);
  },
};
