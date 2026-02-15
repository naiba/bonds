export interface Contact {
  id: number;
  vault_id: number;
  first_name: string;
  last_name: string;
  nickname: string;
  is_archived: boolean;
  is_favorite: boolean;
  gender_id?: number;
  pronoun_id?: number;
  template_id?: number;
  company_id?: number;
  file_id?: number;
  religion_id?: number;
  job_position?: string;
  labels?: ContactLabel[];
  avatar_url?: string;
  created_at: string;
  updated_at: string;
}

export interface ContactLabel {
  id: number;
  label_id: number;
  name: string;
  bg_color: string;
  text_color: string;
  created_at: string;
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

export interface AddContactLabelRequest {
  label_id: number;
}

export interface UpdateJobInfoRequest {
  company_id?: number;
  job_position?: string;
}

export interface UpdateContactReligionRequest {
  religion_id?: number;
}

export interface MoveContactRequest {
  target_vault_id: string;
}

export interface UpdateContactSortRequest {
  sort_order: string;
}
