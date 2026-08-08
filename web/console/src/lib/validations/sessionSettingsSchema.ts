import * as yup from 'yup'

export const sessionSettingsSchema = yup.object({
  access_token_ttl_minutes: yup
    .number()
    .required()
    .min(1, 'Must be at least 1')
    .max(60, 'Cannot exceed 60'),
  refresh_token_ttl_days: yup
    .number()
    .required()
    .min(1, 'Must be at least 1')
    .max(365, 'Cannot exceed 365'),
  max_concurrent_sessions: yup
    .number()
    .required()
    .min(0, 'Cannot be negative'),
  idle_timeout_minutes: yup
    .number()
    .required()
    .min(1, 'Must be at least 1'),
  absolute_timeout_hours: yup
    .number()
    .required()
    .min(1, 'Must be at least 1'),
  rotate_refresh_tokens: yup.boolean().required(),
  refresh_token_reuse_interval_seconds: yup
    .number()
    .required()
    .min(0, 'Cannot be negative'),
  cookie_secure: yup.boolean().required(),
  cookie_http_only: yup.boolean().required(),
  cookie_same_site: yup
    .string()
    .required()
    .oneOf(['Strict', 'Lax', 'None']),
  revoke_sessions_on_password_change: yup.boolean().required(),
}).required()

export type SessionSettingsFormData = yup.InferType<typeof sessionSettingsSchema>
