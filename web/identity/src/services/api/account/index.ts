import { ApiError, get, post, put, deleteRequest } from '@/services/api/client'
import { unwrap, assertSuccess } from '@/services/api/_lib/unwrap'
import type { ApiResponse } from '@/services/api/types'
import type { AccountEntity } from '@/services/api/auth/types'

// ---------------------------------------------------------------------------
// Profiles
// ---------------------------------------------------------------------------

// Mirrors user.ProfileResponseDTO (internal/user/types.go).
export interface UserProfile {
  profile_id: string
  display_name?: string
  first_name?: string
  middle_name?: string
  last_name?: string
  /** YYYY-MM-DD — a calendar date, not an instant. Same shape the write DTO takes. */
  birthdate?: string
  gender?: string
  email?: string
  timezone?: string
  language?: string
  profile_url?: string
  /** Extended data. Carries the OIDC `address` claim under `metadata.address`. */
  metadata?: Record<string, unknown>
  is_default: boolean
  created_at: string
}

// Mirrors user.ProfileRequestDTO. This is a genuine partial update: the service
// (applyProfileFields) assigns only the pointers the caller actually sent, so an
// omitted key means "leave this field alone". `""` is NOT the way to clear a
// field — NilOrNotEmpty rejects it — so clearing a stored value is not
// expressible through this DTO at all.
export interface ProfileRequest {
  first_name: string
  middle_name?: string
  last_name?: string
  display_name?: string
  /** YYYY-MM-DD. Any other format fails validateDateFormat server-side. */
  birthdate?: string
  gender?: string
  email?: string
  timezone?: string
  language?: string
  profile_url?: string
  metadata?: Record<string, unknown>
}

export const fetchProfiles = (): Promise<UserProfile[]> =>
  get<ApiResponse<UserProfile[]>>('/profiles').then(r => unwrap(r, 'fetch profiles') as UserProfile[])

export const createProfile = (data: ProfileRequest): Promise<UserProfile> =>
  post<ApiResponse<UserProfile>>('/profiles', data).then(r => unwrap(r, 'create profile') as UserProfile)

export const updateProfile = (uuid: string, data: ProfileRequest): Promise<UserProfile> =>
  put<ApiResponse<UserProfile>>(`/profiles/${uuid}`, data).then(r => unwrap(r, 'update profile') as UserProfile)

export const deleteProfile = (uuid: string): Promise<void> =>
  deleteRequest<ApiResponse<void>>(`/profiles/${uuid}`).then(r => assertSuccess(r, 'delete profile'))

/**
 * Uploads an avatar and returns the URL the profile now points at.
 *
 * Sent as multipart rather than base64 JSON: base64 inflates the payload by a
 * third and forces the whole image through a JSON parser on both ends.
 *
 * Content-Type is left unset on purpose — the browser must add the multipart
 * boundary, and providing the header manually omits it, which makes the body
 * unparseable server-side.
 */
export const uploadProfilePicture = (uuid: string, file: File): Promise<{ profile_url: string }> => {
  const form = new FormData()
  form.append('file', file)
  // Content-Type must be UNSET, not set: the axios instance defaults every
  // request to application/json, and axios only computes the
  // multipart/form-data boundary when the header is absent. Leaving the default
  // in place sends a multipart body labelled JSON with no boundary, which the
  // server cannot parse.
  return post<ApiResponse<{ profile_url: string }>>(`/profiles/${uuid}/picture`, form, {
    headers: { 'Content-Type': undefined },
  })
    .then(r => unwrap(r, 'upload profile picture') as { profile_url: string })
}

/** Removes an uploaded avatar. A linked URL is unaffected. */
export const deleteProfilePicture = (uuid: string): Promise<void> =>
  deleteRequest<ApiResponse<void>>(`/profiles/${uuid}/picture`).then(r => assertSuccess(r, 'delete profile picture'))

export const setDefaultProfile = (uuid: string): Promise<UserProfile> =>
  put<ApiResponse<UserProfile>>(`/profiles/${uuid}/set-default`, {}).then(r => unwrap(r, 'set default profile') as UserProfile)

// ---------------------------------------------------------------------------
// Account info
// ---------------------------------------------------------------------------

// GET /account returns user.AccountResponseDTO, which the auth service already
// models as AccountEntity — so this is an alias, not a second declaration.
// The hand-written shape it replaces (user_id / username / is_email_verified /
// is_phone_verified / status / created_at) matched no field the endpoint sends;
// the response was cast rather than parsed, so every read silently produced
// undefined — an unverified-looking account for a verified user.
export type AccountInfo = AccountEntity

// Errors propagate. Swallowing them with `.catch(() => null)` rendered a 401, a
// 429 and a 500 all as "no account data", which left dependent controls (the
// password-reset button) disabled with nothing on screen to explain why.
export const fetchAccountInfo = (): Promise<AccountInfo> =>
  get<ApiResponse<AccountInfo>>('/account').then(r => unwrap(r, 'load your account'))

// ---------------------------------------------------------------------------
// Username change
// ---------------------------------------------------------------------------

// Field names mirror user.ChangeUsernameDTO. This sent `{ username }`, which
// the DTO reads as `new_username` — so the server saw an empty value and
// rejected every request — and it omitted current_password entirely, which the
// service compares against the stored hash. The feature could never succeed.
export const changeUsername = (newUsername: string, currentPassword: string): Promise<void> =>
  put<ApiResponse<void>>('/account/username', {
    new_username: newUsername,
    current_password: currentPassword,
  }).then(r => assertSuccess(r, 'change username'))

