import { PROVIDER_LABELS } from '@/components/provider-config'

/**
 * Get provider display name
 */
export function getProviderDisplayName(provider: string, isSystem: boolean): string {
  if (isSystem) return 'Built-in Authentication'

  return PROVIDER_LABELS[provider] || provider
}
