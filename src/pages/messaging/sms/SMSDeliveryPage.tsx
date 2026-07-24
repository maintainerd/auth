import { useNavigate } from "react-router-dom"
import { MessageSquare, Server, Phone, Hash, CalendarDays, Settings, FlaskConical } from "lucide-react"
import { useQuery } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { DetailsContainer } from "@/components/container"
import { DetailHeaderCard, StatusBadge, EmptyState, type DetailAttribute } from "@/components/details"
import { safeFormat } from "@/lib/formatDate"
import { PageHeader } from "@/components/layout"
import { Skeleton } from "@/components/ui/skeleton"
import { fetchSMSConfig } from "@/services/api/notifier"

const PROVIDER_LABELS: Record<string, string> = {
  twilio: "Twilio",
  sns: "AWS SNS",
  vonage: "Vonage",
  messagebird: "MessageBird",
  log: "Log (testing)",
}

export default function SMSDeliveryPage({ standalone = true }: { standalone?: boolean }) {
  const navigate = useNavigate()

  const { data, isLoading, isError } = useQuery({
    queryKey: ["sms-config"],
    queryFn: fetchSMSConfig,
    retry: false,
  })

  const isConfigured = Boolean(data?.provider)
  const notConfigured = (!data && !isLoading && !isError) || (data && !isConfigured) || isError
  const providerLabel = PROVIDER_LABELS[data?.provider ?? ""] ?? data?.provider
  const configureUrl = `/messaging/sms/configure`

  const attributes: DetailAttribute[] = data && isConfigured
    ? [
        { icon: Server, label: "Provider", value: providerLabel },
        ...(data.account_sid
          ? [{ icon: Hash, label: "Account / Key ID", value: <span className="font-mono text-xs">{data.account_sid}</span> }]
          : []),
        ...(data.from_number
          ? [{ icon: Phone, label: "From Number", value: <span className="font-mono text-xs">{data.from_number}</span> }]
          : []),
        ...(data.sender_id
          ? [{ icon: Hash, label: "Sender ID", value: data.sender_id }]
          : []),
        { icon: Hash, label: "Daily Send Limit", value: String(data.daily_send_limit ?? 1000) },
        { icon: CalendarDays, label: "Created", value: safeFormat(data.created_at, "PPp") },
        { icon: CalendarDays, label: "Last updated", value: safeFormat(data.updated_at, "PPp") },
      ]
    : []

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

      {!isLoading && notConfigured && (
        <Card>
          <CardContent>
            <EmptyState
              icon={MessageSquare}
              title="SMS delivery is not configured"
              description="Connect an SMS provider so the platform can send one-time codes, phone verification, and security alerts."
              action={
                <Button className="gap-2" onClick={() => navigate(configureUrl)}>
                  <Settings className="size-4" />
                  Configure
                </Button>
              }
            />
          </CardContent>
        </Card>
      )}

      {!isLoading && data && isConfigured && (
        <DetailHeaderCard
          leading={
            <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
              <MessageSquare className="size-6" />
            </div>
          }
          title={providerLabel}
          badge={
            <div className="flex items-center gap-2">
              <StatusBadge status={data.status || "active"} />
              {data.test_mode && (
                <Badge variant="secondary" className="gap-1 border-amber-500/30 bg-amber-500/10 font-normal text-amber-600">
                  <FlaskConical className="size-3" />
                  Test mode
                </Badge>
              )}
            </div>
          }
          subtitle={data.from_number || data.sender_id || "No sender configured"}
          attributes={attributes}
          actions={
            <Button
              variant="outline"
              size="sm"
              className="h-9 gap-2"
              onClick={() => navigate(configureUrl)}
            >
              <Settings className="size-4" />
              Configure
            </Button>
          }
        />
      )}
    </>
  )

  if (!standalone) return content

  return (
    <DetailsContainer>
      <div className="flex flex-col gap-6">
        <PageHeader
          title="SMS Delivery"
          description="Configure how the platform sends one-time codes and security alerts by text message."
        />
        {content}
      </div>
    </DetailsContainer>
  )
}
