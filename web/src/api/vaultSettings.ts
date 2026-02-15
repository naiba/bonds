import client from "./client";
import type { APIResponse } from "@/types/api";
import type {
  VaultSettingsResponse,
  UpdateVaultSettingsRequest,
  UpdateTabVisibilityRequest,
  VaultUserResponse,
  AddVaultUserRequest,
  UpdateVaultUserPermRequest,
  LabelResponse,
  CreateLabelRequest,
  UpdateLabelRequest,
  TagResponse,
  CreateTagRequest,
  UpdateTagRequest,
  ImportantDateTypeResponse,
  CreateImportantDateTypeRequest,
  UpdateImportantDateTypeRequest,
  MoodTrackingParameterResponse,
  CreateMoodTrackingParameterRequest,
  UpdateMoodTrackingParameterRequest,
  LifeEventCategoryResponse,
  CreateLifeEventCategoryRequest,
  UpdateLifeEventCategoryRequest,
  CreateLifeEventCategoryTypeRequest,
  UpdateLifeEventCategoryTypeRequest,
  QuickFactTemplateResponse,
  CreateQuickFactTemplateRequest,
  UpdateQuickFactTemplateRequest,
} from "@/types/vaultSettings";

export const vaultSettingsApi = {
  getSettings(vaultId: number) {
    return client.get<APIResponse<VaultSettingsResponse>>(
      `/vaults/${vaultId}/settings`
    );
  },

  updateSettings(vaultId: number, data: UpdateVaultSettingsRequest) {
    return client.put<APIResponse<VaultSettingsResponse>>(
      `/vaults/${vaultId}/settings`,
      data
    );
  },

  updateDefaultTemplate(vaultId: number, templateId: number) {
    return client.put<APIResponse<VaultSettingsResponse>>(
      `/vaults/${vaultId}/settings/template`,
      { default_template_id: templateId }
    );
  },

  updateTabVisibility(vaultId: number, data: UpdateTabVisibilityRequest) {
    return client.put<APIResponse<VaultSettingsResponse>>(
      `/vaults/${vaultId}/settings/visibility`,
      data
    );
  },

  listUsers(vaultId: number) {
    return client.get<APIResponse<VaultUserResponse[]>>(
      `/vaults/${vaultId}/settings/users`
    );
  },

  inviteUser(vaultId: number, data: AddVaultUserRequest) {
    return client.post<APIResponse<VaultUserResponse>>(
      `/vaults/${vaultId}/settings/users`,
      data
    );
  },

  updateUserPermission(
    vaultId: number,
    userId: number,
    data: UpdateVaultUserPermRequest
  ) {
    return client.put<APIResponse<VaultUserResponse>>(
      `/vaults/${vaultId}/settings/users/${userId}`,
      data
    );
  },

  removeUser(vaultId: number, userId: number) {
    return client.delete(`/vaults/${vaultId}/settings/users/${userId}`);
  },

  listLabels(vaultId: number) {
    return client.get<APIResponse<LabelResponse[]>>(
      `/vaults/${vaultId}/settings/labels`
    );
  },

  createLabel(vaultId: number, data: CreateLabelRequest) {
    return client.post<APIResponse<LabelResponse>>(
      `/vaults/${vaultId}/settings/labels`,
      data
    );
  },

  updateLabel(vaultId: number, labelId: number, data: UpdateLabelRequest) {
    return client.put<APIResponse<LabelResponse>>(
      `/vaults/${vaultId}/settings/labels/${labelId}`,
      data
    );
  },

  deleteLabel(vaultId: number, labelId: number) {
    return client.delete(`/vaults/${vaultId}/settings/labels/${labelId}`);
  },

  listTags(vaultId: number) {
    return client.get<APIResponse<TagResponse[]>>(
      `/vaults/${vaultId}/settings/tags`
    );
  },

  createTag(vaultId: number, data: CreateTagRequest) {
    return client.post<APIResponse<TagResponse>>(
      `/vaults/${vaultId}/settings/tags`,
      data
    );
  },

  updateTag(vaultId: number, tagId: number, data: UpdateTagRequest) {
    return client.put<APIResponse<TagResponse>>(
      `/vaults/${vaultId}/settings/tags/${tagId}`,
      data
    );
  },

  deleteTag(vaultId: number, tagId: number) {
    return client.delete(`/vaults/${vaultId}/settings/tags/${tagId}`);
  },

  listImportantDateTypes(vaultId: number) {
    return client.get<APIResponse<ImportantDateTypeResponse[]>>(
      `/vaults/${vaultId}/settings/contactImportantDateTypes`
    );
  },

  createImportantDateType(
    vaultId: number,
    data: CreateImportantDateTypeRequest
  ) {
    return client.post<APIResponse<ImportantDateTypeResponse>>(
      `/vaults/${vaultId}/settings/contactImportantDateTypes`,
      data
    );
  },

  updateImportantDateType(
    vaultId: number,
    typeId: number,
    data: UpdateImportantDateTypeRequest
  ) {
    return client.put<APIResponse<ImportantDateTypeResponse>>(
      `/vaults/${vaultId}/settings/contactImportantDateTypes/${typeId}`,
      data
    );
  },

  deleteImportantDateType(vaultId: number, typeId: number) {
    return client.delete(
      `/vaults/${vaultId}/settings/contactImportantDateTypes/${typeId}`
    );
  },

  listMoodTrackingParameters(vaultId: number) {
    return client.get<APIResponse<MoodTrackingParameterResponse[]>>(
      `/vaults/${vaultId}/settings/moodTrackingParameters`
    );
  },

  createMoodTrackingParameter(
    vaultId: number,
    data: CreateMoodTrackingParameterRequest
  ) {
    return client.post<APIResponse<MoodTrackingParameterResponse>>(
      `/vaults/${vaultId}/settings/moodTrackingParameters`,
      data
    );
  },

  updateMoodTrackingParameter(
    vaultId: number,
    paramId: number,
    data: UpdateMoodTrackingParameterRequest
  ) {
    return client.put<APIResponse<MoodTrackingParameterResponse>>(
      `/vaults/${vaultId}/settings/moodTrackingParameters/${paramId}`,
      data
    );
  },

  deleteMoodTrackingParameter(vaultId: number, paramId: number) {
    return client.delete(
      `/vaults/${vaultId}/settings/moodTrackingParameters/${paramId}`
    );
  },

  listLifeEventCategories(vaultId: number) {
    return client.get<APIResponse<LifeEventCategoryResponse[]>>(
      `/vaults/${vaultId}/settings/lifeEventCategories`
    );
  },

  createLifeEventCategory(
    vaultId: number,
    data: CreateLifeEventCategoryRequest
  ) {
    return client.post<APIResponse<LifeEventCategoryResponse>>(
      `/vaults/${vaultId}/settings/lifeEventCategories`,
      data
    );
  },

  updateLifeEventCategory(
    vaultId: number,
    categoryId: number,
    data: UpdateLifeEventCategoryRequest
  ) {
    return client.put<APIResponse<LifeEventCategoryResponse>>(
      `/vaults/${vaultId}/settings/lifeEventCategories/${categoryId}`,
      data
    );
  },

  deleteLifeEventCategory(vaultId: number, categoryId: number) {
    return client.delete(
      `/vaults/${vaultId}/settings/lifeEventCategories/${categoryId}`
    );
  },

  createLifeEventCategoryType(
    vaultId: number,
    categoryId: number,
    data: CreateLifeEventCategoryTypeRequest
  ) {
    return client.post<APIResponse<LifeEventCategoryResponse>>(
      `/vaults/${vaultId}/settings/lifeEventCategories/${categoryId}/types`,
      data
    );
  },

  updateLifeEventCategoryType(
    vaultId: number,
    categoryId: number,
    typeId: number,
    data: UpdateLifeEventCategoryTypeRequest
  ) {
    return client.put<APIResponse<LifeEventCategoryResponse>>(
      `/vaults/${vaultId}/settings/lifeEventCategories/${categoryId}/types/${typeId}`,
      data
    );
  },

  deleteLifeEventCategoryType(
    vaultId: number,
    categoryId: number,
    typeId: number
  ) {
    return client.delete(
      `/vaults/${vaultId}/settings/lifeEventCategories/${categoryId}/types/${typeId}`
    );
  },

  listQuickFactTemplates(vaultId: number) {
    return client.get<APIResponse<QuickFactTemplateResponse[]>>(
      `/vaults/${vaultId}/settings/quickFactTemplates`
    );
  },

  createQuickFactTemplate(
    vaultId: number,
    data: CreateQuickFactTemplateRequest
  ) {
    return client.post<APIResponse<QuickFactTemplateResponse>>(
      `/vaults/${vaultId}/settings/quickFactTemplates`,
      data
    );
  },

  updateQuickFactTemplate(
    vaultId: number,
    templateId: number,
    data: UpdateQuickFactTemplateRequest
  ) {
    return client.put<APIResponse<QuickFactTemplateResponse>>(
      `/vaults/${vaultId}/settings/quickFactTemplates/${templateId}`,
      data
    );
  },

  deleteQuickFactTemplate(vaultId: number, templateId: number) {
    return client.delete(
      `/vaults/${vaultId}/settings/quickFactTemplates/${templateId}`
    );
  },
};
