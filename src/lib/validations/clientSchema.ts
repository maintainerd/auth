/**
 * Client Form Validation Schema
 *
 * Kept in LOCK-STEP with the backend so the two never disagree about what a valid
 * client is. Backend counterparts:
 *   - internal/client/validation_client.go  — per-field rules
 *   - internal/client/client_matrix.go      — the cross-field validity matrix
 *   - internal/client/redirect_match.go     — redirect-URI rules per client type
 *   - migration/019_create_clients_table.go — CHECK constraints
 *
 * A client's fields are only coherent as a combination: client_type decides
 * whether a secret can exist, which auth methods are legal, which grants make
 * sense, and what a redirect URI may look like. Validating each field alone lets
 * an operator save a client that can never authenticate, so the cross-field rules
 * live here too rather than being discovered as a 400 on save.
 */

import * as yup from 'yup'

// ─── Enumerations (mirror the DB CHECK constraints) ─────────────────────────

export const CLIENT_TYPES = ['traditional', 'spa', 'mobile', 'm2m'] as const
export type ClientTypeValue = (typeof CLIENT_TYPES)[number]

/** Public clients cannot keep a secret: their code is readable by the user. */
export const PUBLIC_CLIENT_TYPES: readonly ClientTypeValue[] = ['spa', 'mobile']

export const isPublicClientType = (t?: string): boolean =>
  PUBLIC_CLIENT_TYPES.includes(t as ClientTypeValue)

/**
 * Auth methods the token endpoint can actually perform.
 *
 * tls_client_auth and self_signed_tls_client_auth are deliberately ABSENT: the
 * registry and the CHECK constraint accept them, but there is no
 * certificate-binding implementation, so the backend now refuses them at write
 * time. Offering them here would only produce a client that cannot authenticate.
 */
export const CLIENT_AUTH_METHODS = [
  'none',
  'client_secret_basic',
  'client_secret_post',
  'client_secret_jwt',
  'private_key_jwt',
] as const
export type ClientAuthMethod = (typeof CLIENT_AUTH_METHODS)[number]

/** Methods that authenticate with the shared secret, so one must exist. */
export const SECRET_BASED_AUTH_METHODS: readonly ClientAuthMethod[] = [
  'client_secret_basic',
  'client_secret_post',
  'client_secret_jwt',
]

export const authMethodRequiresSecret = (m?: string): boolean =>
  SECRET_BASED_AUTH_METHODS.includes(m as ClientAuthMethod)

/** Mirrors chk_clients_grant_types. */
export const GRANT_TYPES = [
  'authorization_code',
  'refresh_token',
  'client_credentials',
  'urn:ietf:params:oauth:grant-type:device_code',
  'urn:openid:params:grant-type:ciba',
  'urn:ietf:params:oauth:grant-type:token-exchange',
] as const

/** Mirrors chk_clients_response_types. Only `code` is implemented end to end. */
export const RESPONSE_TYPES = ['code'] as const

export const REQUIRED_ACR_VALUES = ['1', '2'] as const

// ─── Field-level patterns ───────────────────────────────────────────────────

