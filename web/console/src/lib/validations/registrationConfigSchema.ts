import * as yup from 'yup'

export const registrationConfigSchema = yup.object({
  self_registration_enabled: yup.boolean().required(),
  require_email_verification: yup.boolean().required(),
  require_phone_verification: yup.boolean().required(),
  auto_confirm_enabled: yup.boolean().required(),
  verification_token_ttl_hours: yup
    .number()
    .required()
    .min(1, 'Must be at least 1'),
  captcha_on_signup: yup.boolean().required(),
  registration_rate_limit_per_ip_per_hour: yup
    .number()
    .required()
    .min(1, 'Must be at least 1'),
  allowed_email_domains: yup.array().of(yup.string().defined()).required(),
  blocked_email_domains: yup.array().of(yup.string().defined()).required(),
}).required()

export type RegistrationConfigFormData = yup.InferType<typeof registrationConfigSchema>
