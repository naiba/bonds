import client from "./client";
import type { APIResponse } from "@/types/api";
import type {
  Journal,
  Post,
  CreateJournalRequest,
  CreatePostRequest,
} from "@/types/modules";

export const journalsApi = {
  list(vaultId: string | number) {
    return client.get<APIResponse<Journal[]>>(`/vaults/${vaultId}/journals`);
  },

  get(vaultId: string | number, journalId: string | number) {
    return client.get<APIResponse<Journal>>(
      `/vaults/${vaultId}/journals/${journalId}`,
    );
  },

  create(vaultId: string | number, data: CreateJournalRequest) {
    return client.post<APIResponse<Journal>>(
      `/vaults/${vaultId}/journals`,
      data,
    );
  },

  update(vaultId: string | number, journalId: string | number, data: CreateJournalRequest) {
    return client.put<APIResponse<Journal>>(
      `/vaults/${vaultId}/journals/${journalId}`,
      data,
    );
  },

  delete(vaultId: string | number, journalId: string | number) {
    return client.delete(`/vaults/${vaultId}/journals/${journalId}`);
  },

  listPosts(vaultId: string | number, journalId: string | number) {
    return client.get<APIResponse<Post[]>>(
      `/vaults/${vaultId}/journals/${journalId}/posts`,
    );
  },

  getPost(vaultId: string | number, journalId: string | number, postId: string | number) {
    return client.get<APIResponse<Post>>(
      `/vaults/${vaultId}/journals/${journalId}/posts/${postId}`,
    );
  },

  createPost(vaultId: string | number, journalId: string | number, data: CreatePostRequest) {
    return client.post<APIResponse<Post>>(
      `/vaults/${vaultId}/journals/${journalId}/posts`,
      data,
    );
  },

  updatePost(
    vaultId: string | number,
    journalId: string | number,
    postId: string | number,
    data: CreatePostRequest,
  ) {
    return client.put<APIResponse<Post>>(
      `/vaults/${vaultId}/journals/${journalId}/posts/${postId}`,
      data,
    );
  },

  deletePost(vaultId: string | number, journalId: string | number, postId: string | number) {
    return client.delete(
      `/vaults/${vaultId}/journals/${journalId}/posts/${postId}`,
    );
  },
};
