/**
 * User API Service
 *
 * Standard CRUD goes through the shared resource factory; the bespoke endpoints
 * (status, roles, identities, profiles, verification) compose the same
 * unwrap/buildQuery helpers so the whole service is consistent.
 */

import { get, post, put, patch, deleteRequest } from '../client'
import { createResourceApi } from '../_lib/resource'
import { unwrap, assertSuccess } from '../_lib/unwrap'
import { buildQuery } from '../_lib/query'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type {
  UserListResponse,
  UserQueryParams,
  User,
  CreateUserRequest,
  UpdateUserRequest,
  UpdateUserStatusRequest,
  UserRolesQueryParams,
  UserRolesResponse,
  UserIdentitiesQueryParams,
  UserIdentitiesResponse,
  UserProfilesQueryParams,
  UserProfilesResponse,
  CreateUserProfileRequest,
  UpdateUserProfileRequest,
  UserProfile,
  UserActivityQueryParams,
  UserActivityResponse,
  UserSession,
  UserMFAResponse,
  UserConsent,
  TrustedDevice,
} from './types'

const userApi = createResourceApi<User, CreateUserRequest, UpdateUserRequest, UserListResponse>(
  API_ENDPOINTS.USER,
  'user',
)

// ── Standard CRUD ───────────────────────────────────────────────────────────
export const fetchUsers = (params?: UserQueryParams): Promise<UserListResponse> =>
  userApi.list(params as Record<string, unknown> | undefined)
export const fetchUserById = (userId: string): Promise<User> => userApi.getById(userId)
export const createUser = (data: CreateUserRequest): Promise<User> => userApi.create(data)
export const updateUser = (userId: string, data: UpdateUserRequest): Promise<User> =>
  userApi.update(userId, data)
export const deleteUser = (userId: string): Promise<void> => userApi.remove(userId)

// ── Bespoke endpoints ─────────────────────────────────────────────────────────
const base = API_ENDPOINTS.USER

export const updateUserStatus = (userId: string, data: UpdateUserStatusRequest): Promise<User> =>
  patch<ApiResponse<User>>(`${base}/${userId}/status`, data).then((r) => unwrap(r, 'update user status'))

export const fetchUserRoles = (userId: string, params?: UserRolesQueryParams): Promise<UserRolesResponse> =>
  get<ApiResponse<UserRolesResponse>>(`${base}/${userId}/roles${buildQuery(params as Record<string, unknown>)}`).then(
    (r) => unwrap(r, 'fetch user roles'),
  )

export const fetchUserIdentities = (
  userId: string,
  params?: UserIdentitiesQueryParams,
): Promise<UserIdentitiesResponse> =>
  get<ApiResponse<UserIdentitiesResponse>>(
    `${base}/${userId}/identities${buildQuery(params as Record<string, unknown>)}`,
  ).then((r) => unwrap(r, 'fetch user identities'))

export const assignUserRoles = (userId: string, data: { role_ids: string[] }): Promise<void> =>
  post<ApiResponse<void>>(`${base}/${userId}/roles`, data).then((r) => assertSuccess(r, 'assign roles to user'))

export const removeUserRole = (userId: string, roleId: string): Promise<void> =>
  deleteRequest<ApiResponse<void>>(`${base}/${userId}/roles/${roleId}`).then((r) =>
    assertSuccess(r, 'remove role from user'),
  )

export const fetchUserProfiles = (
  userId: string,
  params?: UserProfilesQueryParams,
): Promise<UserProfilesResponse> =>
  get<ApiResponse<UserProfilesResponse>>(
    `${base}/${userId}/profiles${buildQuery(params as Record<string, unknown>)}`,
  ).then((r) => unwrap(r, 'fetch user profiles'))

export const createUserProfile = (userId: string, data: CreateUserProfileRequest): Promise<UserProfile> =>
  post<ApiResponse<UserProfile>>(`${base}/${userId}/profiles`, data).then((r) => unwrap(r, 'create user profile'))

