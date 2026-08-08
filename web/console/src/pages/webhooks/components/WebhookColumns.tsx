import type { ColumnDef } from "@tanstack/react-table"
import { Webhook as WebhookIcon } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { Badge } from "@/components/ui/badge"
import { WebhookActions } from "./WebhookActions"
import { DataTableColumnHeader } from "@/components/data-table"
import { StatusBadge } from "@/components/details/StatusBadge"
import type { Webhook } from "@/services/api/webhooks/types"

export const webhookColumns: ColumnDef<Webhook>[] = [
  {
    id: "url",
    accessorKey: "url",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Endpoint" />,
    cell: ({ row }) => {
      const webhook = row.original
      return (
        <div className="flex items-center gap-3 px-3 py-1 max-w-md">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <WebhookIcon className="size-5" />
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            <span className="font-mono text-sm font-medium truncate">{webhook.url}</span>
            {webhook.description && (
              <span className="text-xs text-muted-foreground truncate">{webhook.description}</span>
            )}
          </div>
        </div>
      )
    },
  },
  {
    id: "events",
    accessorKey: "subscribe_all",
    enableSorting: false,
    header: ({ column }) => <DataTableColumnHeader column={column} title="Events" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <Badge variant="secondary" className="text-xs font-normal">
          {row.original.subscribe_all ? "All events" : "Selected events"}
        </Badge>
      </div>
    ),
  },
  {
    id: "status",
    accessorKey: "status",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <StatusBadge status={row.original.status} />
      </div>
    ),
  },
  {
    id: "last_triggered_at",
    accessorKey: "last_triggered_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Last triggered" />,
    cell: ({ row }) => {
      const last = row.original.last_triggered_at
      return (
        <div className="px-3 py-1">
          {last ? (
            <span className="text-sm">{formatDistanceToNow(new Date(last), { addSuffix: true })}</span>
          ) : (
            <span className="text-sm text-muted-foreground">Never</span>
          )}
        </div>
      )
    },
  },
  {
    id: "created_at",
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => {
      const webhook = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(new Date(webhook.created_at), { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {new Date(webhook.created_at).toLocaleDateString()}
          </span>
        </div>
      )
    },
  },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <WebhookActions webhook={row.original} />
      </div>
    ),
  },
]
