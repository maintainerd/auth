import type { ColumnDef } from "@tanstack/react-table"
import { Badge } from "@/components/ui/badge"
import { Network } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { DataTableColumnHeader } from "@/components/data-table"
import { StatusBadge } from "@/components/details/StatusBadge"
import { WorkloadIdentityActions } from "./WorkloadIdentityActions"
import type { WorkloadIdentityFederation } from "@/services/api/workload-identity/types"

/**
 * Renders the issuer host rather than the full URL — the scheme is always https and
 * the host is what identifies the trusted platform (GitHub, GitLab, a cluster).
 */
function issuerHost(issuerUrl: string): string {
  try {
    return new URL(issuerUrl).host
  } catch {
    return issuerUrl
  }
}

export const workloadIdentityColumns: ColumnDef<WorkloadIdentityFederation>[] = [
  {
    id: "Federation",
    accessorKey: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Federation" />,
    cell: ({ row }) => {
      const federation = row.original
      return (
        <div className="flex items-center gap-3 px-3 py-1 max-w-xs">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <Network className="size-5" />
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            <span className="font-medium truncate">{federation.name}</span>
            <span className="text-sm text-muted-foreground truncate">
              {issuerHost(federation.issuer_url)}
            </span>
          </div>
        </div>
      )
    },
  },
  {
    // The subject pattern IS the trust boundary for this federation, so it belongs in
    // the listing rather than buried on the details page.
    id: "Subject Pattern",
    accessorKey: "subject_pattern",
    enableSorting: false,
    header: "Subject Pattern",
    cell: ({ row }) => (
      <div className="px-3 py-1 max-w-xs">
        <code className="text-xs font-mono text-muted-foreground break-all">
          {row.original.subject_pattern}
        </code>
      </div>
    ),
  },
  {
    id: "Scopes",
    accessorKey: "allowed_scopes",
    enableSorting: false,
    header: "Scopes",
    cell: ({ row }) => {
      const scopes = row.original.allowed_scopes ?? []
      if (scopes.length === 0) {
        return <span className="px-3 py-1 text-sm text-muted-foreground">None</span>
      }
      return (
        <div className="flex flex-wrap gap-1 px-3 py-1 max-w-xs">
          {scopes.slice(0, 2).map((scope) => (
            <Badge key={scope} variant="secondary" className="text-xs">
              {scope}
            </Badge>
          ))}
          {scopes.length > 2 && (
            <Badge variant="secondary" className="text-xs">
              +{scopes.length - 2}
            </Badge>
          )}
        </div>
      )
    },
  },
  {
    id: "Status",
    accessorKey: "is_active",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <StatusBadge status={row.original.is_active ? "active" : "inactive"} />
      </div>
    ),
  },
  {
    id: "Created",
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => {
      const federation = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(new Date(federation.created_at), { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {new Date(federation.created_at).toLocaleDateString()}
          </span>
        </div>
      )
    },
  },
  {
    id: "actions",
    enableSorting: false,
    enableHiding: false,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <WorkloadIdentityActions federation={row.original} />
      </div>
    ),
  },
]
