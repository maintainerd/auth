import { Loader2 } from 'lucide-react'
import { BrandLockup } from '@/components/brand/BrandLockup'
import { useTenant } from '@/hooks/useTenant'
import { authUiTemplatePresentationFromMetadata } from '@/lib/branding/authUiTemplates'
import type { BrandingPublic } from '@/services/api/tenants/types'
import { resolveBrandingLogoUrl } from '@/utils/branding'

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
  const companyName = resolvedBranding?.company_name || 'Maintainerd'
  const logoLabel = resolvedBranding?.identity_logo_label || resolvedBranding?.logo_label || companyName
  const showLogoLabel = resolvedBranding?.identity_show_logo_label ?? resolvedBranding?.show_logo_label ?? true
  const logoUrl = resolveBrandingLogoUrl(resolvedBranding?.logo_url)
  const presentation = authUiTemplatePresentationFromMetadata(resolvedBranding?.metadata)

  return (
    <div
      data-auth-identity-shell
      className="relative flex min-h-svh flex-col items-center justify-center overflow-hidden px-4"
    >
      <div className="auth-page-background pointer-events-none absolute inset-0" />

      <div className="relative z-10 flex flex-col items-center gap-6 text-center">
        <BrandLockup
          companyName={companyName}
          logoLabel={logoLabel}
          showLogoLabel={showLogoLabel}
          logoUrl={logoUrl}
          logoDetail={presentation.logoDetail}
          centered
        />
        <div className="text-muted-foreground flex items-center gap-2">
          <Loader2 className="size-4 animate-spin" />
          <span className="text-sm">Loading…</span>
        </div>
      </div>
    </div>
  )
}

export default AppLoadingScreen
