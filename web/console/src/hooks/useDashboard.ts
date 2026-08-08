import { useQuery } from '@tanstack/react-query'
import { fetchDashboardSummary } from '@/services/api/dashboard'

export const dashboardKeys = {
  all: ['dashboard'] as const,
  summary: () => [...dashboardKeys.all, 'summary'] as const,
}

export function useDashboardSummary() {
  return useQuery({
    queryKey: dashboardKeys.summary(),
    queryFn: fetchDashboardSummary,
  })
}
