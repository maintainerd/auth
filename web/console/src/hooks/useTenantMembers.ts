import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchTenantMembers,
  addTenantMember,
  updateTenantMemberRole,
  deleteTenantMember,
  transferTenantOwnership,
} from '@/services/api/tenants/members'
import type { TenantMembersListParams } from '@/services/api/tenants/members'

export function useTenantMembers(tenantId: string, params: TenantMembersListParams = {}) {
  return useQuery({
    queryKey: ['tenant-members', tenantId, params],
    queryFn: () => fetchTenantMembers(tenantId, params),
    enabled: !!tenantId,
  })
}

export function useAddTenantMember(tenantId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ user_id, role }: { user_id: string; role: 'owner' | 'admin' | 'member' }) =>
      addTenantMember(tenantId, user_id, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-members', tenantId] })
    },
  })
}

export function useUpdateTenantMemberRole(tenantId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ tenant_member_id, role }: { tenant_member_id: string; role: 'owner' | 'admin' | 'member' }) =>
      updateTenantMemberRole(tenantId, tenant_member_id, role),
    onSuccess: () => {
      // Only invalidate tenant-members query, not tenant query
      queryClient.invalidateQueries({ queryKey: ['tenant-members', tenantId] })
    },
  })
}

export function useDeleteTenantMember(tenantId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (tenant_member_id: string) => deleteTenantMember(tenantId, tenant_member_id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-members', tenantId] })
    },
  })
}

export function useTransferTenantOwnership(tenantId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (new_owner_member_id: string) =>
      transferTenantOwnership(tenantId, new_owner_member_id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-members', tenantId] })
    },
  })
}
