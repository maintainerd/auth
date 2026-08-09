import { useEffect } from 'react'
import { ExternalLink } from 'lucide-react'
import { BrandLockup } from '@/components/brand/BrandLockup'
import { Card, CardContent } from '@/components/ui/card'
import type { BrandingPublic } from '@/services/api/tenants/types'
import { resolveBrandingLogoUrl } from '@/utils/branding'

type Props = {
  children: React.ReactNode
  branding?: BrandingPublic
}

const LoginLayout = ({ children, branding }: Props) => {
  const companyName = branding?.company_name || 'Maintainerd'
  const logoLabel = branding?.logo_label || companyName
  const logoDetail = branding?.logo_detail
  const showLogoLabel = branding?.show_logo_label ?? true
  const logoUrl = resolveBrandingLogoUrl(branding?.logo_url)

  const year = new Date().getFullYear()

  const legalLinks = [
    branding?.support_url && { label: 'Support', href: branding.support_url },
    branding?.privacy_policy_url && { label: 'Privacy', href: branding.privacy_policy_url },
    branding?.terms_of_service_url && { label: 'Terms', href: branding.terms_of_service_url },
  ].filter(Boolean) as { label: string; href: string }[]

  useEffect(() => {
    if (branding?.favicon_url) {
      const link = document.querySelector<HTMLLinkElement>("link[rel*='icon']")
      if (link) link.href = branding.favicon_url
    }
  }, [branding?.favicon_url])

  return (
    <div
      data-console-auth-shell
      className="flex min-h-svh flex-col items-center justify-center bg-background px-4 py-12 text-foreground"
    >
      <div className="w-full max-w-md">
        {/* Brand mark */}
        <div className="mb-8">
          <BrandLockup
            logoLabel={logoLabel}
            companyName={companyName}
            showLogoLabel={showLogoLabel}
            logoUrl={logoUrl}
            logoDetail={logoDetail}
            iconSize={48}
            imgClassName="h-11 w-auto"
          />
        </div>

        {/* Form card */}
        <Card data-console-auth-card className="border-border shadow-sm">
          <CardContent className="p-7 sm:p-9">{children}</CardContent>
        </Card>

        {/* Footer */}
        {(legalLinks.length > 0 || companyName) && (
          <div className="mt-8 flex flex-col items-center gap-3 text-center">
            {legalLinks.length > 0 && (
              <div className="flex flex-wrap justify-center gap-5 text-sm text-muted-foreground">
                {legalLinks.map((link) => (
                  <a
                    key={link.label}
                    href={link.href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 transition-colors hover:text-foreground"
                  >
                    {link.label} <ExternalLink className="size-3" />
                  </a>
                ))}
              </div>
            )}
            {companyName && (
              <span className="text-xs text-muted-foreground">© {year} {companyName}</span>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

export default LoginLayout;
