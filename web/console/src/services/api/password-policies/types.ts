export interface PasswordPolicies {
  min_length: number
  max_length: number
  require_uppercase: boolean
  require_lowercase: boolean
  require_number: boolean
  require_symbol: boolean
  reject_common_passwords: boolean
  check_hibp: boolean
  password_history_count: number
  max_age_days: number
  temporary_password_validity_hours: number
  hash_algorithm: string
  min_strength_score: number
}

export interface PasswordPoliciesPayload {
  min_length?: number
  max_length?: number
  require_uppercase?: boolean
  require_lowercase?: boolean
  require_number?: boolean
  require_symbol?: boolean
  reject_common_passwords?: boolean
  check_hibp?: boolean
  password_history_count?: number
  max_age_days?: number
  temporary_password_validity_hours?: number
  hash_algorithm?: string
  min_strength_score?: number
}

export interface PasswordPoliciesResponse {
  success: boolean
  data: PasswordPolicies
  message: string
}