export const updateUserProfile = (
  userId: string,
  profileId: string,
  data: UpdateUserProfileRequest,
): Promise<UserProfile> =>
  put<ApiResponse<UserProfile>>(`${base}/${userId}/profiles/${profileId}`, data).then((r) =>
    unwrap(r, 'update user profile'),
  )

export const deleteUserProfile = (userId: string, profileId: string): Promise<void> =>
  deleteRequest<ApiResponse<void>>(`${base}/${userId}/profiles/${profileId}`).then((r) =>
    assertSuccess(r, 'delete user profile'),
  )

export const setUserProfileAsDefault = (userId: string, profileId: string): Promise<UserProfile> =>
  put<ApiResponse<UserProfile>>(`${base}/${userId}/profiles/${profileId}/set-default`, {}).then((r) =>
    unwrap(r, 'set profile as default'),
  )

export const verifyUserEmail = (userId: string): Promise<User> =>
  patch<ApiResponse<User>>(`${base}/${userId}/verify-email`).then((r) => unwrap(r, 'verify email'))

export const verifyUserPhone = (userId: string): Promise<User> =>
  patch<ApiResponse<User>>(`${base}/${userId}/verify-phone`).then((r) => unwrap(r, 'verify phone'))

// Admin action: reset a user's MFA enrollment (POST /mfa/admin/users/{uuid}/reset).
// Requires the "user:mfa:reset" permission + step-up; clears every factor at once.
export const resetUserMfa = (userId: string): Promise<void> =>
  post<ApiResponse<void>>(`/mfa/admin/users/${userId}/reset`).then((r) => assertSuccess(r, 'reset MFA'))

// The individual MFA factors an admin can reset for a user.
export type UserMfaMethod = 'totp' | 'sms' | 'email_otp' | 'webauthn' | 'backup_code'

// Admin action: reset a single MFA factor for a user
// (POST /mfa/admin/users/{uuid}/reset/{method}). Same permission + step-up as the
// full reset, but leaves the user's other factors intact.
export const resetUserMfaMethod = (userId: string, method: UserMfaMethod): Promise<void> =>
  post<ApiResponse<void>>(`/mfa/admin/users/${userId}/reset/${method}`).then((r) =>
    assertSuccess(r, 'reset MFA method'),
  )

// Activity: the auth-events recorded against this user (read-only audit trail).
export const fetchUserActivity = (userId: string, params?: UserActivityQueryParams): Promise<UserActivityResponse> =>
  get<ApiResponse<UserActivityResponse>>(
    `${API_ENDPOINTS.AUTH_EVENTS}${buildQuery({ ...params, user: userId } as Record<string, unknown>)}`,
  ).then((r) => unwrap(r, 'fetch user activity'))

// Recent activity: the latest auth-events across the tenant (no actor filter),
// backend-scoped to the caller's session. Powers the dashboard activity feed.
export const fetchRecentActivity = (params?: UserActivityQueryParams): Promise<UserActivityResponse> =>
  get<ApiResponse<UserActivityResponse>>(
    `${API_ENDPOINTS.AUTH_EVENTS}${buildQuery({ ...params } as Record<string, unknown>)}`,
  ).then((r) => unwrap(r, 'fetch recent activity'))

// Sessions: active sessions for this user, with admin revoke.
export const fetchUserSessions = (userId: string): Promise<UserSession[]> =>
  get<ApiResponse<UserSession[]>>(`${base}/${userId}/sessions`).then((r) => unwrap(r, 'fetch user sessions'))

export const revokeUserSession = (userId: string, sessionId: string): Promise<void> =>
  deleteRequest<ApiResponse<void>>(`${base}/${userId}/sessions/${sessionId}`).then((r) =>
    assertSuccess(r, 'revoke session'),
  )

