import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { User, Shield, Monitor, Smartphone, Settings, Database, Link2, ShieldCheck, CheckCircle2 } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { Card, CardContent } from '@/components/ui/card'
import { fetchAccountInfo } from '@/services/api/account'

const sections = [
  { href: '/account/profile', label: 'Profile', description: 'Names and display preferences', icon: User },
  { href: '/account/security', label: 'Security', description: 'Password and email address', icon: Shield },
  { href: '/account/sessions', label: 'Sessions', description: 'Active sign-ins and history', icon: Monitor },
  { href: '/account/devices', label: 'Trusted Devices', description: 'Devices that skip MFA', icon: Smartphone },
  { href: '/account/mfa', label: 'Two-Factor Auth', description: 'TOTP, passkeys, SMS', icon: ShieldCheck },
  { href: '/account/identities', label: 'Linked Accounts', description: 'Social and external logins', icon: Link2 },
  { href: '/account/settings', label: 'Preferences', description: 'Language, timezone and notifications', icon: Settings },
  { href: '/account/data', label: 'Data & Privacy', description: 'Export or delete your data', icon: Database },
]

export default function AccountOverviewPage() {
  const { data: account, isLoading } = useQuery({
    queryKey: ['account', 'info'],
    queryFn: fetchAccountInfo,
  })

  const initials = account?.email?.slice(0, 2).toUpperCase() ?? '??'

  return (
    <AccountLayout title="Account">
      <div className="space-y-6">
        {/* Account info header */}
        <div data-md-listing-item className="flex items-center gap-4 rounded-lg border bg-card p-4 shadow-sm">
          <div className="flex size-14 shrink-0 items-center justify-center rounded-full bg-primary/10 text-lg font-semibold text-primary">
            {isLoading ? '…' : initials}
          </div>
          {isLoading ? (
            <div className="space-y-1.5">
              <div className="h-4 w-40 animate-pulse rounded bg-muted" />
              <div className="h-3 w-28 animate-pulse rounded bg-muted" />
            </div>
          ) : account ? (
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-semibold">{account.email}</p>
                <span
                  className={`rounded px-1.5 py-0.5 text-xs font-medium ${
                    account.status === 'active'
                      ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                      : 'bg-muted text-muted-foreground'
                  }`}
                >
                  {account.status}
                </span>
              </div>
              <div className="mt-0.5 flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
                {account.username && <span>@{account.username}</span>}
                {account.is_email_verified && (
                  <span className="flex items-center gap-1 text-emerald-600 dark:text-emerald-400">
                    <CheckCircle2 className="size-3.5" />
                    Verified
                  </span>
                )}
              </div>
            </div>
          ) : null}
        </div>

        {/* Section cards */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {sections.map(({ href, label, description, icon: Icon }) => (
            <Link key={href} to={href} className="block">
              <Card data-md-listing-item className="h-full transition-colors hover:bg-muted/50">
                <CardContent className="flex items-center gap-4 p-4">
                  <div data-md-listing-icon className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                    <Icon className="size-5 text-muted-foreground" />
                  </div>
                  <div>
                    <p className="text-sm font-medium">{label}</p>
                    <p className="text-xs text-muted-foreground">{description}</p>
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      </div>
    </AccountLayout>
  )
}
