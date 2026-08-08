import * as yup from 'yup'

export const lockoutConfigSchema = yup.object({
  enabled: yup.boolean().required(),
  max_failed_attempts: yup
    .number()
    .required()
    .min(1, 'Must be at least 1')
    .max(100, 'Cannot exceed 100'),
  lockout_duration_minutes: yup
    .number()
    .required()
    .min(1, 'Must be at least 1'),
  progressive_lockout: yup.boolean().required(),
  auto_unlock: yup.boolean().required(),
  reset_count_on_success: yup.boolean().required(),
  observation_window_minutes: yup
    .number()
    .required()
    .min(1, 'Must be at least 1'),
  max_lockout_duration_minutes: yup
    .number()
    .required()
    .min(1, 'Must be at least 1'),
  progression_reset_hours: yup
    .number()
    .required()
    .min(1, 'Must be at least 1'),
  notify_user_on_lockout: yup.boolean().required(),
}).required()

export type LockoutConfigFormData = yup.InferType<typeof lockoutConfigSchema>