// Admin: revoke ALL of a user's active sessions (force global sign-out).
export const revokeAllUserSessions = (userId: string): Promise<void> =>
  deleteRequest<ApiResponse<void>>(`${base}/${userId}/sessions`).then((r) =>
    assertSuccess(r, 'revoke all sessions'),
  )

// MFA: the user's MFA configuration (TOTP, WebAuthn keys, backup codes).
export const fetchUserMFA = (userId: string): Promise<UserMFAResponse> =>
  get<ApiResponse<UserMFAResponse>>(`${base}/${userId}/mfa`).then((r) => unwrap(r, 'fetch user MFA'))

export const forcePasswordChange = (userId: string): Promise<void> =>
  put<ApiResponse<void>>(`${base}/${userId}/force-password-change`, {}).then((r) =>
    assertSuccess(r, 'force password change'),
  )

export const unlinkUserIdentity = (userId: string, identityId: string): Promise<void> =>
  deleteRequest<ApiResponse<void>>(`${base}/${userId}/identities/${identityId}`).then((r) =>
    assertSuccess(r, 'unlink identity'),
  )

// The API returns a bare array of all consents for the user (not paginated).
export const fetchUserConsents = (userId: string): Promise<UserConsent[]> =>
  get<ApiResponse<UserConsent[]>>(`${base}/${userId}/consents`).then((r) =>
    unwrap(r, 'fetch user consents') ?? [],
  )

// The API returns a bare array of devices (same shape as identity's /me/devices).
export const fetchUserTrustedDevices = (userId: string): Promise<TrustedDevice[]> =>
  get<ApiResponse<TrustedDevice[]>>(`${base}/${userId}/devices`).then((r) =>
    unwrap(r, 'fetch user devices') ?? [],
  )

export const createErasureRequest = (userId: string): Promise<void> =>
  post<ApiResponse<void>>(`${base}/${userId}/erasure-requests`).then((r) =>
    assertSuccess(r, 'create erasure request'),
  )

// Admin: revoke a user's trusted device.
export const revokeUserDevice = (userId: string, deviceId: string): Promise<void> =>
  deleteRequest<ApiResponse<void>>(`${base}/${userId}/devices/${deviceId}`).then((r) =>
    assertSuccess(r, 'revoke device'),
  )

// Admin: withdraw a user's consent (GDPR right to withdraw — logged, not erased).
export const withdrawUserConsent = (userId: string, consentType: string): Promise<void> =>
  post<ApiResponse<void>>(`${base}/${userId}/consents/withdraw`, { consent_type: consentType }).then((r) =>
    assertSuccess(r, 'withdraw consent'),
  )

// Admin: clear a user's failed-login lockout.
export const unlockUser = (userId: string): Promise<void> =>
  post<ApiResponse<void>>(`${base}/${userId}/unlock`).then((r) =>
    assertSuccess(r, 'unlock account'),
  )

/**
 * Membership candidates: the SYSTEM-tenant users that a tenant may add as
 * members. `CreateByUserUUID` accepts only those, and the ordinary user list is
 * pinned to the caller's own tenant — so this is the only source that yields
 * choices which will actually succeed. The tenant is resolved server-side, so
 * there is nothing to pass and nothing to point elsewhere.
 */
export interface MembershipCandidate {
  user_id: string
  username: string
  email: string
  fullname?: string
}

export interface MembershipCandidateListResponse {
  rows: MembershipCandidate[]
  total: number
  page: number
  limit: number
  total_pages: number
}

export const fetchMembershipCandidates = (
  params?: { search?: string; page?: number; limit?: number },
): Promise<MembershipCandidateListResponse> =>
  get<ApiResponse<MembershipCandidateListResponse>>(
    `${API_ENDPOINTS.USER}/membership-candidates`,
    params as Record<string, unknown> | undefined,
  ).then((r) => unwrap(r, 'fetch membership candidates'))
