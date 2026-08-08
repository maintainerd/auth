import { format } from "date-fns"
import { FileText, Globe, Calendar, Ban } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { RowActions, type RowActionItem } from "@/components/data-table"
import { useUserConsents, useWithdrawUserConsent } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"

interface UserConsentsProps {
  userId: string
}

interface ConsentRow {
  uuid: string
  consent_type: string
  accepted: boolean
  policy_version: string
  ip_address?: string | null
  created_at: string
}

function formatDate(value?: string | null) {
  if (!value) return "—"
  try {
    return format(new Date(value), "PPp")
  } catch {
    return "—"
  }
}

export function UserConsents({ userId }: UserConsentsProps) {
  const { data, isLoading, isError } = useUserConsents(userId)
  const consents = (data ?? []) as ConsentRow[]
  const { showSuccess, showError } = useToast()
  const withdrawMutation = useWithdrawUserConsent()

  const withdraw = async (consent: ConsentRow) => {
    try {
      await withdrawMutation.mutateAsync({ userId, consentType: consent.consent_type })
      showSuccess("Consent withdrawn")
    } catch (e) {
      showError(e)
    }
  }

  const consentActions = (consent: ConsentRow): RowActionItem[] => [
    {
      key: "withdraw",
      label: "Withdraw consent",
      icon: Ban,
      destructive: true,
      onSelect: () => withdraw(consent),
      confirm: {
        title: "Withdraw consent",
        description:
          "This records a withdrawal of the user's consent (the original acceptance is kept for the audit trail). Per GDPR, withdrawal must be as easy as granting. Continue?",
        confirmText: "Withdraw",
      },
    },
  ]

  return (
    <InformationCard
      title="Consents"
      description="Consent agreements recorded for this user. Withdrawing a consent is logged as a new entry — the original acceptance is preserved for audit."
      icon={FileText}
    >
      <div className="space-y-4">
        {isLoading && <ListSkeleton />}

        {isError && <p className="py-8 text-center text-sm text-destructive">Failed to load consents</p>}

        {!isLoading && !isError && consents.length === 0 && (
          <EmptyState
            icon={FileText}
            title="No consents"
            description="This user has not accepted any consent agreements."
          />
        )}

        {consents.length > 0 && (
          <div className="space-y-3">
            {consents.map((consent) => (
              <div
                key={consent.uuid}
                data-md-listing-item
                className="flex items-start justify-between gap-3 rounded-lg border p-4"
              >
                <div className="flex min-w-0 items-start gap-3">
                  <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <FileText className="size-4" />
                  </div>
                  <div className="min-w-0 space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="text-sm font-medium capitalize">
                        {consent.consent_type.replace(/_/g, " ")}
                      </p>
                      <Badge variant={consent.accepted ? "default" : "secondary"} className="text-xs">
                        {consent.accepted ? "Accepted" : "Withdrawn"}
                      </Badge>
                      <Badge variant="outline" className="text-xs font-normal">
                        v{consent.policy_version}
                      </Badge>
                    </div>
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                      <span className="inline-flex items-center gap-1">
                        <Calendar className="size-3" />
                        {formatDate(consent.created_at)}
                      </span>
                      {consent.ip_address && (
                        <span className="inline-flex items-center gap-1 font-mono">
                          <Globe className="size-3" />
                          {consent.ip_address}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
                {consent.accepted && <RowActions items={consentActions(consent)} />}
              </div>
            ))}
          </div>
        )}
      </div>
    </InformationCard>
  )
}
