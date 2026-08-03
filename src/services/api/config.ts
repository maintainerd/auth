/**
 * API Configuration
 * Centralized configuration for API endpoints and settings
 */

// Runtime environment injected by docker-entrypoint.sh into window.__ENV__.
// Lets a single built image target different API origins per deployment without
// a rebuild. Values are optional; build-time import.meta.env is the fallback.
declare global {
  interface Window {
    __ENV__?: Record<string, string | undefined>
  }
}

function runtimeEnv(key: string): string | undefined {
  if (typeof window === 'undefined') return undefined
  const value = window.__ENV__?.[key]
  // Ignore empty placeholders left by the local-dev config.js.
  return value && value.trim() !== '' ? value : undefined
}

// Get base URL from environment variables
// In development, use relative path to go through Vite proxy
// In production, prefer runtime config, then the build-time value, then a default.
// Both planes are served SAME-ORIGIN, in dev via the Vite proxy and in
// production via the container's nginx (see nginx.conf). This is not a
// convenience: auth cookies are __Host- prefixed, so they are host-only and a
// cookie set by a sibling API host would never be sent back with the console's
// own requests. Keeping the API on this origin is what makes the session work
// at all — and it removes CORS. An absolute URL may still be injected for
// deployments that terminate the proxy elsewhere.
const getBaseUrl = () => {
  // Dev always uses the Vite proxy, regardless of what .env holds.
  if (import.meta.env.DEV) return '/api/v1'
  return (
    runtimeEnv('VITE_AUTH_API_BASE_URL') ||
    import.meta.env.VITE_AUTH_API_BASE_URL ||
    '/api/v1'
  )
}

// The data plane (OAuth2/OIDC + tenant bootstrap), also same-origin. The token
// exchange runs here — a public authorization-server endpoint, never the
// control plane — and its Set-Cookie must land on THIS origin.
const getPublicBaseUrl = () => {
  // Dev always uses the Vite proxy, regardless of what .env holds.
  if (import.meta.env.DEV) return '/public-api/api/v1'
  return (
    runtimeEnv('VITE_AUTH_PUBLIC_API_BASE_URL') ||
    import.meta.env.VITE_AUTH_PUBLIC_API_BASE_URL ||
    '/public-api/api/v1'
  )
}

// Last-resort fallback identity origin. The real identity host is per-tenant and
// comes from the tenant-bootstrap response (`identity_url`); this env var is only
// used when that per-tenant value is unavailable.
const getIdentityBaseUrl = () => {
  return (
    runtimeEnv('VITE_AUTH_IDENTITY_BASE_URL') ||
    import.meta.env.VITE_AUTH_IDENTITY_BASE_URL ||
    'https://auth.maintainerd.local'
  ).replace(/\/$/, '')
}

export const API_CONFIG = {
  BASE_URL: getBaseUrl(),
  PUBLIC_BASE_URL: getPublicBaseUrl(),
  IDENTITY_BASE_URL: getIdentityBaseUrl(),
  TIMEOUT: 30000, // 30 seconds
  HEADERS: {
    'Content-Type': 'application/json',
  },
} as const

// Token delivery mode for this app. Sent on every token-issuing request
// (login, register, refresh) so the backend delivers tokens as httpOnly cookies
// instead of in the response body. Single source of truth — reuse everywhere.
export const TOKEN_DELIVERY_HEADER = { 'X-Token-Delivery': 'cookie' } as const

// API Endpoints
export const API_ENDPOINTS = {
  SETUP: {
    STATUS: '/setup/status',
    CREATE_TENANT: '/setup/create_tenant',
    CREATE_ADMIN: '/setup/create_admin',
    // Required before COMPLETE: the backend gates /setup/complete on
    // IsProfileSetup, so skipping this leaves the tenant stuck in `pending`.
    CREATE_PROFILE: '/setup/create_profile',
    COMPLETE: '/setup/complete',
  },
  AUTH: {
    LOGIN: '/login',
    // Login MFA second step (issues an acr=2 session on success).
    LOGIN_MFA_VERIFY: '/login/mfa/verify',
    LOGIN_MFA_SEND_SMS: '/login/mfa/send-sms',
    LOGIN_MFA_SEND_EMAIL_OTP: '/login/mfa/send-email-otp',
    LOGIN_MFA_WEBAUTHN_BEGIN: '/login/mfa/webauthn/begin',
    REGISTER: '/register',
    REGISTER_INVITE: '/register/invite',
    LOGOUT: '/logout',
    // POST /api/v1/refresh-token — rotates the session using the httpOnly
    // refresh-token cookie (scoped to this path) and Set-Cookies fresh tokens
    // when called with `X-Token-Delivery: cookie`.
    REFRESH: '/refresh-token',
    PROFILE: '/profile',
    ACCOUNT: '/account',
    FORGOT_PASSWORD: '/forgot-password',
    RESET_PASSWORD: '/reset-password',
  },
  TENANT: '/tenant',
  SERVICE: '/services',
  API: '/apis',
  PERMISSION: '/permissions',
  POLICY: '/policies',
  IDENTITY_PROVIDER: '/identity_providers',
  IDENTITY_PROVIDER_TEST: '/identity_providers/test',
  CLIENT: '/clients',
  ROLE: '/roles',
  USER: '/users',
  REGISTRATION_FLOW: '/registration_flows',
  INVITE: '/invite',
  BRANDING: '/branding',
  EMAIL_TEMPLATE: '/email_templates',
  SMS_TEMPLATE: '/sms_templates',
  LOGIN_TEMPLATE: '/login_templates',
  AUTH_EVENTS: '/auth-events',
  WEBHOOK_ENDPOINT: '/webhook-endpoints',
  WEBHOOK_REPLAY: '/webhook-replay',
  EVENT_TYPE: '/event-types',
  TENANT_EVENT_TYPE: '/tenant-event-types',
  EVENT_ROUTE: '/event-routes',
  DASHBOARD: '/dashboard',
  AUTH_EVENTS_EXPORT: '/auth-events/export',
  AUTH_EVENTS_COUNT: '/auth-events/count',
  MANAGEMENT_AUDIT_LOG: '/management-audit-log',
  WORKLOAD_IDENTITY_FEDERATIONS: '/workload-identity-federations',
  CLIENT_ROLES: (id: string) => `/clients/${id}/roles`,
  USER_CONSENTS: (id: string) => `/users/${id}/consents`,
  USER_TRUSTED_DEVICES: (id: string) => `/users/${id}/devices`,
  USER_ERASURE_REQUESTS: (id: string) => `/users/${id}/erasure-requests`,
  POLICY_HISTORY: (id: string) => `/policies/${id}/history`,
} as const
