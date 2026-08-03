import type { LucideIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Users,
  Shield,
  ShieldCheck,
  ShieldAlert,
  Server,
  KeyRound,
  Layers,
  Settings,
  Lock,
  Clock,
  Globe,
  Activity,
  CheckCircle,
  BookOpen,
  ExternalLink,
  ChevronRight,
} from "lucide-react"
import { useNavigate } from "react-router-dom"
import { useQuery } from "@tanstack/react-query"
import { fetchMFAStatus } from "@/services/api/mfa"
import { DetailsContainer } from "@/components/container"
import { RecentActivityCard } from "./components/RecentActivityCard"
import { useDashboardSummary } from "@/hooks/useDashboard"

const fmt = (n: number) => n.toLocaleString()

const QUICK_ACTIONS = [
  { icon: Users, title: "Users & Roles", desc: "Manage identities and access grants", to: "/users" },
  { icon: Shield, title: "Security Policies", desc: "Review MFA, sessions, and threat controls", to: "/security" },
  { icon: Server, title: "Services & APIs", desc: "Register protected resources", to: "/services" },
  { icon: KeyRound, title: "Identity Providers", desc: "Connect enterprise and social IdPs", to: "/providers/identity" },
]

// Security areas that exist in the backend (/security-settings/*, /ip-restriction-rules)
// AND have a built console page.
const SECURITY_LINKS = [
  { title: "Multi-Factor Auth", desc: "Require a second authentication factor", icon: ShieldCheck, to: "/security?tab=mfa" },
  { title: "Password Policy", desc: "Set password strength requirements", icon: Lock, to: "/security?tab=password" },
  { title: "Sessions", desc: "Session lifetime and concurrency limits", icon: Clock, to: "/security?tab=sessions" },
  { title: "Threat Protection", desc: "Detect and block suspicious activity", icon: ShieldAlert, to: "/security?tab=threat" },
  { title: "IP Restrictions", desc: "Allow or block access by IP range", icon: Globe, to: "/security?tab=ip-restrictions" },
]

// Shared icon-tile + chevron row used by both Quick Actions and Security, so the
// two sections stay visually identical.
function NavRow({
  icon: Icon,
  title,
  desc,
  onClick,
}: {
  icon: LucideIcon
  title: string
  desc: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      data-md-option-card
      onClick={onClick}
      className="flex items-center justify-between rounded-lg border bg-card p-3 text-left transition-colors hover:border-border hover:bg-accent"
    >
      <div className="flex items-center gap-3">
        <div className="flex size-9 items-center justify-center rounded-lg bg-muted text-muted-foreground">
          <Icon className="size-5" />
        </div>
        <div className="space-y-0.5">
          <div className="font-medium">{title}</div>
          <div className="text-sm text-muted-foreground">{desc}</div>
        </div>
      </div>
      <ChevronRight className="size-4 shrink-0 text-muted-foreground" />
    </button>
  )
}

// Reflects the account's MFA posture on the dashboard. When no second factor is
// active it shows an action-oriented amber prompt (securing the account reads as
// a required onboarding step); once a factor is enrolled it collapses to a quiet
// green confirmation rather than a loud banner, so the "secure" state stays calm
// and the prompt only shouts when action is actually needed.
function MfaStatusBanner({ onSetup }: { onSetup: () => void }) {
  const { data, isLoading, isError } = useQuery({ queryKey: ["mfa", "status"], queryFn: fetchMFAStatus, retry: false })
  if (isLoading || isError || !data) return null

  const active = (data.is_totp_enabled ? 1 : 0) + (data.is_sms_available ? 1 : 0) + ((data.webauthn_keys?.length ?? 0) > 0 ? 1 : 0)

  if (active > 0) {
    return (
      <Alert>
        <ShieldCheck className="size-4" style={{ color: "#10b981" }} />
        <AlertDescription>
          <p className="text-sm font-medium">Two-factor authentication is enabled</p>
        </AlertDescription>
      </Alert>
    )
  }

  return (
    <Card className="border-amber-500/40 bg-amber-500/[0.04]">
      <CardContent className="flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-start gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-amber-500/10 text-amber-600">
            <ShieldAlert className="size-5" />
          </div>
          <div className="space-y-0.5">
            <p className="font-medium">Secure your account with two-factor authentication</p>
            <p className="text-sm text-muted-foreground">
              Add a second step at sign-in to protect your account. This is strongly recommended for all administrators.
            </p>
          </div>
        </div>
        <Button onClick={onSetup} className="shrink-0 self-start sm:self-auto">
          <ShieldCheck className="mr-2 size-4" />
          Set up MFA
        </Button>
      </CardContent>
    </Card>
  )
}

