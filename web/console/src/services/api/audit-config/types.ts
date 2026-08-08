export interface AuditConfig {
  enabled: boolean
  retention_days: number
  pii_masking: boolean
  log_level: string
  event_types: string[]
}

export interface AuditConfigResponse {
  success: boolean
  data: AuditConfig
  message: string
}

export type AuditConfigPayload = Partial<AuditConfig>