/** Mirrors clientDomainPattern: a bare hostname or an absolute https URL. */
const DOMAIN_PATTERN =
  /^(https:\/\/)?[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?(:[0-9]{1,5})?(\/.*)?$/

const CLIENT_NAME_PATTERN = /^[a-z0-9\-_]+$/

/** Reverse-domain private-use scheme, per RFC 8252 §7.1 (e.g. com.example.app). */
const REVERSE_DOMAIN_SCHEME = /^[a-z0-9]+(\.[a-z0-9-]+)+$/

const LOOPBACK_HOSTS = new Set(['127.0.0.1', '[::1]', '::1'])

/**
 * Validates a redirect URI the way the backend does at registration time
 * (ValidateRegisteredRedirectURI). Returns null when valid, else the reason.
 *
 * The rules differ per client type, which is why this cannot be a single regex:
 * a mobile app legitimately receives the response on a private-use scheme or an
 * ephemeral loopback port, while a browser client must use https.
 */
export function validateRedirectUri(
  clientType: string | undefined,
  raw: string
): string | null {
  const value = raw.trim()
  if (!value) return 'Redirect URI is required'

  // Code-executing schemes would turn an open redirect into script execution.
  const lower = value.toLowerCase()
  for (const scheme of ['javascript:', 'data:', 'vbscript:', 'file:']) {
    if (lower.startsWith(scheme)) return `Redirect URI must not use the ${scheme} scheme`
  }

  // The authorization response appends its own fragment (OIDC Core §3.1.2.1).
  if (value.includes('#')) return 'Redirect URI must not contain a fragment'

  let url: URL
  try {
    url = new URL(value)
  } catch {
    return 'Redirect URI must be an absolute URI including a scheme'
  }

  if (url.username || url.password) return 'Redirect URI must not contain embedded credentials'

  const scheme = url.protocol.replace(':', '').toLowerCase()

  if (scheme === 'https') {
    return url.hostname ? null : 'Redirect URI must include a host'
  }

  if (scheme === 'http') {
    // Loopback only — plain http elsewhere exposes the authorization code.
    return LOOPBACK_HOSTS.has(url.hostname)
      ? null
      : 'Redirect URI must use https (http is only allowed for 127.0.0.1 and [::1])'
  }

  if (clientType !== 'mobile') {
    return `Redirect URI scheme "${scheme}" is only allowed for mobile clients`
  }
  return REVERSE_DOMAIN_SCHEME.test(scheme)
    ? null
    : 'A custom scheme must be a reverse-domain name (e.g. com.example.app)'
}

// ─── The form schema ────────────────────────────────────────────────────────

export const clientSchema = yup.object({
  name: yup
    .string()
    .required('Client name is required')
    .min(3, 'Client name must be at least 3 characters')
    .max(50, 'Client name must not exceed 50 characters')
    .matches(
      CLIENT_NAME_PATTERN,
      'Client name can only contain lowercase letters, numbers, hyphens, and underscores'
    ),
  displayName: yup
    .string()
    .required('Display name is required')
    // Mirrors validation_client.go Length(8, 200).
    .min(8, 'Display name must be at least 8 characters')
    .max(200, 'Display name must not exceed 200 characters'),
  clientType: yup
    .string()
    .required('Client type is required')
    .oneOf(CLIENT_TYPES, 'Invalid client type'),
  domain: yup
    .string()
    .required('Domain is required')
    .min(3, 'Domain must be at least 3 characters')
    // 253 and a host pattern, matching the column and clientDomainPattern. The
    // domain becomes the token issuer, so free text here is load-bearing.
    .max(253, 'Domain must not exceed 253 characters')
    .matches(DOMAIN_PATTERN, 'Domain must be a hostname or an https URL'),
  status: yup
    .string()
    .required('Status is required')
    .oneOf(['active', 'inactive'], 'Invalid status'),
})

export type ClientFormData = yup.InferType<typeof clientSchema>

// ─── Cross-field OAuth rules ────────────────────────────────────────────────
//
// These fields are held in component state rather than react-hook-form, so they
// cannot be validated by the resolver above. They are validated here instead and
// called explicitly at submit, which keeps every rule in one file alongside the
// backend references — the alternative (rules scattered as ad-hoc checks in the
// submit handler) is how the FE and BE drifted apart in the first place.

export interface ClientOAuthConfigValues {
  clientType?: string
  tokenEndpointAuthMethod?: string
  grantTypes?: string[]
  allowedScopes?: string[]
  accessTokenTtl?: number
  refreshTokenTtl?: number
  sessionIdleTimeout?: number
  sessionAbsoluteTimeout?: number
  backchannelLogoutUri?: string
  backchannelLogoutSessionRequired?: boolean
  redirectUris?: string[]
  /** Inline JWK Set, as the operator typed it. Verified for shape, not for keys. */
  jwks?: string
  jwksUri?: string
}

/**
 * Returns a message per invalid field, or an empty object when the combination is
 * valid. Mirrors ValidateClientOAuthMatrix and the DB CHECK constraints, so the
 * operator sees the same verdict the server would give — before saving.
 */
export function validateClientOAuthConfig(
  v: ClientOAuthConfigValues
): Record<string, string> {
  const errors: Record<string, string> = {}
  const publicClient = isPublicClientType(v.clientType)
  const method = v.tokenEndpointAuthMethod
  const grants = v.grantTypes ?? []

  // "none" means no credential, which only a client incapable of holding one may
  // use — client_id is public, so anything else leaves the token endpoint open.
  if (method === 'none' && !publicClient) {
    errors.tokenEndpointAuthMethod =
      'Only public clients (SPA, mobile) may use "none"; a confidential client must authenticate'
  }
  if (authMethodRequiresSecret(method) && publicClient) {
    errors.tokenEndpointAuthMethod =
      'A public client cannot keep a secret — use "none" with PKCE'
  }
  if (!method) {
    errors.tokenEndpointAuthMethod = 'Client authentication method is required'
  }

  if (grants.length === 0) {
    errors.grantTypes = 'Select at least one grant type'
  }
  if (grants.includes('client_credentials')) {
    if (publicClient) {
      errors.grantTypes =
        'client_credentials requires client authentication, so it is not valid for a public client'
    } else if (method === 'none') {
      errors.grantTypes =
        'client_credentials requires client authentication; "none" would let anyone holding the public client ID mint tokens'
    }
    if ((v.allowedScopes?.length ?? 0) === 0) {
      errors.allowedScopes = 'A client using client_credentials must declare its allowed scopes'
    }
  }
  if (grants.includes('authorization_code') && v.clientType === 'm2m') {
    errors.grantTypes =
      'authorization_code is not valid for an m2m client: there is no user to authorize'
  }

  // Mirrors chk_clients_token_ttl_order and chk_clients_session_timeout_order.
  if (v.accessTokenTtl != null && v.accessTokenTtl <= 0) {
    errors.accessTokenTtl = 'Access token lifetime must be greater than 0 seconds'
  }
  if (v.refreshTokenTtl != null && v.refreshTokenTtl <= 0) {
    errors.refreshTokenTtl = 'Refresh token lifetime must be greater than 0 seconds'
  }
  if (
    v.accessTokenTtl != null &&
    v.refreshTokenTtl != null &&
    v.refreshTokenTtl < v.accessTokenTtl
  ) {
    errors.refreshTokenTtl =
      'Refresh token lifetime must be greater than or equal to the access token lifetime'
  }
  if (
    v.sessionIdleTimeout != null &&
    v.sessionAbsoluteTimeout != null &&
    v.sessionAbsoluteTimeout < v.sessionIdleTimeout
  ) {
    errors.sessionAbsoluteTimeout =
      'Absolute timeout must be greater than or equal to the idle timeout'
  }

  // A session requirement is meaningless without an endpoint to notify.
  if (v.backchannelLogoutSessionRequired && !v.backchannelLogoutUri?.trim()) {
    errors.backchannelLogoutSessionRequired =
      'Back-channel logout session required needs a back-channel logout URI'
  }

  // Redirect URIs follow per-type rules: a mobile app may use a private-use
  // scheme or an ephemeral loopback port; a browser client must use https.
  for (const uri of v.redirectUris ?? []) {
    if (!uri.trim()) continue
    const reason = validateRedirectUri(v.clientType, uri)
    if (reason) {
      errors.redirectUris = reason
      break
    }
  }

  Object.assign(errors, validateClientKeys(v))

  // Deliberately NOT enforced here: "an authorization_code client must have at
  // least one redirect URI". It is true at runtime — MatchClientRedirectURI fails
  // closed with "no redirect URIs registered" — but the form creates URIs in a
  // second step after the client exists, so blocking creation on it would make a
  // legitimate two-phase workflow impossible. What matters for security is the
  // FORMAT of any URI that is provided, which is enforced above.

  return errors
}

/**
 * Private-key components that must never be pasted into a JWKS. A JWK Set is the
 * client's PUBLIC keys; a private component means the operator is handing us the
 * signing key, which the server rejects and which would be a credential leak.
 */
const JWK_PRIVATE_COMPONENTS = ['d', 'p', 'q', 'dp', 'dq', 'qi', 'k'] as const

/**
 * Mirrors validateAdvancedClientConfig and the private_key_jwt arm of
 * ValidateClientOAuthMatrix. Without registered keys the token endpoint rejects
 * every client assertion, so a private_key_jwt client saved without them can never
 * authenticate — the server refuses the write, and this reports it before the round
 * trip.
 */
export function validateClientKeys(v: ClientOAuthConfigValues): Record<string, string> {
  const errors: Record<string, string> = {}
  const jwks = v.jwks?.trim() ?? ''
  const jwksUri = v.jwksUri?.trim() ?? ''

  // RFC 7591 §2 — with both set, which source verifies an assertion depends on
  // lookup order rather than on intent.
  if (jwks && jwksUri) {
    errors.jwks = 'Provide either an inline JWK Set or a JWKS URL, not both'
    return errors
  }

  if (v.tokenEndpointAuthMethod === 'private_key_jwt' && !jwks && !jwksUri) {
    errors.jwks =
      'private_key_jwt verifies the client assertion against the client\'s public keys — add a JWK Set or a JWKS URL'
    return errors
  }

  if (jwks) {
    const reason = describeInvalidJwks(jwks)
    if (reason) errors.jwks = reason
  }

  if (jwksUri) {
    const reason = describeInvalidJwksUri(jwksUri)
    if (reason) errors.jwksUri = reason
  }

  return errors
}

function describeInvalidJwks(raw: string): string | null {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return 'JWK Set must be valid JSON'
  }
  if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
    return 'JWK Set must be a JSON object with a "keys" array'
  }
  const keys = (parsed as { keys?: unknown }).keys
  if (!Array.isArray(keys) || keys.length === 0) {
    return 'JWK Set must contain a non-empty "keys" array'
  }
  for (const [index, key] of keys.entries()) {
    if (typeof key !== 'object' || key === null || Array.isArray(key)) {
      return `keys[${index}] must be a JWK object`
    }
    const jwk = key as Record<string, unknown>
    if (typeof jwk.kty !== 'string' || !jwk.kty.trim()) {
      return `keys[${index}] must declare a "kty"`
    }
    const priv = JWK_PRIVATE_COMPONENTS.find((component) => component in jwk)
    if (priv) {
      return `keys[${index}] contains the private key component "${priv}" — publish only the public JWK`
    }
  }
  return null
}

function describeInvalidJwksUri(raw: string): string | null {
  let url: URL
  try {
    url = new URL(raw)
  } catch {
    return 'JWKS URL must be an absolute URL'
  }
  // The keys it serves decide whether an assertion is accepted, so the fetch must
  // be authenticated and tamper-proof.
  if (url.protocol !== 'https:') {
    return 'JWKS URL must use https'
  }
  if (url.hash) {
    return 'JWKS URL must not contain a fragment'
  }
  if (raw.length > 2048) {
    return 'JWKS URL must be at most 2048 characters'
  }
  return null
}
