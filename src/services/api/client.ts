/**
 * API Client
 * Base HTTP client with common functionality like error handling, timeouts, etc.
 */

import axios, { type AxiosError, type AxiosRequestConfig, type InternalAxiosRequestConfig } from 'axios'
import { API_CONFIG, API_ENDPOINTS, TOKEN_DELIVERY_HEADER } from './config'
import { clearOAuthSession } from './oauth-session'
import { requestStepUp } from './stepUp'

// Debug helpers are development-only — never ship them in a production bundle.
if (import.meta.env.DEV) {
  void import('./debug')
}

// Custom error class
export class ApiError extends Error {
  public status: number
  public code?: string
  public retryAfter?: number
  public responseData?: {
    error: string | object
    details?: string | object
    success?: boolean
  }

  constructor({ message, status, code, retryAfter }: { message: string; status: number; code?: string; retryAfter?: number }) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.retryAfter = retryAfter
  }
}

// Maps an HTTP status to a distinct, user-facing message. Never surface the raw
// `HTTP <status>` string to users — it leaks nothing useful and reads like a bug.
function friendlyMessageForStatus(status: number): string {
  switch (status) {
    case 400:
      return 'The request was invalid. Please check your input and try again.'
    case 401:
      return 'Your session has expired. Please sign in again.'
    case 403:
      return 'You do not have permission to perform this action.'
    case 404:
      return 'The requested resource could not be found.'
    case 409:
      return 'This action conflicts with the current state. Please refresh and try again.'
    case 422:
      return 'Some of the information provided was invalid. Please review and try again.'
    case 429:
      return 'Too many requests. Please wait a moment and try again.'
    default:
      if (status >= 500) return 'The server ran into a problem. Please try again in a moment.'
      return 'Something went wrong. Please try again.'
  }
}

// Parses a `Retry-After` header (delta-seconds or an HTTP date) into seconds.
function parseRetryAfter(value: unknown): number | undefined {
  if (typeof value !== 'string' || value.trim() === '') return undefined
  const seconds = Number(value)
  if (Number.isFinite(seconds) && seconds >= 0) return Math.ceil(seconds)
  const dateMs = Date.parse(value)
  if (!Number.isNaN(dateMs)) {
    const delta = Math.ceil((dateMs - Date.now()) / 1000)
    return delta > 0 ? delta : 0
  }
  return undefined
}

// Create axios instance with default configuration
const axiosInstance = axios.create({
  baseURL: API_CONFIG.BASE_URL,
  timeout: API_CONFIG.TIMEOUT,
  headers: API_CONFIG.HEADERS,
  withCredentials: true, // Include cookies for authentication
})

// Authentication now rides entirely on httpOnly cookies (withCredentials above).
// The access / refresh / id tokens are delivered as cookies via
// `X-Token-Delivery: cookie` on the token-exchange and refresh requests, so
// there is no Authorization header to attach from JS on normal requests — the
// only exception is the step-up retry below, which attaches a short-lived
// elevated (acr=2) access token returned in the step-up response body.

// Endpoints where a 401 means a genuine credential failure (not an expired
// session), so we must NOT attempt a token refresh — including the refresh
// endpoint itself, to avoid an infinite loop.
const NO_REFRESH_ENDPOINTS = [
  API_ENDPOINTS.AUTH.LOGIN,
  API_ENDPOINTS.AUTH.REGISTER,
  API_ENDPOINTS.AUTH.LOGOUT,
  API_ENDPOINTS.AUTH.REFRESH,
  API_ENDPOINTS.AUTH.FORGOT_PASSWORD,
  API_ENDPOINTS.AUTH.RESET_PASSWORD,
]

// Single-flight refresh: concurrent 401s share one refresh request instead of
// stampeding the refresh endpoint.
let refreshPromise: Promise<void> | null = null

