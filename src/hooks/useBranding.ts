/**
 * Branding hooks (TanStack Query)
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchBrandings,
  createBranding,
  updateBranding,
  activateBranding,
  restoreBranding,
  deleteBranding,
} from '@/services/api/branding'
import type { BrandingRequest } from '@/services/api/branding/types'
import type { Branding } from '@/services/api/branding/types'
import type { BrandingPublic } from '@/services/api/tenants/types'
import { useAppDispatch } from '@/store/hooks'
import { setTenantBranding } from '@/store/tenant/slice'

export const brandingKeys = {
  all: ['branding'] as const,
  list: () => [...brandingKeys.all, 'list'] as const,
}

export function useBrandings() {
  return useQuery({
    queryKey: brandingKeys.list(),
    queryFn: fetchBrandings,
  })
}

// Single branding, derived from the list (there is no GET-by-id endpoint).
export function useBranding(brandingId: string | undefined) {
  const query = useBrandings()
  const data = brandingId ? query.data?.find((b) => b.branding_id === brandingId) : undefined
  return { ...query, data }
}

function toPublicBranding(branding: Branding): BrandingPublic {
  return {
    layout: branding.layout,
    company_name: branding.company_name,
    logo_label: branding.logo_label,
    show_logo_label: branding.show_logo_label,
    logo_url: branding.logo_url,
    favicon_url: branding.favicon_url,
    support_url: branding.support_url,
    privacy_policy_url: branding.privacy_policy_url,
    terms_of_service_url: branding.terms_of_service_url,
    metadata: branding.metadata ?? {},
  }
}

export function useCreateBranding() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: BrandingRequest) => createBranding(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: brandingKeys.list() })
    },
  })
}

export function useUpdateBranding() {
  const queryClient = useQueryClient()
  const dispatch = useAppDispatch()
  return useMutation({
    mutationFn: ({ brandingId, data }: { brandingId: string; data: BrandingRequest }) =>
      updateBranding(brandingId, data),
    onSuccess: (branding) => {
      queryClient.invalidateQueries({ queryKey: brandingKeys.list() })
      if (branding.is_active) {
        dispatch(setTenantBranding(toPublicBranding(branding)))
      }
    },
  })
}

export function useActivateBranding() {
  const queryClient = useQueryClient()
  const dispatch = useAppDispatch()
  return useMutation({
    mutationFn: (brandingId: string) => activateBranding(brandingId),
    onSuccess: (branding) => {
      queryClient.invalidateQueries({ queryKey: brandingKeys.list() })
      dispatch(setTenantBranding(toPublicBranding(branding)))
    },
  })
}

export function useRestoreBranding() {
  const queryClient = useQueryClient()
  const dispatch = useAppDispatch()
  return useMutation({
    mutationFn: (brandingId: string) => restoreBranding(brandingId),
    onSuccess: (branding) => {
      queryClient.invalidateQueries({ queryKey: brandingKeys.list() })
      if (branding.is_active) {
        dispatch(setTenantBranding(toPublicBranding(branding)))
      }
    },
  })
}

export function useDeleteBranding() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (brandingId: string) => deleteBranding(brandingId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: brandingKeys.list() })
    },
  })
}
