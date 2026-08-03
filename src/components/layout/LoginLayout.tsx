import { useEffect, type ReactNode } from 'react'
import { ExternalLink } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { useTenant } from '@/hooks/useTenant'
import type { BrandingLayout, BrandingPublic } from '@/services/api/tenants/types'
import MaintainedAuthIcon from '../icon/MaintainedAuthIcon'

type Props = {
  children: ReactNode
  branding?: BrandingPublic | null
}

type BrandMarkProps = {
  companyName: string
  logoLabel: string
  showLogoLabel: boolean
  logoUrl?: string
  panel?: boolean
}

type FooterProps = {
  companyName: string
  legalLinks: { label: string; href: string }[]
  panel?: boolean
}

function resolvedLayout(layout: BrandingPublic['layout'] | undefined): BrandingLayout {
  return layout === 'full_page' || layout === 'split' ? layout : 'centered'
}

function BrandMark({ companyName, logoLabel, showLogoLabel, logoUrl, panel = false }: BrandMarkProps) {
  return (
    <div className="flex flex-col items-center gap-3 text-center">
      {logoUrl ? (
        <img src={logoUrl} alt={logoLabel || companyName || 'Logo'} className="h-11 w-auto max-w-full object-contain" />
      ) : panel ? (
        <div className="auth-brand-icon-panel rounded-2xl p-3 shadow-sm">
          <MaintainedAuthIcon width={48} height={48} />
        </div>
      ) : (
        <MaintainedAuthIcon width={48} height={48} />
      )}
      {showLogoLabel && logoLabel && (
        <span className={panel ? 'text-xl font-semibold tracking-tight' : 'text-foreground text-lg font-semibold tracking-tight'}>
          {logoLabel}
        </span>
      )}
    </div>
  )
}

function Footer({ companyName, legalLinks, panel = false }: FooterProps) {
  if (legalLinks.length === 0 && !companyName) return null

  return (
    <div className={panel ? 'flex flex-col items-center gap-3 text-center text-current/75' : 'text-muted-foreground flex flex-col items-center gap-3 text-center'}>
      {legalLinks.length > 0 && (
        <div className="flex flex-wrap justify-center gap-5 text-sm">
          {legalLinks.map((link) => (
            <a
              key={link.label}
              href={link.href}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 transition-colors hover:text-current"
            >
              {link.label} <ExternalLink className="size-3" />
            </a>
          ))}
        </div>
      )}
      {companyName && (
        <span className={panel ? 'text-xs' : 'text-muted-foreground/70 text-xs'}>
          © {new Date().getFullYear()} {companyName}
        </span>
      )}
    </div>
  )
}

function CenteredLayout({
  children,
  companyName,
  logoLabel,
  showLogoLabel,
  logoUrl,
  legalLinks,
}: {
  children: ReactNode
  companyName: string
  logoLabel: string
  showLogoLabel: boolean
  logoUrl?: string
  legalLinks: FooterProps['legalLinks']
}) {
  return (
    <main data-layout="centered" className="relative flex min-h-svh flex-col items-center justify-center overflow-hidden px-4 py-12">
      <div className="auth-page-background pointer-events-none absolute inset-0" />
      <div className="relative z-10 w-full max-w-md">
        <div className="mb-8">
          <BrandMark companyName={companyName} logoLabel={logoLabel} showLogoLabel={showLogoLabel} logoUrl={logoUrl} />
        </div>
        <Card className="auth-form-panel border-border/70 shadow-[0_1px_2px_rgba(15,23,42,0.04),0_16px_40px_-20px_rgba(15,23,42,0.25)]">
          <CardContent className="p-7 sm:p-9">{children}</CardContent>
        </Card>
        <div className="mt-8">
          <Footer companyName={companyName} legalLinks={legalLinks} />
        </div>
      </div>
    </main>
  )
}

