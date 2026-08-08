/**
 * IP Restriction Rules API Types
 */

export interface IpRestrictionRule {
  ipRestrictionRuleId: string
  description: string
  type: 'allow' | 'deny'
  ipAddress: string
  status: 'active' | 'inactive'
  createdAt: string
  updatedAt: string
}

export interface IpRestrictionRulePayload {
  ip_restriction_rule_id: string
  description: string
  type: 'allow' | 'deny'
  ip_address: string
  status: 'active' | 'inactive'
  created_at: string
  updated_at: string
}

export interface IpRestrictionRulesResponse {
  success: boolean
  data: {
    rows: IpRestrictionRulePayload[]
    total: number
    page: number
    limit: number
    total_pages: number
  }
  message: string
}

export interface CreateIpRestrictionRuleRequest {
  description: string
  type: 'allow' | 'deny'
  ip_address: string
  status: 'active' | 'inactive'
}

export interface UpdateIpRestrictionRuleRequest {
  description: string
  type: 'allow' | 'deny'
  ip_address: string
  status: 'active' | 'inactive'
}

export interface UpdateIpRestrictionRuleStatusRequest {
  status: 'active' | 'inactive'
}

export interface IpRestrictionRulesQueryParams {
  type?: 'allow' | 'deny'
  status?: 'active' | 'inactive'
  ip_address?: string
  description?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}
