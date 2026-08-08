export interface SessionSettings {
  access_token_ttl_minutes: number
  refresh_token_ttl_days: number
  max_concurrent_sessions: number
  idle_timeout_minutes: number
  absolute_timeout_hours: number
  rotate_refresh_tokens: boolean
  refresh_token_reuse_interval_seconds: number
  cookie_secure: boolean
  cookie_http_only: boolean
  cookie_same_site: string
  revoke_sessions_on_password_change: boolean
}

export interface SessionSettingsPayload {
  access_token_ttl_minutes?: number
  refresh_token_ttl_days?: number
  max_concurrent_sessions?: number
  idle_timeout_minutes?: number
  absolute_timeout_hours?: number
  rotate_refresh_tokens?: boolean
  refresh_token_reuse_interval_seconds?: number
  cookie_secure?: boolean
  cookie_http_only?: boolean
  cookie_same_site?: string
  revoke_sessions_on_password_change?: boolean
}

export interface SessionSettingsResponse {
  success: boolean
  data: SessionSettings
  message: string
}
