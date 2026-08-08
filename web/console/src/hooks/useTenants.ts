import { useQuery, useMutation, useQueryClient, keepPreviousData } from "@tanstack/react-query"
import { get } from "@/services/api/client"
import { API_ENDPOINTS } from "@/services/api/config"
import {
  fetchTenantList,
  fetchTenantById,
  createTenant,
  updateTenant,
  updateTenantStatus,
  deleteTenant,
} from "@/services/api/tenants"
import type {
  TenantEntity,
  TenantListParams,
  CreateTenantRequest,
  UpdateTenantRequest,
} from "@/services/api/tenants/types"
import type { ApiResponse, PaginatedResponse } from "@/services/api/types"

export interface TenantListResult {
  rows: TenantEntity[]
  total: number
}

async function fetchTenantsForListing(
  params: TenantListParams = {},
): Promise<TenantListResult> {
  const qp = new URLSearchParams()
  if (params.name) qp.set("name", params.name)
  if (params.display_name) qp.set("display_name", params.display_name)
  if (params.description) qp.set("description", params.description)
  if (params.status) qp.set("status", params.status)
  if (params.is_system !== undefined) qp.set("is_system", String(params.is_system))
  if (params.page) qp.set("page", String(params.page))
  if (params.limit) qp.set("limit", String(params.limit))
  if (params.sort_by) qp.set("sort_by", params.sort_by)
  if (params.sort_order) qp.set("sort_order", params.sort_order)

  const url = `${API_ENDPOINTS.TENANT}s${qp.toString() ? `?${qp.toString()}` : ""}`
  const response = await get<ApiResponse<PaginatedResponse<TenantEntity>>>(url)

  if (!response.success || !response.data) {
    throw new Error(response.message || "Failed to fetch tenants")
  }

  return { rows: response.data.rows, total: response.data.total }
}

export const tenantKeys = {
  all: ["tenants"] as const,
  lists: () => [...tenantKeys.all, "list"] as const,
  list: (params?: TenantListParams) => [...tenantKeys.lists(), params] as const,
  details: () => [...tenantKeys.all, "detail"] as const,
  detail: (id: string) => [...tenantKeys.details(), id] as const,
}

export function useTenants(params?: TenantListParams) {
  return useQuery({
    queryKey: tenantKeys.list(params),
    queryFn: () => fetchTenantsForListing(params),
    placeholderData: keepPreviousData,
  })
}

/** Raw paginated list (full ApiResponse) — for the tenant switcher's `data.data.rows`. */
export function useTenantsList(params?: TenantListParams) {
  return useQuery({
    queryKey: [...tenantKeys.list(params), "raw"] as const,
    queryFn: () => fetchTenantList(params),
    placeholderData: keepPreviousData,
  })
}

/** Single tenant by id — disabled until an id is available. */
export function useTenantById(tenantId?: string) {
  return useQuery({
    queryKey: tenantKeys.detail(tenantId ?? ""),
    queryFn: () => fetchTenantById(tenantId as string),
    enabled: !!tenantId,
  })
}

export function useCreateTenant() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateTenantRequest) => createTenant(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tenantKeys.lists() })
    },
  })
}

export function useUpdateTenant() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ tenantId, data }: { tenantId: string; data: UpdateTenantRequest }) =>
      updateTenant(tenantId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tenantKeys.lists() })
    },
  })
}

export function useUpdateTenantStatus() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ tenantId, status }: { tenantId: string; status: string }) =>
      updateTenantStatus(tenantId, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tenantKeys.lists() })
    },
  })
}

export function useDeleteTenant() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (tenantId: string) => deleteTenant(tenantId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tenantKeys.lists() })
    },
  })
}
