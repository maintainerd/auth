/**
 * Provider Configuration Schemas
 *
 * Declarative, per-provider field maps. Shared connection fields live as
 * top-level identity provider columns on the backend (`issuer`,
 * `provider_client_id`, `provider_client_secret`, `allow_jit_provisioning`, and `email_domains`). The schemas
 * below only describe provider-specific config JSON keys the backend actually
 * reads: OAuth scopes and OAuth2 endpoints (optional overrides for OIDC
 * providers with discovery; required for OAuth2-only providers without it).
 * The backend rejects unknown config keys, so no other keys are collected.
 *
 * Note: this is distinct from the Clients resource, which models the apps that
 * authenticate *against* Maintainerd and whose credentials Maintainerd issues.
 */

import type { ProviderOption } from "@/services/api/identity-providers/types"

export type ProviderFieldType = "text" | "password" | "url" | "list" | "switch" | "scopes" | "textarea"

/** Backend `provider_type` discriminator. */
export type ProviderKind = "system" | "social" | "enterprise" | "saml"
export type ProviderConnectionFieldKey =
  | "issuer"
  | "provider_client_id"
  | "provider_client_secret"
  | "allow_jit_provisioning"
  | "email_domains"

export interface ProviderConfigField {
  /** Key written into the provider `config` JSON. */
  key: string
  label: string
  type: ProviderFieldType
  required?: boolean
  placeholder?: string
  description?: string
  /** Seed value used on create (never overrides an existing/edited value). */
  default?: string
  /** For `scopes` fields: selectable suggestions shown in the picker. */
  suggestions?: string[]
}

export interface ProviderConfigGroup {
  /** Subsection heading, e.g. "Connection", "User Pool", or "Directory". */
  title: string
  description?: string
  fields: ProviderConfigField[]
}

export interface ProviderConfigSchema {
  /** Backend `provider_type` value this provider is created with. */
  kind: ProviderKind
  /** Explanatory line shown under the section header. */
  summary: string
  /** Optional external setup documentation. */
  docsLabel?: string
  docsUrl?: string
  /** Grouped, well-known fields. Empty for providers with no provider-level config. */
  groups: ProviderConfigGroup[]
}

export interface ProviderConnectionField {
  key: ProviderConnectionFieldKey
  label: string
  type: ProviderFieldType
  required?: boolean
  placeholder?: string
  description?: string
}

export interface ProviderConnectionSchema {
  summary: string
  fields: ProviderConnectionField[]
}

/** Human-friendly provider names, shared across forms, tables, and detail pages. */
export const PROVIDER_LABELS: Record<string, string> = {
  maintainerd: "Maintainerd",
  saml: "SAML 2.0",
  cognito: "AWS Cognito",
  auth0: "Auth0",
  microsoft: "Microsoft Entra ID",
  google: "Google",
  github: "GitHub",
  gitlab: "GitLab",
  facebook: "Facebook",
  linkedin: "LinkedIn",
  twitter: "X (Twitter)",
}

/**
 * Display order for the provider dropdown — social providers first, then the
 * enterprise identity providers, then SAML 2.0 last.
 */
export const PROVIDER_ORDER: ProviderOption[] = [
  // Social sign-in
  "google",
  "github",
  "gitlab",
  "facebook",
  "linkedin",
  "twitter",
  // Enterprise identity providers (OIDC federation)
  "cognito",
  "auth0",
  "microsoft",
  "maintainerd",
  // SAML 2.0
  "saml",
]

/**
 * Providers that have no OIDC discovery document, so the broker needs explicit
 * OAuth2 endpoints. Mirrors the backend's OAUTH2_ONLY provider class. For these
 * providers the config-level authorization/token/userinfo endpoints are REQUIRED
 * and the top-level issuer connection field is NOT required.
 */
const OAUTH2_ONLY_PROVIDERS: ReadonlySet<ProviderOption> = new Set([
  "github",
  "facebook",
  "twitter",
])

/** True when a provider is OAuth2-only (no OIDC issuer/discovery). */
export function isOAuth2OnlyProvider(provider: string): boolean {
  return OAUTH2_ONLY_PROVIDERS.has(provider as ProviderOption)
}

/**
 * Per-provider official host allow-list — kept in lock-step with the backend's
 * providerAllowedHosts (internal/idp/validation_provider.go). For fixed-domain
 * providers the ISSUER (OIDC) or the ENDPOINTS (OAuth2-only) must live on these
 * hosts, so a known provider can't be pointed at an attacker-controlled host.
 * Providers deliberately ABSENT are host-unrestricted because they use custom /
 * self-managed / arbitrary domains (auth0, gitlab, external maintainerd, saml).
 * Cognito uses a regex because its issuer host is regional.
 */
