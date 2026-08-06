/**
 * Authentication Service
 * Handles authentication API calls and storage operations
 */

import { post, get } from '../client'
import { API_ENDPOINTS, TOKEN_DELIVERY_HEADER } from '../config'
import type { ApiResponse } from '../types'
import { unwrap } from '../_lib/unwrap'
import type { WebAuthnAssertionOptions } from '@/lib/webauthn'
import type { ProfileEntity, AccountEntity, LoginRequest, LoginResponse, MFALoginVerifyRequest, LogoutResponse, RegisterRequest, RegisterResponse, RegisterInviteRequest, RegisterInviteQueryParams, CreateProfileRequest, CreateProfileResponse, ForgotPasswordRequest, ForgotPasswordResponse, ResetPasswordRequest, ResetPasswordResponse, ResetPasswordQueryParams, ProfileResponse, RegistrationContext } from './types'
import { appendPublicAuthContext, publicAuthQuery, type PublicAuthContext } from '@/utils/clientContext'

type AccountResponse = ApiResponse<AccountEntity>

/**
 * Login user with credentials
 * @param username - User's email/username
 * @param password - User's password
 * @param tenantId - Optional tenant identifier for tenant-scoped login
 * @returns Promise<LoginResponse>
 */
export async function login(data: LoginRequest): Promise<LoginResponse> {
	const endpoint = `${API_ENDPOINTS.AUTH.LOGIN}?${publicAuthQuery()}`
	const response = await post<LoginResponse>(
		endpoint,
		data,
		{
			headers: { ...TOKEN_DELIVERY_HEADER }
		}
	)
	return response
}

/**
 * Complete the login MFA second step. On success the backend Set-Cookies an
 * acr=2 session (with X-Token-Delivery: cookie), so no token handling is needed
 * client-side — callers just refresh the auth state afterwards.
 */
export async function verifyMFALogin(
  data: MFALoginVerifyRequest,
  tenantId?: string,
  clientId?: string,
): Promise<LoginResponse> {
  const endpoint = `${API_ENDPOINTS.AUTH.LOGIN_MFA_VERIFY}?${publicAuthQuery({ tenantId, clientId })}`
  return post<LoginResponse>(endpoint, data, {
    headers: { ...TOKEN_DELIVERY_HEADER },
  })
}

/** Send an SMS OTP for the in-flight login MFA challenge. */
export async function sendMFALoginSMS(challengeToken: string, context?: PublicAuthContext): Promise<void> {
  await post<ApiResponse<void>>(`${API_ENDPOINTS.AUTH.LOGIN_MFA_SEND_SMS}?${publicAuthQuery(context)}`, {
    mfa_challenge_token: challengeToken,
  })
}

/** Send an Email OTP for the in-flight login MFA challenge. */
export async function sendMFALoginEmailOtp(challengeToken: string, context?: PublicAuthContext): Promise<void> {
  await post<ApiResponse<void>>(`${API_ENDPOINTS.AUTH.LOGIN_MFA_SEND_EMAIL_OTP}?${publicAuthQuery(context)}`, {
    mfa_challenge_token: challengeToken,
  }, { headers: TOKEN_DELIVERY_HEADER })
}

export async function verifyMagicLink(queryString: string): Promise<LoginResponse> {
  return post<LoginResponse>(`${API_ENDPOINTS.AUTH.MAGIC_LINK_VERIFY}?${queryString}`, {}, {
    headers: TOKEN_DELIVERY_HEADER,
  })
}

export interface SendMagicLinkContext {
  clientId?: string
  tenantId?: string
}

export async function sendMagicLink(email: string, context: SendMagicLinkContext = {}): Promise<void> {
  const query = publicAuthQuery(context)
  const endpoint = `${API_ENDPOINTS.AUTH.MAGIC_LINK_SEND}${query ? `?${query}` : ''}`
  const response = await post<ApiResponse<unknown>>(endpoint, { email })
  if (!response.success) {
    throw new Error(typeof response.error === 'string' ? response.error : 'Failed to send sign-in link')
  }
}

/** Begin a passkey assertion ceremony for the in-flight login MFA challenge. */
export async function beginMFALoginWebAuthn(challengeToken: string, context?: PublicAuthContext): Promise<WebAuthnAssertionOptions> {
  const r = await post<ApiResponse<WebAuthnAssertionOptions>>(`${API_ENDPOINTS.AUTH.LOGIN_MFA_WEBAUTHN_BEGIN}?${publicAuthQuery(context)}`, {
    mfa_challenge_token: challengeToken,
  })
  if (!r.success || !r.data) {
    throw new Error(typeof r.error === 'string' ? r.error : 'Failed to begin WebAuthn authentication')
  }
  return r.data
}

// Extended register request with optional query parameters
export interface RegisterServiceRequest extends Omit<RegisterRequest, 'username'> {
  clientId?: string
  tenantId?: string
  registrationFlow?: string
}

