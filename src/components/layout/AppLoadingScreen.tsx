import { Loader2 } from 'lucide-react'
import type { BrandingPublic } from '@/services/api/tenants/types'
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
  const logoUrl = branding?.logo_url

  return (
    <div className="relative flex min-h-svh flex-col items-center justify-center overflow-hidden px-4">
      {/* Same calm gradient backdrop as the login layout. */}
      <div
        className="pointer-events-none absolute inset-0"
        style={{ background: 'linear-gradient(180deg, #f7f9fc 0%, #eef2f8 55%, #e4ebf6 100%)' }}
      />

      <div className="relative z-10 flex flex-col items-center gap-6 text-center">
        {logoUrl ? (
          <img src={logoUrl} alt={companyName} className="h-12 w-auto" />
        ) : (
          <MaintainedAuthIcon width={56} height={56} />
        )}
        <div className="flex items-center gap-2 text-slate-500">
          <Loader2 className="size-4 animate-spin" />
          <span className="text-sm">Loading…</span>
        </div>
      </div>
    </div>
  )
}

export default AppLoadingScreen
