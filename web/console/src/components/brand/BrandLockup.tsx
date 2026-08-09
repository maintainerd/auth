import MaintainedAuthIcon from '@/components/icon/MaintainedAuthIcon'

type BrandLockupProps = {
  logoLabel?: string | null
  companyName?: string | null
  showLogoLabel?: boolean
  logoUrl?: string | null
  logoDetail?: string | null
  /** Pixel size for the fallback icon; the tenant logo <img> uses imgClassName. */
  iconSize?: number
  imgClassName?: string
}

/**
 * Canonical console brand mark: the tenant logo (or the Maintainerd icon
 * fallback) stacked above the logo label and optional detail. Shared by the
 * login layout and the bootstrap loading screen so every brand surface renders
 * the same logo-and-label pattern instead of each reimplementing it.
 */
export function BrandLockup({
  logoLabel,
  companyName,
  showLogoLabel = true,
  logoUrl,
  logoDetail,
  iconSize = 48,
  imgClassName = 'h-11 w-auto',
}: BrandLockupProps) {
  const label = logoLabel || companyName || 'Maintainerd-Auth'
  const detail = logoDetail?.trim()

  return (
    <div className="flex flex-col items-center gap-3 text-center">
      {logoUrl ? (
        <img src={logoUrl} alt={label} className={`${imgClassName} shrink-0 object-contain`} />
      ) : (
        <MaintainedAuthIcon width={iconSize} height={iconSize} />
      )}
      {showLogoLabel && label && (
        <span className="text-center">
          <span
            className={`block font-semibold tracking-tight text-foreground ${detail ? 'text-sm' : 'text-lg'}`}
          >
            {label}
          </span>
          {detail && <span className="mt-1 block text-xs text-muted-foreground">{detail}</span>}
        </span>
      )}
    </div>
  )
}
