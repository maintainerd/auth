import * as yup from 'yup'

export const threatDetectionSettingsSchema = yup.object({
  brute_force_detection_enabled: yup.boolean().required(),
  impossible_travel_detection_enabled: yup.boolean().required(),
  new_device_notification_enabled: yup.boolean().required(),
  velocity_check_enabled: yup.boolean().required(),
  risk_based_step_up_enabled: yup.boolean().required(),
  compromised_credential_monitoring_enabled: yup.boolean().required(),
  ip_reputation_check_enabled: yup.boolean().required(),
  block_tor_exit_nodes: yup.boolean().required(),
  risk_step_up_threshold: yup
    .number()
    .required()
    .min(0, 'Cannot be negative')
    .max(100, 'Cannot exceed 100'),
  risk_block_threshold: yup
    .number()
    .required()
    .min(0, 'Cannot be negative')
    .max(100, 'Cannot exceed 100'),
  velocity_failures_per_ip_per_hour: yup
    .number()
    .required()
    .min(1, 'Must be at least 1'),
  distinct_accounts_per_ip_per_hour: yup
    .number()
    .required()
    .min(1, 'Must be at least 1'),
}).required()

export type ThreatDetectionSettingsFormData = yup.InferType<typeof threatDetectionSettingsSchema>
