import client from "./client";

export const twofactorApi = {
  getStatus: () => client.get("/settings/2fa/status"),
  enable: () => client.post("/settings/2fa/enable"),
  confirm: (code: string) => client.post("/settings/2fa/confirm", { code }),
  disable: (code: string) => client.post("/settings/2fa/disable", { code }),
};
