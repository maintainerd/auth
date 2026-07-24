export const EMAIL_REGEX = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

export const SLUG_REGEX = /^[a-z0-9:-]+$/

export function sanitizeSlug(raw: string): string {
  return raw.replace(/[^a-z0-9:-]/g, '').toLowerCase()
}

// Like sanitizeSlug but without the colon — for names that are plain
// lowercase/digit/hyphen slugs (e.g. identity provider names), matching the
// backend idpNamePattern which does not permit colons.
export function sanitizeName(raw: string): string {
  return raw.replace(/[^a-z0-9-]/g, '').toLowerCase()
}

// A registration-flow name. The name IS the public registration-link selector
// (?registration_flow=<name>), so it must satisfy the backend's
// registrationFlowNamePattern: lowercase alphanumerics separated by single
// hyphens or underscores. Whitespace becomes a hyphen rather than vanishing, so
// typing "partner signup" yields "partner-signup" instead of "partnersignup".
export function sanitizeFlowName(raw: string): string {
  return raw
    .toLowerCase()
    .replace(/\s+/g, '-')
    .replace(/[^a-z0-9-_]/g, '')
}

// An OAuth2 / OIDC scope token. Permits the shapes real providers use — plain
// (`openid`), namespaced (`read:user`), dotted (`tweet.read`), and URL-style
// (`https://www.googleapis.com/auth/userinfo.email`) — while rejecting spaces,
// quotes, and other junk. Tokens are separated by commas in the UI.
export const SCOPE_TOKEN_REGEX = /^[A-Za-z0-9._:/-]+$/

export function isValidScopeToken(token: string): boolean {
  return SCOPE_TOKEN_REGEX.test(token)
}

/**
 * True only when `value` is a well-formed URL whose scheme is https, EXCEPT that
 * plain http is allowed when the host is localhost / 127.0.0.1 (local dev). This
 * is the IAM best-practice rule the backend enforces in lock-step: external
 * identity-provider URLs (issuer, OAuth2/SAML endpoints) must be TLS-protected,
 * so `http://provider.com` is rejected while `https://provider.com` and
 * `http://localhost:8080` are accepted. Empty/malformed input returns false;
 * callers gate on "value present" before requiring a valid URL.
 */
export function isHttpsUrl(value: string): boolean {
  let url: URL
  try {
    url = new URL(value.trim())
  } catch {
    return false
  }
  if (url.protocol === "https:") return true
  if (url.protocol === "http:") {
    return url.hostname === "localhost" || url.hostname === "127.0.0.1"
  }
  return false
}

// A PEM-encoded X.509 certificate block. The backend fully parses the
// certificate; the frontend only shape-checks that a BEGIN/END CERTIFICATE
// block is present so obvious garbage is caught before submit.
const PEM_CERTIFICATE_REGEX = /-----BEGIN CERTIFICATE-----[\s\S]+?-----END CERTIFICATE-----/

export function isPemCertificate(value: string): boolean {
  return PEM_CERTIFICATE_REGEX.test(value.trim())
}

export const PHONE_REGEX = /^\+?[1-9][\d\s\-().]{6,20}$/

export const PASSWORD_SPECIAL_REGEX = /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/

const NON_DIGIT_REGEX = /\D/g

const PHONE_DIGIT_LIMIT = 15

export function isValidEmail(email: string): boolean {
  if (email.length > 254) return false
  return EMAIL_REGEX.test(email)
}

export function isValidPhone(phone: string): boolean {
  const digitsOnly = phone.replace(NON_DIGIT_REGEX, '')
  if (digitsOnly.length < 7 || digitsOnly.length > 15) return false
  return PHONE_REGEX.test(phone)
}

export function formatPhone(value: string): string {
  let digits = value.replace(/\D/g, '').slice(0, PHONE_DIGIT_LIMIT)
  digits = digits.replace(/^0+/, '')
  if (digits.length === 0) return ''

  if (digits.length <= 3) return digits
  if (digits.length <= 6) return `(${digits.slice(0, 3)}) ${digits.slice(3)}`
  if (digits.length <= 10) return `(${digits.slice(0, 3)}) ${digits.slice(3, 6)}-${digits.slice(6, 10)}`

  return `+${digits.slice(0, 1)} (${digits.slice(1, 4)}) ${digits.slice(4, 7)}-${digits.slice(7, PHONE_DIGIT_LIMIT)}`
}

export const DEFAULT_PASSWORD_CONFIG = {
  min_length: 8,
  max_length: 128,
  require_uppercase: true,
  require_lowercase: true,
  require_number: true,
  require_symbol: true,
} as const