const PROVIDER_ALLOWED_HOSTS: Partial<Record<ProviderOption, string[]>> = {
  google: ["accounts.google.com"],
  linkedin: ["www.linkedin.com"],
  microsoft: ["login.microsoftonline.com", "login.microsoftonline.us", "login.partner.microsoftonline.cn"],
  github: ["github.com", "api.github.com"],
  facebook: ["facebook.com", "www.facebook.com", "graph.facebook.com"],
  twitter: ["x.com", "twitter.com", "api.x.com", "api.twitter.com"],
}

const COGNITO_ISSUER_HOST_REGEX = /^(issuer-)?cognito-idp\.[a-z0-9-]+\.amazonaws\.com(\.cn)?$/

/**
 * True when `url`'s host is permitted for `provider`. Exact host match (never
 * suffix-contains, which is bypassable by evilgithub.com / github.com.evil.com).
 * Providers with no allow-list are unrestricted. An empty or unparseable value
 * returns true — presence and https/format are enforced separately, so this only
 * adds the host constraint. Mirrors the backend requireAllowedHost.
 */
export function isAllowedProviderHost(provider: string, url: string | null | undefined): boolean {
  const v = (url ?? "").trim()
  if (v === "") return true
  let host: string
  try {
    host = new URL(v).hostname.toLowerCase()
  } catch {
    return true
  }
  if (provider === "cognito") return COGNITO_ISSUER_HOST_REGEX.test(host)
  const allowed = PROVIDER_ALLOWED_HOSTS[provider as ProviderOption]
  if (!allowed) return true
  return allowed.includes(host)
}

const DEFAULT_ISSUERS: Partial<Record<ProviderOption, string>> = {
  maintainerd: "https://auth.example-org.com",
  cognito: "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_aB12cD34",
  auth0: "https://your-tenant.us.auth0.com/",
  microsoft: "https://login.microsoftonline.com/<tenant-id>/v2.0",
  google: "https://accounts.google.com",
  github: "https://github.com",
  gitlab: "https://gitlab.com",
  facebook: "https://www.facebook.com",
  // LinkedIn's OIDC issuer is /oauth — the bare host 404s on discovery.
  linkedin: "https://www.linkedin.com/oauth",
  twitter: "https://x.com",
}

/**
 * Providers whose issuer is a FIXED, ready-to-use constant (not a per-install
 * value like a Cognito pool URL or an Auth0 tenant). Their Issuer URL is
 * pre-filled on create so the operator doesn't retype a well-known value —
 * still fully editable (e.g. a self-hosted GitLab, or a single Entra tenant).
 * Cognito / Auth0 / Maintainerd are intentionally excluded: their issuer is
 * unique per installation and must be entered by hand.
 */
const PREFILL_ISSUER_PROVIDERS: ReadonlySet<ProviderOption> = new Set([
  "google",
  "gitlab",
  "linkedin",
  // Microsoft is intentionally NOT here: its issuer is tenant-specific
  // (https://login.microsoftonline.com/{tenant-id}/v2.0) and Azure AD tokens
  // carry a tenant-specific `iss`, so a fixed default would fail validation.
])

/** Issuer URL to pre-fill on create, or "" when it must be entered by hand. */
export function getPrefillIssuer(provider: string): string {
  if (!PREFILL_ISSUER_PROVIDERS.has(provider as ProviderOption)) return ""
  return DEFAULT_ISSUERS[provider as ProviderOption] ?? ""
}

const CLIENT_ID_LABELS: Partial<Record<ProviderOption, string>> = {
  cognito: "App Client ID",
  auth0: "Application Client ID",
}

const CLIENT_SECRET_LABELS: Partial<Record<ProviderOption, string>> = {
  cognito: "App Client Secret",
}

