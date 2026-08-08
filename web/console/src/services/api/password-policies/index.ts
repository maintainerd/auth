import { get, put } from '@/services'
import type {
  PasswordPolicies,
  PasswordPoliciesPayload,
  PasswordPoliciesResponse,
} from './types'

const API_ENDPOINTS = {
  PASSWORD_POLICIES: '/security-settings/password',
}

export async function fetchPasswordPolicies(): Promise<PasswordPolicies> {
  const response = await get<PasswordPoliciesResponse>(API_ENDPOINTS.PASSWORD_POLICIES)

  if (!response.success) {
    throw new Error(response.message || 'Failed to fetch password policies')
  }

  return response.data
}

export async function updatePasswordPolicies(
  data: PasswordPoliciesPayload
): Promise<PasswordPolicies> {
  const response = await put<PasswordPoliciesResponse>(
    API_ENDPOINTS.PASSWORD_POLICIES,
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to update password policies')
  }

  return response.data
}
