export interface TokenConfig {
  clock_skew_leeway_seconds: number
  additional_id_token_claims: string[]
  additional_access_token_claims: string[]
  signing_algorithm: string
  require_pkce: boolean
}

export interface TokenConfigPayload {
  clock_skew_leeway_seconds?: number
  additional_id_token_claims?: string[]
  additional_access_token_claims?: string[]
  signing_algorithm?: string
  require_pkce?: boolean
}

export interface TokenConfigResponse {
  success: boolean
  data: TokenConfig
  message: string
}
