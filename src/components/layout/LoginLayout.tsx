import { useEffect, type ReactNode } from 'react'
import { ExternalLink } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { useTenant } from '@/hooks/useTenant'
import {
  authUiTemplateIdFromMetadata,
  authUiTemplatePresentationFromMetadata,
  safeAuthTemplateImageUrl,
  type AuthTemplatePresentation,
  type LogoPlacement,
  type SplitVisualStyle,
} from '@/lib/branding/authUiTemplates'
import type { BrandingLayout, BrandingPublic } from '@/services/api/tenants/types'
import { resolveBrandingLogoUrl } from '@/utils/branding'
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

type LayoutProps = {
  children: ReactNode
  companyName: string
  logoLabel: string
  showLogoLabel: boolean
  logoUrl?: string
  legalLinks: FooterProps['legalLinks']
  presentation: AuthTemplatePresentation
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

function LoginFormPanel({
  children,
  companyName,
  logoLabel,
  showLogoLabel,
  logoUrl,
  logoPlacement = 'above-form',
  embedded = false,
}: {
  children: ReactNode
  companyName: string
  logoLabel: string
  showLogoLabel: boolean
  logoUrl?: string
  logoPlacement?: LogoPlacement | 'none'
  embedded?: boolean
}) {
  const showLogo = logoPlacement !== 'none'
  const logo = showLogo ? (
    <div className={logoPlacement === 'inside-form' ? 'mb-7' : 'mb-8'}>
      <BrandMark companyName={companyName} logoLabel={logoLabel} showLogoLabel={showLogoLabel} logoUrl={logoUrl} />
    </div>
  ) : null

  if (embedded) {
    return (
      <div className="w-full">
        {logo}
        {children}
      </div>
    )
  }

  return (
    <div className="w-full max-w-md">
      {logoPlacement === 'above-form' && logo}
      <Card
        data-auth-identity-card
        className="auth-form-panel border-border/70 shadow-[0_1px_2px_rgba(15,23,42,0.04),0_16px_40px_-20px_rgba(15,23,42,0.25)]"
      >
        <CardContent className="p-7 sm:p-9">
          {logoPlacement === 'inside-form' && logo}
          {children}
        </CardContent>
      </Card>
    </div>
  )
}

function SplitVisualDesign({ visualStyle, imageUrl }: { visualStyle: SplitVisualStyle; imageUrl: string }) {
  const image = safeAuthTemplateImageUrl(imageUrl)

  if (visualStyle === 'image' && image) {
    return (
      <>
        <img src={image} alt="" className="pointer-events-none absolute inset-0 size-full object-cover" />
        <div className="auth-split-visual-overlay pointer-events-none absolute inset-0" />
      </>
    )
  }

  if (visualStyle === 'identity-mesh') {
    return (
      <>
        <div className="auth-split-mesh pointer-events-none absolute inset-0" />
        {[['12%', '18%'], ['36%', '12%'], ['64%', '20%'], ['84%', '42%'], ['66%', '72%'], ['34%', '80%'], ['14%', '56%'], ['48%', '48%']].map(([left, top], index) => (
          <span
            key={`${left}-${top}`}
            className="auth-split-node pointer-events-none absolute rounded-full border"
            style={{ left, top, width: index === 7 ? '46px' : '24px', height: index === 7 ? '46px' : '24px' }}
          />
        ))}
      </>
    )
  }

  if (visualStyle === 'access-grid') {
    return (
      <>
        <div className="auth-split-grid pointer-events-none absolute inset-0" />
        {[0, 1, 2, 3, 4, 5].map((item) => (
          <span
            key={item}
            className="auth-split-tile pointer-events-none absolute rounded-md border"
            style={{
              left: `${14 + (item % 3) * 22}%`,
              top: `${18 + Math.floor(item / 3) * 28}%`,
              width: item === 1 ? '92px' : '68px',
              height: item === 1 ? '46px' : '34px',
            }}
          />
        ))}
      </>
    )
  }

  if (visualStyle === 'security-radar' || visualStyle === 'session-orbit') {
    return (
      <>
        <div className="auth-split-radar pointer-events-none absolute left-1/2 top-1/2 size-[34rem] -translate-x-1/2 -translate-y-1/2 rounded-full" />
        <span className="auth-split-axis pointer-events-none absolute left-0 top-1/2 h-px w-full" />
        <span className="auth-split-axis pointer-events-none absolute left-1/2 top-0 h-full w-px" />
      </>
    )
  }

  if (visualStyle === 'trust-circuit' || visualStyle === 'audit-trail') {
    return (
      <>
        <div className="auth-split-circuit pointer-events-none absolute inset-0" />
        {[14, 26, 38, 50, 62, 74].map((top, index) => (
          <span
            key={top}
            className="auth-split-trace pointer-events-none absolute h-px"
            style={{ left: index % 2 === 0 ? '10%' : '22%', right: index % 2 === 0 ? '18%' : '8%', top: `${top}%` }}
          />
        ))}
      </>
    )
  }

  return (
    <>
      <div className="auth-split-decoration-light pointer-events-none absolute -right-24 -top-24 size-80 rounded-full opacity-10" />
      <div className="auth-split-decoration-dark pointer-events-none absolute -bottom-40 -left-20 size-96 rounded-full opacity-10" />
    </>
  )
}

function VisualPanel({
  companyName,
  logoLabel,
  showLogoLabel,
  logoUrl,
  legalLinks,
  presentation,
  showBrand,
  visibleFrom = 'lg',
}: Omit<LayoutProps, 'children'> & { showBrand: boolean; visibleFrom?: 'md' | 'lg' }) {
  return (
    <section
      data-testid="split-brand-panel"
      className={`auth-split-brand-panel relative hidden min-h-[280px] overflow-hidden p-8 ${visibleFrom === 'md' ? 'md:flex' : 'lg:flex'} flex-col justify-between`}
    >
      <SplitVisualDesign
        visualStyle={presentation.splitShowcaseVisualStyle}
        imageUrl={presentation.splitShowcaseImageUrl}
      />
      {showBrand ? (
        <div className="relative">
          <BrandMark companyName={companyName} logoLabel={logoLabel} showLogoLabel={showLogoLabel} logoUrl={logoUrl} panel />
        </div>
      ) : (
        <div className="relative" />
      )}
      <div className="relative max-w-sm space-y-3">
        <p className="text-3xl font-semibold leading-tight">{presentation.splitShowcaseTitle}</p>
        <p className="text-sm opacity-80">{presentation.splitShowcaseSubtitle}</p>
      </div>
      <div className="relative">
        <Footer companyName={companyName} legalLinks={legalLinks} panel />
      </div>
    </section>
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
  presentation,
}: LayoutProps) {
  const logoPlacement = presentation.logoPlacement
  return (
    <main
      data-auth-identity-shell
      data-auth-ui-template="centered-card"
      data-layout="centered"
      className="relative flex min-h-svh flex-col items-center justify-center overflow-hidden px-4 py-12"
    >
      <div className="auth-page-background pointer-events-none absolute inset-0" />
      <div className="relative z-10 w-full max-w-md">
        <LoginFormPanel
          companyName={companyName}
          logoLabel={logoLabel}
          showLogoLabel={showLogoLabel}
          logoUrl={logoUrl}
          logoPlacement={logoPlacement}
        >
          {children}
        </LoginFormPanel>
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
  presentation,
}: LayoutProps) {
  return (
    <main
      data-auth-identity-shell
      data-auth-ui-template="stepper-flow"
      data-layout="full_page"
      className="relative flex min-h-svh items-center justify-center overflow-hidden p-4 sm:p-6 lg:p-10"
    >
      <div className="auth-page-background pointer-events-none absolute inset-0" />
      <div
        data-auth-identity-card
        className="auth-full-page-panel relative z-10 grid min-h-[calc(100svh-2rem)] w-full max-w-sm overflow-hidden rounded-xl border shadow-2xl md:min-h-[440px] md:max-w-4xl md:grid-cols-2 sm:min-h-[calc(100svh-3rem)] lg:min-h-[520px]"
      >
        <section className="flex min-h-[440px] items-center justify-center px-6 py-10 md:px-8">
          <div className="w-full max-w-md">
            <LoginFormPanel
              companyName={companyName}
              logoLabel={logoLabel}
              showLogoLabel={showLogoLabel}
              logoUrl={logoUrl}
              logoPlacement="inside-form"
              embedded
            >
              {children}
            </LoginFormPanel>
          </div>
        </section>
        <VisualPanel
          companyName={companyName}
          logoLabel={logoLabel}
          showLogoLabel={showLogoLabel}
          logoUrl={logoUrl}
          legalLinks={legalLinks}
          presentation={presentation}
          showBrand={false}
          visibleFrom="md"
        />
      </div>
    </main>
  )
}

function SplitShowcaseLayout({
  children,
  companyName,
  logoLabel,
  showLogoLabel,
  logoUrl,
  legalLinks,
  presentation,
}: LayoutProps) {
  return (
    <main
      data-auth-identity-shell
      data-auth-ui-template="split-showcase"
      data-layout="split"
      className="auth-form-panel grid min-h-svh lg:grid-cols-[minmax(0,1.1fr)_minmax(30rem,0.9fr)]"
    >
      <VisualPanel
        companyName={companyName}
        logoLabel={logoLabel}
        showLogoLabel={showLogoLabel}
        logoUrl={logoUrl}
        legalLinks={legalLinks}
        presentation={presentation}
        showBrand
      />

      <section
        data-auth-identity-card
        className="auth-form-panel flex min-h-svh items-center justify-center px-6 py-12 sm:px-10 lg:px-14"
      >
        <div className="w-full max-w-md">
          <div className="mb-8 lg:hidden">
            <BrandMark companyName={companyName} logoLabel={logoLabel} showLogoLabel={showLogoLabel} logoUrl={logoUrl} />
          </div>
          <LoginFormPanel
            companyName={companyName}
            logoLabel={logoLabel}
            showLogoLabel={showLogoLabel}
            logoUrl={logoUrl}
            logoPlacement="none"
          >
            {children}
          </LoginFormPanel>
          <div className="mt-8 lg:hidden">
            <Footer companyName={companyName} legalLinks={legalLinks} />
          </div>
        </div>
      </section>
    </main>
  )
}

function EditorialCoverLayout({
  children,
  companyName,
  logoLabel,
  showLogoLabel,
  logoUrl,
  legalLinks,
  presentation,
}: LayoutProps) {
  return (
    <main
      data-auth-identity-shell
      data-auth-ui-template="editorial-cover"
      data-layout="split"
      className="auth-form-panel grid min-h-svh lg:grid-cols-[minmax(30rem,0.9fr)_minmax(0,1.1fr)]"
    >
      <section
        data-auth-identity-card
        className="auth-form-panel flex min-h-svh flex-col px-6 py-8 sm:px-10 lg:px-14"
      >
        <div className="shrink-0">
          <BrandMark companyName={companyName} logoLabel={logoLabel} showLogoLabel={showLogoLabel} logoUrl={logoUrl} />
        </div>
        <div className="flex flex-1 items-center justify-center py-8">
          <LoginFormPanel
            companyName={companyName}
            logoLabel={logoLabel}
            showLogoLabel={showLogoLabel}
            logoUrl={logoUrl}
            logoPlacement="none"
          >
            {children}
          </LoginFormPanel>
        </div>
        <div className="shrink-0">
          <Footer companyName={companyName} legalLinks={legalLinks} />
        </div>
      </section>
      <VisualPanel
        companyName={companyName}
        logoLabel={logoLabel}
        showLogoLabel={showLogoLabel}
        logoUrl={logoUrl}
        legalLinks={legalLinks}
        presentation={presentation}
        showBrand={false}
      />
    </main>
  )
}

const LoginLayout = ({ children, branding }: Props) => {
  const { currentTenant } = useTenant()
  const resolvedBranding = branding === undefined ? currentTenant?.branding : branding
  const companyName = resolvedBranding?.company_name || 'Maintainerd-Auth'
  const logoLabel = resolvedBranding?.logo_label || companyName
  const showLogoLabel = resolvedBranding?.show_logo_label ?? true
  const logoUrl = resolveBrandingLogoUrl(resolvedBranding?.logo_url) ?? undefined
  const layout = resolvedLayout(resolvedBranding?.layout)
  const templateId = authUiTemplateIdFromMetadata(resolvedBranding?.metadata, layout)
  const presentation = authUiTemplatePresentationFromMetadata(resolvedBranding?.metadata)
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

  const layoutProps = { children, companyName, logoLabel, showLogoLabel, logoUrl, legalLinks, presentation }
  if (templateId === 'stepper-flow') return <FullPageLayout {...layoutProps} />
  if (templateId === 'editorial-cover') return <EditorialCoverLayout {...layoutProps} />
  if (templateId === 'split-showcase') return <SplitShowcaseLayout {...layoutProps} />
  return <CenteredLayout {...layoutProps} />
}

export default LoginLayout
