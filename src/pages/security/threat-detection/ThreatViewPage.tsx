import { useNavigate } from "react-router-dom"
import { ShieldAlert, Hash, Settings, Check, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { DetailsContainer } from "@/components/container"
import { PageHeader } from "@/components/layout"
import { Skeleton } from "@/components/ui/skeleton"
import { useThreatDetectionSettings } from "@/hooks/useThreatDetectionSettings"
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

export default function ThreatViewPage({ standalone = true }: { standalone?: boolean }) {
  const navigate = useNavigate()
  const { data, isLoading, isError } = useThreatDetectionSettings()
  const configureUrl = `/security/threat/configure`

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
            Failed to load attack protection settings.
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
                  <h1 className="text-lg font-semibold tracking-tight">Threat Protection</h1>
                </div>
              </div>
              <Button variant="outline" size="sm" className="h-9 gap-2" onClick={() => navigate(configureUrl)}>
                <Settings className="size-4" />
                Configure
              </Button>
            </div>

            <Separator className="my-5" />
            <SectionLabel>Detection Engines</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2">
              <Attr icon={data.brute_force_detection_enabled ? Check : X} label="Brute Force" value={data.brute_force_detection_enabled ? "Enabled" : "Disabled"} />
              <Attr icon={data.impossible_travel_detection_enabled ? Check : X} label="Impossible Travel" value={data.impossible_travel_detection_enabled ? "Enabled" : "Disabled"} />
              <Attr icon={data.new_device_notification_enabled ? Check : X} label="New Device Notification" value={data.new_device_notification_enabled ? "Enabled" : "Disabled"} />
              <Attr icon={data.velocity_check_enabled ? Check : X} label="Velocity Check" value={data.velocity_check_enabled ? "Enabled" : "Disabled"} />
            </div>

            <Separator className="my-5" />
            <SectionLabel>Risk-Based Controls</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2">
              <Attr icon={data.risk_based_step_up_enabled ? Check : X} label="Risk-Based Step-Up" value={data.risk_based_step_up_enabled ? "Enabled" : "Disabled"} />
              <Attr icon={data.compromised_credential_monitoring_enabled ? Check : X} label="Compromised Credentials" value={data.compromised_credential_monitoring_enabled ? "Monitored" : "Disabled"} />
              <Attr icon={Hash} label="Step-Up Threshold" value={`Score ≥ ${data.risk_step_up_threshold}`} />
              <Attr icon={Hash} label="Block Threshold" value={`Score ≥ ${data.risk_block_threshold}`} />
            </div>

            <Separator className="my-5" />
            <SectionLabel>Network & IP</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={data.ip_reputation_check_enabled ? Check : X} label="IP Reputation Check" value={data.ip_reputation_check_enabled ? "Enabled" : "Disabled"} />
              <Attr icon={data.block_tor_exit_nodes ? Check : X} label="Block Tor Exit Nodes" value={data.block_tor_exit_nodes ? "Blocked" : "Allowed"} />
              <Attr icon={Hash} label="Velocity Limit" value={`${data.velocity_failures_per_ip_per_hour} failures/IP/hour`} />
              <Attr icon={Hash} label="Distinct Accounts Limit" value={`${data.distinct_accounts_per_ip_per_hour} accounts/IP/hour`} />
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
          title="Threat Protection"
          description="Brute-force detection, anomaly monitoring, and security alert behaviors."
        />
        {content}
      </div>
    </DetailsContainer>
  )
}
