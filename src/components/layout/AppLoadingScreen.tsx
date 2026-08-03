import { Loader2 } from 'lucide-react'
import type { BrandingPublic } from '@/services/api/tenants/types'
import { resolveBrandingLogoUrl } from '@/utils/branding'
import MaintainedAuthIcon from '../icon/MaintainedAuthIcon'

type Props = {
  branding?: BrandingPublic
}

/**
 * Full-screen bootstrap splash shown once while the app figures out where the
 * user belongs (auth + tenant initialization). Mirrors the login page brand
 * mark: tenant logo when configured, otherwise the Maintainerd icon.
 */
const AppLoadingScreen = ({ branding }: Props) => {
  const companyName = branding?.company_name || 'Maintainerd IAM'
  const logoLabel = branding?.logo_label || companyName
  const logoUrl = resolveBrandingLogoUrl(branding?.logo_url)

  return (
    <div
      data-console-auth-shell
      className="flex min-h-svh flex-col items-center justify-center bg-background px-4 text-foreground"
    >
      <div className="flex flex-col items-center gap-6 text-center">
        {logoUrl ? (
          <img src={logoUrl} alt={logoLabel} className="h-12 w-auto" />
        ) : (
          <MaintainedAuthIcon width={56} height={56} />
        )}
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          <span className="text-sm">Loading…</span>
        </div>
      </div>
    </div>
  )
}

export default AppLoadingScreen
