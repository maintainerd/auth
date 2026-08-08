import { useNavigate } from "react-router-dom"
import { Key, Clock, Shield, Settings, Check, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { DetailsContainer } from "@/components/container"
import { PageHeader } from "@/components/layout"
import { Skeleton } from "@/components/ui/skeleton"
import { useTokenConfig } from "@/hooks/useTokenConfig"
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

export default function TokenViewPage({ standalone = true }: { standalone?: boolean }) {
  const navigate = useNavigate()
  const { data, isLoading, isError } = useTokenConfig()
  const configureUrl = `/security/token/configure`

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
            Failed to load token configuration.
          </CardContent>
        </Card>
      )}

      {!isLoading && data && (
        <Card>
          <CardContent>
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div className="flex min-w-0 items-center gap-4">
                <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  <Key className="size-6" />
                </div>
                <div className="min-w-0 space-y-1">
                  <h1 className="text-lg font-semibold tracking-tight">Token Configuration</h1>
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
            <SectionLabel>JWT Settings</SectionLabel>
            <div className="mt-3 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
              <Attr icon={Shield} label="Signing Algorithm" value={data.signing_algorithm} />
              <Attr icon={Clock} label="Clock Skew Leeway" value={`${data.clock_skew_leeway_seconds} seconds`} />
              <Attr icon={data.require_pkce ? Check : X} label="Require PKCE" value={data.require_pkce ? "Enforced" : "Disabled"} />
            </div>

            <Separator className="my-5" />
            <SectionLabel>ID Token Claims</SectionLabel>
            <div className="mt-3 flex flex-wrap gap-2">
              {(data.additional_id_token_claims ?? []).length > 0
                ? data.additional_id_token_claims.map((claim) => (
                    <Badge key={claim} variant="secondary" className="font-mono text-xs">{claim}</Badge>
                  ))
                : <span className="text-sm text-muted-foreground">No additional claims</span>
              }
            </div>

            <Separator className="my-5" />
            <SectionLabel>Access Token Claims</SectionLabel>
            <div className="mt-3 flex flex-wrap gap-2">
              {(data.additional_access_token_claims ?? []).length > 0
                ? data.additional_access_token_claims.map((claim) => (
                    <Badge key={claim} variant="secondary" className="font-mono text-xs">{claim}</Badge>
                  ))
                : <span className="text-sm text-muted-foreground">No additional claims</span>
              }
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
          title="Token Configuration"
          description="Access/refresh token lifetimes, rotation, and security settings."
        />
        {content}
      </div>
    </DetailsContainer>
  )
}