const DashboardPage = () => {
  const navigate = useNavigate()
  const { data: summary, isLoading } = useDashboardSummary()

  const to = (path: string) => navigate(`${path}`)

  const stats = [
    {
      label: "Users",
      value: summary ? fmt(summary.users.total) : undefined,
      detail: summary ? `${fmt(summary.users.active)} active / ${fmt(summary.users.pending)} pending` : undefined,
      icon: Users,
    },
    {
      label: "Applications",
      value: summary ? fmt(summary.clients.total) : undefined,
      detail: summary ? `${fmt(summary.clients.active)} active clients` : undefined,
      icon: Layers,
    },
    {
      label: "Sign-ins (24h)",
      value: summary ? fmt(summary.recent_logins_24h) : undefined,
      detail: summary ? "Successful authentication events" : undefined,
      icon: Activity,
    },
    {
      label: "Failed sign-ins (24h)",
      value: summary ? fmt(summary.failed_logins_24h) : undefined,
      detail: summary ? "Watch for brute-force activity" : undefined,
      icon: ShieldAlert,
    },
  ]

  return (
    <DetailsContainer>
      <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="text-sm text-muted-foreground">
          Monitor tenant identity posture, application access, and authentication activity for Maintainerd IAM.
        </p>
      </div>

      {/* MFA status — amber prompt until a factor is enrolled, then a quiet green confirmation */}
      <MfaStatusBanner onSetup={() => to("/account/mfa?from=dashboard")} />

      {/* KPI stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <Card key={stat.label} className="py-0">
            <CardContent className="flex items-center justify-between p-5">
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">{stat.label}</p>
                {isLoading || stat.value === undefined ? (
                  <Skeleton className="h-8 w-16" />
                ) : (
                  <p className="text-2xl font-semibold tracking-tight">{stat.value}</p>
                )}
                {stat.detail && (
                  <p className="text-xs text-muted-foreground">{stat.detail}</p>
                )}
              </div>
              <div className="flex size-10 items-center justify-center rounded-xl bg-muted text-muted-foreground">
                <stat.icon className="size-5" />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Quick Actions */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Settings className="h-4 w-4" />
            IAM Shortcuts
          </CardTitle>
          <CardDescription>
            Common control-plane workflows for identity and access administration
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-3 sm:grid-cols-2">
            {QUICK_ACTIONS.map((action) => (
              <NavRow
                key={action.title}
                icon={action.icon}
                title={action.title}
                desc={action.desc}
                onClick={() => to(action.to)}
              />
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Integration Guide */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <BookOpen className="h-4 w-4" />
            Application Access
          </CardTitle>
          <CardDescription>
            Connect applications to Maintainerd IAM using standard protocols
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <div data-md-listing-nested className="flex items-center justify-between rounded-lg border p-3">
              <div className="space-y-1">
                <div className="font-medium">OAuth 2.0 / OpenID Connect</div>
                <div className="text-sm text-muted-foreground">Standard authentication flows</div>
              </div>
              <CheckCircle className="h-5 w-5 text-muted-foreground" />
            </div>

            <div data-md-listing-nested className="flex items-center justify-between rounded-lg border p-3">
              <div className="space-y-1">
                <div className="font-medium">REST APIs</div>
                <div className="text-sm text-muted-foreground">Direct API integration</div>
              </div>
              <CheckCircle className="h-5 w-5 text-muted-foreground" />
            </div>
          </div>

          <div className="flex flex-wrap gap-3 pt-2">
            <Button onClick={() => to("/clients")}>
              Create OAuth Client
            </Button>
            <Button variant="ghost">
              <ExternalLink className="mr-2 h-4 w-4" />
              View Documentation
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Security */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Shield className="h-4 w-4" />
            Security Posture
          </CardTitle>
          <CardDescription>
            Configure authentication policy, session controls, and access protections
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-3 sm:grid-cols-2">
            {SECURITY_LINKS.map((item) => (
              <NavRow
                key={item.title}
                icon={item.icon}
                title={item.title}
                desc={item.desc}
                onClick={() => to(item.to)}
              />
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Recent activity — latest tenant auth-events; supporting context, kept below the setup actions */}
      <RecentActivityCard onViewAll={() => to("/monitoring?tab=logs")} />
      </div>
    </DetailsContainer>
  )
}

export default DashboardPage