/** Provider-specific labels for the backend-owned connection columns. */
export function getProviderConnectionSchema(provider: string): ProviderConnectionSchema | undefined {
  const option = provider as ProviderOption
  const schema = getProviderConfigSchema(option)
  // System (built-in) and SAML providers have no upstream OIDC connection: SAML
  // uses its own config fields (sso_url / entity_id / certificate) instead of an
  // issuer + client id/secret, so it must not render the OIDC connection card.
  if (!schema || schema.kind === "system" || schema.kind === "saml") return undefined

  const fields: ProviderConnectionField[] = []

  // The issuer only exists for OIDC providers: the broker discovers the
  // authorization/token/userinfo endpoints from it. OAuth2-only providers
  // (github/facebook/twitter) publish no discovery document and have no issuer —
  // the broker uses the explicit OAuth2 endpoints (collected in the config
  // section below) instead. So the issuer field is OMITTED for them rather than
  // shown as an inert, confusing near-duplicate of the authorization endpoint.
  if (!isOAuth2OnlyProvider(option)) {
    fields.push({
      key: "issuer",
      label: option === "cognito" ? "User Pool Issuer URL" : "Issuer URL",
      type: "url",
      required: true,
      placeholder: DEFAULT_ISSUERS[option] ?? "https://provider.example.com",
      description:
        "Issuer or authority URL used by the broker to discover metadata and validate tokens.",
    })
  }

  fields.push(
    {
      key: "provider_client_id",
      label: CLIENT_ID_LABELS[option] ?? "OAuth Client ID",
      type: "text",
      required: true,
      placeholder: option === "cognito" ? "your-app-client-id" : "your-oauth-client-id",
      description: "OAuth application identifier issued by the external provider.",
    },
    {
      key: "provider_client_secret",
      label: CLIENT_SECRET_LABELS[option] ?? "OAuth Client Secret",
      type: "password",
      required: true,
      placeholder: "••••••••",
      description:
        "Stored encrypted by the backend. Leave blank when editing to keep the existing secret.",
    },
    {
      key: "allow_jit_provisioning",
      label: "Just-in-Time Provisioning",
      type: "switch",
      description: "Create a local user automatically on first successful broker sign-in.",
    },
  )

  // Email domains drive home-realm discovery — routing a corporate email domain
  // to the org's identity provider. That is an ENTERPRISE-federation concept:
  // social sign-in (google/github/gitlab/facebook/linkedin/twitter) is a
  // user-chosen button, not something an org owns a domain for, so the field is
  // shown for enterprise providers only. (Google Workspace domain routing, if
  // ever needed, belongs on a dedicated enterprise provider, not the social one.)
  if (schema.kind === "enterprise") {
    fields.push({
      key: "email_domains",
      label: "Email Domains",
      type: "list",
      placeholder: "example.com, corp.example.com",
      description: "Domains routed to this provider during home-realm discovery.",
    })
  }

  return {
    summary:
      "Connection details are stored as dedicated provider fields. Provider-specific options below are saved into config JSON.",
    fields,
  }
}

export function getPromotedProviderFieldKeys(): string[] {
  return [
    "issuer",
    "provider_client_id",
    "provider_client_secret",
    "allow_jit_provisioning",
    "email_domains",
    // Legacy config keys from before the backend promoted provider credentials.
    "client_id",
    "client_secret",
  ]
}

// Scopes an OIDC provider commonly supports, offered as picker suggestions in
// addition to whatever the provider's default set is.
const COMMON_OIDC_SCOPES = ["openid", "profile", "email", "offline_access", "address", "phone", "groups"]

function parseScopeString(value: string): string[] {
  return value.split(/[\n,]/).map((s) => s.trim()).filter(Boolean)
}

function scopesGroup(defaultScopes = "openid, profile, email"): ProviderConfigGroup {
  const base = parseScopeString(defaultScopes)
  // Suggest the provider's own defaults first, then the common OIDC scopes when
  // this is an OIDC provider (its default set includes `openid`).
  const isOidc = base.includes("openid")
  const suggestions = Array.from(new Set([...base, ...(isOidc ? COMMON_OIDC_SCOPES : [])]))
  return {
    title: "OAuth",
    description: "Scopes requested during sign-in.",
    fields: [
      {
        key: "scopes",
        label: "Scopes",
        type: "scopes",
        // Pre-filled on create so operators don't have to guess; editable, and
        // never overrides an existing provider's saved scopes on edit.
        default: defaultScopes,
        suggestions,
        placeholder: defaultScopes,
        description: "Comma-separated scopes requested during sign-in. Use Browse to pick common scopes.",
      },
    ],
  }
}

interface EndpointDefaults {
  authorization_endpoint: string
  token_endpoint: string
  userinfo_endpoint: string
}

/**
 * REQUIRED OAuth2 endpoints for OAuth2-only providers (github/facebook/twitter)
 * which publish no discovery document. Pre-filled with each provider's known
 * endpoints (like scopes) so operators rarely need to change them.
 */
