/**
 * Identity Provider Form Validation Schema
 * Yup validation schema for identity provider-related forms.
 *
 * IMPORTANT: these rules are kept in lock-step with the backend validation in
 * internal/idp/validation_provider.go — every field that is required (or format-
 * checked) on the backend is required (and format-checked) here, and vice versa,
 * so the two never disagree about what a valid provider is.
 */

import * as yup from 'yup'
import { getProviderKind, isOAuth2OnlyProvider, isAllowedProviderHost } from '@/components/provider-config/providerConfigSchemas'
import { isHttpsUrl } from './regex'

// A provider needs upstream OAuth credentials (client id/secret) only when its
// backend provider_type is social or enterprise. This mirrors the backend's
// isExternalProviderType() and is derived from the SAME source of truth
// (getProviderKind) that the form uses to send provider_type — so SAML
// (kind "saml") and the built-in system provider are correctly excluded.
function isExternalProvider(provider: unknown): boolean {
  if (typeof provider !== 'string') return false
  const kind = getProviderKind(provider)
  return kind === 'social' || kind === 'enterprise'
}

function activeExternalProvider(provider: unknown, status: unknown): boolean {
  return isExternalProvider(provider) && status === 'active'
}

// OAuth2-only providers (github/facebook/twitter) have no OIDC discovery, so
// they carry no issuer — the OAuth2 endpoints live in config instead. Mirrors
// the backend's OAUTH2_ONLY provider class.
function isOAuth2Only(provider: unknown): boolean {
  return typeof provider === 'string' && isOAuth2OnlyProvider(provider)
}

// The issuer is required only for active OIDC (discovery) external providers —
// enterprise/social providers that are NOT OAuth2-only. OAuth2-only providers
// are excluded even though they are external + active.
function activeDiscoveryProvider(provider: unknown, status: unknown): boolean {
  return activeExternalProvider(provider, status) && !isOAuth2Only(provider)
}

function activeTokenFederation(allowTokenFederation: unknown, status: unknown): boolean {
  return allowTokenFederation === true && status === 'active'
}

// Matches the backend is.Domain check applied per email domain entry.
const DOMAIN_REGEX = /^(?=.{1,253}$)([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$/i

// Identity Provider Form Schema
export const identityProviderSchema = yup.object({
  name: yup
    .string()
    .required('Provider name is required')
    // Length 3–50 mirrors backend validation.Length(3, 50).
    .min(3, 'Provider name must be at least 3 characters')
    .max(50, 'Provider name must not exceed 50 characters')
    // Mirrors backend idpNamePattern: lowercase letters, digits and hyphens,
    // and cannot start or end with a hyphen.
    .matches(
      /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/,
      'Provider name must contain only lowercase letters, numbers, and hyphens, and cannot start or end with a hyphen'
    ),
  displayName: yup
    .string()
    .required('Display name is required')
    // Length 2–100 mirrors backend validation.Length(2, 100).
    .min(2, 'Display name must be at least 2 characters')
    .max(100, 'Display name must not exceed 100 characters'),
  provider: yup
    .string()
    .oneOf(
      [
        'maintainerd',
        'cognito',
        'auth0',
        'google',
        'facebook',
        'github',
        'gitlab',
        'microsoft',
        'linkedin',
        'twitter',
        'saml',
      ],
      'Invalid provider'
    )
    .required('Provider is required'),
  status: yup
    .string()
    .oneOf(['active', 'inactive'], 'Invalid status')
    .required('Status is required'),
  issuer: yup
    .string()
    .nullable()
    // Treat an empty string as "absent" so the https check is skipped when the
    // field is optional and left blank (matches backend "URL only when set").
    .transform((value) => (value === '' ? undefined : value))
    // HTTPS enforcement (IAM best practice): the issuer must be a valid https://
    // URL — except http://localhost / 127.0.0.1 for local dev. Mirrors the
    // backend's isHttpsUrl rule; rejects http://provider.com.
    .test(
      'issuer-https',
      'Issuer must be a valid https:// URL',
      (value) => value == null || isHttpsUrl(value)
    )
    // Per-provider host binding (mirrors backend requireAllowedHost): for a
    // fixed-domain provider (google, microsoft, cognito, linkedin) the issuer
    // must live on that provider's official host, so a known provider can't be
    // pointed at a fake issuer. Variable-domain providers are unrestricted.
    .test(
      'issuer-host',
      'Issuer host is not permitted for this provider',
      function (value) {
        return value == null || isAllowedProviderHost(this.parent.provider, value)
      }
    )
    .when(['provider', 'status', 'allowTokenFederation'], ([provider, status, allowTokenFederation], schema) =>
      activeDiscoveryProvider(provider, status) || activeTokenFederation(allowTokenFederation, status)
        ? schema.required('Issuer URL is required for active OIDC or token-federation providers')
        : schema.notRequired()
    ),
  clientId: yup
    .string()
    .nullable()
    .when(['provider', 'status'], ([provider, status], schema) =>
      activeExternalProvider(provider, status)
        ? schema.required('Client ID is required for active external providers')
        : schema.notRequired()
    ),
  clientSecret: yup.string().nullable(),
  allowJITProvisioning: yup.boolean().default(false),
  allowRegistration: yup.boolean().default(true),
  allowTokenFederation: yup.boolean().default(false),
  allowedAudiences: yup
    .string()
    .nullable()
    .when(['allowTokenFederation', 'status'], ([allowTokenFederation, status], schema) =>
      activeTokenFederation(allowTokenFederation, status)
        ? schema.required('At least one allowed audience is required when token federation is enabled')
        : schema.notRequired()
    ),
  emailDomains: yup
    .string()
    .nullable()
    // Mirrors backend validation.Each(is.Domain): every entry (comma/space/
    // newline-separated) must be a valid domain.
    .test(
      'valid-email-domains',
      'Each email domain must be a valid domain (e.g. example.com)',
      (value) => {
        if (!value) return true
        const domains = value.split(/[\s,\n]+/).map((d) => d.trim()).filter(Boolean)
        return domains.every((d) => DOMAIN_REGEX.test(d))
      }
    ),
})

export type IdentityProviderFormData = yup.InferType<typeof identityProviderSchema>
