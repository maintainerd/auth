/**
 * SMS Templates Hook
 * Custom hook for fetching SMS templates using TanStack Query
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchSmsTemplates,
  fetchSmsTemplateById,
  updateSmsTemplate,
  updateSmsTemplateStatus,
} from '@/services/api/sms-templates'
import type {
  SmsTemplateQueryParams,
  UpdateSmsTemplateRequest,
  UpdateSmsTemplateStatusRequest,
} from '@/services/api/sms-templates/types'
import { useToast } from './useToast'

/**
 * Query key factory for SMS templates
 */
export const smsTemplateKeys = {
  all: ['smsTemplates'] as const,
  lists: () => [...smsTemplateKeys.all, 'list'] as const,
  list: (params?: SmsTemplateQueryParams) => [...smsTemplateKeys.lists(), params] as const,
  details: () => [...smsTemplateKeys.all, 'detail'] as const,
  detail: (id: string) => [...smsTemplateKeys.details(), id] as const,
}

/**
 * Hook to fetch SMS templates with optional filters
 */
export function useSmsTemplates(params?: SmsTemplateQueryParams) {
  return useQuery({
    queryKey: smsTemplateKeys.list(params),
    queryFn: () => fetchSmsTemplates(params),
    staleTime: 5 * 60 * 1000,
  })
}

export function useSmsTemplatesList(params: Record<string, unknown>) {
  return useSmsTemplates(params as SmsTemplateQueryParams)
}

/**
 * Hook to fetch a single SMS template by ID
 */
export function useSmsTemplate(id: string) {
  return useQuery({
    queryKey: smsTemplateKeys.detail(id),
    queryFn: () => fetchSmsTemplateById(id),
    enabled: !!id,
    staleTime: 5 * 60 * 1000, // 5 minutes
  })
}

/**
 * Hook to update an SMS template
 */
export function useUpdateSmsTemplate() {
  const queryClient = useQueryClient()
  const { showSuccess } = useToast()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateSmsTemplateRequest }) =>
      updateSmsTemplate(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: smsTemplateKeys.lists() })
      queryClient.invalidateQueries({ queryKey: smsTemplateKeys.detail(variables.id) })
      showSuccess('SMS template updated successfully')
    },
  })
}

/**
 * Hook to update SMS template status
 */
export function useUpdateSmsTemplateStatus() {
  const queryClient = useQueryClient()
  const { showSuccess, showError } = useToast()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateSmsTemplateStatusRequest }) =>
      updateSmsTemplateStatus(id, data),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: smsTemplateKeys.lists() })
      queryClient.invalidateQueries({ queryKey: smsTemplateKeys.detail(result.smsTemplateId) })
      showSuccess(`SMS template ${result.status === 'active' ? 'activated' : 'deactivated'} successfully`)
    },
    onError: (e) => showError(e),
  })
}
