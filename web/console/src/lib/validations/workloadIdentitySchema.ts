/**
 * Workload identity federation validation.
 *
 * Mirrors the backend rules in
 * `internal/federation/validation_workload_identity_federation.go` so the operator
 * sees the same verdict here that the server would give — before a round trip that
 * includes a live OIDC discovery probe of the issuer.
 *
 * The security-relevant rules are `subject_pattern` breadth and the
 * `attribute_mapping` destination names; both are explained where they are defined.
 */

import * as yup from "yup"

/** Backend column limits (see migration 028). */
export const WORKLOAD_LIMITS = {
  name: 100,
  description: 2000,
  issuerUrl: 512,
  audience: 512,
  subjectClaim: 100,
  subjectPattern: 512,
  scope: 128,
} as const

/**
 * Claim names the token issuer owns. A mapping that writes one of these forges the
 * token's own identity: `sub` + `client_id` decide which user, tenant and roles the
 * request resolves to, and `svc` is the service principal the gRPC surface
 * authorizes on. The server drops them at issuance and refuses them on write; this
 * refuses them before the request so the operator gets a field error.
 *
 * Keep in sync with `internal/platform/jwt/reserved_claims.go`.
 */
export const RESERVED_CLAIM_NAMES: readonly string[] = [
  "iss", "sub", "aud", "exp", "nbf", "iat", "jti",
  "azp", "nonce", "acr", "amr", "auth_time",
  "at_hash", "c_hash", "s_hash",
  "scope", "scp", "client_id", "sub_type",
  "tenant_id", "permissions", "roles", "sid",
  "cnf",
  "svc", "provider_id", "token_type", "act", "token_use",
]

export const isReservedClaimName = (name: string): boolean =>
  RESERVED_CLAIM_NAMES.includes(name.trim().toLowerCase())

/** A mapped destination claim name must be a plain lowercase identifier. */
const CLAIM_NAME_PATTERN = /^[a-z][a-z0-9_]{0,63}$/

/** Matches the backend's minSubjectPatternLiterals. */
const MIN_SUBJECT_PATTERN_LITERALS = 6

/** Matches the backend's maxAttributeMappingEntries. */
export const MAX_ATTRIBUTE_MAPPING_ENTRIES = 16

/**
 * Validates an issuer URL: absolute https with a host, no trailing slash.
 *
 * Deliberately NOT reusing `isHttpsUrl` from ./regex — that helper permits
 * `http://localhost`, which this backend rule rejects outright, so reusing it would
 * accept a value the server refuses. The trailing-slash rule matches a DB CHECK:
 * OIDC discovery compares the issuer byte-for-byte, so a stored trailing slash can
 * only ever produce a federation that never matches.
 */
export function validateIssuerUrl(raw: string): string | null {
  const value = raw.trim()
  if (!value) return "Issuer URL is required"
  if (value.length > WORKLOAD_LIMITS.issuerUrl) {
    return `Issuer URL must not exceed ${WORKLOAD_LIMITS.issuerUrl} characters`
  }
  let url: URL
  try {
    url = new URL(value)
  } catch {
    return "Issuer URL must be an absolute URL, e.g. https://token.actions.githubusercontent.com"
  }
  if (url.protocol !== "https:") {
    return "Issuer URL must use https"
  }
  if (!url.host) return "Issuer URL must include a host"
  if (value.endsWith("/")) {
    return "Issuer URL must not end with a slash — OIDC discovery compares it exactly"
  }
  return null
}

/**
 * Validates subject-pattern breadth.
 *
 * `subject_pattern` is the ONLY thing that distinguishes one workload from another on
 * a shared issuer. Public issuers like token.actions.githubusercontent.com will issue
 * a token to anyone, and the audience is chosen by the requesting workflow — so an
 * unanchored pattern lets any workload on that issuer exchange its token for this
 * tenant's access token, with no client credentials.
 */
export function validateSubjectPattern(raw: string): string | null {
  const pattern = raw.trim()
  if (!pattern) return "Subject pattern is required"
  if (pattern.length > WORKLOAD_LIMITS.subjectPattern) {
    return `Subject pattern must not exceed ${WORKLOAD_LIMITS.subjectPattern} characters`
  }
  // An exact pattern is always safe, however short.
  if (!/[*?]/.test(pattern)) return null

  if (pattern.startsWith("*") || pattern.startsWith("?")) {
    return 'Subject pattern must not start with a wildcard — it would match every workload from this issuer. Anchor it, e.g. "repo:my-org/my-repo:*"'
  }
  const literals = pattern.replace(/[*?]/g, "").length
  if (literals < MIN_SUBJECT_PATTERN_LITERALS) {
    return "Subject pattern is too broad to identify a workload — include the organisation or namespace before the wildcard"
  }
  return null
}

