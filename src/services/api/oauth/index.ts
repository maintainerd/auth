import { deleteRequest, get, post } from '@/services/api/client'
import { API_CONFIG } from '@/services/api/config'
import { assertSuccess, unwrap } from '@/services/api/_lib/unwrap'
import type { ApiResponse } from '@/services/api/types'
import type {
  OAuthAuthorizeResult,
  OAuthConnections,
  OAuthConsentChallenge,
  OAuthConsentDecisionResult,
  OAuthConsentGrant,
} from './types'

const BASE = '/oauth'
const FORM_HEADERS = { 'Content-Type': 'application/x-www-form-urlencoded' } as const

function formBody(values: Record<string, string | undefined>): URLSearchParams {
  const body = new URLSearchParams()
  for (const [key, value] of Object.entries(values)) {
    if (value) body.set(key, value)
  }
  return body
}

export async function authorizeOAuth(queryString: string): Promise<OAuthAuthorizeResult> {
  const query = queryString.startsWith('?') ? queryString.slice(1) : queryString
  const endpoint = query ? `${BASE}/authorize?${query}` : `${BASE}/authorize`
  const response = await get<ApiResponse<OAuthAuthorizeResult>>(endpoint)
  return unwrap(response, 'authorize OAuth request')
}

export async function fetchOAuthConnections(clientId: string): Promise<OAuthConnections> {
  const params = new URLSearchParams({ client_id: clientId })
  const response = await get<ApiResponse<OAuthConnections>>(`${BASE}/connections?${params.toString()}`)
  return unwrap(response, 'fetch OAuth connections')
}

export async function fetchOAuthConsentChallenge(challengeId: string): Promise<OAuthConsentChallenge> {
  const response = await get<ApiResponse<OAuthConsentChallenge>>(`${BASE}/consent/${encodeURIComponent(challengeId)}`)
  return unwrap(response, 'fetch OAuth consent challenge')
}

export async function submitOAuthConsent(challengeId: string, approved: boolean): Promise<OAuthConsentDecisionResult> {
  const response = await post<ApiResponse<OAuthConsentDecisionResult>>(`${BASE}/consent`, {
    challenge_id: challengeId,
    approved,
  })
  return unwrap(response, 'submit OAuth consent')
}

export async function listOAuthConsentGrants(): Promise<OAuthConsentGrant[]> {
  const response = await get<ApiResponse<OAuthConsentGrant[]>>(`${BASE}/consent/grants`)
  return unwrap(response, 'list OAuth consent grants')
}

export async function revokeOAuthConsentGrant(grantId: string): Promise<void> {
  const response = await deleteRequest<ApiResponse<void>>(`${BASE}/consent/grants/${encodeURIComponent(grantId)}`)
  assertSuccess(response, 'revoke OAuth consent grant')
}

export async function approveOAuthDevice(userCode: string): Promise<void> {
  const response = await post<ApiResponse<void>>(
    `${BASE}/device`,
    formBody({ user_code: userCode }),
    { headers: FORM_HEADERS },
  )
  assertSuccess(response, 'approve OAuth device request')
}

export async function denyOAuthDevice(userCode: string): Promise<void> {
  const response = await post<ApiResponse<void>>(
    `${BASE}/device/deny`,
    formBody({ user_code: userCode }),
    { headers: FORM_HEADERS },
  )
  assertSuccess(response, 'deny OAuth device request')
}

export async function approveOAuthCIBA(authReqId: string): Promise<void> {
  const response = await post<ApiResponse<void>>(
    `${BASE}/ciba/approve`,
    formBody({ auth_req_id: authReqId }),
    { headers: FORM_HEADERS },
  )
  assertSuccess(response, 'approve OAuth CIBA request')
}

export async function denyOAuthCIBA(authReqId: string): Promise<void> {
  const response = await post<ApiResponse<void>>(
    `${BASE}/ciba/deny`,
    formBody({ auth_req_id: authReqId }),
    { headers: FORM_HEADERS },
  )
  assertSuccess(response, 'deny OAuth CIBA request')
}

export function oauthEndSessionURL(queryString: string): string {
  if (!queryString) return `${API_CONFIG.BASE_URL}${BASE}/end_session`
  const query = queryString.startsWith('?') ? queryString : `?${queryString}`
  return `${API_CONFIG.BASE_URL}${BASE}/end_session${query}`
}

export async function continueOAuth(requestId: string): Promise<OAuthAuthorizeResult> {
  const response = await post<ApiResponse<OAuthAuthorizeResult>>(`${BASE}/authorize/continue`, {
    request_id: requestId,
  })
  return unwrap(response, 'continue OAuth authorize')
}
