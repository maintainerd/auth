/**
 * Invitation API
 * Admin endpoints for inviting users and managing pending invitations.
 */

import { get, post, deleteRequest } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import { unwrap, assertSuccess } from '../_lib/unwrap'
import type { Invite, SendInviteRequest } from './types'

// List all invitations for the current tenant (returns a plain array).
export async function fetchInvites(): Promise<Invite[]> {
  const r = await get<ApiResponse<Invite[]>>(API_ENDPOINTS.INVITE)
  return unwrap(r, 'fetch invites')
}

// Send an invitation. Requires step-up (acr=2) on the backend.
export async function sendInvite(data: SendInviteRequest): Promise<void> {
  const r = await post<ApiResponse<void>>(API_ENDPOINTS.INVITE, data)
  assertSuccess(r, 'send invite')
}

// Resend an invitation — regenerates the token and re-sends the email.
export async function resendInvite(inviteId: string): Promise<void> {
  const r = await post<ApiResponse<void>>(`${API_ENDPOINTS.INVITE}/${inviteId}/resend`)
  assertSuccess(r, 'resend invite')
}

// Revoke a pending invitation.
export async function revokeInvite(inviteId: string): Promise<void> {
  const r = await deleteRequest<ApiResponse<void>>(`${API_ENDPOINTS.INVITE}/${inviteId}`)
  assertSuccess(r, 'revoke invite')
}

// Fetch a single invitation by ID.
export async function fetchInviteById(inviteId: string): Promise<Invite> {
  const r = await get<ApiResponse<Invite>>(`${API_ENDPOINTS.INVITE}/${inviteId}`)
  return unwrap(r, 'fetch invite')
}
