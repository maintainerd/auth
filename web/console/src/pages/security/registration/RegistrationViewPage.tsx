import { useNavigate } from "react-router-dom"
import { UserPlus, Globe, Shield, Clock, Settings, Check, X, Hash } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { DetailsContainer } from "@/components/container"
import { PageHeader } from "@/components/layout"
import { Skeleton } from "@/components/ui/skeleton"
import { useRegistrationConfig } from "@/hooks/useRegistrationConfig"
import type { LucideIcon } from "lucide-react"
import type { ReactNode } from "react"

function Attr({ icon: Icon, label, value }: { icon: LucideIcon; label: string; value: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">
        <Icon className="size-3.5" />
        {label}
      </div>
      <div className="text-sm text-foreground">{value}</div>
    </div>
  )
}

function SectionLabel({ children }: { children: ReactNode }) {
  return <h3 className="text-sm font-semibold text-foreground">{children}</h3>
}

export default function RegistrationViewPage({ standalone = true }: { standalone?: boolean }) {
  const navigate = useNavigate()
  const { data, isLoading, isError } = useRegistrationConfig()
  const configureUrl = `/security/registration/configure`

  const content = (
    <>
      {isLoading && (
        <Card>
          <CardContent className="space-y-4 pt-6">
            <Skeleton className="h-5 w-48" />
            <div className="grid gap-4 md:grid-cols-3">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full" />
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {isError && !isLoading && (
        <Card>
          <CardContent className="py-12 text-center text-sm text-destructive">
            Failed to load registration configuration.
          </CardContent>
        </Card>
      )}

      {!isLoading && data && (
        <Card>
          <CardContent>
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div className="flex min-w-0 items-center gap-4">
                <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  <UserPlus className="size-6" />
                </div>
                <div className="min-w-0 space-y-1">
                  <div className="flex flex-wrap items-center gap-2.5">
                    <h1 className="text-lg font-semibold tracking-tight">Registration</h1>
                    <Badge
                      variant="secondary"
                      className={data.self_registration_enabled
                        ? "gap-1 border-emerald-500/30 bg-emerald-500/10 text-emerald-600"
                        : "gap-1"
                      }
                    >
                      {data.self_registration_enabled ? "Open" : "Invite only"}
                    </Badge>
                  </div>
                </div>
              </div>
              <Button variant="outline" size="sm" className="h-9 gap-2" onClick={() => navigate(configureUrl)}>
                <Settings className="size-4" />
                Configure
              </Button>
            </div>

            <Separator className="my-5" />
            <SectionLabel>Registration Settings</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={data.self_registration_enabled ? Check : X} label="Self Registration" value={data.self_registration_enabled ? "Enabled" : "Disabled"} />
              <Attr icon={X} label="CAPTCHA" value="Coming soon" />
              <Attr icon={Hash} label="Rate Limit" value={`${data.registration_rate_limit_per_ip_per_hour} per IP per hour`} />
            </div>

            <Separator className="my-5" />
            <SectionLabel>Domain Rules</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2">
              <Attr icon={Globe} label="Allowed Domains" value={
                (data.allowed_email_domains ?? []).length > 0
                  ? <div className="flex flex-wrap gap-1">{data.allowed_email_domains.map((d) => <Badge key={d} variant="secondary" className="font-mono text-xs">{d}</Badge>)}</div>
                  : "All domains allowed"
              } />
              <Attr icon={Shield} label="Blocked Domains" value={
                (data.blocked_email_domains ?? []).length > 0
                  ? <div className="flex flex-wrap gap-1">{data.blocked_email_domains.map((d) => <Badge key={d} variant="destructive" className="font-mono text-xs">{d}</Badge>)}</div>
                  : "None"
              } />
            </div>

            <Separator className="my-5" />
            <SectionLabel>Verification</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={data.require_email_verification ? Check : X} label="Email Verification" value={data.require_email_verification ? "Required" : "Not required"} />
              <Attr icon={data.require_phone_verification ? Check : X} label="Phone Verification" value={data.require_phone_verification ? "Required" : "Not required"} />
              <Attr icon={data.auto_confirm_enabled ? Check : X} label="Auto-Confirm" value={data.auto_confirm_enabled ? "Enabled" : "Disabled"} />
              <Attr icon={Clock} label="Verification Token TTL" value={`${data.verification_token_ttl_hours} hours`} />
            </div>
          </CardContent>
        </Card>
      )}
    </>
  )

  if (!standalone) return content

  return (
    <DetailsContainer>
      <div className="flex flex-col gap-6">
        <PageHeader
          title="Registration Settings"
          description="Self-service registration policy, domain control, and required contact information."
        />
        {content}
      </div>
    </DetailsContainer>
  )
}
