export interface TwoFactorStatus {
  enabled: boolean;
}

export interface TwoFactorSetup {
  secret: string;
  qr_code?: string;
  recovery_codes: string[];
}