function refreshSession(): Promise<void> {
  if (!refreshPromise) {
    // Cookie-based rotation: `withCredentials` sends the httpOnly refresh-token
    // cookie and `X-Token-Delivery: cookie` tells the backend to rotate and
    // Set-Cookie the fresh access/id/refresh tokens. The refresh token rides in
    // the cookie, never in localStorage. Send `{}` (not null) so axios keeps the
    // `Content-Type: application/json` header the backend requires on POST.
    //
    // TODO(auth-cookies): `/refresh-token` lives on the public API and its
    // `__Secure-refresh_token` cookie is scoped to `/api/v1/refresh-token`. In
    // deployments where the console reaches the public API on a different origin
    // (or via the `/public-api` proxy prefix) the cookie path may not match this
    // request path, so silent refresh can fail and the user is bounced back
    // through the hosted-identity login. The console, private API, and public API
    // must be served same-site for httpOnly cookie auth to round-trip fully.
    refreshPromise = axios
      .post(`${API_CONFIG.PUBLIC_BASE_URL}${API_ENDPOINTS.AUTH.REFRESH}`, {}, {
        headers: { ...TOKEN_DELIVERY_HEADER },
        withCredentials: true,
      })
      .then(() => undefined)
      .catch((error) => {
        clearOAuthSession()
        throw error
      })
      .finally(() => {
        refreshPromise = null
      })
  }
  return refreshPromise
}

// Routes that are allowed to see a 401 without being bounced. These are the
// pages that exist precisely because there is no session yet — redirecting from
// them would either be a no-op or, on the OAuth callback, abort the exchange
// that is in the middle of establishing the session.
const UNAUTHENTICATED_ROUTES = ['/login', '/logout', '/auth/callback', '/setup/', '/no-access', '/service-unavailable']

// Fire once. Several requests typically 401 together (a dashboard fans out), and
// each would otherwise call location.replace and fight over the navigation.
let sessionEnded = false

function endDeadSession(): void {
  if (sessionEnded) return
  const path = window.location.pathname
  if (UNAUTHENTICATED_ROUTES.some((route) => path === route || path.startsWith(route))) return
  sessionEnded = true
  clearOAuthSession()
  // Hard navigation, not react-router: flipping auth state while still mounted
  // on a protected route makes the guard see "unauthenticated on a protected
  // route" and fire the hosted-identity OAuth redirect, bouncing the user
  // straight back in. This mirrors logoutAndRedirect, inlined because
  // services/api/auth imports this module and importing it back would be a
  // require cycle.
  window.location.replace('/login')
}

type RetriableRequestConfig = InternalAxiosRequestConfig & { _retry?: boolean; _stepUpRetry?: boolean }

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
        // Refresh failed: the session is gone server-side and cannot be revived.
        //
        // The most common way to get here is a sign-out in the OTHER surface of
        // this same browser — console and identity share one session, so ending
        // it from identity kills the console too. Previously nothing acted on
        // that: the request 401'd, the refresh 401'd, and the console sat on a
        // fully-rendered admin page showing stale data, still believing it was
        // signed in until the user happened to reload. Now it lands on /login,
        // which is what "logged out" is supposed to look like.
        endDeadSession()
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
      // Server responded with error status
      const status = error.response.status
      const data = error.response.data as {
        error?: string
        details?: string | object
        success?: boolean
        code?: string
      } | undefined

      const retryAfter = status === 429
        ? parseRetryAfter(error.response.headers?.['retry-after'])
        : undefined

      // Prefer a meaningful message from the backend; otherwise fall back to a
      // distinct per-status message. Never expose the raw `HTTP <status>` text.
      const backendMessage = typeof data?.error === 'string' && data.error.trim() !== '' ? data.error : undefined
      let errorMessage = backendMessage || friendlyMessageForStatus(status)
      if (status === 429 && !backendMessage && retryAfter && retryAfter > 0) {
        errorMessage = `Too many requests. Please try again in ${retryAfter} second${retryAfter === 1 ? '' : 's'}.`
      }
      const errorDetails = data?.details || undefined

      const apiError = new ApiError({
        message: errorMessage,
        status,
        code: data?.code,
        retryAfter,
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
