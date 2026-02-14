import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Note, CreateNoteRequest } from "@/types/modules";

export const notesApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<Note[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/notes`,
    );
  },

  get(vaultId: string | number, contactId: string | number, noteId: number) {
    return client.get<APIResponse<Note>>(
      `/vaults/${vaultId}/contacts/${contactId}/notes/${noteId}`,
    );
  },

  create(vaultId: string | number, contactId: string | number, data: CreateNoteRequest) {
    return client.post<APIResponse<Note>>(
      `/vaults/${vaultId}/contacts/${contactId}/notes`,
      data,
    );
  },

  update(
    vaultId: string | number,
    contactId: string | number,
    noteId: number,
    data: CreateNoteRequest,
  ) {
    return client.put<APIResponse<Note>>(
      `/vaults/${vaultId}/contacts/${contactId}/notes/${noteId}`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number, noteId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/notes/${noteId}`,
    );
  },
};
