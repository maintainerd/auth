import { get, put } from '@/services'
import type {
  TokenConfig,
  TokenConfigPayload,
  TokenConfigResponse,
} from './types'

const API_ENDPOINTS = {
  TOKEN_CONFIG: '/security-settings/token',
}

export async function fetchTokenConfig(): Promise<TokenConfig> {
  const response = await get<TokenConfigResponse>(API_ENDPOINTS.TOKEN_CONFIG)

  if (!response.success) {
    throw new Error(response.message || 'Failed to fetch token config')
  }

  return response.data
}

export async function updateTokenConfig(
  data: TokenConfigPayload
): Promise<TokenConfig> {
  const response = await put<TokenConfigResponse>(
    API_ENDPOINTS.TOKEN_CONFIG,
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to update token config')
  }

  return response.data
}
