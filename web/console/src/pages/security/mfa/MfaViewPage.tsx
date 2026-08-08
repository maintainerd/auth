import { useNavigate } from "react-router-dom"
import { ShieldCheck, Smartphone, Clock, KeyRound, Hash, Settings, Check, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { DetailsContainer } from "@/components/container"
import { PageHeader } from "@/components/layout"
import { Skeleton } from "@/components/ui/skeleton"
import { useMfaConfig } from "@/hooks/useMfaConfig"
import type { LucideIcon } from "lucide-react"
import type { ReactNode } from "react"

const METHOD_LABELS: Record<string, string> = {
  totp: "Authenticator (TOTP)",
  webauthn: "Passkeys",
  sms: "SMS",
  email_otp: "Email OTP",
  recovery_code: "Recovery Codes",
}

const MODE_LABELS: Record<string, string> = {
  disabled: "Disabled",
  optional: "Optional",
  enforced: "Enforced",
}

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

export default function MfaViewPage({ standalone = true }: { standalone?: boolean }) {
  const navigate = useNavigate()

  const { data, isLoading, isError } = useMfaConfig()

  const configureUrl = `/security/mfa/configure`

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
            Failed to load MFA configuration.
          </CardContent>
        </Card>
      )}

      {!isLoading && data && (
        <Card>
          <CardContent>
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div className="flex min-w-0 items-center gap-4">
                <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  <ShieldCheck className="size-6" />
                </div>
                <div className="min-w-0 space-y-1">
                  <div className="flex flex-wrap items-center gap-2.5">
                    <h1 className="text-lg font-semibold tracking-tight">MFA</h1>
                    <Badge
                      variant="secondary"
                      className={
                        data.mode === "enforced"
                          ? "gap-1 border-emerald-500/30 bg-emerald-500/10 text-emerald-600"
                          : data.mode === "optional"
                            ? "gap-1 border-amber-500/30 bg-amber-500/10 text-amber-600"
                            : "gap-1"
                      }
                    >
                      {MODE_LABELS[data.mode] ?? data.mode}
                    </Badge>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    {(data.allowed_methods ?? []).map((m) => (
                      <Badge key={m} variant="secondary" className="text-xs">
                        {METHOD_LABELS[m] ?? m}
                      </Badge>
                    ))}
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
            <div className="grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={ShieldCheck} label="Mode" value={MODE_LABELS[data.mode] ?? data.mode} />
              <Attr icon={Smartphone} label="Preferred Method" value={METHOD_LABELS[data.preferred_method] ?? data.preferred_method} />
            </div>

            <Separator className="my-5" />
            <SectionLabel>TOTP Parameters</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={KeyRound} label="Issuer" value={data.totp_issuer} />
              <Attr icon={Hash} label="Digits" value={`${data.totp_digits} digits`} />
              <Attr icon={Clock} label="Period" value={`${data.totp_period_seconds} seconds`} />
            </div>

            <Separator className="my-5" />
            <SectionLabel>Trust, Recovery & Options</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={Clock} label="Trusted Device" value={data.trusted_device_period_days > 0 ? `${data.trusted_device_period_days} days` : "Disabled"} />
              <Attr icon={Clock} label="Enrollment Grace" value={data.grace_period_days > 0 ? `${data.grace_period_days} days` : "None"} />
              <Attr icon={KeyRound} label="Recovery Codes" value={data.recovery_codes_count > 0 ? `${data.recovery_codes_count} codes` : "Disabled"} />
              <Attr icon={Clock} label="Step-Up TTL" value={`${data.step_up_ttl_minutes} minutes`} />
              <Attr icon={data.allow_sms ? Check : X} label="Allow SMS" value={data.allow_sms ? "Enabled" : "Disabled"} />
              <Attr icon={data.allow_email_otp ? Check : X} label="Allow Email OTP" value={data.allow_email_otp ? "Enabled" : "Disabled"} />
              <Attr icon={data.require_mfa_for_sensitive_actions ? Check : X} label="Sensitive Actions" value={data.require_mfa_for_sensitive_actions ? "Step-up required" : "Not required"} />
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
          title="Multi-Factor Authentication"
          description="Configure MFA enforcement, allowed methods, and device trust policies."
        />
        {content}
      </div>
    </DetailsContainer>
  )
}
