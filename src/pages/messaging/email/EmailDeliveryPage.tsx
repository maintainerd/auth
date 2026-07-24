import { useNavigate } from "react-router-dom"
import { Mail, Server, UserRound, CalendarDays, Settings, FlaskConical } from "lucide-react"
import { useQuery } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { DetailsContainer } from "@/components/container"
import { DetailHeaderCard, StatusBadge, EmptyState, type DetailAttribute } from "@/components/details"
import { safeFormat } from "@/lib/formatDate"
import { PageHeader } from "@/components/layout"
import { Skeleton } from "@/components/ui/skeleton"
import { fetchEmailConfig } from "@/services/api/notifier"

const PROVIDER_LABELS: Record<string, string> = {
  smtp: "SMTP",
  ses: "Amazon SES",
  sendgrid: "SendGrid",
  mailgun: "Mailgun",
  postmark: "Postmark",
  resend: "Resend",
}

export default function EmailDeliveryPage({ standalone = true }: { standalone?: boolean }) {
  const navigate = useNavigate()

  const { data, isLoading, isError } = useQuery({
    queryKey: ["email-config"],
    queryFn: fetchEmailConfig,
    retry: false,
  })

  const isConfigured = Boolean(data?.provider && data?.from_address)
  const notConfigured = (!data && !isLoading && !isError) || (data && !isConfigured) || isError
  const providerLabel = PROVIDER_LABELS[data?.provider ?? ""] ?? data?.provider
  const isSmtp = data?.provider === "smtp"
  const configureUrl = `/messaging/email/configure`

  const attributes: DetailAttribute[] = data && isConfigured
    ? [
        { icon: Server, label: "Provider", value: providerLabel },
        ...(isSmtp
          ? [
              { icon: Server, label: "Host", value: <span className="font-mono text-xs">{data.host || "—"}</span> },
              { icon: Server, label: "Port", value: <span className="font-mono text-xs">{data.port || "—"}</span> },
              { icon: Server, label: "Encryption", value: (data.encryption || "none").toUpperCase() },
            ]
          : []),
        ...(data.username
          ? [{ icon: UserRound, label: isSmtp ? "Username" : "Key ID", value: <span className="font-mono text-xs">{data.username}</span> }]
          : []),
        { icon: Mail, label: "From Address", value: data.from_address || "—" },
        ...(data.from_name ? [{ icon: UserRound, label: "From Name", value: data.from_name }] : []),
        ...(data.reply_to ? [{ icon: Mail, label: "Reply-To", value: data.reply_to }] : []),
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
              icon={Mail}
              title="Email delivery is not configured"
              description="Connect an email provider so the platform can send verification, password reset, and notification emails."
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
              <Mail className="size-6" />
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
          subtitle={data.from_address}
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
          title="Email Delivery"
          description="Configure how the platform sends verification, security, and notification emails."
        />
        {content}
      </div>
    </DetailsContainer>
  )
}
