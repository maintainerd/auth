/**
 * Client API Types
 */

import type { Status } from '@/types/status'

/**
 * Client config type. The backend returns the stored JSON object directly
 * inside the API response envelope.
 */
export type ClientConfig = Record<string, unknown>

/**
 * Client status type - defines valid statuses for clients only
 */
export type ClientStatus = Extract<Status, 'active' | 'inactive'>

/**
 * Client type enum
 */
export type ClientType = 'traditional' | 'spa' | 'mobile' | 'm2m'

/**
 * Required authentication context class (ACR) override.
 * '1' = password / single factor, '2' = step-up / MFA.
 * null/undefined = inherit the tenant security policy.
 */
export type RequiredAcr = '1' | '2'

/**
 * Client URI type enum
 */
export type ClientUriType = 'redirect_uri' | 'origin_uri' | 'logout_uri' | 'login_uri' | 'cors_origin_uri'

/**
 * URI type (used in client list response)
 */
export type Uri = {
  uri_id: string
  uri: string
  type: ClientUriType
  created_at: string
  updated_at: string
}

/**
 * Identity Provider type (nested in client response)
 */
export type ClientIdentityProvider = {
  identity_provider_id: string
  name: string
  display_name: string
  provider: string
  provider_type: string
  identifier: string
  status: string
  is_default: boolean
  is_system: boolean
  created_at: string
  updated_at: string
}

export type ClientIdentityProviderConnection = {
  client_identity_provider_id: string
  identity_provider: ClientIdentityProvider
  is_default: boolean
  enabled: boolean
  display_order: number
  created_at: string
  updated_at: string
}

/**
 * Client type
 */
export type Client = {
  client_id: string
  name: string
  display_name: string
  client_type: ClientType
  domain?: string | null
  uris?: Uri[]
  identity_provider?: ClientIdentityProvider
  connections?: ClientIdentityProviderConnection[]
  status: ClientStatus
  is_default: boolean
  is_system: boolean
  branding_id?: string | null
  allow_registration: boolean
  allow_magic_link: boolean
  /**
   * OAuth metadata as ENFORCED by the runtime. These come from real columns; the
   * `config` blob is only mirrored into them on write, so reading the blob can show
   * values the server rejected.
   */
  token_endpoint_auth_method?: string
  grant_types?: string[]
  response_types?: string[]
  allowed_scopes?: string[]
  require_consent?: boolean | null
  access_token_ttl?: number | null
  refresh_token_ttl?: number | null

  // Security posture / per-client overrides. null = inherits the tenant default.
  require_pkce?: boolean | null
  required_acr?: RequiredAcr | null
  session_idle_timeout?: number | null
  session_absolute_timeout?: number | null
  created_at: string
  updated_at: string
  backchannel_logout_uri?: string | null
  frontchannel_logout_uri?: string | null
  backchannel_logout_session_required?: boolean
  dpop_required?: boolean
  /**
   * Binds this client to a service, making it that service's credential. A token
   * issued to a bound client carries the `svc` claim, which is the principal the
   * policy bundle and the gRPC authorizer resolve — this is what makes
   * service-to-service authorization reachable. m2m clients only.
   */
  service_id?: string
}

export type RotateClientSecretRequest = {
  grace_period_hours: number
}

export type RotateClientSecretResponse = {
  client_id?: string
  client_secret: string
  previous_secret_expires_at?: string
}

/**
 * Client list query parameters interface
 */
