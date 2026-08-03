import { Link, useLocation, useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'
import {
  User, Shield, Smartphone, Monitor, Settings, Database,
  LogOut, Link2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useMutation } from '@tanstack/react-query'
import { logout } from '@/services/api/auth'
import { useTenant } from '@/hooks/useTenant'
import MaintainedAuthIcon from '@/components/icon/MaintainedAuthIcon'

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

  const logoutMutation = useMutation({
    mutationFn: logout,
    onSuccess: () => navigate('/login'),
  })

  const companyName = currentTenant?.branding?.company_name || 'Maintainerd-Auth'
  const logoLabel = currentTenant?.branding?.logo_label || companyName
  const showLogoLabel = currentTenant?.branding?.show_logo_label ?? true
  const logoUrl = currentTenant?.branding?.logo_url

  return (
    <div className="auth-account-shell min-h-screen">
      {/* Top navigation */}
      <header className="auth-account-header fixed inset-x-0 top-0 z-50 h-14 border-b">
        <div className="mx-auto flex h-14 max-w-5xl items-center px-4">
          <Link to="/account" className="flex items-center gap-2">
            {logoUrl ? (
              <img src={logoUrl} alt={logoLabel || companyName} className="h-7 w-auto object-contain" />
            ) : (
              <MaintainedAuthIcon width={28} height={28} className="shrink-0" />
            )}
            {showLogoLabel && logoLabel && (
              <span className="text-base font-semibold tracking-tight">{logoLabel}</span>
            )}
          </Link>

          <div className="ml-auto flex items-center gap-3">
            <Button
              variant="ghost"
              size="sm"
              className="gap-2 text-current hover:bg-current/10 hover:text-current"
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
          <aside className="w-56 shrink-0 space-y-1 self-start">
            <p className="mb-3 px-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
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
                  className={cn(
                    'flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    active
                      ? 'auth-account-nav-active font-semibold shadow-sm'
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
