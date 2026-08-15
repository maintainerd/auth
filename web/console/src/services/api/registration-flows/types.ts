/**
 * Registration Flow API Types
 *
 * These types mirror the backend DTOs in internal/idp/types.go
 * (RegistrationFlowResponseDTO / RegistrationFlowDetailResponseDTO and friends).
 * Two shapes matter and must not be conflated:
 *
 *   - LIST rows are lean: no `required_fields`, no resolved `client`.
 *   - DETAIL adds `required_fields` and the nested `client` summary.
 *
 * Typing them separately is what keeps a component from reading a field the
 * listing endpoint never sends.
 */

import type { Status } from '@/types/status'

/**
 * Registration flow status.
 *
 * Only 'active' and 'inactive' exist — the backend validation
 * (validation_registration_flow.go) rejects anything else with a 400 on create,
 * update, status-patch, and the list filter. There is no 'draft'.
 */
export type RegistrationFlowStatus = Extract<Status, 'active' | 'inactive'>

/**
 * The additional registration fields a flow may demand on top of the always-
 * required username + password. Mirrors the backend allow-list in
 * validateRequiredFieldsJSON.
 */
export type RegistrationFlowRequiredField = 'fullname' | 'email' | 'phone'

/**
 * Role attached to a registration flow (backend RoleResponseDTO).
 */
export interface RegistrationFlowRole {
  role_id: string
  name: string
  description: string
  is_default: boolean
  is_system: boolean
  status: string
  created_at: string
  updated_at: string
}

/**
 * The client a registration flow belongs to, as resolved on the detail response.
 *
 * `identifier` is the OAuth client_id an external app puts in a registration
 * link; `client_id` here is the internal UUID (the backend names the UUID field
 * `client_id` in JSON). Do not mix them up — see RegistrationFlowLink.
 */
export interface RegistrationFlowClientSummary {
  client_id: string
  name: string
  display_name?: string
  identifier: string
  status: string
}

/**
 * Registration flow list row (lean projection).
 */
export interface RegistrationFlow {
  registration_flow_id: string
  /**
   * Slug-shaped and tenant-unique: this IS the public selector an external app
   * puts in a registration link (?registration_flow=<name>). Renaming a flow
   * therefore changes its link.
   */
  name: string
  description: string
  status: RegistrationFlowStatus
  client_id?: string
  verification_required: boolean
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * Registration flow detail projection: the list shape plus `required_fields` and
 * the resolved client. Returned by GET/POST/PUT /{uuid}, PATCH /{uuid}/status,
 * and DELETE /{uuid}/roles/{role_id}.
 */
export interface RegistrationFlowDetail extends RegistrationFlow {
  required_fields: string[]
  client?: RegistrationFlowClientSummary
}

export interface RegistrationFlowListResponse {
  rows: RegistrationFlow[]
  limit: number
  page: number
  total: number
  total_pages: number
}

export interface RegistrationFlowRolesResponse {
  rows: RegistrationFlowRole[]
  limit: number
  page: number
  total: number
  total_pages: number
}

/**
 * Registration flow list query parameters.
 *
 * `status` is a plain string because the backend accepts a comma-separated list
 * ("active,inactive") and the shared listing toolbar joins multi-select values
 * that way — same as IdentityProviderQueryParams.
 */
export interface RegistrationFlowQueryParams {
  /** Free-text match over name + description. */
  search?: string
  name?: string
  status?: string
  client_id?: string
  is_system?: boolean
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/** Query params for a flow's paginated roles listing. */
export interface RegistrationFlowRolesQueryParams {
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Create request.
 *
 * There is no separate identifier — `name` is the public registration-link
 * selector, so the backend validates it as a slug and requires it to be unique
 * within the tenant.
 */
export interface CreateRegistrationFlowRequest {
  name: string
  description?: string
  status?: RegistrationFlowStatus
  client_id: string
  verification_required?: boolean
  required_fields?: string[]
  role_ids?: string[]
}

/**
 * Update request — every field is optional with omitted-means-unchanged
 * semantics on the backend. `client_id` is immutable after create and is not
 * accepted here.
 *
 * Renaming a flow changes its public registration link, so any link an external
 * app already published stops resolving. Warn before submitting a rename.
 */
export interface UpdateRegistrationFlowRequest {
  name?: string
  description?: string
  status?: RegistrationFlowStatus
  verification_required?: boolean
  required_fields?: string[]
  role_ids?: string[]
}

export interface UpdateRegistrationFlowStatusRequest {
  status: RegistrationFlowStatus
}
