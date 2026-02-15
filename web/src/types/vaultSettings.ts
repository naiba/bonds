export interface VaultSettingsResponse {
  id: number;
  name: string;
  description: string;
  default_template_id: number;
  show_group_tab: boolean;
  show_tasks_tab: boolean;
  show_files_tab: boolean;
  show_journal_tab: boolean;
  show_companies_tab: boolean;
  show_reports_tab: boolean;
  show_calendar_tab: boolean;
  created_at: string;
  updated_at: string;
}

export interface LabelResponse {
  id: number;
  name: string;
  slug: string;
  description: string;
  bg_color: string;
  text_color: string;
  created_at: string;
  updated_at: string;
}

export interface TagResponse {
  id: number;
  name: string;
  slug: string;
  created_at: string;
  updated_at: string;
}

export interface ImportantDateTypeResponse {
  id: number;
  label: string;
  internal_type: string;
  can_be_deleted: boolean;
  created_at: string;
  updated_at: string;
}

export interface MoodTrackingParameterResponse {
  id: number;
  label: string;
  hex_color: string;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface LifeEventCategoryTypeResponse {
  id: number;
  category_id: number;
  label: string;
  can_be_deleted: boolean;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface LifeEventCategoryResponse {
  id: number;
  label: string;
  can_be_deleted: boolean;
  position: number;
  types: LifeEventCategoryTypeResponse[];
  created_at: string;
  updated_at: string;
}

export interface QuickFactTemplateResponse {
  id: number;
  label: string;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface VaultUserResponse {
  id: number;
  user_id: number;
  email: string;
  first_name: string;
  last_name: string;
  permission: number;
}

export interface UpdateVaultSettingsRequest {
  name: string;
  description?: string;
  default_template_id?: number;
}

export interface UpdateTabVisibilityRequest {
  show_group_tab?: boolean;
  show_tasks_tab?: boolean;
  show_files_tab?: boolean;
  show_journal_tab?: boolean;
  show_companies_tab?: boolean;
  show_reports_tab?: boolean;
  show_calendar_tab?: boolean;
}

export interface AddVaultUserRequest {
  email: string;
  permission: number;
}

export interface UpdateVaultUserPermRequest {
  permission: number;
}

export interface CreateLabelRequest {
  name: string;
  description?: string;
  bg_color: string;
  text_color: string;
}

export type UpdateLabelRequest = Partial<CreateLabelRequest>;

export interface CreateTagRequest {
  name: string;
}

export type UpdateTagRequest = CreateTagRequest;

export interface CreateImportantDateTypeRequest {
  label: string;
}

export type UpdateImportantDateTypeRequest = CreateImportantDateTypeRequest;

export interface CreateMoodTrackingParameterRequest {
  label: string;
  hex_color: string;
}

export type UpdateMoodTrackingParameterRequest = CreateMoodTrackingParameterRequest;

export interface CreateLifeEventCategoryRequest {
  label: string;
}

export type UpdateLifeEventCategoryRequest = CreateLifeEventCategoryRequest;

export interface CreateLifeEventCategoryTypeRequest {
  label: string;
}

export type UpdateLifeEventCategoryTypeRequest = CreateLifeEventCategoryTypeRequest;

export interface CreateQuickFactTemplateRequest {
  label: string;
}

export type UpdateQuickFactTemplateRequest = CreateQuickFactTemplateRequest;
