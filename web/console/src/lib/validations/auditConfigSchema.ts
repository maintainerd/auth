import * as yup from 'yup'

export const auditConfigSchema = yup.object({
  enabled: yup.boolean().required(),
  retention_days: yup.number().required().min(1, 'Must be at least 1').integer(),
  pii_masking: yup.boolean().required(),
  log_level: yup.string().required().oneOf(['debug', 'info', 'warn', 'critical'], 'Invalid log level'),
}).required()

export type AuditConfigFormData = yup.InferType<typeof auditConfigSchema>
