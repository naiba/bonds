export interface WebAuthnCredential {
  id: number;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface WebAuthnRegistrationStartResponse {
  publicKey: PublicKeyCredentialCreationOptionsJSON;
}

export interface WebAuthnLoginStartResponse {
  publicKey: PublicKeyCredentialRequestOptionsJSON;
}

// WebAuthn credential option types — using Record<string, unknown> as these
// are complex protocol objects passed through to the browser's navigator.credentials API
export type PublicKeyCredentialCreationOptionsJSON = Record<string, unknown>;
export type PublicKeyCredentialRequestOptionsJSON = Record<string, unknown>;
export type RegistrationResponseJSON = Record<string, unknown>;
export type AuthenticationResponseJSON = Record<string, unknown>;

export interface OAuthProvider {
  driver: string;
  id: string;
  name: string;
  avatar_url?: string;
  created_at: string;
}

export interface Currency {
  id: number;
  code: string;
  name: string;
  symbol: string;
}

export interface StorageUsage {
  used_bytes: number;
  limit_bytes: number;
}
