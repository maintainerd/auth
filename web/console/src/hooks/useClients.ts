/**
 * Clients Hook
 * Custom hook for fetching clients using TanStack Query
 */

import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import {
  fetchClients,
  fetchClientById,
  createClient,
  updateClient,
  deleteClient,
  addClientIdentityProvider,
  removeClientIdentityProvider,
  updateClientIdentityProvider,
  updateClientStatus,
  rotateClientSecret,
  fetchClientConfig,
  fetchClientUris,
  createClientUri,
  updateClientUri,
  deleteClientUri,
  fetchClientApis,
  addClientApis,
  removeClientApi,
  addClientApiPermissions,
  removeClientApiPermission,
  fetchClientRoles,
  addClientRole,
  removeClientRole,
} from '@/services/api/clients'
import type {
  ClientQueryParams,
  CreateClientRequest,
  UpdateClientRequest,
  UpdateClientStatusRequest,
  RotateClientSecretRequest,
  CreateClientUriRequest,
  UpdateClientUriRequest,
  AddClientApisRequest,
  AddClientApiPermissionsRequest,
  AddClientIdentityProviderRequest
} from '@/services/api/clients/types'

/**
 * Query key factory for clients
 */
export const clientKeys = {
  all: ['clients'] as const,
  lists: () => [...clientKeys.all, 'list'] as const,
  list: (params?: ClientQueryParams) => [...clientKeys.lists(), params] as const,
  details: () => [...clientKeys.all, 'detail'] as const,
  detail: (id: string) => [...clientKeys.details(), id] as const,
  config: (id: string) => [...clientKeys.all, 'config', id] as const,
  uris: (id: string) => [...clientKeys.all, 'uris', id] as const,
  apis: (id: string) => [...clientKeys.all, 'apis', id] as const,
  roles: (id: string) => [...clientKeys.all, 'roles', id] as const,
}

/**
 * Hook to fetch clients with optional filters and pagination
 */
export function useClients(params?: ClientQueryParams) {
  return useQuery({
    queryKey: clientKeys.list(params),
    queryFn: () => fetchClients(params),
    placeholderData: keepPreviousData,
  })
}

/**
 * Hook to fetch clients for the standard listing page.
 * The shared listing filter uses human labels for system/regular, while the
 * API expects an `is_system` boolean.
 */
export function useClientsList(params: Record<string, unknown>) {
  const { is_system, ...rest } = params
  const queryParams: ClientQueryParams = {
    ...rest as ClientQueryParams,
  }

  if (typeof is_system === 'string') {
    if (is_system === 'system') queryParams.is_system = true
    else if (is_system === 'regular') queryParams.is_system = false
  }

  return useClients(queryParams)
}

/**
 * Hook to fetch a single client by ID
 */
export function useClient(clientId: string) {
  return useQuery({
    queryKey: clientKeys.detail(clientId),
    queryFn: () => fetchClientById(clientId),
    enabled: !!clientId,
  })
}

/**
 * Hook to create a new client
 */
export function useCreateClient() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateClientRequest) => createClient(data),
    onSuccess: () => {
      // Invalidate clients list to refetch
      queryClient.invalidateQueries({ queryKey: clientKeys.lists() })
    },
  })
}

/**
 * Hook to update an existing client
 */
export function useUpdateClient() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, data }: { clientId: string; data: UpdateClientRequest }) =>
      updateClient(clientId, data),
    onSuccess: (_, variables) => {
      // Invalidate both the specific client and the clients list
      queryClient.invalidateQueries({ queryKey: clientKeys.detail(variables.clientId) })
      queryClient.invalidateQueries({ queryKey: clientKeys.lists() })
    },
  })
}

/**
 * Hook to delete a client
 */
export function useDeleteClient() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (clientId: string) => deleteClient(clientId),
    onSuccess: () => {
      // Invalidate clients list to refetch
      queryClient.invalidateQueries({ queryKey: clientKeys.lists() })
    },
  })
}

/**
 * Hook to connect an identity provider to an existing client.
 */
export function useAddClientIdentityProvider() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, data }: { clientId: string; data: AddClientIdentityProviderRequest }) =>
      addClientIdentityProvider(clientId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: clientKeys.detail(variables.clientId) })
      queryClient.invalidateQueries({ queryKey: clientKeys.lists() })
    },
  })
}

/**
 * Hook to disconnect an identity provider from a client without deleting the client.
 */
export function useRemoveClientIdentityProvider() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, connectionId }: { clientId: string; connectionId: string }) =>
      removeClientIdentityProvider(clientId, connectionId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: clientKeys.detail(variables.clientId) })
      queryClient.invalidateQueries({ queryKey: clientKeys.lists() })
    },
  })
}

export function useUpdateClientIdentityProvider() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ clientId, connectionId, data }: {
      clientId: string
      connectionId: string
      data: { is_default?: boolean; enabled?: boolean; display_order?: number }
    }) => updateClientIdentityProvider(clientId, connectionId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: clientKeys.detail(variables.clientId) })
    },
  })
}

/**
 * Hook to update client status
 */
