/**
 * Email Template API Types
 */

import type { Status } from '@/types/status'

/**
 * Email Template status type
 */
export type EmailTemplateStatus = Extract<Status, 'active' | 'inactive'>

/**
 * Email Template type (camelCase for frontend)
 */
export type EmailTemplate = {
  emailTemplateId: string
  name: string
  subject: string
  bodyHtml: string
  bodyPlain: string
  parametersDoc: string | null
  status: EmailTemplateStatus
  isDefault: boolean
  isSystem: boolean
  createdAt: string
  updatedAt: string
}

/**
 * Email Template Payload type (snake_case from backend)
 */
export type EmailTemplatePayload = {
  email_template_id: string
  name: string
  subject: string
  body_html: string
  body_plain: string
  parameters_doc: string | null
  status: EmailTemplateStatus
  is_default: boolean
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * Email Template Query Params
 */
export interface EmailTemplateQueryParams {
  name?: string
  status?: EmailTemplateStatus
  is_default?: boolean
  is_system?: boolean
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Email Template List Response
 */
export interface EmailTemplateListResponse {
  rows: EmailTemplate[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Update Email Template Request
 */
export interface UpdateEmailTemplateRequest {
  subject: string
  body_html: string
  body_plain: string
  status: EmailTemplateStatus
}

/**
 * Update Email Template Status Request
 */
export interface UpdateEmailTemplateStatusRequest {
  status: EmailTemplateStatus
}
