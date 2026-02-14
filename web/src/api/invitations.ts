import client from "./client";

export const invitationsApi = {
  list: () => client.get("/settings/invitations"),
  create: (data: { email: string; permission: number }) =>
    client.post("/settings/invitations", data),
  delete: (id: number) => client.delete(`/settings/invitations/${id}`),
  accept: (data: {
    token: string;
    first_name: string;
    last_name?: string;
    password: string;
  }) => client.post("/invitations/accept", data),
};
