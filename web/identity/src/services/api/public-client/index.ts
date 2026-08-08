import { get } from '../client'
import type { ApiResponse } from '../types'

export interface PublicClient {
  client_id: string
  name: string
  display_name: string
  client_type: string
  domain?: string
  tenant_id: string
}

export async function fetchPublicClient(clientId: string): Promise<PublicClient> {
  const response = await get<ApiResponse<PublicClient>>(`/client?client_id=${encodeURIComponent(clientId)}`)
  if (!response.success || !response.data) throw new Error('Auth client not found')
  return response.data
}
