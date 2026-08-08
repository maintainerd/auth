/**
 * Workload Identity Federation API Types
 */

export interface WorkloadIdentityFederation {
  workload_identity_federation_uuid: string
  client_uuid: string
  name: string
  description: string
  issuer_url: string
  audience: string
  subject_claim: string
  subject_pattern: string
  allowed_scopes: string[]
  attribute_mapping: Record<string, string>
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface WorkloadIdentityQueryParams {
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface WorkloadIdentityListResponse {
  rows: WorkloadIdentityFederation[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Create request — `client_uuid`, `name`, `issuer_url`, `audience`, and
 * `subject_pattern` are required by the backend.
 */
export interface CreateWorkloadIdentityRequest {
  client_uuid: string
  name: string
  description?: string
  issuer_url: string
  audience: string
  subject_claim?: string
  subject_pattern: string
  allowed_scopes?: string[]
  attribute_mapping?: Record<string, string>
  is_active?: boolean
}

/**
 * Update request — `client_uuid` cannot change after creation; all other
 * required fields stay required.
 */
export interface UpdateWorkloadIdentityRequest {
  name: string
  description?: string
  issuer_url: string
  audience: string
  subject_claim?: string
  subject_pattern: string
  allowed_scopes?: string[]
  attribute_mapping?: Record<string, string>
  is_active?: boolean
}