function requiredEndpointsGroup(defaults: EndpointDefaults): ProviderConfigGroup {
  return {
    title: "Endpoints",
    description:
      "OAuth2 endpoints used by the broker. Required because this provider has no OIDC discovery document.",
    fields: [
      {
        key: "authorization_endpoint",
        label: "Authorization Endpoint",
        type: "url",
        required: true,
        default: defaults.authorization_endpoint,
        placeholder: defaults.authorization_endpoint,
      },
      {
        key: "token_endpoint",
        label: "Token Endpoint",
        type: "url",
        required: true,
        default: defaults.token_endpoint,
        placeholder: defaults.token_endpoint,
      },
      {
        key: "userinfo_endpoint",
        label: "UserInfo Endpoint",
        type: "url",
        required: true,
        default: defaults.userinfo_endpoint,
        placeholder: defaults.userinfo_endpoint,
      },
    ],
  }
}

/**
 * OIDC providers: scopes only. Endpoints come from OIDC discovery — the backend
 * REJECTS endpoint overrides for OIDC providers (they'd let the runtime be
 * pointed at an attacker host), so we must not collect them here.
 */
function commonExternalGroups(defaultScopes?: string): ProviderConfigGroup[] {
  return [scopesGroup(defaultScopes)]
}

/** OAuth2-only providers: scopes + REQUIRED, pre-filled OAuth2 endpoints. */
function oauth2OnlyGroups(defaultScopes: string, endpoints: EndpointDefaults): ProviderConfigGroup[] {
  return [scopesGroup(defaultScopes), requiredEndpointsGroup(endpoints)]
}

