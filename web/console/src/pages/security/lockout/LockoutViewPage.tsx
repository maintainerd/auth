import { useNavigate } from "react-router-dom"
import { ShieldAlert, Hash, Clock, Settings, Check, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { DetailsContainer } from "@/components/container"
import { PageHeader } from "@/components/layout"
import { Skeleton } from "@/components/ui/skeleton"
import { useLockoutConfig } from "@/hooks/useLockoutConfig"
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

export default function LockoutViewPage({ standalone = true }: { standalone?: boolean }) {
  const navigate = useNavigate()
  const { data, isLoading, isError } = useLockoutConfig()
  const configureUrl = `/security/lockout/configure`

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
            Failed to load lockout configuration.
          </CardContent>
        </Card>
      )}

      {!isLoading && data && (
        <Card>
          <CardContent>
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div className="flex min-w-0 items-center gap-4">
                <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  <ShieldAlert className="size-6" />
                </div>
                <div className="min-w-0 space-y-1">
                  <div className="flex flex-wrap items-center gap-2.5">
                    <h1 className="text-lg font-semibold tracking-tight">Account Lockout</h1>
                    <Badge
                      variant="secondary"
                      className={data.enabled
                        ? "gap-1 border-emerald-500/30 bg-emerald-500/10 text-emerald-600"
                        : "gap-1"
                      }
                    >
                      {data.enabled ? "Enabled" : "Disabled"}
                    </Badge>
                  </div>
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
            <SectionLabel>Lockout Policy</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={Hash} label="Max Failed Attempts" value={String(data.max_failed_attempts)} />
              <Attr icon={Clock} label="Lockout Duration" value={`${data.lockout_duration_minutes} minutes`} />
              <Attr icon={Clock} label="Observation Window" value={`${data.observation_window_minutes} minutes`} />
              <Attr icon={Clock} label="Max Lockout Duration" value={`${data.max_lockout_duration_minutes} minutes`} />
            </div>

            <Separator className="my-5" />
            <SectionLabel>Progressive Lockout</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={data.progressive_lockout ? Check : X} label="Progressive Lockout" value={data.progressive_lockout ? "Enabled" : "Disabled"} />
              <Attr icon={Clock} label="Progression Reset" value={`${data.progression_reset_hours} hours`} />
            </div>

            <Separator className="my-5" />
            <SectionLabel>Behavior</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={data.auto_unlock ? Check : X} label="Auto Unlock" value={data.auto_unlock ? "After duration expires" : "Manual only"} />
              <Attr icon={data.reset_count_on_success ? Check : X} label="Reset on Success" value={data.reset_count_on_success ? "Enabled" : "Disabled"} />
              <Attr icon={data.notify_user_on_lockout ? Check : X} label="Notify User" value={data.notify_user_on_lockout ? "Enabled" : "Disabled"} />
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
          title="Account Lockout"
          description="Failed-login lockout policies, progressive escalation, and auto-unlock behavior."
        />
        {content}
      </div>
    </DetailsContainer>
  )
}
