/**
 * Email Template API Service
 */

import { get, put, patch } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type {
  EmailTemplate,
  EmailTemplatePayload,
  EmailTemplateQueryParams,
  EmailTemplateListResponse,
  UpdateEmailTemplateRequest,
  UpdateEmailTemplateStatusRequest,
} from './types'

/**
 * Transform backend payload to frontend EmailTemplate
 */
function transformPayloadToEmailTemplate(payload: EmailTemplatePayload): EmailTemplate {
  return {
    emailTemplateId: payload.email_template_id,
    name: payload.name,
    subject: payload.subject,
    bodyHtml: payload.body_html,
    bodyPlain: payload.body_plain,
    parametersDoc: payload.parameters_doc,
    status: payload.status,
    isDefault: payload.is_default,
    isSystem: payload.is_system,
    createdAt: payload.created_at,
    updatedAt: payload.updated_at,
  }
}

/**
 * Fetch email templates with optional filters and pagination
 */
export async function fetchEmailTemplates(
  params?: EmailTemplateQueryParams
): Promise<EmailTemplateListResponse> {
  const queryParams = new URLSearchParams()

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  const endpoint = `${API_ENDPOINTS.EMAIL_TEMPLATE}${
    queryParams.toString() ? `?${queryParams.toString()}` : ''
  }`
  const response = await get<
    ApiResponse<{
      rows: EmailTemplatePayload[] | null
      total: number
      page: number
      limit: number
      total_pages: number
    }>
  >(endpoint)

  if (response.success && response.data) {
    return {
      rows: (response.data.rows ?? []).map(transformPayloadToEmailTemplate),
      total: response.data.total,
      page: response.data.page,
      limit: response.data.limit,
      total_pages: response.data.total_pages,
    }
  }

  throw new Error(response.message || 'Failed to fetch email templates')
}

/**
 * Fetch a single email template by ID
 */
export async function fetchEmailTemplateById(id: string): Promise<EmailTemplate> {
  const endpoint = `${API_ENDPOINTS.EMAIL_TEMPLATE}/${id}`
  const response = await get<ApiResponse<EmailTemplatePayload>>(endpoint)

  if (response.success && response.data) {
    return transformPayloadToEmailTemplate(response.data)
  }

  throw new Error(response.message || 'Failed to fetch email template')
}

/**
 * Update an existing email template
 */
export async function updateEmailTemplate(
  id: string,
  data: UpdateEmailTemplateRequest
): Promise<EmailTemplate> {
  const endpoint = `${API_ENDPOINTS.EMAIL_TEMPLATE}/${id}`
  const response = await put<ApiResponse<EmailTemplatePayload>>(endpoint, data)

  if (response.success && response.data) {
    return transformPayloadToEmailTemplate(response.data)
  }

  throw new Error(response.message || 'Failed to update email template')
}

/**
 * Update email template status
 */
export async function updateEmailTemplateStatus(
  id: string,
  data: UpdateEmailTemplateStatusRequest
): Promise<EmailTemplate> {
  const endpoint = `${API_ENDPOINTS.EMAIL_TEMPLATE}/${id}/status`
  const response = await patch<ApiResponse<EmailTemplatePayload>>(endpoint, data)

  if (response.success && response.data) {
    return transformPayloadToEmailTemplate(response.data)
  }

  throw new Error(response.message || 'Failed to update email template status')
}