export interface ClientQueryParams {
  name?: string
  display_name?: string
  client_type?: string
  identity_provider_id?: string
  status?: string
  is_default?: boolean
  is_system?: boolean
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Paginated client list response interface
 */
export interface ClientListResponse {
  rows: Client[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Single client response interface
 */
export interface ClientResponse {
  /**
   * The MANAGEMENT handle (clients.client_uuid). Named client_id in JSON for
   * backward compatibility — it is NOT the OAuth client_id. See `identifier`.
   */
  client_id: string
  /**
   * The OAuth 2.0 client_id: the value an application presents at /authorize and
   * /oauth/token, and what /oauth/connections resolves a client by. Server
   * generated, so absent until the client exists.
   */
  identifier?: string | null
  name: string
  display_name: string
  client_type: ClientType
  domain?: string | null
  uris?: Uri[]
  identity_provider?: ClientIdentityProvider
  connections?: ClientIdentityProviderConnection[]
  status: ClientStatus
  is_default: boolean
  is_system: boolean
  branding_id?: string | null
  allow_registration: boolean
  allow_magic_link: boolean
  /**
   * OAuth metadata as ENFORCED by the runtime. These come from real columns; the
   * `config` blob is only mirrored into them on write, so reading the blob can show
   * values the server rejected.
   */
  token_endpoint_auth_method?: string
  grant_types?: string[]
  response_types?: string[]
  allowed_scopes?: string[]
  require_consent?: boolean | null
  access_token_ttl?: number | null
  refresh_token_ttl?: number | null

  // Security posture / per-client overrides. null = inherits the tenant default.
  require_pkce?: boolean | null
  required_acr?: RequiredAcr | null
  session_idle_timeout?: number | null
  session_absolute_timeout?: number | null
  created_at: string
  updated_at: string
  backchannel_logout_uri?: string | null
  frontchannel_logout_uri?: string | null
  backchannel_logout_session_required?: boolean
  dpop_required?: boolean
  /**
   * Binds this client to a service, making it that service's credential. A token
   * issued to a bound client carries the `svc` claim, which is the principal the
   * policy bundle and the gRPC authorizer resolve — this is what makes
   * service-to-service authorization reachable. m2m clients only.
   */
  service_id?: string
}

/**
 * Create client response. The backend returns plaintext credentials exactly
 * once at creation, alongside the created client resource.
 */
export interface ClientCredentialsResponse {
  client_uuid: string
  /** The OAuth client_id (clients.identifier). */
  client_id: string
  /** Plaintext, returned exactly once. Unrecoverable afterwards. */
  client_secret: string
}

export interface ClientCreateResponse {
  client: ClientResponse
  /**
   * Absent for a public client (spa/mobile): the backend no longer issues a secret
   * to a client that cannot keep one.
   */
  credentials?: ClientCredentialsResponse
}

/**
 * Create client request interface
 */
export interface CreateClientRequest {
  name: string
  display_name: string
  client_type: ClientType
  domain: string
  identity_provider_id?: string
  branding_id?: string
  allow_registration?: boolean
  allow_magic_link?: boolean
  status: ClientStatus
  config: Record<string, unknown>
  backchannel_logout_uri?: string | null
  frontchannel_logout_uri?: string | null
  backchannel_logout_session_required?: boolean
  dpop_required?: boolean
  /**
   * Binds this client to a service, making it that service's credential. A token
   * issued to a bound client carries the `svc` claim, which is the principal the
   * policy bundle and the gRPC authorizer resolve — this is what makes
   * service-to-service authorization reachable. m2m clients only.
   */
  service_id?: string
}

/**
 * Update client request interface
 */
export interface UpdateClientRequest {
  name: string
  display_name: string
  client_type: ClientType
  domain: string
  branding_id?: string
  allow_registration?: boolean
  allow_magic_link?: boolean
  status: ClientStatus
  config: Record<string, unknown>
  backchannel_logout_uri?: string | null
  frontchannel_logout_uri?: string | null
  backchannel_logout_session_required?: boolean
  dpop_required?: boolean
  /** Binds this client to a service. "" unbinds. m2m only. */
  service_id?: string
  /**
   * Optimistic-concurrency token: the `updated_at` this edit was loaded from. An
   * update replaces the whole client, so without it two operators editing at once
   * silently overwrite each other. The server answers 409 when the row moved on.
   */
  expected_updated_at?: string
}

/**
 * Update client status request interface
 */
export interface UpdateClientStatusRequest {
  status: ClientStatus
}

export interface AddClientIdentityProviderRequest {
  identity_provider_id: string
  is_default: boolean
  enabled?: boolean
  display_order: number
}

/**
 * Client URI type
 */
export type ClientUri = {
  // The API returns the client URI's UUID under `uri_id`.
  uri_id: string
  type: ClientUriType
  uri: string
  created_at: string
  updated_at: string
}

/**
 * Client URIs response interface
 */
export interface ClientUrisResponse {
  uris: ClientUri[]
}

/**
 * Create client URI request interface
 */
export interface CreateClientUriRequest {
  uri: string
  type: ClientUriType
}

/**
 * Update client URI request interface
 */
export interface UpdateClientUriRequest {
  uri: string
  type: ClientUriType
}

/**
 * Permission type (simplified for client API context)
 */
export type ClientApiPermission = {
  permission_id: string
  name: string
  description: string
  status: 'active' | 'inactive'
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * API entity (simplified for client API context)
 */
export type ClientApi = {
  api_id: string
  name: string
  display_name: string
  description: string
  identifier?: string
  status: 'active' | 'inactive'
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * Client API association with permissions
 */
export type ClientApiAssociation = {
  client_api_id: string
  api: ClientApi
  permissions: ClientApiPermission[]
  created_at: string
}

/**
 * Client APIs response interface
 */
export interface ClientApisResponse {
  apis: ClientApiAssociation[]
}

/**
 * Add APIs to client request interface
 */
export interface AddClientApisRequest {
  api_uuids: string[]
}

/**
 * Add permissions to client API request interface
 */
export interface AddClientApiPermissionsRequest {
  permission_uuids: string[]
}
