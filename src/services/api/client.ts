/**
 * API Client
 * Base HTTP client with common functionality like error handling, timeouts, etc.
 */

import axios, { type AxiosError, type AxiosRequestConfig, type InternalAxiosRequestConfig } from 'axios'
import { API_CONFIG, API_ENDPOINTS, TOKEN_DELIVERY_HEADER } from './config'
import { requestStepUp } from './stepUp'

// Debug utilities are development-only. Dynamically importing them behind the
// DEV flag keeps the debug helper out of the production bundle entirely.
if (import.meta.env.DEV) {
  void import('./debug')
}

// Custom error class
export class ApiError extends Error {
  public status: number
  public code?: string
  public requestId?: string
  // Seconds to wait before retrying, parsed from the `Retry-After` response
  // header (present on 429/503). Undefined when the server did not send one.
  public retryAfter?: number
  public responseData?: {
    error: string | object
    details?: string | object
    success?: boolean
  }

  constructor({ message, status, code, requestId, retryAfter }: { message: string; status: number; code?: string; requestId?: string; retryAfter?: number }) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.requestId = requestId
    this.retryAfter = retryAfter
  }
}

// Parse the HTTP `Retry-After` header, which may be either a number of seconds
// or an HTTP-date. Returns the delay in whole seconds, or undefined when absent
// or unparseable.
function parseRetryAfter(headerValue: unknown): number | undefined {
  if (typeof headerValue !== 'string' || headerValue.trim() === '') return undefined
  const asSeconds = Number(headerValue)
  if (Number.isFinite(asSeconds)) return Math.max(0, Math.round(asSeconds))
  const asDate = Date.parse(headerValue)
  if (Number.isNaN(asDate)) return undefined
  return Math.max(0, Math.round((asDate - Date.now()) / 1000))
}

// Create axios instance with default configuration
const axiosInstance = axios.create({
  baseURL: API_CONFIG.BASE_URL,
  timeout: API_CONFIG.TIMEOUT,
  headers: API_CONFIG.HEADERS,
  withCredentials: true, // Include cookies for authentication
})

// CSRF (double-submit cookie). The backend protects cookie-auth state-changing
// routes (profile, account, user-settings, mfa, federation) with CSRFMiddleware:
// it issues a JS-readable `__Host-csrf` cookie on safe requests and requires the
// same value echoed in the `X-CSRF-Token` header on unsafe ones. axios' built-in
// XSRF support uses different cookie/header names, so we wire it explicitly.
const CSRF_COOKIE_NAME = '__Host-csrf'
const CSRF_HEADER_NAME = 'X-CSRF-Token'
const CSRF_SAFE_METHODS = new Set(['get', 'head', 'options', 'trace'])

function readCsrfToken(): string | null {
  const cookies = document.cookie ? document.cookie.split('; ') : []
  for (const cookie of cookies) {
    const eq = cookie.indexOf('=')
    if (eq !== -1 && cookie.slice(0, eq) === CSRF_COOKIE_NAME) {
      return decodeURIComponent(cookie.slice(eq + 1))
    }
  }
  return null
}

axiosInstance.interceptors.request.use((config) => {
  const method = (config.method ?? 'get').toLowerCase()
  if (!CSRF_SAFE_METHODS.has(method)) {
    const token = readCsrfToken()
    if (token) {
      config.headers = config.headers ?? {}
      config.headers[CSRF_HEADER_NAME] = token
    }
  }
  return config
})

// Endpoints where a 401 means a genuine credential failure (not an expired
// session), so we must NOT attempt a token refresh — including the refresh
// endpoint itself, to avoid an infinite loop.
const NO_REFRESH_ENDPOINTS = [
  API_ENDPOINTS.AUTH.LOGIN,
  API_ENDPOINTS.AUTH.REGISTER,
  API_ENDPOINTS.AUTH.LOGOUT,
  API_ENDPOINTS.AUTH.REFRESH,
  API_ENDPOINTS.AUTH.MAGIC_LINK_SEND,
  API_ENDPOINTS.AUTH.MAGIC_LINK_VERIFY,
  API_ENDPOINTS.AUTH.FORGOT_PASSWORD,
  API_ENDPOINTS.AUTH.RESET_PASSWORD,
]

// Signed-in, in-page surfaces whose 429 must NOT navigate away.
//
// The /account endpoints share a per-account password-verification throttle
// (checkAccountPasswordThrottle, internal/user/handler_account.go). Sending the
// user to the full-screen /too-many-requests page for one of those throws away
// the form they were half-way through and offers them a "Back to Login" button
// while they are already signed in. The page still owns login-flow 429s, where
// there is no form state to lose and signing in IS the next step.
const NO_LIMIT_REDIRECT_ENDPOINTS = ['/account', '/profile', '/me/', '/recovery/']

