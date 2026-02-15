import client from "./client";
import type { APIResponse } from "@/types/api";
import type { 
  Contact, 
  UpdateJobInfoRequest, 
  UpdateContactReligionRequest, 
  MoveContactRequest
} from "@/types/contact";
import type { FeedItem } from "@/types/modules";

export const contactExtraApi = {
  updateJobInfo(vaultId: string | number, contactId: string | number, data: UpdateJobInfoRequest) {
    return client.put<APIResponse<Contact>>(
      `/vaults/${vaultId}/contacts/${contactId}/jobInformation`,
      data
    );
  },

  deleteJobInfo(vaultId: string | number, contactId: string | number) {
    return client.delete<APIResponse<Contact>>(
      `/vaults/${vaultId}/contacts/${contactId}/jobInformation`
    );
  },

  updateReligion(vaultId: string | number, contactId: string | number, data: UpdateContactReligionRequest) {
    return client.put<APIResponse<Contact>>(
      `/vaults/${vaultId}/contacts/${contactId}/religion`,
      data
    );
  },

  getFeed(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<FeedItem[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/feed`
    );
  },

  uploadAvatar(vaultId: string | number, contactId: string | number, file: File) {
    const formData = new FormData();
    formData.append("avatar", file);
    return client.put<APIResponse<Contact>>(
      `/vaults/${vaultId}/contacts/${contactId}/avatar`,
      formData,
      {
        headers: {
          "Content-Type": "multipart/form-data",
        },
      }
    );
  },

  deleteAvatar(vaultId: string | number, contactId: string | number) {
    return client.delete<APIResponse<Contact>>(
      `/vaults/${vaultId}/contacts/${contactId}/avatar`
    );
  },
  
  move(vaultId: string | number, contactId: string | number, data: MoveContactRequest) {
    return client.post<APIResponse<Contact>>(
      `/vaults/${vaultId}/contacts/${contactId}/move`,
      data
    );
  },
};
