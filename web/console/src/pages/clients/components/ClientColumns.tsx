import type { ColumnDef } from "@tanstack/react-table"
import { Badge } from "@/components/ui/badge"
import { AppWindow, Monitor, Smartphone, Globe, Cog, type LucideIcon } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { ClientActions } from "./ClientActions"
import { DataTableColumnHeader } from "@/components/data-table"
import { SystemBadge } from "@/components/badges"
import { StatusBadge } from "@/components/details/StatusBadge"
import type { Client, ClientType } from "@/services/api/clients/types"

// Neutral secondary badge + icon, matching the identity-provider "Provider"
// column — the pastel block variant is retired app-wide.
const TYPE_CONFIG: Record<ClientType, { icon: LucideIcon; label: string }> = {
  traditional: { icon: Globe, label: "Traditional Web" },
  mobile: { icon: Smartphone, label: "Native Mobile" },
  spa: { icon: Monitor, label: "Single Page App" },
  m2m: { icon: Cog, label: "Machine to Machine" },
}

const getTypeBadge = (type: ClientType) => {
  const config = TYPE_CONFIG[type] ?? { icon: Monitor, label: type }
  const Icon = config.icon

  return (
    <Badge variant="secondary" className="text-xs">
      <Icon className="h-3 w-3 mr-1" />
      {config.label}
    </Badge>
  )
}

export const clientColumns: ColumnDef<Client>[] = [
  {
    id: "Client",
    accessorKey: "display_name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Client" />,
    cell: ({ row }) => {
      const client = row.original
      return (
        <div className="flex items-center gap-3 px-3 py-1 max-w-xs">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <AppWindow className="size-5" />
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            <div className="flex items-center gap-2">
              <span className="font-medium truncate">{client.display_name}</span>
              <SystemBadge isSystem={client.is_system} />
            </div>
            <span className="text-sm text-muted-foreground truncate">{client.name}</span>
          </div>
        </div>
      )
    },
  },
  {
    id: "Type",
    accessorKey: "client_type",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
    cell: ({ row }) => {
      return (
        <div className="px-3 py-1">
          {getTypeBadge(row.original.client_type)}
        </div>
      )
    },
  },
  {
    id: "Status",
    accessorKey: "status",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => {
      return (
        <div className="px-3 py-1">
          <StatusBadge status={row.original.status} />
        </div>
      )
    },
  },
  {
    id: "Created",
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => {
      const client = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(new Date(client.created_at), { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {new Date(client.created_at).toLocaleDateString()}
          </span>
        </div>
      )
    },
  },
  {
    id: "actions",
    enableSorting: false,
    enableHiding: false,
    cell: ({ row }) => {
      const client = row.original
      return (
        <div className="px-3 py-1">
          <ClientActions client={client} />
        </div>
      )
    },
  },
]
