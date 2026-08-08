export interface ThreatDetectionSettings {
  brute_force_detection_enabled: boolean
  impossible_travel_detection_enabled: boolean
  new_device_notification_enabled: boolean
  velocity_check_enabled: boolean
  risk_based_step_up_enabled: boolean
  compromised_credential_monitoring_enabled: boolean
  ip_reputation_check_enabled: boolean
  block_tor_exit_nodes: boolean
  risk_step_up_threshold: number
  risk_block_threshold: number
  velocity_failures_per_ip_per_hour: number
  distinct_accounts_per_ip_per_hour: number
}

export interface ThreatDetectionSettingsPayload {
  brute_force_detection_enabled?: boolean
  impossible_travel_detection_enabled?: boolean
  new_device_notification_enabled?: boolean
  velocity_check_enabled?: boolean
  risk_based_step_up_enabled?: boolean
  compromised_credential_monitoring_enabled?: boolean
  ip_reputation_check_enabled?: boolean
  block_tor_exit_nodes?: boolean
  risk_step_up_threshold?: number
  risk_block_threshold?: number
  velocity_failures_per_ip_per_hour?: number
  distinct_accounts_per_ip_per_hour?: number
}

export interface ThreatDetectionSettingsResponse {
  success: boolean
  data: ThreatDetectionSettings
  message: string
}
