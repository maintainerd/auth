/**
 * IP Restriction Rules Hook
 */

import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import {
  fetchIpRestrictionRules,
  createIpRestrictionRule,
  updateIpRestrictionRule,
  deleteIpRestrictionRule,
  updateIpRestrictionRuleStatus,
} from '@/services/api/ip-restriction-rules'
import type {
  IpRestrictionRulesQueryParams,
  CreateIpRestrictionRuleRequest,
  UpdateIpRestrictionRuleRequest,
  UpdateIpRestrictionRuleStatusRequest,
} from '@/services/api/ip-restriction-rules/types'

export const ipRestrictionRulesKeys = {
  all: ['ipRestrictionRules'] as const,
  lists: () => [...ipRestrictionRulesKeys.all, 'list'] as const,
  list: (params: IpRestrictionRulesQueryParams) => [...ipRestrictionRulesKeys.lists(), params] as const,
}

export const useIpRestrictionRules = (params: IpRestrictionRulesQueryParams = {}) => {
  return useQuery({
    queryKey: ipRestrictionRulesKeys.list(params),
    queryFn: () => fetchIpRestrictionRules(params),
    placeholderData: keepPreviousData,
  })
}

export const useCreateIpRestrictionRule = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateIpRestrictionRuleRequest) => createIpRestrictionRule(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ipRestrictionRulesKeys.lists() })
    },
  })
}

export const useUpdateIpRestrictionRule = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateIpRestrictionRuleRequest }) =>
      updateIpRestrictionRule(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ipRestrictionRulesKeys.lists() })
    },
  })
}

export const useDeleteIpRestrictionRule = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => deleteIpRestrictionRule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ipRestrictionRulesKeys.lists() })
    },
  })
}

export const useUpdateIpRestrictionRuleStatus = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateIpRestrictionRuleStatusRequest }) =>
      updateIpRestrictionRuleStatus(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ipRestrictionRulesKeys.lists() })
    },
  })
}
