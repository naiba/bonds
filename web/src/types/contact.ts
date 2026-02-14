export interface Contact {
  id: number;
  vault_id: number;
  first_name: string;
  last_name: string;
  nickname: string;
  is_archived: boolean;
  is_favorite: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateContactRequest {
  first_name: string;
  last_name: string;
  nickname: string;
}

export interface UpdateContactRequest {
  first_name: string;
  last_name: string;
  nickname: string;
}
