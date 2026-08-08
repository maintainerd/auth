/**
 * ProviderLogo — brand mark badge for an identity / social provider.
 * Brand glyph + colour come from `providerBrand.ts`.
 */

import { cn } from "@/lib/utils"
import { getProviderBrand } from "./providerBrand"

export interface ProviderLogoProps {
  provider: string
  /** Classes for the badge container (controls size, e.g. "size-10"). */
  className?: string
  /** Classes for the glyph itself (controls icon size, e.g. "size-5"). */
  iconClassName?: string
}

export function ProviderLogo({ provider, className, iconClassName }: ProviderLogoProps) {
  const { Icon, color } = getProviderBrand(provider)

  return (
    <div
      className={cn(
        "flex size-10 shrink-0 items-center justify-center rounded-lg border bg-background",
        className
      )}
    >
      <Icon className={cn("size-5", color, iconClassName)} />
    </div>
  )
}
