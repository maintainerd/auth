export interface AuthEvent {
  auth_event_id: string
  tenant_id: number
  actor_user_id?: number | null
  target_user_id?: number | null
  ip_address: string
  user_agent?: string | null
  category: string
  event_type: string
  severity: string
  result: string
  description?: string | null
  error_reason?: string | null
  trace_id?: string | null
  metadata?: Record<string, unknown> | null
  created_at: string
}

export interface AuthEventQueryParams {
  category?: string
  event_type?: string
  severity?: string
  result?: string
  date_from?: string
  date_to?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  after_id?: number
}

export interface AuthEventListResponse {
  rows: AuthEvent[]
  total: number
  page: number
  limit: number
  total_pages: number
}
