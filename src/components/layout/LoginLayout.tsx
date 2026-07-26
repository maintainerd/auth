import { useEffect } from 'react'
import { ExternalLink } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import type { BrandingPublic } from '@/services/api/tenants/types'
import MaintainedAuthIcon from "../icon/MaintainedAuthIcon"

type Props = {
  children: React.ReactNode
  branding?: BrandingPublic
}

const LoginLayout = ({ children, branding }: Props) => {
  const companyName = branding?.company_name || 'Maintainerd IAM'
  const logoUrl = branding?.logo_url

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
    <div className="relative flex min-h-svh flex-col items-center justify-center overflow-hidden px-4 py-12">
      {/* Light, lightly-blue gradient backdrop — calm and trustworthy, not flashy */}
      <div
        className="pointer-events-none absolute inset-0"
        style={{ background: 'linear-gradient(180deg, #f7f9fc 0%, #eef2f8 55%, #e4ebf6 100%)' }}
      />

      <div className="relative z-10 w-full max-w-md">
        {/* Brand mark */}
        <div className="mb-8 flex flex-col items-center gap-3 text-center">
          {logoUrl ? (
            <img src={logoUrl} alt={companyName || 'Logo'} className="h-11 w-auto" />
          ) : (
            <MaintainedAuthIcon width={48} height={48} />
          )}
          {companyName && (
            <span className="text-lg font-semibold tracking-tight text-slate-900">{companyName}</span>
          )}
        </div>

        {/* Form card */}
        <Card className="border-slate-200/70 shadow-[0_1px_2px_rgba(15,23,42,0.04),0_16px_40px_-20px_rgba(15,23,42,0.25)]">
          <CardContent className="p-7 sm:p-9">{children}</CardContent>
        </Card>

        {/* Footer */}
        {(legalLinks.length > 0 || companyName) && (
          <div className="mt-8 flex flex-col items-center gap-3 text-center">
            {legalLinks.length > 0 && (
              <div className="flex flex-wrap justify-center gap-5 text-sm text-slate-500">
                {legalLinks.map((link) => (
                  <a
                    key={link.label}
                    href={link.href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 transition-colors hover:text-slate-900"
                  >
                    {link.label} <ExternalLink className="size-3" />
                  </a>
                ))}
              </div>
            )}
            {companyName && (
              <span className="text-xs text-slate-400">© {year} {companyName}</span>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

export default LoginLayout;
