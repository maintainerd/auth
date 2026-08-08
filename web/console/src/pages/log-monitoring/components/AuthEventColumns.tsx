import type { ColumnDef } from "@tanstack/react-table"
import { ShieldAlert, ShieldCheck } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { Badge } from "@/components/ui/badge"
import { DataTableColumnHeader } from "@/components/data-table"
import type { AuthEvent } from "@/services/api/auth-events/types"

const severityVariant: Record<string, "default" | "destructive" | "secondary" | "outline"> = {
  INFO: "secondary",
  WARN: "outline",
  CRITICAL: "destructive",
}

export const authEventColumns: ColumnDef<AuthEvent>[] = [
  {
    id: "event_type",
    accessorKey: "event_type",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Event" />,
    cell: ({ row }) => {
      const event = row.original
      return (
        <div className="flex items-center gap-3 px-3 py-1 max-w-md">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            {event.result === "success" ? (
              <ShieldCheck className="size-5 text-emerald-500" />
            ) : (
              <ShieldAlert className="size-5 text-destructive" />
            )}
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            <span className="font-mono text-sm font-medium truncate">{event.event_type}</span>
            <span className="text-xs text-muted-foreground">{event.category}</span>
          </div>
        </div>
      )
    },
  },
  {
    id: "severity",
    accessorKey: "severity",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Severity" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <Badge variant={severityVariant[row.original.severity] ?? "outline"} className="text-xs font-normal">
          {row.original.severity}
        </Badge>
      </div>
    ),
  },
  {
    id: "result",
    accessorKey: "result",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Result" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <Badge
          variant={row.original.result === "success" ? "default" : "destructive"}
          className="text-xs font-normal"
        >
          {row.original.result}
        </Badge>
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
      const event = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(new Date(event.created_at), { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {new Date(event.created_at).toLocaleString()}
          </span>
        </div>
      )
    },
  },
]
