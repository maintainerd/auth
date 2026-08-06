import { get, post, patch, deleteRequest } from '../client'
import { API_ENDPOINTS } from '../config'

export interface TenantMemberUser {
  user_id: string
  username: string
  fullname: string
  email: string
  phone: string
  is_email_verified: boolean
  is_phone_verified: boolean
  // No is_profile_completed / is_account_completed: member rows serialize
  // MemberUserResponseDTO, which emits neither (maintainerd-auth
  // internal/tenant/types.go:188-200). They were always undefined at runtime.
  status: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface TenantMember {
  tenant_member_id: string
  role: 'owner' | 'admin' | 'member'
  user: TenantMemberUser
  created_at: string
  updated_at: string
}

export interface TenantMembersListParams {
  role?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface TenantMembersListResponse {
  success: boolean
  data: {
    rows: TenantMember[]
    total: number
    page: number
    limit: number
    // snake_case, not totalPages: the paginated envelope is emitted by
    // PaginatedResponseDTO with a `total_pages` json tag (maintainerd-auth
    // internal/platform/pagination/pagination.go:53). Reading `totalPages` gave
    // undefined, so page counts had to be guessed from the current page's rows.
    total_pages: number
  }
  message?: string
}

export async function fetchTenantMembers(tenantId: string, params: TenantMembersListParams = {}): Promise<TenantMembersListResponse> {
  const qp: Record<string, string> = {
    role: params.role ?? '',
    page: params.page !== undefined ? String(params.page) : '1',
    limit: params.limit !== undefined ? String(params.limit) : '10',
    sort_by: params.sort_by ?? 'created_at',
    sort_order: params.sort_order ?? 'desc',
  }
  const queryParams = new URLSearchParams(qp)
  const url = `${API_ENDPOINTS.TENANT}s/${tenantId}/members?${queryParams.toString()}`
  return get<TenantMembersListResponse>(url)
}

export async function addTenantMember(tenantId: string, user_id: string, role: 'owner' | 'admin' | 'member') {
  return post(`${API_ENDPOINTS.TENANT}s/${tenantId}/members`, { user_id, role })
}

export async function updateTenantMemberRole(tenantId: string, tenant_member_id: string, role: 'owner' | 'admin' | 'member') {
  return patch(`${API_ENDPOINTS.TENANT}s/${tenantId}/members/${tenant_member_id}/role`, { role })
}

export async function transferTenantOwnership(tenantId: string, new_owner_member_id: string) {
  return patch(`${API_ENDPOINTS.TENANT}s/${tenantId}/members/${new_owner_member_id}/role`, { role: 'owner' })
}

export async function deleteTenantMember(tenantId: string, tenant_member_id: string) {
  return deleteRequest(`${API_ENDPOINTS.TENANT}s/${tenantId}/members/${tenant_member_id}`)
}
