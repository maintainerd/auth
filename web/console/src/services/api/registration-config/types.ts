export interface RegistrationConfig {
  self_registration_enabled: boolean
  require_email_verification: boolean
  require_phone_verification: boolean
  allowed_email_domains: string[]
  blocked_email_domains: string[]
  auto_confirm_enabled: boolean
  verification_token_ttl_hours: number
  captcha_on_signup: boolean
  registration_rate_limit_per_ip_per_hour: number
}

export interface RegistrationConfigPayload {
  self_registration_enabled?: boolean
  require_email_verification?: boolean
  require_phone_verification?: boolean
  allowed_email_domains?: string[]
  blocked_email_domains?: string[]
  auto_confirm_enabled?: boolean
  verification_token_ttl_hours?: number
  captcha_on_signup?: boolean
  registration_rate_limit_per_ip_per_hour?: number
}

export interface RegistrationConfigResponse {
  success: boolean
  data: RegistrationConfig
  message: string
}
