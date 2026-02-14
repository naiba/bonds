import client from "./client";

export const vcardApi = {
  exportContact: (
    vaultId: string | number,
    contactId: string | number,
  ) =>
    client.get(`/vaults/${vaultId}/contacts/${contactId}/vcard`, {
      responseType: "blob",
    }),
  exportVault: (vaultId: string | number) =>
    client.get(`/vaults/${vaultId}/contacts/export`, {
      responseType: "blob",
    }),
  importVCard: (vaultId: string | number, file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    return client.post(`/vaults/${vaultId}/contacts/import`, formData);
  },
};
