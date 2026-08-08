/**
 * Client Type Capabilities
 *
 * Declarative, per-client-type descriptor that drives the client form. A
 * client's type (SPA / Traditional / Mobile / M2M) determines which OAuth
 * flows, credential model, and application URIs are relevant — mirroring how
 * `provider-config` drives the identity-provider form by provider.
 *
 * This keeps the form free of scattered `if (clientType === ...)` checks: the
 * form reads a single capability object and renders/defaults accordingly. The
 * backend stores these as standard OAuth client settings, so the values here
 * are the well-known OAuth/OIDC defaults for each application type.
 */

import type { ClientType } from "@/services/api/clients/types"
import type { SelectOption } from "@/components/form"

export type PkceMode = "required" | "optional" | "none"

export interface ClientTypeCapability {
  /** Card summary shown under the client-type selector. */
  summary: string
  /** Public clients cannot hold a secret; confidential clients can. */
  isPublic: boolean
  /** Whether the type uses a browser/redirect-based interactive login. */
  interactive: boolean
  /** PKCE posture: forced on for public interactive clients. */
  pkce: PkceMode
  /** Grant types offered for this type, in display order. */
  grantTypeOptions: SelectOption[]
  /** Sensible default grant selection applied when the type is chosen. */
  defaultGrantTypes: string[]
  /** Response types are irrelevant to non-interactive (M2M) clients. */
  showResponseTypes: boolean
  defaultResponseTypes: string[]
  /** Token endpoint auth methods valid for this type. */
  authMethodOptions: SelectOption[]
  defaultAuthMethod: string
  /** Which Application URI sections apply to this type. */
  showRedirectUris: boolean
  showLoginUri: boolean
  showLogoutUrls: boolean
  showAllowedOrigins: boolean
  showCors: boolean
}

const GRANT_AUTH_CODE: SelectOption = { value: "authorization_code", label: "Authorization Code" }
const GRANT_REFRESH: SelectOption = { value: "refresh_token", label: "Refresh Token" }
const GRANT_CLIENT_CREDENTIALS: SelectOption = { value: "client_credentials", label: "Client Credentials" }

const PUBLIC_AUTH_METHODS: SelectOption[] = [{ value: "none", label: "None (public client)" }]

const CONFIDENTIAL_AUTH_METHODS: SelectOption[] = [
  { value: "client_secret_basic", label: "Client Secret Basic" },
  { value: "client_secret_post", label: "Client Secret Post" },
  { value: "private_key_jwt", label: "Private Key JWT" },
  { value: "client_secret_jwt", label: "Client Secret JWT" },
  { value: "tls_client_auth", label: "TLS Client Auth" },
  { value: "self_signed_tls_client_auth", label: "Self-Signed TLS Client Auth" },
]

export const CLIENT_TYPE_CAPABILITIES: Record<ClientType, ClientTypeCapability> = {
  spa: {
    summary:
      "Browser-based single-page app. Public client — no client secret; PKCE is required and tokens are returned via the authorization code flow.",
    isPublic: true,
    interactive: true,
    pkce: "required",
    grantTypeOptions: [GRANT_AUTH_CODE, GRANT_REFRESH],
    defaultGrantTypes: ["authorization_code", "refresh_token"],
    showResponseTypes: true,
    defaultResponseTypes: ["code"],
    authMethodOptions: PUBLIC_AUTH_METHODS,
    defaultAuthMethod: "none",
    showRedirectUris: true,
    showLoginUri: true,
    showLogoutUrls: true,
    showAllowedOrigins: true,
    showCors: true,
  },

  traditional: {
    summary:
      "Server-side web application. Confidential client — holds a client secret and authenticates at the token endpoint.",
    isPublic: false,
    interactive: true,
    pkce: "optional",
    grantTypeOptions: [GRANT_AUTH_CODE, GRANT_REFRESH],
    defaultGrantTypes: ["authorization_code", "refresh_token"],
    showResponseTypes: true,
    defaultResponseTypes: ["code"],
    authMethodOptions: CONFIDENTIAL_AUTH_METHODS,
    defaultAuthMethod: "client_secret_basic",
    showRedirectUris: true,
    showLoginUri: true,
    showLogoutUrls: true,
    showAllowedOrigins: false,
    showCors: false,
  },

  mobile: {
    summary:
      "Native mobile or desktop app. Public client — no client secret; PKCE is required and redirects use the app's custom scheme or app links.",
    isPublic: true,
    interactive: true,
    pkce: "required",
    grantTypeOptions: [GRANT_AUTH_CODE, GRANT_REFRESH],
    defaultGrantTypes: ["authorization_code", "refresh_token"],
    showResponseTypes: true,
    defaultResponseTypes: ["code"],
    authMethodOptions: PUBLIC_AUTH_METHODS,
    defaultAuthMethod: "none",
    showRedirectUris: true,
    showLoginUri: false,
    showLogoutUrls: true,
    showAllowedOrigins: false,
    showCors: false,
  },

  m2m: {
    summary:
      "Machine-to-machine service. Confidential client using the client credentials grant — no interactive login, redirects, or browser origins.",
    isPublic: false,
    interactive: false,
    pkce: "none",
    grantTypeOptions: [GRANT_CLIENT_CREDENTIALS],
    defaultGrantTypes: ["client_credentials"],
    showResponseTypes: false,
    defaultResponseTypes: [],
    authMethodOptions: CONFIDENTIAL_AUTH_METHODS,
    defaultAuthMethod: "client_secret_basic",
    showRedirectUris: false,
    showLoginUri: false,
    showLogoutUrls: false,
    showAllowedOrigins: false,
    showCors: false,
  },
}

export const CLIENT_TYPE_OPTIONS: SelectOption[] = [
  { value: "spa", label: "Single Page Application" },
  { value: "traditional", label: "Traditional Web Application" },
  { value: "mobile", label: "Native Mobile Application" },
  { value: "m2m", label: "Machine to Machine" },
]

export function getClientTypeCapability(type: ClientType): ClientTypeCapability {
  return CLIENT_TYPE_CAPABILITIES[type] ?? CLIENT_TYPE_CAPABILITIES.spa
}

/** Whether any Application URI section is relevant for the given type. */
export function hasApplicationUris(cap: ClientTypeCapability): boolean {
  return cap.showRedirectUris || cap.showLoginUri || cap.showLogoutUrls || cap.showAllowedOrigins
}
