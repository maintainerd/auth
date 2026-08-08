import { get } from '../client'
import type { ApiResponse } from '../types'
import { unwrap } from '../_lib/unwrap'
import { API_ENDPOINTS } from '../config'
import type { DashboardSummary } from './types'

export const fetchDashboardSummary = (): Promise<DashboardSummary> =>
  get<ApiResponse<DashboardSummary>>(`${API_ENDPOINTS.DASHBOARD}/summary`)
    .then((r) => unwrap(r, 'fetch dashboard summary'))