// Single-flight refresh: concurrent 401s share one refresh request instead of
// stampeding the refresh endpoint.
let refreshPromise: Promise<void> | null = null

function refreshSession(): Promise<void> {
  if (!refreshPromise) {
    // withCredentials sends the httpOnly refresh-token cookie; the
    // `X-Token-Delivery: cookie` header tells the backend to rotate and
    // Set-Cookie the new access/refresh tokens (cookie-based session).
    // `/refresh-token` is NOT in the middleware's form-encoded exempt list, so
    // it requires `Content-Type: application/json`. Send `{}` (not null) so axios
    // keeps that header — the actual refresh token rides in the httpOnly cookie.
    refreshPromise = axiosInstance
      .post(API_ENDPOINTS.AUTH.REFRESH, {}, {
        headers: { ...TOKEN_DELIVERY_HEADER },
      })
      .then(() => undefined)
      .finally(() => {
        refreshPromise = null
      })
  }
  return refreshPromise
}

// D5: lockout / rate-limit redirect. The axios interceptor lives outside the
// React tree, so it can't call useNavigate() directly. A component (AppBootstrap)
// registers a navigation callback here; the interceptor invokes it when the
// backend returns 423 (account locked) or 429 (rate limited) so the user is
// taken to the matching screen instead of only seeing a toast.
export type LimitRedirectKind = 'locked' | 'rate_limited' | 'maintenance'
type LimitRedirectHandler = (kind: LimitRedirectKind, retryAfterSeconds?: number) => void
let limitRedirectHandler: LimitRedirectHandler | null = null

export function setLimitRedirectHandler(handler: LimitRedirectHandler | null): void {
  limitRedirectHandler = handler
}

type RetriableRequestConfig = InternalAxiosRequestConfig & { _retry?: boolean; _stepUpRetry?: boolean }

// User-facing fallback messages by HTTP status. Used only when the backend does
// not supply its own `error` message, so we never surface a raw `HTTP <status>`
// string to end users. 429/423 are handled ahead of this map with dedicated
// lockout/rate-limit flows.
function defaultMessageForStatus(status: number): string {
  switch (status) {
    case 400:
      return 'The request was invalid. Please check your input and try again.'
    case 401:
      return 'Your session has expired or your credentials are invalid. Please sign in again.'
    case 403:
      return "You don't have permission to perform this action."
    case 404:
      return 'The requested resource could not be found.'
    case 409:
      return 'This action conflicts with the current state. Please refresh and try again.'
    case 422:
      return 'Some of the information provided is invalid. Please review and try again.'
    default:
      if (status >= 500) {
        return 'Something went wrong on our end. Please try again in a moment.'
      }
      return 'Something went wrong. Please try again.'
  }
}

