import * as yup from 'yup'

// Every key here is submitted verbatim as the rate_limit_config body, and
// ValidateRateLimitConfig in internal/tenant/validation_setting.go 422s on the
// first field it does not recognise. Only enabled, per_ip, requests_per_window,
// window_duration_seconds, exempt_ips and endpoint_overrides are accepted — a
// field the backend has no concept of (there was a per_api_key here) makes the
// whole panel unsaveable, not just that one switch.
export const rateLimitConfigSchema = yup.object({
  enabled: yup.boolean().required(),
  requests_per_window: yup.number().required().min(1, 'Must be at least 1').integer(),
  window_duration_seconds: yup.number().required().min(1, 'Must be at least 1').integer(),
  per_ip: yup.boolean().required(),
}).required()

export type RateLimitConfigFormData = yup.InferType<typeof rateLimitConfigSchema>
