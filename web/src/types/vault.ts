export interface Vault {
  id: number;
  account_id: number;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateVaultRequest {
  name: string;
  description: string;
}

export interface UpdateVaultRequest {
  name: string;
  description: string;
}
