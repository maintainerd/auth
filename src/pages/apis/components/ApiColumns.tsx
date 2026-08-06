import type { ColumnDef } from "@tanstack/react-table"
import { Server } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { ApiActions } from "./ApiActions"
import { DataTableColumnHeader } from "@/components/data-table"
import { StatusBadge } from "@/components/details"
import { SystemBadge } from "@/components/badges"
import type { Api } from "@/services/api/api/types"

export const apiColumns: ColumnDef<Api>[] = [
  {
    id: "API",
    accessorKey: "display_name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="API" />,
    cell: ({ row }) => {
      const api = row.original
      return (
        <div className="flex items-center gap-3 px-3 py-1 max-w-xs">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <Server className="size-5" />
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            <div className="flex items-center gap-2">
              <span className="truncate font-medium">{api.display_name}</span>
              <SystemBadge isSystem={api.is_system} />
            </div>
            <span className="truncate text-sm text-muted-foreground">{api.name}</span>
          </div>
        </div>
      )
    },
  },
  {
    id: "Service",
    accessorKey: "service",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Service" />,
    cell: ({ row }) => {
      const service = row.original.service
      // `service` is omitted from the payload when the API has no owning
      // service, so reading `.display_name` off it threw and took down the
      // whole listing render, not just this cell.
      if (!service) {
        return <span className="px-3 py-1 text-muted-foreground">&mdash;</span>
      }
      return (
        <div className="flex items-center gap-2 px-3 py-1">
          <Server className="size-4 text-muted-foreground" />
          <span className="font-medium">{service.display_name}</span>
        </div>
      )
    },
  },
  {
    id: "Status",
    accessorKey: "status",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <StatusBadge status={row.original.status} />
      </div>
    ),
  },
  {
    id: "Created",
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => {
      const createdAt = new Date(row.original.created_at)
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(createdAt, { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {createdAt.toLocaleDateString()}
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
        <ApiActions api={row.original} />
      </div>
    ),
  },
]
