import client from "./client";
import type { APIResponse } from "@/types/api";
import type {
  WebAuthnCredential,
  WebAuthnRegistrationStartResponse,
  WebAuthnLoginStartResponse,
  RegistrationResponseJSON,
  AuthenticationResponseJSON,
} from "@/types/settings_extra";

export const webauthnApi = {
  listCredentials() {
    return client.get<APIResponse<WebAuthnCredential[]>>("/settings/webauthn/credentials");
  },

  deleteCredential(id: number) {
    return client.delete<APIResponse<void>>(`/settings/webauthn/credentials/${id}`);
  },

  registerBegin() {
    return client.post<APIResponse<WebAuthnRegistrationStartResponse>>(
      "/settings/webauthn/register/begin"
    );
  },

  registerFinish(data: RegistrationResponseJSON) {
    return client.post<APIResponse<void>>("/settings/webauthn/register/finish", data);
  },

  loginBegin(email: string) {
    return client.post<APIResponse<WebAuthnLoginStartResponse>>(
      "/auth/webauthn/login/begin",
      { email }
    );
  },

  loginFinish(data: AuthenticationResponseJSON) {
    return client.post<APIResponse<{ token: string }>>(
      "/auth/webauthn/login/finish",
      data
    );
  },
};
