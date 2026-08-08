export interface ResourceCount {
  total: number
  active: number
  inactive?: number
}

export interface UserCount {
  total: number
  active: number
  inactive: number
  suspended: number
  pending: number
}

export interface DashboardSummary {
  users: UserCount
  services: ResourceCount
  clients: ResourceCount
  identity_providers: ResourceCount
  roles: ResourceCount
  api_keys: ResourceCount
  recent_logins_24h: number
  failed_logins_24h: number
}
