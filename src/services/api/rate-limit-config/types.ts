// The exact key set ValidateRateLimitConfig accepts (internal/tenant/validation_setting.go);
// the stored config carries no others either (see the rate_limit_config default
// in internal/platform/database/migration/003_create_tenant_settings_table.go).
// Because RateLimitConfigPayload is derived from this interface, an extra field
// here is not a harmless unused property — it is a 422 on every save.
export interface RateLimitConfig {
  enabled: boolean
  requests_per_window: number
  window_duration_seconds: number
  per_ip: boolean
  exempt_ips: string[]
  endpoint_overrides: Record<string, number>
}

export interface RateLimitConfigResponse {
  success: boolean
  data: RateLimitConfig
  message: string
}

export type RateLimitConfigPayload = Partial<RateLimitConfig>
