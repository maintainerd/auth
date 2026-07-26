/**
 * SMS Template API Service
 */

import { get, put, patch } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type {
  SmsTemplate,
  SmsTemplatePayload,
  SmsTemplateQueryParams,
  SmsTemplateListResponse,
  UpdateSmsTemplateRequest,
  UpdateSmsTemplateStatusRequest,
} from './types'

/**
 * Transform backend payload to frontend SmsTemplate
 */
function transformPayloadToSmsTemplate(payload: SmsTemplatePayload): SmsTemplate {
  return {
    smsTemplateId: payload.sms_template_id,
    name: payload.name,
    description: payload.description,
    message: payload.message,
    parametersDoc: payload.parameters_doc,
    status: payload.status,
    isDefault: payload.is_default,
    isSystem: payload.is_system,
    createdAt: payload.created_at,
    updatedAt: payload.updated_at,
  }
}

/**
 * Fetch SMS templates with optional filters and pagination
 */
export async function fetchSmsTemplates(
  params?: SmsTemplateQueryParams
): Promise<SmsTemplateListResponse> {
  const queryParams = new URLSearchParams()

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  const endpoint = `${API_ENDPOINTS.SMS_TEMPLATE}${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const response = await get<ApiResponse<{
    rows: SmsTemplatePayload[] | null
    total: number
    page: number
    limit: number
    total_pages: number
  }>>(endpoint)

  if (response.success && response.data) {
    return {
      rows: (response.data.rows ?? []).map(transformPayloadToSmsTemplate),
      total: response.data.total,
      page: response.data.page,
      limit: response.data.limit,
      total_pages: response.data.total_pages,
    }
  }

  throw new Error(response.message || 'Failed to fetch SMS templates')
}

/**
 * Fetch a single SMS template by ID
 */
export async function fetchSmsTemplateById(id: string): Promise<SmsTemplate> {
  const endpoint = `${API_ENDPOINTS.SMS_TEMPLATE}/${id}`
  const response = await get<ApiResponse<SmsTemplatePayload>>(endpoint)

  if (response.success && response.data) {
    return transformPayloadToSmsTemplate(response.data)
  }

  throw new Error(response.message || 'Failed to fetch SMS template')
}

/**
 * Update an existing SMS template
 */
export async function updateSmsTemplate(
  id: string,
  data: UpdateSmsTemplateRequest
): Promise<SmsTemplate> {
  const endpoint = `${API_ENDPOINTS.SMS_TEMPLATE}/${id}`
  const response = await put<ApiResponse<SmsTemplatePayload>>(endpoint, data)

  if (response.success && response.data) {
    return transformPayloadToSmsTemplate(response.data)
  }

  throw new Error(response.message || 'Failed to update SMS template')
}

/**
 * Update SMS template status
 */
export async function updateSmsTemplateStatus(
  id: string,
  data: UpdateSmsTemplateStatusRequest
): Promise<SmsTemplate> {
  const endpoint = `${API_ENDPOINTS.SMS_TEMPLATE}/${id}/status`
  const response = await patch<ApiResponse<SmsTemplatePayload>>(endpoint, data)

  if (response.success && response.data) {
    return transformPayloadToSmsTemplate(response.data)
  }

  throw new Error(response.message || 'Failed to update SMS template status')
}
