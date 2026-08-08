/**
 * SMS Template API Types
 */

import type { Status } from '@/types/status'

/**
 * SMS Template status type
 */
export type SmsTemplateStatus = Extract<Status, 'active' | 'inactive'>

/**
 * SMS Template type (camelCase for frontend)
 */
export type SmsTemplate = {
  smsTemplateId: string
  name: string
  description: string
  message: string
  parametersDoc: string | null
  status: SmsTemplateStatus
  isDefault: boolean
  isSystem: boolean
  createdAt: string
  updatedAt: string
}

/**
 * SMS Template Payload type (snake_case from backend)
 */
export type SmsTemplatePayload = {
  sms_template_id: string
  name: string
  description: string
  message: string
  parameters_doc: string | null
  status: SmsTemplateStatus
  is_default: boolean
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * SMS Template Query Params
 */
export interface SmsTemplateQueryParams {
  name?: string
  status?: SmsTemplateStatus
  is_default?: boolean
  is_system?: boolean
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * SMS Template List Response
 */
export interface SmsTemplateListResponse {
  rows: SmsTemplate[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Update SMS Template Request
 */
export interface UpdateSmsTemplateRequest {
  description: string
  message: string
  status: SmsTemplateStatus
}

/**
 * Update SMS Template Status Request
 */
export interface UpdateSmsTemplateStatusRequest {
  status: SmsTemplateStatus
}
