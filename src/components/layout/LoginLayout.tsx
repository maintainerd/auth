import { useEffect, type ReactNode } from 'react'
import { ExternalLink } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { BrandLockup } from '@/components/brand/BrandLockup'
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

type Props = {
  children: ReactNode
  branding?: BrandingPublic | null
}

type BrandMarkProps = {
  companyName: string
  logoLabel: string
  showLogoLabel: boolean
  logoUrl?: string
  logoDetail: string
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
  logoDetail: string
  legalLinks: FooterProps['legalLinks']
  presentation: AuthTemplatePresentation
}

function resolvedLayout(layout: BrandingPublic['layout'] | undefined): BrandingLayout {
  return layout === 'full_page' || layout === 'split' ? layout : 'centered'
}

function BrandMark({ companyName, logoLabel, showLogoLabel, logoUrl, logoDetail, panel = false }: BrandMarkProps) {
  return (
    <BrandLockup
      companyName={companyName}
      logoLabel={logoLabel}
      showLogoLabel={showLogoLabel}
      logoUrl={logoUrl}
      logoDetail={logoDetail}
      panel={panel}
      centered
    />
  )
}

function LoginFormPanel({
  children,
  companyName,
  logoLabel,
  showLogoLabel,
  logoUrl,
  logoDetail,
  logoPlacement = 'above-form',
  embedded = false,
}: {
  children: ReactNode
  companyName: string
  logoLabel: string
  showLogoLabel: boolean
  logoUrl?: string
  logoDetail: string
  logoPlacement?: LogoPlacement | 'none'
  embedded?: boolean
}) {
  const showLogo = logoPlacement !== 'none'
  const logo = showLogo ? (
    <div className={logoPlacement === 'inside-form' ? 'mb-7' : 'mb-8'}>
      <BrandMark
        companyName={companyName}
        logoLabel={logoLabel}
        showLogoLabel={showLogoLabel}
        logoUrl={logoUrl}
        logoDetail={logoDetail}
      />
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

  if (visualStyle === 'image') {
    return (
      <>
        {image ? (
          <img src={image} alt="" className="pointer-events-none absolute inset-0 size-full object-cover" />
        ) : (
          <div className="auth-split-image-fallback pointer-events-none absolute inset-0" />
        )}
        <div className="auth-split-visual-overlay pointer-events-none absolute inset-0" />
      </>
    )
  }

  if (visualStyle === 'identity-mesh') {
    const lines = [
      ['left-[12%] top-[18%] h-px w-[72%] rotate-[17deg]', 'high'],
      ['left-[13%] top-[56%] h-px w-[68%] -rotate-[13deg]', 'medium'],
      ['left-[34%] top-[80%] h-px w-[48%] -rotate-[34deg]', 'low'],
      ['left-[48%] top-[16%] h-[70%] w-px rotate-[12deg]', 'low'],
    ] as const
    const nodes = [
      ['12%', '18%', '26px'],
      ['36%', '12%', '18px'],
      ['64%', '20%', '34px'],
      ['84%', '42%', '22px'],
      ['66%', '72%', '26px'],
      ['34%', '80%', '20px'],
      ['14%', '56%', '32px'],
      ['48%', '48%', '46px'],
    ] as const

    return (
      <>
        <div className="auth-split-mesh-bg pointer-events-none absolute inset-0" />
        <div className="auth-split-mesh-grid pointer-events-none absolute inset-0 opacity-30" />
        {lines.map(([lineClassName, tone]) => (
          <span
            key={lineClassName}
            className={`auth-split-link pointer-events-none absolute ${lineClassName}`}
            data-tone={tone}
          />
        ))}
        {nodes.map(([left, top, size], index) => (
          <span
            key={`${left}-${top}`}
            className="auth-split-node pointer-events-none absolute rounded-full border"
            data-tone={index % 2 === 0 ? 'light' : 'dark'}
            style={{ left, top, width: size, height: size }}
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
            data-featured={item === 1 ? true : undefined}
            data-tone={item === 1 ? 'light' : 'text'}
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

  if (visualStyle === 'security-radar') {
    const dots = [
      ['24%', '28%', 'light', '16px'],
      ['72%', '22%', 'dark', '10px'],
      ['80%', '60%', 'text', '12px'],
      ['44%', '70%', 'light', '9px'],
      ['18%', '58%', 'text', '7px'],
    ] as const

    return (
      <>
        <div className="auth-split-radar-field pointer-events-none absolute inset-0" />
        <div className="auth-split-radar pointer-events-none absolute left-1/2 top-1/2 size-[34rem] -translate-x-1/2 -translate-y-1/2 rounded-full" />
        <span className="auth-split-axis pointer-events-none absolute left-0 top-1/2 h-px w-full" />
        <span className="auth-split-axis pointer-events-none absolute left-1/2 top-0 h-full w-px" />
        {dots.map(([left, top, tone, size]) => (
          <span
            key={`${left}-${top}`}
            className="auth-split-radar-dot pointer-events-none absolute rounded-full"
            data-tone={tone}
            style={{ left, top, width: size, height: size }}
          />
        ))}
        <div className="auth-split-radar-card pointer-events-none absolute bottom-[14%] left-[12%] right-[12%] rounded-md border p-3">
          <div className="grid grid-cols-4 gap-2">
            {[0, 1, 2, 3].map((item) => (
              <span key={item} className="h-9 rounded-sm border" data-featured={item === 2 ? true : undefined} />
            ))}
          </div>
        </div>
      </>
    )
  }

  if (visualStyle === 'trust-circuit') {
    return (
      <>
        <div className="auth-split-circuit pointer-events-none absolute inset-0" />
        {[14, 26, 38, 50, 62, 74].map((top, index) => (
          <div
            key={top}
            className="auth-split-trace-row pointer-events-none absolute"
            style={{
              left: index % 2 === 0 ? '10%' : '22%',
              right: index % 2 === 0 ? '18%' : '8%',
              top: `${top}%`,
            }}
          >
            <span className="absolute left-0 right-0 top-1/2 h-px" data-index={index} />
            <span
              className="absolute left-0 top-1/2 size-4 -translate-y-1/2 rounded-sm border"
              data-tone={index % 2 === 0 ? 'light' : 'dark'}
            />
            <span
              className="absolute right-0 top-1/2 size-4 -translate-y-1/2 rounded-sm border"
              data-tone={index % 2 === 0 ? 'dark' : 'light'}
            />
          </div>
        ))}
        {[0, 1, 2, 3].map((item) => (
          <span
            key={item}
            className="auth-split-circuit-card pointer-events-none absolute rounded-md border"
            data-tone={item % 2 === 0 ? 'light' : 'dark'}
            style={{
              left: `${18 + item * 17}%`,
              top: `${20 + (item % 2) * 42}%`,
              width: '76px',
              height: '44px',
            }}
          />
        ))}
      </>
    )
  }

  if (visualStyle === 'audit-trail') {
    return (
      <>
        <div className="auth-split-audit-field pointer-events-none absolute inset-0" />
        <span className="auth-split-timeline pointer-events-none absolute bottom-[10%] left-[22%] top-[12%] w-px" data-tone="strong" />
        <span className="auth-split-timeline pointer-events-none absolute bottom-[16%] right-[18%] top-[18%] w-px" />
        {[0, 1, 2, 3, 4].map((item) => (
          <div
            key={item}
            className="auth-split-audit-card pointer-events-none absolute rounded-md border p-3"
            data-tone={item % 2 === 0 ? 'light' : 'dark'}
            style={{
              left: item % 2 === 0 ? '26%' : '44%',
              right: item % 2 === 0 ? '16%' : '8%',
              top: `${13 + item * 15}%`,
            }}
          >
            <div className="flex items-center gap-3">
              <span className="size-5 rounded-full border" />
              <span className="h-2 flex-1 rounded-full" />
              <span className="h-2 w-14 rounded-full" />
            </div>
          </div>
        ))}
      </>
    )
  }

  if (visualStyle === 'session-orbit') {
    const tiles = [
      ['18%', '20%', '52px', '34px', 'dark'],
      ['72%', '18%', '34px', '56px', 'light'],
      ['78%', '66%', '54px', '34px', 'dark'],
      ['18%', '68%', '58px', '34px', 'light'],
      ['50%', '10%', '42px', '30px', 'text'],
      ['46%', '80%', '42px', '30px', 'text'],
    ] as const

    return (
      <>
        <div className="auth-split-orbit-field pointer-events-none absolute inset-0" />
        <div className="auth-split-orbit-core pointer-events-none absolute left-1/2 top-1/2 w-36 -translate-x-1/2 -translate-y-1/2 rounded-md border p-3">
          <div className="mb-3 h-2 w-20 rounded-full" />
          <div className="space-y-1.5">
            <span className="block h-1.5 w-full rounded-full" />
            <span className="block h-1.5 w-2/3 rounded-full" />
          </div>
        </div>
        {tiles.map(([left, top, width, height, tone]) => (
          <span
            key={`${left}-${top}`}
            className="auth-split-orbit-tile pointer-events-none absolute rounded-md border"
            data-tone={tone}
            style={{ left, top, width, height }}
          />
        ))}
        <span className="auth-split-orbit-line pointer-events-none absolute left-[22%] top-[29%] h-px w-[56%] rotate-[14deg]" data-tone="strong" />
        <span className="auth-split-orbit-line pointer-events-none absolute left-[22%] top-[70%] h-px w-[56%] -rotate-[12deg]" />
      </>
    )
  }

  return (
    <>
      <span className="auth-split-decoration-light pointer-events-none absolute -right-24 -top-24 size-80 rounded-full opacity-15" />
      <span className="auth-split-decoration-dark pointer-events-none absolute -bottom-40 -left-20 size-96 rounded-full opacity-15" />
      <div className="auth-split-visual-overlay pointer-events-none absolute inset-0 opacity-25" />
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
}: Omit<LayoutProps, 'children' | 'logoDetail'> & { showBrand: boolean; visibleFrom?: 'md' | 'lg' }) {
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
          <BrandMark
            companyName={companyName}
            logoLabel={logoLabel}
            showLogoLabel={showLogoLabel}
            logoUrl={logoUrl}
            logoDetail={presentation.logoDetail}
            panel
          />
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
          logoDetail={presentation.logoDetail}
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
      className="relative flex min-h-svh flex-col items-center justify-center overflow-hidden p-6 md:p-10"
    >
      <div className="auth-page-background pointer-events-none absolute inset-0" />
      <div
        data-auth-identity-card
        className="auth-full-page-panel relative z-10 grid w-full max-w-sm items-stretch overflow-hidden rounded-md border md:min-h-[440px] md:max-w-4xl md:grid-cols-2"
      >
        <section className="flex min-h-[440px] items-center justify-center p-6 md:p-8">
          <div className="w-full max-w-md">
            <LoginFormPanel
              companyName={companyName}
              logoLabel={logoLabel}
              showLogoLabel={showLogoLabel}
              logoUrl={logoUrl}
              logoDetail={presentation.logoDetail}
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
      className="auth-form-panel grid min-h-svh lg:grid-cols-[0.95fr_1.05fr]"
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
        className="auth-form-panel flex min-h-svh items-center justify-center p-6"
      >
        <div className="w-full max-w-md">
          <div className="mb-8 lg:hidden">
            <BrandMark
              companyName={companyName}
              logoLabel={logoLabel}
              showLogoLabel={showLogoLabel}
              logoUrl={logoUrl}
              logoDetail={presentation.logoDetail}
            />
          </div>
          <LoginFormPanel
            companyName={companyName}
            logoLabel={logoLabel}
            showLogoLabel={showLogoLabel}
            logoUrl={logoUrl}
            logoDetail={presentation.logoDetail}
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
      className="auth-form-panel grid min-h-svh lg:grid-cols-[1.05fr_0.95fr]"
    >
      <section
        data-auth-identity-card
        className="auth-form-panel flex min-h-svh flex-col p-6"
      >
        <div className="shrink-0">
          <BrandMark
            companyName={companyName}
            logoLabel={logoLabel}
            showLogoLabel={showLogoLabel}
            logoUrl={logoUrl}
            logoDetail={presentation.logoDetail}
          />
        </div>
        <div className="flex flex-1 items-center justify-center py-6">
          <LoginFormPanel
            companyName={companyName}
            logoLabel={logoLabel}
            showLogoLabel={showLogoLabel}
            logoUrl={logoUrl}
            logoDetail={presentation.logoDetail}
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
  const companyName = resolvedBranding?.company_name || 'Maintainerd'
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

  const layoutProps = { children, companyName, logoLabel, showLogoLabel, logoUrl, logoDetail: presentation.logoDetail, legalLinks, presentation }
  if (templateId === 'stepper-flow') return <FullPageLayout {...layoutProps} />
  if (templateId === 'editorial-cover') return <EditorialCoverLayout {...layoutProps} />
  if (templateId === 'split-showcase') return <SplitShowcaseLayout {...layoutProps} />
  return <CenteredLayout {...layoutProps} />
}

export default LoginLayout
