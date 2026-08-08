export interface EventRoute {
  uuid: string
  event_type_uuid: string
  event_type_key: string
  channel: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateEventRouteRequest {
  event_type_uuid: string
}

export interface UpdateEventRouteRequest {
  enabled: boolean
}