// Response interceptor for error handling
axiosInstance.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    // On an expired access token (401), transparently refresh once and retry the
    // original request. Skipped for auth endpoints and already-retried requests.
    const original = error.config as RetriableRequestConfig | undefined
    const requestUrl = original?.url || ''
    const isAuthEndpoint = NO_REFRESH_ENDPOINTS.some((endpoint) => requestUrl.includes(endpoint))

    if (error.response?.status === 401 && original && !original._retry && !isAuthEndpoint) {
      original._retry = true
      try {
        await refreshSession()
        return axiosInstance(original)
      } catch {
        // Refresh failed — fall through to normal error handling below.
      }
    }

    // Step-up elevation. Sensitive actions (assign role, delete user, revoke
    // sessions, admin MFA reset, …) require an acr=2 token. When the backend
    // signals `step_up_required`, prompt for a second factor once, then retry
    // the original request with the elevated Bearer token. The ceremony is
    // single-flighted in requestStepUp(), so concurrent gated calls share it.
    const stepUpCode = (error.response?.data as { code?: string } | undefined)?.code
    if (error.response?.status === 403 && stepUpCode === 'step_up_required' && original && !original._stepUpRetry) {
      original._stepUpRetry = true
      try {
        const elevatedToken = await requestStepUp()
        original.headers = original.headers ?? {}
        original.headers.Authorization = `Bearer ${elevatedToken}`
        return axiosInstance(original)
      } catch {
        // User cancelled or step-up unavailable — fall through to error handling.
      }
    }

    if (error.response) {
      if (error.response.status === 429) {
        const retryAfter = parseRetryAfter(error.response.headers?.['retry-after'])
        if (!NO_LIMIT_REDIRECT_ENDPOINTS.some((endpoint) => requestUrl.includes(endpoint))) {
          limitRedirectHandler?.('rate_limited', retryAfter)
        }
        return Promise.reject(new ApiError({
          message: retryAfter !== undefined
            ? `Too many requests — please wait ${retryAfter}s and try again`
            : "Too many requests — please wait and try again",
          status: 429,
          code: "rate_limited",
          retryAfter,
        }))
      }

      if (error.response.status === 423) {
        limitRedirectHandler?.('locked')
        return Promise.reject(new ApiError({
          message: "Account temporarily locked due to too many attempts",
          status: 423,
          code: "account_locked",
        }))
      }

      // Tenant maintenance window: the backend returns 503 with code
      // "maintenance_mode" (e.g. on login when the tenant is in maintenance).
      // Route to the service-unavailable page rather than surfacing a raw error.
      if (error.response.status === 503) {
        const body = error.response.data as { code?: string; error?: string; message?: string } | undefined
        if (body?.code === 'maintenance_mode') {
          limitRedirectHandler?.('maintenance')
          return Promise.reject(new ApiError({
            message: body.error ?? body.message ?? 'The system is currently undergoing maintenance.',
            status: 503,
            code: 'maintenance_mode',
          }))
        }
      }

      // Server responded with error status
      const data = error.response.data as {
        error?: string
        details?: string | object
        success?: boolean
        code?: string
        request_id?: string
      } | undefined
      const errorMessage = data?.error || defaultMessageForStatus(error.response.status)
      const errorDetails = data?.details || undefined

      const apiError = new ApiError({
        message: errorMessage,
        status: error.response.status,
        code: data?.code,
        requestId: data?.request_id,
      })

      // Attach the original response data for more detailed error handling
      apiError.responseData = {
        error: errorMessage,
        details: errorDetails,
        success: data?.success
      }

      throw apiError
    } else if (error.code === 'ECONNABORTED') {
      // Request timeout
      throw new ApiError({
        message: 'Request timeout',
        status: 408,
        code: 'TIMEOUT',
      })
    } else if (error.request) {
      // Request was made but no response received
      throw new ApiError({
        message: error.message || 'Network error',
        status: 0,
        code: 'NETWORK_ERROR',
      })
    } else {
      // Something else happened
      throw new ApiError({
        message: error.message || 'Unknown error occurred',
        status: 0,
        code: 'UNKNOWN_ERROR',
      })
    }
  }
)

/**
 * HTTP GET request
 */
export async function get<T>(endpoint: string, config?: AxiosRequestConfig): Promise<T> {
  const response = await axiosInstance.get<T>(endpoint, config)
  return response.data || ({ success: true, message: 'Request completed successfully' } as T)
}

/**
 * HTTP POST request
 *
 * Defaults the body to `{}` so axios keeps the `Content-Type: application/json`
 * header (axios strips it when there is no body). The backend middleware
 * requires that header on every POST/PUT/PATCH, so bodyless admin actions still
 * send a truthful JSON content type. Callers passing form data (URLSearchParams)
 * are unaffected — their body is already defined.
 */
export async function post<T>(endpoint: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  const response = await axiosInstance.post<T>(endpoint, data ?? {}, config)
  return response.data || ({ success: true, message: 'Request completed successfully' } as T)
}

/**
 * HTTP PUT request
 *
 * Defaults the body to `{}` — see `post` for why (Content-Type compliance).
 */
export async function put<T>(endpoint: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  const response = await axiosInstance.put<T>(endpoint, data ?? {}, config)
  return response.data || ({ success: true, message: 'Request completed successfully' } as T)
}

/**
 * HTTP DELETE request
 */
export async function deleteRequest<T>(endpoint: string, config?: AxiosRequestConfig): Promise<T> {
  const response = await axiosInstance.delete<T>(endpoint, config)
  return response.data || ({ success: true, message: 'Request completed successfully' } as T)
}

/**
 * HTTP PATCH request
 *
 * Defaults the body to `{}` — see `post` for why (Content-Type compliance).
 * This covers the bodyless admin actions (verify-email / verify-phone /
 * complete-account) without each call site needing to pass `{}`.
 */
export async function patch<T>(endpoint: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  const response = await axiosInstance.patch<T>(endpoint, data ?? {}, config)
  return response.data || ({ success: true, message: 'Request completed successfully' } as T)
}



// Export API functions as a convenient object (for backward compatibility)
export const apiClient = {
  get,
  post,
  put,
  delete: deleteRequest,
  patch,
}
