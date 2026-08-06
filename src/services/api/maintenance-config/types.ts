export interface MaintenanceConfig {
  enabled: boolean
  message: string
  // RFC3339 on the wire, in both directions — the API parses these with Go's
  // time.RFC3339 and rejects "" as well as null-less empties, so a raw
  // datetime-local value ("2026-08-05T14:30") must be run through
  // toRfc3339 in @/lib/datetime before it lands in this shape.
  scheduled_start: string | null
  scheduled_end: string | null
}

export interface MaintenanceConfigResponse {
  success: boolean
  data: MaintenanceConfig
  message: string
}

export type MaintenanceConfigPayload = Partial<MaintenanceConfig>