export function useUpdateClientStatus() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, data }: { clientId: string; data: UpdateClientStatusRequest }) =>
      updateClientStatus(clientId, data),
    onSuccess: (_, variables) => {
      // Invalidate both the specific client and the clients list
      queryClient.invalidateQueries({ queryKey: clientKeys.detail(variables.clientId) })
      queryClient.invalidateQueries({ queryKey: clientKeys.lists() })
    },
  })
}

/**
 * Hook to rotate a client secret. The returned secret is shown only once.
 */
export function useRotateClientSecret() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, data }: { clientId: string; data?: RotateClientSecretRequest }) =>
      rotateClientSecret(clientId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: clientKeys.detail(variables.clientId) })
    },
  })
}

/**
 * Hook to fetch client config
 */
export function useClientConfig(clientId: string) {
  return useQuery({
    queryKey: clientKeys.config(clientId),
    queryFn: () => fetchClientConfig(clientId),
    enabled: !!clientId,
  })
}

/**
 * Hook to fetch client URIs
 */
export function useClientUris(clientId: string) {
  return useQuery({
    queryKey: clientKeys.uris(clientId),
    queryFn: () => fetchClientUris(clientId),
    enabled: !!clientId,
  })
}

/**
 * Hook to create a new client URI
 */
export function useCreateClientUri() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, data }: { clientId: string; data: CreateClientUriRequest }) =>
      createClientUri(clientId, data),
    onSuccess: (_, variables) => {
      // Invalidate client URIs to refetch
      queryClient.invalidateQueries({ queryKey: clientKeys.uris(variables.clientId) })
    },
  })
}

/**
 * Hook to update an existing client URI
 */
export function useUpdateClientUri() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, clientUriId, data }: { clientId: string; clientUriId: string; data: UpdateClientUriRequest }) =>
      updateClientUri(clientId, clientUriId, data),
    onSuccess: (_, variables) => {
      // Invalidate client URIs to refetch
      queryClient.invalidateQueries({ queryKey: clientKeys.uris(variables.clientId) })
    },
  })
}

/**
 * Hook to delete a client URI
 */
export function useDeleteClientUri() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, clientUriId }: { clientId: string; clientUriId: string }) =>
      deleteClientUri(clientId, clientUriId),
    onSuccess: (_, variables) => {
      // Invalidate client URIs to refetch
      queryClient.invalidateQueries({ queryKey: clientKeys.uris(variables.clientId) })
    },
  })
}

/**
 * Hook to fetch client APIs
 */
export function useClientApis(clientId: string) {
  return useQuery({
    queryKey: clientKeys.apis(clientId),
    queryFn: () => fetchClientApis(clientId),
    enabled: !!clientId,
  })
}

/**
 * Hook to add APIs to a client
 */
export function useAddClientApis() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, data }: { clientId: string; data: AddClientApisRequest }) =>
      addClientApis(clientId, data),
    onSuccess: (_, variables) => {
      // Invalidate client APIs to refetch
      queryClient.invalidateQueries({ queryKey: clientKeys.apis(variables.clientId) })
    },
  })
}

/**
 * Hook to remove an API from a client
 */
export function useRemoveClientApi() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, apiId }: { clientId: string; apiId: string }) =>
      removeClientApi(clientId, apiId),
    onSuccess: (_, variables) => {
      // Invalidate client APIs to refetch
      queryClient.invalidateQueries({ queryKey: clientKeys.apis(variables.clientId) })
    },
  })
}

/**
 * Hook to add permissions to a client API
 */
export function useAddClientApiPermissions() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, apiId, data }: { clientId: string; apiId: string; data: AddClientApiPermissionsRequest }) =>
      addClientApiPermissions(clientId, apiId, data),
    onSuccess: (_, variables) => {
      // Invalidate client APIs to refetch
      queryClient.invalidateQueries({ queryKey: clientKeys.apis(variables.clientId) })
    },
  })
}

/**
 * Hook to remove a permission from a client API
 */
export function useRemoveClientApiPermission() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ clientId, apiId, permissionId }: { clientId: string; apiId: string; permissionId: string }) =>
      removeClientApiPermission(clientId, apiId, permissionId),
    onSuccess: (_, variables) => {
      // Invalidate client APIs to refetch
      queryClient.invalidateQueries({ queryKey: clientKeys.apis(variables.clientId) })
    },
  })
}

/**
 * Hook to fetch roles assigned to a client
 */
export function useClientRoles(clientId: string) {
  return useQuery({
    queryKey: clientKeys.roles(clientId),
    queryFn: () => fetchClientRoles(clientId),
    enabled: !!clientId,
  })
}

/**
 * Hook to add a role to a client
 */
export function useAddClientRole(clientId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (roleUuid: string) => addClientRole(clientId, roleUuid),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: clientKeys.roles(clientId) })
    },
  })
}

/**
 * Hook to remove a role from a client
 */
export function useRemoveClientRole(clientId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (roleUuid: string) => removeClientRole(clientId, roleUuid),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: clientKeys.roles(clientId) })
    },
  })
}
