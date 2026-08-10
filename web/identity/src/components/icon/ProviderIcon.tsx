/**
 * ProviderIcon — the brand glyph for an identity / social provider, in the
 * provider's brand colour. Falls back to a neutral shield for unmapped
 * providers. Brand glyph + colour come from `@/lib/providerBrand`.
 */

import { cn } from '@/lib/utils'
import { getProviderBrand } from '@/lib/providerBrand'

interface ProviderIconProps {
  provider: string
  className?: string
}

const ProviderIcon = ({ provider, className }: ProviderIconProps) => {
  const { Icon, color } = getProviderBrand(provider)
  return <Icon className={cn(color, className)} />
}

export default ProviderIcon