export const PROVIDER_CONFIG_SCHEMAS: Record<ProviderOption, ProviderConfigSchema> = {
  maintainerd: {
    // A user-created Maintainerd provider federates to ANOTHER organization's
    // self-hosted Maintainerd instance over standard OIDC (enterprise). The
    // built-in local provider that ships with this system is a separate,
    // undeletable system record and is shown read-only — it needs no config.
    kind: "enterprise",
    summary:
      "Federate to another organization's self-hosted Maintainerd instance over OpenID Connect. Provide that instance's issuer URL and an OAuth client registered there. (The built-in Maintainerd provider is managed separately and needs no configuration.)",
    groups: commonExternalGroups("openid, profile, email"),
  },

  cognito: {
    kind: "enterprise",
    summary: "Connect an Amazon Cognito User Pool over OpenID Connect. Provide the pool's issuer URL and OAuth credentials; the broker discovers the rest from its metadata.",
    docsLabel: "AWS Cognito user pool guide",
    docsUrl:
      "https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-identity-pools.html",
    groups: commonExternalGroups(),
  },

  auth0: {
    kind: "enterprise",
    summary: "Connect an Auth0 tenant over OpenID Connect. Provide the tenant issuer URL and application credentials.",
    docsLabel: "Auth0 application setup",
    docsUrl: "https://auth0.com/docs/get-started/applications",
    groups: commonExternalGroups(),
  },

  microsoft: {
    kind: "enterprise",
    summary: "Sign in with Microsoft Entra ID (Azure AD) over OpenID Connect. Provide the tenant issuer URL and application credentials.",
    docsLabel: "Microsoft Entra ID platform",
    docsUrl: "https://learn.microsoft.com/en-us/entra/identity-platform/",
    groups: commonExternalGroups(),
  },

  google: {
    kind: "social",
    summary: "Sign in with Google. Maintainerd uses Google's published OIDC metadata with your OAuth credentials.",
    docsLabel: "Google OAuth 2.0",
    docsUrl: "https://developers.google.com/identity/protocols/oauth2",
    groups: commonExternalGroups(),
  },

  github: {
    kind: "social",
    summary: "Sign in with GitHub using your OAuth App credentials. GitHub has no OIDC discovery, so the OAuth2 endpoints below are required (pre-filled with github.com defaults).",
    docsLabel: "GitHub OAuth Apps",
    docsUrl:
      "https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app",
    groups: oauth2OnlyGroups("read:user, user:email", {
      authorization_endpoint: "https://github.com/login/oauth/authorize",
      token_endpoint: "https://github.com/login/oauth/access_token",
      userinfo_endpoint: "https://api.github.com/user",
    }),
  },

  gitlab: {
    kind: "social",
    summary: "Sign in with GitLab using your OAuth application credentials over OpenID Connect.",
    docsLabel: "GitLab OAuth provider",
    docsUrl: "https://docs.gitlab.com/ee/integration/oauth_provider.html",
    groups: commonExternalGroups("openid, profile, email"),
  },

  facebook: {
    kind: "social",
    summary: "Sign in with Facebook Login using your app credentials. Facebook has no OIDC discovery, so the OAuth2 endpoints below are required (pre-filled with Graph API defaults).",
    docsLabel: "Facebook Login setup",
    docsUrl: "https://developers.facebook.com/docs/facebook-login/",
    groups: oauth2OnlyGroups("public_profile, email", {
      authorization_endpoint: "https://www.facebook.com/v25.0/dialog/oauth",
      token_endpoint: "https://graph.facebook.com/v25.0/oauth/access_token",
      userinfo_endpoint: "https://graph.facebook.com/v25.0/me?fields=id,name,email",
    }),
  },

  linkedin: {
    kind: "social",
    summary: "Sign in with LinkedIn using your OAuth application credentials.",
    docsLabel: "Sign in with LinkedIn",
    docsUrl:
      "https://learn.microsoft.com/en-us/linkedin/consumer/integrations/self-serve/sign-in-with-linkedin",
    groups: commonExternalGroups("openid, profile, email"),
  },

  twitter: {
    kind: "social",
    summary: "Sign in with X (Twitter) using your OAuth 2.0 application credentials. X has no OIDC discovery, so the OAuth2 endpoints below are required (pre-filled with X API defaults).",
    docsLabel: "X OAuth 2.0",
    docsUrl: "https://developer.twitter.com/en/docs/authentication/oauth-2-0",
    groups: oauth2OnlyGroups("tweet.read, users.read, offline.access, users.email", {
      authorization_endpoint: "https://x.com/i/oauth2/authorize",
      token_endpoint: "https://api.x.com/2/oauth2/token",
      userinfo_endpoint: "https://api.x.com/2/users/me?user.fields=confirmed_email",
    }),
  },

  saml: {
    kind: "saml",
    // A SAML connection is two-way, and the outbound half was invisible: after
    // saving, the Connection tab publishes the SP values the IdP needs back.
    summary: "Connect a SAML 2.0 identity provider. Maintainerd acts as the service provider and validates assertions signed by your enterprise IdP. Save the provider first — its Connection tab then publishes the entity ID, ACS URL and metadata URL your IdP needs in return.",
    groups: [
      {
        title: "Identity Provider",
        description: "Endpoints and credentials provided by your SAML IdP.",
        fields: [
          {
            key: "sso_url",
            label: "SSO URL",
            type: "url",
            required: true,
            placeholder: "https://idp.example.com/saml2/sso",
            description: "Single sign-on URL where Maintainerd sends authentication requests.",
          },
          {
            key: "entity_id",
            label: "IdP Entity ID",
            type: "text",
            required: true,
            placeholder: "https://idp.example.com/saml2/metadata",
            description: "Unique identifier for the identity provider (from its metadata).",
          },
          {
            key: "certificate",
            label: "X.509 Certificate",
            type: "textarea",
            required: true,
            placeholder: "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
            description: "Public certificate (PEM) used to verify SAML assertion signatures.",
          },
        ],
      },
      {
        title: "Options",
        description: "Attribute mapping and assertion handling.",
        fields: [
          {
            key: "name_id_format",
            label: "NameID Format",
            type: "text",
            placeholder: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
            description: "NameID format requested in authentication requests. Optional.",
          },
          {
            key: "email_attribute",
            label: "Email Attribute",
            type: "text",
            placeholder: "email",
            description: "SAML attribute name that carries the user's email. Defaults to NameID.",
          },
        ],
      },
    ],
  },
}

/** Select options for the provider dropdown, in display order. */
export const PROVIDER_SELECT_OPTIONS = PROVIDER_ORDER.map((value) => ({
  value,
  label: PROVIDER_LABELS[value] ?? value,
}))

/** Resolve the config schema for a provider, if one is defined. */
export function getProviderConfigSchema(provider: string): ProviderConfigSchema | undefined {
  return PROVIDER_CONFIG_SCHEMAS[provider as ProviderOption]
}

/** Flat list of every well-known config field for a provider. */
export function getProviderFields(provider: string): ProviderConfigField[] {
  const schema = getProviderConfigSchema(provider)
  if (!schema) return []
  return schema.groups.flatMap((group) => group.fields)
}

/** Flat list of every well-known config key for a provider. */
export function getProviderFieldKeys(provider: string): string[] {
  return getProviderFields(provider).map((field) => field.key)
}

/** The backend `provider_type` a given provider should be created with. */
export function getProviderKind(provider: string): ProviderKind {
  return getProviderConfigSchema(provider)?.kind ?? "social"
}
