import { Loader2 } from 'lucide-react'
import { useTenant } from '@/hooks/useTenant'
import type { BrandingPublic } from '@/services/api/tenants/types'
import MaintainedAuthIcon from '../icon/MaintainedAuthIcon'

type Props = {
  branding?: BrandingPublic | null
}

/**
 * Full-screen bootstrap splash shown once while the app figures out where the
 * user belongs (auth + tenant initialization). Mirrors the login page brand
 * mark: tenant logo when configured, otherwise the Maintainerd icon.
 */
const AppLoadingScreen = ({ branding }: Props) => {
  const { currentTenant } = useTenant()
  const resolvedBranding = branding === undefined ? currentTenant?.branding : branding
  const companyName = resolvedBranding?.company_name || 'Maintainerd-Auth'
  const logoLabel = resolvedBranding?.logo_label || companyName
  const showLogoLabel = resolvedBranding?.show_logo_label ?? true
  const logoUrl = resolvedBranding?.logo_url

  return (
    <div className="relative flex min-h-svh flex-col items-center justify-center overflow-hidden px-4">
      <div className="auth-page-background pointer-events-none absolute inset-0" />

      <div className="relative z-10 flex flex-col items-center gap-6 text-center">
        {logoUrl ? (
          <img src={logoUrl} alt={logoLabel || companyName} className="h-12 w-auto" />
        ) : (
          <MaintainedAuthIcon width={56} height={56} />
        )}
        {showLogoLabel && logoLabel && (
          <span className="text-foreground text-lg font-semibold tracking-tight">{logoLabel}</span>
        )}
        <div className="text-muted-foreground flex items-center gap-2">
          <Loader2 className="size-4 animate-spin" />
          <span className="text-sm">Loading…</span>
        </div>
      </div>
    </div>
  )
}

export default AppLoadingScreen
