export interface AuditLogEntry {
  uuid: string
  action: string
  resource_type: string
  resource_id: string
  changes: Record<string, unknown> | string | null
  ip_address: string | null
  actor_user_id?: number | null
  actor_user_name?: string | null
  actor_client_id?: number | null
  actor_client_name?: string | null
  outcome: 'success' | 'failure' | 'partial'
  created_at: string
}

export interface AuditLogQueryParams {
  resource_type?: string
  actor_user_id?: string
  date_from?: string
  date_to?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface AuditLogListResponse {
  rows: AuditLogEntry[]
  total: number
  page: number
  limit: number
  total_pages: number
}