/**
 * Validates the external-claim → token-claim mapping.
 *
 * The VALUES are destination claim names in the issued token, which is why a
 * reserved value is refused rather than silently ignored.
 */
export function validateAttributeMapping(
  mapping: Record<string, string>,
): string | null {
  const entries = Object.entries(mapping)
  if (entries.length === 0) return null
  if (entries.length > MAX_ATTRIBUTE_MAPPING_ENTRIES) {
    return `Attribute mapping must not contain more than ${MAX_ATTRIBUTE_MAPPING_ENTRIES} entries`
  }
  for (const [externalClaim, internalClaim] of entries) {
    const external = externalClaim.trim()
    const internal = internalClaim.trim()
    if (!external) return "Attribute mapping keys must not be empty"
    if (external.length > 128) {
      return "Each attribute mapping key must not exceed 128 characters"
    }
    if (!internal) {
      return `"${external}" has no target claim name — the value is the claim to write`
    }
    if (isReservedClaimName(internal)) {
      return `"${internal}" is set by the token issuer and cannot be overridden — overriding it would forge the token's identity`
    }
    if (!CLAIM_NAME_PATTERN.test(internal)) {
      return `"${internal}" is not a valid claim name — use lowercase letters, digits and underscores, starting with a letter`
    }
  }
  return null
}

/** Splits the comma/space separated scope input into a normalized list. */
export function parseAllowedScopes(raw?: string | null): string[] {
  if (!raw) return []
  return raw
    .split(/[\s,]+/)
    .map((scope) => scope.trim())
    .filter(Boolean)
}

export function validateAllowedScopes(scopes: string[]): string | null {
  for (const scope of scopes) {
    if (scope.length > WORKLOAD_LIMITS.scope) {
      return `Each scope must not exceed ${WORKLOAD_LIMITS.scope} characters`
    }
  }
  return null
}

/**
 * The react-hook-form schema. `attribute_mapping` is deliberately absent: it is
 * edited with the shared metadata editor (structured key/value rows), so an
 * unparseable state is impossible by construction and its rules run in
 * validateAttributeMapping at submit time.
 */
export const workloadIdentitySchema = yup.object({
  // Required only: the value comes from a select of real clients, and yup's .uuid()
  // enforces the RFC 4122 version/variant nibbles, which would reject legitimate
  // UUIDs the server accepts (the backend uses a plain is.UUID check).
  client_uuid: yup.string().required("Client is required"),
  name: yup
    .string()
    .required("Name is required")
    .max(WORKLOAD_LIMITS.name, `Name must not exceed ${WORKLOAD_LIMITS.name} characters`),
  description: yup
    .string()
    .default("")
    .max(
      WORKLOAD_LIMITS.description,
      `Description must not exceed ${WORKLOAD_LIMITS.description} characters`,
    ),
  issuer_url: yup
    .string()
    .required("Issuer URL is required")
    .test("issuer-url", ({ value }) => validateIssuerUrl(String(value ?? "")) ?? "", (value) =>
      validateIssuerUrl(String(value ?? "")) === null,
    ),
  audience: yup
    .string()
    .required("Audience is required")
    .max(
      WORKLOAD_LIMITS.audience,
      `Audience must not exceed ${WORKLOAD_LIMITS.audience} characters`,
    ),
  subject_claim: yup
    .string()
    .default("sub")
    .max(
      WORKLOAD_LIMITS.subjectClaim,
      `Subject claim must not exceed ${WORKLOAD_LIMITS.subjectClaim} characters`,
    ),
  subject_pattern: yup
    .string()
    .required("Subject pattern is required")
    .test(
      "subject-pattern",
      ({ value }) => validateSubjectPattern(String(value ?? "")) ?? "",
      (value) => validateSubjectPattern(String(value ?? "")) === null,
    ),
  allowed_scopes: yup
    .string()
    .default("")
    .test(
      "allowed-scopes",
      ({ value }) => validateAllowedScopes(parseAllowedScopes(String(value ?? ""))) ?? "",
      (value) => validateAllowedScopes(parseAllowedScopes(String(value ?? ""))) === null,
    ),
  is_active: yup.boolean().default(true),
})

export type WorkloadIdentityFormData = yup.InferType<typeof workloadIdentitySchema>
