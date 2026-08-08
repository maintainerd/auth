export interface MfaConfig {
  mode: string
  allowed_methods: string[]
  totp_issuer: string
  trusted_device_period_days: number
  grace_period_days: number
  preferred_method: string
  allow_sms: boolean
  allow_email_otp: boolean
  totp_digits: number
  totp_period_seconds: number
  recovery_codes_count: number
  require_mfa_for_sensitive_actions: boolean
  admin_grace_period_days: number
  step_up_ttl_minutes: number
}

export interface MfaConfigPayload {
  mode?: string
  allowed_methods?: string[]
  totp_issuer?: string
  trusted_device_period_days?: number
  grace_period_days?: number
  preferred_method?: string
  allow_sms?: boolean
  allow_email_otp?: boolean
  totp_digits?: number
  totp_period_seconds?: number
  recovery_codes_count?: number
  require_mfa_for_sensitive_actions?: boolean
  admin_grace_period_days?: number
  step_up_ttl_minutes?: number
}

export interface MfaConfigResponse {
  success: boolean
  data: MfaConfig
  message: string
}