/**
 * Fetch what the signup form must collect for this client and (optional) flow.
 *
 * Public and unauthenticated. Deliberately NOT part of /oauth/connections: login
 * options are a property of the client, and a registration flow must never be
 * able to change them.
 */
export async function fetchRegistrationContext(
  clientId: string,
  registrationFlow?: string,
): Promise<RegistrationContext> {
  const params = new URLSearchParams({ client_id: clientId })
  if (registrationFlow) params.set('registration_flow', registrationFlow)

  const response = await get<ApiResponse<RegistrationContext>>(
    `${API_ENDPOINTS.AUTH.REGISTRATION_CONTEXT}?${params.toString()}`,
  )
  if (response.success && response.data) return response.data
  throw new Error(response.message || 'Failed to load registration context')
}

/**
 * Register a new user
 * @param data - Registration data (email + password; email is sent as the username)
 * @returns Promise<RegisterResponse>
 */
export async function register(data: RegisterServiceRequest): Promise<RegisterResponse> {
  const registerData: RegisterRequest = {
    username: data.email,
    email: data.email,
    password: data.password
  }

  // Omit rather than send "" — an empty string only escapes format validation by
  // accident and is treated as missing anyway, so omission is unambiguous.
  if (data.fullname?.trim()) registerData.fullname = data.fullname.trim()
  if (data.phone?.trim()) registerData.phone = data.phone.trim()

  // Build endpoint URL with query parameters if provided
  let endpoint = API_ENDPOINTS.AUTH.REGISTER
  const queryParams = new URLSearchParams()

  appendPublicAuthContext(queryParams, { clientId: data.clientId, tenantId: data.tenantId })

  if (data.registrationFlow) {
    queryParams.set('registration_flow', data.registrationFlow)
  }

  if (queryParams.toString()) {
    endpoint += `?${queryParams.toString()}`
  }

  const response = await post<RegisterResponse>(
    endpoint,
    registerData,
    {
      headers: { ...TOKEN_DELIVERY_HEADER }
    }
  )
  return response
}

/**
 * Register a new user via invite token (signed URL flow).
 * The query params carry only the signed invite token, expiration, signature,
 * and public auth context. The backend resolves the invited email, optional
 * registration flow, roles, and callback from the signed/stored invite.
 */
export async function registerInvite(
  data: RegisterInviteRequest,
  queryParams: RegisterInviteQueryParams
): Promise<RegisterResponse> {
  const params = new URLSearchParams({
    invite_token: queryParams.invite_token,
    expires: queryParams.expires,
    sig: queryParams.sig,
  })
  // email and callback_url are part of the signed URL, so they must be forwarded
  // verbatim or the server-side signature check fails.
  if (queryParams.email) params.set('email', queryParams.email)
  if (queryParams.callback_url) params.set('callback_url', queryParams.callback_url)
  appendPublicAuthContext(params, {
    clientId: queryParams.client_id,
    tenantId: queryParams.tenant_id,
  })

  const url = `${API_ENDPOINTS.AUTH.REGISTER_INVITE}?${params.toString()}`

  return post<RegisterResponse>(
    url,
    {
      username: data.username,
      password: data.password,
    },
    {
      headers: { ...TOKEN_DELIVERY_HEADER }
    }
  )
}

/**
 * Logout user
 * @returns Promise<LogoutResponse>
 */
export async function logout(): Promise<LogoutResponse> {
  const response = await post<LogoutResponse>(API_ENDPOINTS.AUTH.LOGOUT, {})
  return response
}

/**
 * Fetch user profile from API
 * @returns Promise<ProfileEntity | null> - Returns null if profile doesn't exist (e.g., newly registered user)
 */
export async function fetchProfile(): Promise<ProfileEntity | null> {
  try {
    const response = await get<ProfileResponse>(API_ENDPOINTS.AUTH.PROFILE)

    if (response.success && response.data) {
      return response.data
    }

    return null
  } catch {
    return null
  }
}

/**
 * Create user profile for authenticated users (requires cookie token)
 * @param data - Profile creation data
 * @returns Promise<CreateProfileResponse>
 */
export async function createUserProfile(data: CreateProfileRequest): Promise<CreateProfileResponse> {
  const response = await post<CreateProfileResponse>(
    API_ENDPOINTS.AUTH.PROFILE,
    data
  )
  return response
}

/**
 * Create profile for registered users (dedicated register flow)
 * @param data - Profile creation data
 * @returns Promise<CreateProfileResponse>
 */
export async function createRegisterProfile(data: CreateProfileRequest): Promise<CreateProfileResponse> {
  const response = await post<CreateProfileResponse>(
    API_ENDPOINTS.AUTH.PROFILE,
    data
  )
  return response
}

