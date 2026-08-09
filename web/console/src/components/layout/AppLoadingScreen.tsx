import { Loader2 } from 'lucide-react'
import { BrandLockup } from '@/components/brand/BrandLockup'
import type { BrandingPublic } from '@/services/api/tenants/types'
import { resolveBrandingLogoUrl } from '@/utils/branding'

type Props = {
  branding?: BrandingPublic
}

/**
 * Full-screen bootstrap splash shown once while the app figures out where the
 * user belongs (auth + tenant initialization). Renders the same logo-and-label
 * brand mark as the login page (via BrandLockup) so the loading state matches
 * the rest of the app instead of showing a bare logo.
 */
const AppLoadingScreen = ({ branding }: Props) => {
  const companyName = branding?.company_name || 'Maintainerd'
  const logoLabel = branding?.logo_label || companyName
  const showLogoLabel = branding?.show_logo_label ?? true
  const logoDetail = branding?.logo_detail
  const logoUrl = resolveBrandingLogoUrl(branding?.logo_url)

  return (
    <div
      data-console-auth-shell
      className="flex min-h-svh flex-col items-center justify-center bg-background px-4 text-foreground"
    >
      <div className="flex flex-col items-center gap-6 text-center">
        <BrandLockup
          logoLabel={logoLabel}
          companyName={companyName}
          showLogoLabel={showLogoLabel}
          logoUrl={logoUrl}
          logoDetail={logoDetail}
          iconSize={56}
          imgClassName="h-12 w-auto"
        />
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          <span className="text-sm">Loading…</span>
        </div>
      </div>
    </div>
  )
}

export default AppLoadingScreen
