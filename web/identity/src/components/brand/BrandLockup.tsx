import MaintainedAuthIcon from '@/components/icon/MaintainedAuthIcon'

type BrandLockupProps = {
  companyName: string
  logoLabel: string
  showLogoLabel: boolean
  logoUrl?: string | null
  panel?: boolean
  centered?: boolean
  logoClassName?: string
  labelClassName?: string
  topPanelLabel?: boolean
  logoDetail?: string
}

export function BrandLockup({
  companyName,
  logoLabel,
  showLogoLabel,
  logoUrl,
  panel = false,
  centered = false,
  logoClassName = 'size-9',
  labelClassName,
  topPanelLabel = false,
  logoDetail,
}: BrandLockupProps) {
  const label = logoLabel || companyName || 'Maintainerd'
  const detail = logoDetail?.trim()
  const resolvedLabelClassName = labelClassName ?? (detail ? 'text-sm' : 'text-lg')

  return (
    <div className={`flex items-center gap-3 ${centered ? 'justify-center' : ''}`}>
      {logoUrl ? (
        <img
          src={logoUrl}
          alt={label}
          className={`${logoClassName} shrink-0 rounded-md object-contain ${panel ? 'bg-white/15 p-1' : ''}`}
        />
      ) : (
        <MaintainedAuthIcon
          alt={label}
          width="2.25rem"
          height="2.25rem"
          className={`${logoClassName} shrink-0 rounded-md ${panel ? 'bg-white/15 p-1' : ''}`}
        />
      )}
      {showLogoLabel && label && (
        <div className="min-w-0 text-left">
          <p
            data-md-top-logo-label={topPanelLabel ? true : undefined}
            className={`truncate font-semibold leading-none ${resolvedLabelClassName}`}
          >
            {label}
          </p>
          {detail && (
            <p className={`mt-1 leading-none opacity-75 ${topPanelLabel ? 'text-[11px]' : 'text-xs'}`}>
              {detail}
            </p>
          )}
        </div>
      )}
    </div>
  )
}
