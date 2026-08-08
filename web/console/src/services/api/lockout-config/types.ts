export interface LockoutConfig {
  enabled: boolean
  max_failed_attempts: number
  lockout_duration_minutes: number
  progressive_lockout: boolean
  auto_unlock: boolean
  reset_count_on_success: boolean
  observation_window_minutes: number
  max_lockout_duration_minutes: number
  progression_reset_hours: number
  notify_user_on_lockout: boolean
}

export interface LockoutConfigPayload {
  enabled?: boolean
  max_failed_attempts?: number
  lockout_duration_minutes?: number
  progressive_lockout?: boolean
  auto_unlock?: boolean
  reset_count_on_success?: boolean
  observation_window_minutes?: number
  max_lockout_duration_minutes?: number
  progression_reset_hours?: number
  notify_user_on_lockout?: boolean
}

export interface LockoutConfigResponse {
  success: boolean
  data: LockoutConfig
  message: string
}
