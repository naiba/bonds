import client from "./client";

export const searchApi = {
  search: (vaultId: string | number, query: string, page = 1, perPage = 20) =>
    client.get(`/vaults/${vaultId}/search`, {
      params: { q: query, page, per_page: perPage },
    }),
};
