import { Link, useLocation, useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'
import {
  User, Shield, Smartphone, Monitor, Settings, Database,
  LogOut, Link2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useMutation } from '@tanstack/react-query'
import { useAuth } from '@/hooks/useAuth'
import { useTenant } from '@/hooks/useTenant'
import { BrandLockup } from '@/components/brand/BrandLockup'
import { authUiTemplatePresentationFromMetadata } from '@/lib/branding/authUiTemplates'

const navItems = [
  { href: '/account/profile', label: 'Profile', icon: User },
  { href: '/account/security', label: 'Security', icon: Shield },
  { href: '/account/sessions', label: 'Sessions', icon: Monitor },
  { href: '/account/devices', label: 'Trusted Devices', icon: Smartphone },
  { href: '/account/mfa', label: 'Two-Factor Auth', icon: Shield },
  { href: '/account/identities', label: 'Linked Accounts', icon: Link2 },
  { href: '/account/settings', label: 'Preferences', icon: Settings },
  { href: '/account/data', label: 'Data & Privacy', icon: Database },
]

const accountMenuClass =
  'flex h-9 items-center gap-2.5 rounded-md border border-transparent px-2 text-sm font-medium transition-colors [&>svg]:size-[18px]'

const accountMenuActiveClass =
  'auth-account-nav-active font-semibold shadow-sm'

export default function AccountLayout({
  children,
  title,
}: {
  children: React.ReactNode
  title?: string
}) {
  const location = useLocation()
  const navigate = useNavigate()
  const { currentTenant } = useTenant()

  const { logout } = useAuth()

  // Sign out through the auth store, not the bare API call.
  //
  // This used to call the logout endpoint directly and redirect from onSuccess,
  // which left two holes. The store never saw the logout, so Redux still held
  // isAuthenticated: true and a stale account — the shell kept rendering as a
  // signed-in user. And because onSuccess only fires on a 2xx, any error
  // response (most commonly a 401 when the session was already revoked from the
  // other surface in this browser) skipped the redirect entirely and left the
  // user sitting on an account page that no longer had a session behind it.
  //
  // The store's logout clears auth state on BOTH fulfilled and rejected, so the
  // navigation belongs in onSettled: whatever the server said, this browser is
  // signed out locally and must land on /login.
  const logoutMutation = useMutation({
    mutationFn: logout,
    onSettled: () => navigate('/login', { replace: true }),
  })

  const companyName = currentTenant?.branding?.company_name || 'Maintainerd'
  const logoLabel =
    currentTenant?.branding?.identity_logo_label ||
    currentTenant?.branding?.logo_label ||
    companyName
  const showLogoLabel =
    currentTenant?.branding?.identity_show_logo_label ??
    currentTenant?.branding?.show_logo_label ??
    true
  const presentation = authUiTemplatePresentationFromMetadata(currentTenant?.branding?.metadata)

  return (
    <div data-auth-identity-account-shell className="auth-account-shell min-h-screen">
      {/* Top navigation */}
      <header data-md-top-panel className="auth-account-header fixed inset-x-0 top-0 z-50 h-14 border-b">
        <div className="mx-auto flex h-14 max-w-5xl items-center px-4">
          <Link to="/account/profile" className="flex items-center gap-2">
            <BrandLockup
              companyName={companyName}
              logoLabel={logoLabel}
              showLogoLabel={showLogoLabel}
              logoClassName="size-7"
              logoDetail={presentation.logoDetail}
              topPanelLabel
            />
          </Link>

          <div className="ml-auto flex items-center gap-3">
            <Button
              variant="ghost"
              size="sm"
              data-md-top-logout
              className="gap-2"
              onClick={() => logoutMutation.mutate()}
              disabled={logoutMutation.isPending}
            >
              <LogOut className="size-4" />
              <span className="hidden sm:inline">Sign out</span>
            </Button>
          </div>
        </div>
      </header>

      <div className="mx-auto max-w-5xl px-4 pb-8 pt-20">
        <div className="flex gap-6">
          {/* Sidebar */}
          <aside data-md-sidebar className="w-56 shrink-0 space-y-1 self-start">
            <p
              data-md-sidebar-section
              data-md-sidebar-section-label
              className="mb-3 flex items-center px-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground"
            >
              Account
            </p>
            {navItems.map(({ href, label, icon: Icon }) => {
              const active =
                location.pathname === href ||
                location.pathname.startsWith(href + '/')
              return (
                <Link
                  key={href}
                  to={href}
                  data-md-sidebar-item
                  data-active={active ? 'true' : undefined}
                  className={cn(
                    accountMenuClass,
                    active
                      ? accountMenuActiveClass
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                  )}
                >
                  <Icon className="size-4 shrink-0" />
                  {label}
                </Link>
              )
            })}
          </aside>

          {/* Main content */}
          <main className="min-w-0 flex-1">
            {title && <h1 className="mb-6 text-2xl font-bold">{title}</h1>}
            {children}
          </main>
        </div>
      </div>
    </div>
  )
}
