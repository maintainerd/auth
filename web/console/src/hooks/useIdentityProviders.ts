import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { 
  fetchIdentityProviders, 
  fetchIdentityProviderById, 
  createIdentityProvider, 
  updateIdentityProvider, 
  deleteIdentityProvider,
  updateIdentityProviderStatus,
  fetchSamlServiceProviderMetadata
} from '@/services/api/identity-providers'
import type {
  IdentityProviderQueryParams,
  CreateIdentityProviderRequest,
  UpdateIdentityProviderRequest,
  UpdateIdentityProviderStatusRequest
} from '@/services/api/identity-providers/types'

/**
 * Query key factory for identity providers
 */
export const identityProviderKeys = {
  all: ['identityProviders'] as const,
  lists: () => [...identityProviderKeys.all, 'list'] as const,
  list: (params?: IdentityProviderQueryParams) => [...identityProviderKeys.lists(), params] as const,
  details: () => [...identityProviderKeys.all, 'detail'] as const,
  detail: (id: string) => [...identityProviderKeys.details(), id] as const,
  samlServiceProvider: (identifier: string) =>
    [...identityProviderKeys.all, 'saml-sp-metadata', identifier] as const,
}

/**
 * Hook to fetch identity providers for the listing page.
 * Social connectors are identity providers too, so the listing covers every
 * provider_type; it just maps the is_system filter from string to boolean.
 */
export function useIdentityProvidersList(params: Record<string, unknown>) {
  const { is_system, ...rest } = params
  const queryParams: IdentityProviderQueryParams = {
    ...rest as IdentityProviderQueryParams,
  }

  if (typeof is_system === 'string') {
    if (is_system === 'system') queryParams.is_system = true
    else if (is_system === 'regular') queryParams.is_system = false
  }

  return useIdentityProviders(queryParams)
}

/**
 * Hook to fetch identity providers with optional filters and pagination
 */
export function useIdentityProviders(params?: IdentityProviderQueryParams) {
  return useQuery({
    queryKey: identityProviderKeys.list(params),
    queryFn: () => fetchIdentityProviders(params),
    placeholderData: keepPreviousData,
  })
}

/**
 * Hook to fetch a single identity provider by ID
 */
export function useIdentityProvider(identityProviderId: string) {
  return useQuery({
    queryKey: identityProviderKeys.detail(identityProviderId),
    queryFn: () => fetchIdentityProviderById(identityProviderId),
    enabled: !!identityProviderId,
  })
}

/**
 * Hook to fetch the SAML service-provider metadata published for a provider.
 *
 * Only meaningful for SAML providers, so callers pass `enabled` rather than the
 * hook guessing: a non-SAML identifier would 404 on every detail page view.
 * The document changes only when the backend's public hostname or the provider
 * identifier changes, so it is cached for the session and not retried — a
 * missing document is a configuration fact to surface, not a transient error.
 */
export function useSamlServiceProviderMetadata(identifier: string, enabled: boolean) {
  return useQuery({
    queryKey: identityProviderKeys.samlServiceProvider(identifier),
    queryFn: () => fetchSamlServiceProviderMetadata(identifier),
    enabled: enabled && !!identifier,
    retry: false,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook to create a new identity provider
 */
export function useCreateIdentityProvider() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateIdentityProviderRequest) => createIdentityProvider(data),
    onSuccess: () => {
      // Invalidate identity providers list to refetch
      queryClient.invalidateQueries({ queryKey: identityProviderKeys.lists() })
    },
  })
}

/**
 * Hook to update an existing identity provider
 */
export function useUpdateIdentityProvider() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ identityProviderId, data }: { identityProviderId: string; data: UpdateIdentityProviderRequest }) =>
      updateIdentityProvider(identityProviderId, data),
    onSuccess: (_, variables) => {
      // Invalidate both the specific identity provider and the identity providers list
      queryClient.invalidateQueries({ queryKey: identityProviderKeys.detail(variables.identityProviderId) })
      queryClient.invalidateQueries({ queryKey: identityProviderKeys.lists() })
    },
  })
}

/**
 * Hook to delete an identity provider
 */
export function useDeleteIdentityProvider() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (identityProviderId: string) => deleteIdentityProvider(identityProviderId),
    onSuccess: () => {
      // Invalidate identity providers list to refetch
      queryClient.invalidateQueries({ queryKey: identityProviderKeys.lists() })
    },
  })
}

/**
 * Hook to update identity provider status
 */
export function useUpdateIdentityProviderStatus() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ identityProviderId, data }: { identityProviderId: string; data: UpdateIdentityProviderStatusRequest }) =>
      updateIdentityProviderStatus(identityProviderId, data),
    onSuccess: (_, variables) => {
      // Invalidate both the specific identity provider and the identity providers list
      queryClient.invalidateQueries({ queryKey: identityProviderKeys.detail(variables.identityProviderId) })
      queryClient.invalidateQueries({ queryKey: identityProviderKeys.lists() })
    },
  })
}

