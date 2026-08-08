import { get, put } from '@/services'
import type { RateLimitConfig, RateLimitConfigPayload, RateLimitConfigResponse } from './types'

export async function fetchRateLimitConfig(): Promise<RateLimitConfig> {
  const response = await get<RateLimitConfigResponse>('/tenant-settings/rate-limit')
  if (!response.success) throw new Error(response.message || 'Failed to fetch rate limit config')
  return response.data
}

export async function updateRateLimitConfig(data: RateLimitConfigPayload): Promise<RateLimitConfig> {
  const response = await put<RateLimitConfigResponse>('/tenant-settings/rate-limit', data)
  if (!response.success) throw new Error(response.message || 'Failed to update rate limit config')
  return response.data
}