function FullPageLayout({
  children,
  companyName,
  logoLabel,
  showLogoLabel,
  logoUrl,
  legalLinks,
}: {
  children: ReactNode
  companyName: string
  logoLabel: string
  showLogoLabel: boolean
  logoUrl?: string
  legalLinks: FooterProps['legalLinks']
}) {
  return (
    <main data-layout="full_page" className="relative min-h-svh overflow-hidden p-0 sm:p-6 lg:p-10">
      <div className="auth-page-background pointer-events-none absolute inset-0" />
      <div className="auth-full-page-panel relative z-10 mx-auto flex min-h-svh w-full max-w-4xl flex-col px-6 py-8 shadow-2xl sm:min-h-[calc(100svh-3rem)] sm:rounded-3xl sm:px-12 lg:min-h-[calc(100svh-5rem)] lg:px-20">
        <div className="flex flex-1 items-center justify-center py-10">
          <div className="w-full max-w-md">
            <div className="mb-10">
              <BrandMark companyName={companyName} logoLabel={logoLabel} showLogoLabel={showLogoLabel} logoUrl={logoUrl} />
            </div>
            {children}
          </div>
        </div>
        <Footer companyName={companyName} legalLinks={legalLinks} />
      </div>
    </main>
  )
}

function SplitLayout({
  children,
  companyName,
  logoLabel,
  showLogoLabel,
  logoUrl,
  legalLinks,
}: {
  children: ReactNode
  companyName: string
  logoLabel: string
  showLogoLabel: boolean
  logoUrl?: string
  legalLinks: FooterProps['legalLinks']
}) {
  return (
    <main data-layout="split" className="auth-form-panel grid min-h-svh lg:grid-cols-[minmax(0,1.1fr)_minmax(30rem,0.9fr)]">
      <section data-testid="split-brand-panel" className="auth-split-brand-panel relative hidden overflow-hidden p-12 lg:flex lg:flex-col lg:justify-between">
        <div className="auth-split-decoration-light pointer-events-none absolute -right-24 -top-24 size-80 rounded-full opacity-10" />
        <div className="auth-split-decoration-dark pointer-events-none absolute -bottom-40 -left-20 size-96 rounded-full opacity-10" />
        <div className="relative flex flex-1 items-center justify-center">
          <BrandMark companyName={companyName} logoLabel={logoLabel} showLogoLabel={showLogoLabel} logoUrl={logoUrl} panel />
        </div>
        <div className="relative">
          <Footer companyName={companyName} legalLinks={legalLinks} panel />
        </div>
      </section>

      <section className="auth-form-panel flex min-h-svh items-center justify-center px-6 py-12 sm:px-10 lg:px-14">
        <div className="w-full max-w-md">
          <div className="mb-8 lg:hidden">
            <BrandMark companyName={companyName} logoLabel={logoLabel} showLogoLabel={showLogoLabel} logoUrl={logoUrl} />
          </div>
          <Card className="auth-form-panel border-border/70 shadow-[0_16px_50px_-28px_rgba(15,23,42,0.35)]">
            <CardContent className="p-7 sm:p-9">{children}</CardContent>
          </Card>
          <div className="mt-8 lg:hidden">
            <Footer companyName={companyName} legalLinks={legalLinks} />
          </div>
        </div>
      </section>
    </main>
  )
}

const LoginLayout = ({ children, branding }: Props) => {
  const { currentTenant } = useTenant()
  const resolvedBranding = branding === undefined ? currentTenant?.branding : branding
  const companyName = resolvedBranding?.company_name || 'Maintainerd-Auth'
  const logoLabel = resolvedBranding?.logo_label || companyName
  const showLogoLabel = resolvedBranding?.show_logo_label ?? true
  const logoUrl = resolvedBranding?.logo_url
  const layout = resolvedLayout(resolvedBranding?.layout)
  const legalLinks = [
    resolvedBranding?.support_url && { label: 'Support', href: resolvedBranding.support_url },
    resolvedBranding?.privacy_policy_url && { label: 'Privacy', href: resolvedBranding.privacy_policy_url },
    resolvedBranding?.terms_of_service_url && { label: 'Terms', href: resolvedBranding.terms_of_service_url },
  ].filter(Boolean) as FooterProps['legalLinks']

  useEffect(() => {
    if (resolvedBranding?.favicon_url) {
      const link = document.querySelector<HTMLLinkElement>("link[rel*='icon']")
      if (link) link.href = resolvedBranding.favicon_url
    }
  }, [resolvedBranding?.favicon_url])

  const layoutProps = { children, companyName, logoLabel, showLogoLabel, logoUrl, legalLinks }
  if (layout === 'full_page') return <FullPageLayout {...layoutProps} />
  if (layout === 'split') return <SplitLayout {...layoutProps} />
  return <CenteredLayout {...layoutProps} />
}

export default LoginLayout
