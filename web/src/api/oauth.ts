import client from "./client";
import type { APIResponse } from "@/types/api";
import type { OAuthProvider } from "@/types/settings_extra";

export const oauthApi = {
  listProviders() {
    return client.get<APIResponse<OAuthProvider[]>>("/settings/oauth");
  },

  unlinkProvider(driver: string) {
    return client.delete<APIResponse<void>>(`/settings/oauth/${driver}`);
  },
};
