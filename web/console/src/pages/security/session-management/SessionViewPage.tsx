import { useNavigate } from "react-router-dom"
import { Clock, Hash, Shield, Settings, Check, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { DetailsContainer } from "@/components/container"
import { PageHeader } from "@/components/layout"
import { Skeleton } from "@/components/ui/skeleton"
import { useSessionSettings } from "@/hooks/useSessionSettings"
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

export default function SessionViewPage({ standalone = true }: { standalone?: boolean }) {
  const navigate = useNavigate()
  const { data, isLoading, isError } = useSessionSettings()
  const configureUrl = `/security/session/configure`

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
            Failed to load session settings.
          </CardContent>
        </Card>
      )}

      {!isLoading && data && (
        <Card>
          <CardContent>
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div className="flex min-w-0 items-center gap-4">
                <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  <Clock className="size-6" />
                </div>
                <div className="min-w-0 space-y-1">
                  <h1 className="text-lg font-semibold tracking-tight">Session Management</h1>
                </div>
              </div>
              <Button
                variant="outline"
                size="sm"
                className="h-9 gap-2"
                onClick={() => navigate(configureUrl)}
              >
                <Settings className="size-4" />
                Configure
              </Button>
            </div>

            <Separator className="my-5" />
            <SectionLabel>Token Lifetimes</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={Clock} label="Access Token TTL" value={`${data.access_token_ttl_minutes} minutes`} />
              <Attr icon={Clock} label="Refresh Token TTL" value={`${data.refresh_token_ttl_days} days`} />
            </div>

            <Separator className="my-5" />
            <SectionLabel>Timeouts & Concurrency</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={Clock} label="Idle Timeout" value={`${data.idle_timeout_minutes} minutes`} />
              <Attr icon={Clock} label="Absolute Timeout" value={`${data.absolute_timeout_hours} hours`} />
              <Attr icon={Hash} label="Max Concurrent" value={data.max_concurrent_sessions > 0 ? String(data.max_concurrent_sessions) : "Unlimited"} />
            </div>

            <Separator className="my-5" />
            <SectionLabel>Refresh Rotation</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={data.rotate_refresh_tokens ? Check : X} label="Rotate Tokens" value={data.rotate_refresh_tokens ? "Enabled" : "Disabled"} />
              <Attr icon={Clock} label="Reuse Grace" value={data.refresh_token_reuse_interval_seconds > 0 ? `${data.refresh_token_reuse_interval_seconds} seconds` : "Instant revocation"} />
            </div>

            <Separator className="my-5" />
            <SectionLabel>Cookie Settings</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={data.cookie_secure ? Check : X} label="Secure" value={data.cookie_secure ? "HTTPS only" : "Disabled"} />
              <Attr icon={data.cookie_http_only ? Check : X} label="HttpOnly" value={data.cookie_http_only ? "Enabled" : "Disabled"} />
              <Attr icon={Shield} label="SameSite" value={data.cookie_same_site} />
              <Attr icon={data.revoke_sessions_on_password_change ? Check : X} label="Revoke on Password Change" value={data.revoke_sessions_on_password_change ? "Enabled" : "Disabled"} />
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
          title="Sessions"
          description="Token lifetimes, idle/absolute timeouts, concurrency, refresh rotation, and cookie flags."
        />
        {content}
      </div>
    </DetailsContainer>
  )
}
