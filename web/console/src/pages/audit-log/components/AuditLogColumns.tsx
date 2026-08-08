import type { ColumnDef } from "@tanstack/react-table"
import { FileText, ShieldAlert, ShieldCheck } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { Badge } from "@/components/ui/badge"
import { DataTableColumnHeader } from "@/components/data-table"
import type { AuditLogEntry } from "@/services/api/audit-log/types"
import { formatAuditActor } from "../formatActor"

const outcomeVariant: Record<string, "default" | "destructive" | "secondary" | "outline"> = {
  success: "default",
  failure: "destructive",
  partial: "secondary",
}

function renderOutcomeIcon(outcome: string) {
  if (outcome === "success") return <ShieldCheck className="size-5 text-emerald-500" />
  if (outcome === "failure") return <ShieldAlert className="size-5 text-destructive" />
  return <FileText className="size-5 text-muted-foreground" />
}

export const auditLogColumns: ColumnDef<AuditLogEntry>[] = [
  {
    id: "action",
    accessorKey: "action",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Action" />,
    cell: ({ row }) => {
      const entry = row.original
      return (
        <div className="flex max-w-md items-center gap-3 px-3 py-1">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            {renderOutcomeIcon(entry.outcome)}
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            <span className="truncate text-sm font-medium">{entry.action}</span>
            <span className="truncate text-xs text-muted-foreground">
              {entry.resource_type.replace(/_/g, " ")}
            </span>
          </div>
        </div>
      )
    },
  },
  {
    id: "outcome",
    accessorKey: "outcome",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Outcome" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <Badge
          variant={outcomeVariant[row.original.outcome] ?? "outline"}
          className="text-xs font-normal capitalize"
        >
          {row.original.outcome}
        </Badge>
      </div>
    ),
  },
  {
    id: "actor",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Actor" />,
    enableSorting: false,
    cell: ({ row }) => {
      const actor = formatAuditActor(row.original)
      return (
        <div className="px-3 py-1">
          <div className="flex min-w-0 flex-col gap-1">
            <span className="truncate text-sm font-medium">{actor.label}</span>
            <span className="truncate text-xs text-muted-foreground">{actor.context}</span>
          </div>
        </div>
      )
    },
  },
  {
    id: "resource_id",
    accessorKey: "resource_id",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Resource ID" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <span className="font-mono text-xs text-muted-foreground">
          {row.original.resource_id ? `${row.original.resource_id.slice(0, 8)}...` : "—"}
        </span>
      </div>
    ),
  },
  {
    id: "ip_address",
    accessorKey: "ip_address",
    header: ({ column }) => <DataTableColumnHeader column={column} title="IP Address" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <span className="font-mono text-sm">{row.original.ip_address || "—"}</span>
      </div>
    ),
  },
  {
    id: "created_at",
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Timestamp" />,
    cell: ({ row }) => {
      const createdAt = new Date(row.original.created_at)
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(createdAt, { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">{createdAt.toLocaleString()}</span>
        </div>
      )
    },
  },
]
