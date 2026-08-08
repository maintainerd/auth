/**
 * IP Restriction Rules API Service
 */

import { get, post, put, deleteRequest, patch } from '@/services'
import type {
  IpRestrictionRule,
  IpRestrictionRulesResponse,
  IpRestrictionRulePayload,
  CreateIpRestrictionRuleRequest,
  UpdateIpRestrictionRuleRequest,
  UpdateIpRestrictionRuleStatusRequest,
  IpRestrictionRulesQueryParams,
} from './types'

const transformPayloadToRule = (payload: IpRestrictionRulePayload): IpRestrictionRule => ({
  ipRestrictionRuleId: payload.ip_restriction_rule_id,
  description: payload.description,
  type: payload.type,
  ipAddress: payload.ip_address,
  status: payload.status,
  createdAt: payload.created_at,
  updatedAt: payload.updated_at,
})

export const fetchIpRestrictionRules = async (
  params: IpRestrictionRulesQueryParams = {}
): Promise<{ rows: IpRestrictionRule[]; total: number; page: number; limit: number; totalPages: number }> => {
  const queryParams = new URLSearchParams()
  
  if (params.type) queryParams.append('type', params.type)
  if (params.status) queryParams.append('status', params.status)
  if (params.ip_address) queryParams.append('ip_address', params.ip_address)
  if (params.description) queryParams.append('description', params.description)
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.limit) queryParams.append('limit', params.limit.toString())
  if (params.sort_by) queryParams.append('sort_by', params.sort_by)
  if (params.sort_order) queryParams.append('sort_order', params.sort_order)

  const url = `/ip-restriction-rules${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const response = await get<IpRestrictionRulesResponse>(url)

  if (!response.success) {
    throw new Error(response.message || 'Failed to fetch IP restriction rules')
  }

  return {
    rows: response.data.rows.map(transformPayloadToRule),
    total: response.data.total,
    page: response.data.page,
    limit: response.data.limit,
    totalPages: response.data.total_pages,
  }
}

export const createIpRestrictionRule = async (
  data: CreateIpRestrictionRuleRequest
): Promise<IpRestrictionRule> => {
  const response = await post<{ success: boolean; data: IpRestrictionRulePayload; message: string }>(
    '/ip-restriction-rules',
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to create IP restriction rule')
  }

  return transformPayloadToRule(response.data)
}

export const updateIpRestrictionRule = async (
  id: string,
  data: UpdateIpRestrictionRuleRequest
): Promise<IpRestrictionRule> => {
  const response = await put<{ success: boolean; data: IpRestrictionRulePayload; message: string }>(
    `/ip-restriction-rules/${id}`,
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to update IP restriction rule')
  }

  return transformPayloadToRule(response.data)
}

export const deleteIpRestrictionRule = async (id: string): Promise<void> => {
  const response = await deleteRequest<{ success: boolean; message: string }>(
    `/ip-restriction-rules/${id}`
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to delete IP restriction rule')
  }
}

export const updateIpRestrictionRuleStatus = async (
  id: string,
  data: UpdateIpRestrictionRuleStatusRequest
): Promise<IpRestrictionRule> => {
  const response = await patch<{ success: boolean; data: IpRestrictionRulePayload; message: string }>(
    `/ip-restriction-rules/${id}/status`,
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to update IP restriction rule status')
  }

  return transformPayloadToRule(response.data)
}