/**
 * Validate if the user is still authenticated with the backend
 * This checks if the backend cookie is still valid by calling the profile endpoint
 * @returns Promise<ProfileEntity | null> - Returns profile if authenticated, null otherwise
 */
export async function validateAuthentication(): Promise<AccountEntity | null> {
  try {
    const response = await get<AccountResponse>(API_ENDPOINTS.AUTH.ACCOUNT)
    if (response.success && response.data) return response.data
    return null
  } catch (err: unknown) {
    const apiErr = err as { status?: number }
    // 401/403 is the server's verdict: there is no session. Anything else — a
    // timeout, a 5xx, a network blip, a rate limit — means we do NOT know. It
    // must propagate so boot can record "unknown", not silently resolve to
    // "anonymous" and bounce an authenticated user through SSO.
    if (apiErr?.status === 401 || apiErr?.status === 403) return null
    throw err
  }
}

export async function fetchAccount(): Promise<AccountEntity | null> {
  try {
    const response = await get<AccountResponse>(API_ENDPOINTS.AUTH.ACCOUNT)
    if (response.success && response.data) return response.data
    return null
  } catch { return null }
}

/**
 * Request password reset
 * @param data - Email address for password reset
 * @returns Promise<ForgotPasswordResponse>
 */
export async function forgotPassword(data: ForgotPasswordRequest): Promise<ForgotPasswordResponse> {
  const response = await post<ForgotPasswordResponse>(
    `${API_ENDPOINTS.AUTH.FORGOT_PASSWORD}?${publicAuthQuery()}`,
    data
  )
  return response
}

/**
 * Reset password with token from query params
 * @param data - New password
 * @param queryParams - Query parameters from the reset link (client_id, expires, provider_id, sig, token)
 * @returns Promise<ResetPasswordResponse>
 */
export async function resetPassword(
  data: ResetPasswordRequest,
  queryParams: ResetPasswordQueryParams
): Promise<ResetPasswordResponse> {
  const params = appendPublicAuthContext(new URLSearchParams({
    expires: queryParams.expires,
    sig: queryParams.sig,
    token: queryParams.token
  }), {
    clientId: queryParams.client_id,
    tenantId: queryParams.tenant_id,
  })

  const url = `${API_ENDPOINTS.AUTH.RESET_PASSWORD}?${params.toString()}`

  const response = await post<ResetPasswordResponse>(url, data)
  return response
}

export interface InviteContextResponse {
  invite_token: string
  email: string
  callback_url?: string | null
  expires_at?: string | null
  status: string
}

/**
 * Look up a pending invite by its token.
 *
 * Errors propagate. The backend answers 410 Gone for a revoked or expired
 * invite, and swallowing that meant the caller could never tell a dead link from
 * a live one — the user filled in the whole form and chose a password before the
 * failure surfaced, as a raw backend string, on submit.
 */
export async function fetchInviteContext(inviteToken: string): Promise<InviteContextResponse> {
  const response = await get<ApiResponse<InviteContextResponse>>(`/invite?invite_token=${encodeURIComponent(inviteToken)}`)
  return unwrap(response, 'load this invitation')
}

// Confirm a pending account link request. Requires an authenticated session as
// the existing account. Returns the confirmed provider name and status.
export async function confirmAccountLink(token: string): Promise<{ provider: string; status: string }> {
  const r = await post<ApiResponse<{ provider: string; status: string }>>(
    `/account-link/${token}/confirm`,
    {},
  )
  return unwrap(r, 'confirm account link')
}

// Resume a brokered OAuth flow after a user has confirmed an account link.
// Returns the downstream redirect URL the browser should navigate to.
export async function resumeBrokerSession(data: {
  broker_session_uuid: string
  account_link_token: string
}): Promise<{ redirect_url: string; access_token?: string }> {
  const r = await post<ApiResponse<{ redirect_url: string; access_token?: string }>>(
    API_ENDPOINTS.AUTH.BROKER_RESUME,
    data,
  )
  return unwrap(r, 'resume broker session')
}

// Request erasure of the authenticated user's personal data (GDPR Art.17).
// Schedules anonymisation in 30 days. Idempotent — repeated calls are safe.
export async function requestDataErasure(): Promise<{ uuid: string; status: string; scheduled_at: string }> {
  const r = await post<ApiResponse<{ uuid: string; status: string; scheduled_at: string }>>(
    API_ENDPOINTS.AUTH.ERASURE_REQUEST,
    {},
  )
  return unwrap(r, 'request data erasure')
}

// Export functions as an object for backward compatibility
export const authService = {
  login,
  verifyMFALogin,
  sendMFALoginSMS,
  sendMFALoginEmailOtp,
  beginMFALoginWebAuthn,
  register,
  registerInvite,
  logout,
  fetchProfile,
  fetchAccount,
  createUserProfile,
  createRegisterProfile,
  validateAuthentication,
  forgotPassword,
  resetPassword
}
