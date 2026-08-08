import { useNavigate } from "react-router-dom"
import { Edit, Network, CalendarDays, Building2, Fingerprint } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { Button } from "@/components/ui/button"
import { DetailHeaderCard, StatusBadge, type DetailAttribute } from "@/components/details"
import { WorkloadIdentityActions } from "../../components/WorkloadIdentityActions"
import type { WorkloadIdentityFederation } from "@/services/api/workload-identity/types"

interface WorkloadIdentityHeaderProps {
  federation: WorkloadIdentityFederation
  federationId: string
}

/** The issuer host identifies the trusted platform; the scheme is always https. */
function issuerHost(issuerUrl: string): string {
  try {
    return new URL(issuerUrl).host
  } catch {
    return issuerUrl
  }
}

export function WorkloadIdentityHeader({
  federation,
  federationId,
}: WorkloadIdentityHeaderProps) {
  const navigate = useNavigate()

  const attributes: DetailAttribute[] = [
    { icon: Building2, label: "Issuer", value: issuerHost(federation.issuer_url) },
    { icon: Fingerprint, label: "Subject claim", value: federation.subject_claim || "sub" },
    {
      icon: CalendarDays,
      label: "Created",
      value: formatDistanceToNow(new Date(federation.created_at), { addSuffix: true }),
    },
  ]

  return (
    <DetailHeaderCard
      leading={
        <div className="flex size-12 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
          <Network className="size-6" />
        </div>
      }
      title={federation.name}
      subtitle={federation.description || federation.issuer_url}
      badge={<StatusBadge status={federation.is_active ? "active" : "inactive"} />}
      attributes={attributes}
      actions={
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate(`/workload-identity/${federationId}/edit`)}
          >
            <Edit className="mr-2 size-4" />
            Edit
          </Button>
          <WorkloadIdentityActions federation={federation} />
        </div>
      }
    />
  )
}