// ---------------------------------------------------------------------------
// Password change (authenticated self-service)
// ---------------------------------------------------------------------------

// ChangePasswordResult mirrors the backend ChangePasswordResponseDTO. It reports
// what happened to the user's OTHER sessions so the UI can tell them, and
// whether they must re-authenticate (only when the caller's own session could
// not be identified and everything was revoked).
export interface ChangePasswordResult {
  other_sessions_revoked: boolean
  reauthentication_required: boolean
}

// changePassword rotates the signed-in user's own password via the authenticated
// endpoint (PUT /account/password), which enforces the tenant password policy,
// history, and session revocation. This is distinct from the forgot-password
// email flow — a logged-in user should not have to go through their inbox.
//
// The result is unwrapped rather than defaulted. Defaulting a missing body to
// `{ other_sessions_revoked: false, reauthentication_required: false }` let the
// UI announce a plain "Password changed successfully" for a response that never
// carried that verdict — and when the server had actually set
// reauthentication_required, the user was silently signed out on their next
// request with no warning. No body means we do not know what happened, which is
// an error, not a quiet success.
export const changePassword = (current_password: string, new_password: string): Promise<ChangePasswordResult> =>
  put<ApiResponse<ChangePasswordResult>>('/account/password', { current_password, new_password }).then(r => {
    const result = r.success ? r.data : undefined
    if (!result) {
      throw new ApiError({
        message: 'Your password may have been changed, but we could not confirm it. Sign in again to check before retrying.',
        status: 0,
      })
    }
    return result
  })

// ---------------------------------------------------------------------------
// Email change
// ---------------------------------------------------------------------------

// user.ChangeEmailRequestDTO requires current_password as well; omitting it
// failed validation on every attempt.
export const initiateEmailChange = (newEmail: string, currentPassword: string): Promise<void> =>
  post<ApiResponse<void>>('/account/email/change', {
    new_email: newEmail,
    current_password: currentPassword,
  }).then(r => assertSuccess(r, 'initiate email change'))

// user.VerifyEmailChangeDTO reads `otp`, not `token` — sending the wrong key
// meant the server saw an empty code and rejected every confirmation.
export const verifyEmailChange = (otp: string): Promise<void> =>
  post<ApiResponse<void>>('/account/email/verify', { otp }).then(r => assertSuccess(r, 'verify email change'))

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

// Mirrors user.SessionDataResult on the wire (internal/user/types.go). The
// field names previously guessed at were session_id / last_active_at plus an
// is_current flag: the first made every revoke call POST `undefined` (the
// backend rejected it as an invalid UUID, so revoke silently never worked), the
// second meant "Last active" never rendered, and the third does not exist on
// the response at all — so `!is_current` was always true and the page offered
// to revoke the session the user is currently sitting in.
export interface UserSession {
  session_id: string
  ip_address?: string
  user_agent?: string
  created_at: string
  last_used_at?: string
  expires_at?: string
}

export const fetchSessions = (): Promise<UserSession[]> =>
  get<ApiResponse<UserSession[]>>('/account/sessions').then(r => (r.data ?? []) as UserSession[])

export const revokeSession = (uuid: string): Promise<void> =>
  deleteRequest<ApiResponse<void>>(`/account/sessions/${uuid}`).then(r => assertSuccess(r, 'revoke session'))

export const revokeAllSessions = (): Promise<void> =>
  deleteRequest<ApiResponse<void>>('/account/sessions').then(r => assertSuccess(r, 'revoke all sessions'))

// ---------------------------------------------------------------------------
// Trusted devices
// ---------------------------------------------------------------------------

// Matches the backend UserTrustedDeviceResponseDTO (/me/devices).
export interface TrustedDevice {
  uuid: string
  device_name?: string
  location?: string
  ip_address?: string
  user_agent?: string
  trusted_until?: string | null
  last_seen_at?: string | null
  created_at: string
  /** True for the browser making the request, so the UI can badge "This device". */
  current?: boolean
}

export const fetchTrustedDevices = (): Promise<TrustedDevice[]> =>
  get<ApiResponse<TrustedDevice[]>>('/me/devices').then(r => (r.data ?? []) as TrustedDevice[])

export const revokeTrustedDevice = (uuid: string): Promise<void> =>
  deleteRequest<ApiResponse<void>>(`/me/devices/${uuid}`).then(r => assertSuccess(r, 'revoke device'))

// ---------------------------------------------------------------------------
// User settings
// ---------------------------------------------------------------------------

export interface UserSettings {
  language?: string
  timezone?: string
}

export const fetchUserSettings = (): Promise<UserSettings> =>
  get<ApiResponse<UserSettings>>('/user-settings').then(r => (r.data ?? {}) as UserSettings)

export const updateUserSettings = (data: UserSettings): Promise<void> =>
  post<ApiResponse<void>>('/user-settings', data).then(r => assertSuccess(r, 'update settings'))

// ---------------------------------------------------------------------------
// Data export
// ---------------------------------------------------------------------------

export const requestDataExport = (): Promise<{ download_url?: string; message?: string }> =>
  get<ApiResponse<{ download_url?: string; message?: string }>>('/account/export').then(
    r => (r.data ?? {}) as { download_url?: string; message?: string },
  )
