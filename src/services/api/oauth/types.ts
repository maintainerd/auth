export interface OAuthAuthorizeResult {
  redirect_uri?: string
  consent_challenge?: string
}

export interface OAuthConnection {
  identifier: string
  display_name: string
  provider: string
  provider_type: string
  is_default: boolean
  display_order: number
}

export interface OAuthConnections {
  password_enabled: boolean
  registration_enabled: boolean
  /** Passwordless email sign-in, opt-in per client. */
  magic_link_enabled?: boolean
  branding?: OAuthConnectionsBranding
  connections: OAuthConnection[]
}

export interface OAuthConnectionsBranding {
  branding_id: string
  company_name: string
  logo_url: string
  favicon_url: string
  layout: string
  support_url: string
  privacy_policy_url: string
  terms_of_service_url: string
  metadata: Record<string, unknown> | null
}

export interface OAuthConsentChallenge {
  challenge_id: string
  client_name: string
  client_uuid: string
  scopes: string[]
  redirect_uri: string
  expires_at: number
}

export interface OAuthConsentDecisionResult {
  redirect_uri: string
}

export interface OAuthConsentGrant {
  consent_grant_id: string
  client_name: string
  client_uuid: string
  scopes: string[]
  granted_at: string
  updated_at: string
}
