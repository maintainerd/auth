import type { ColumnDef } from "@tanstack/react-table"
import { Badge } from "@/components/ui/badge"
import { Shield } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { RoleActions } from "./RoleActions"
import { DataTableColumnHeader } from "@/components/data-table"
import { SystemBadge } from "@/components/badges"
import { StatusBadge } from "@/components/details/StatusBadge"
import type { Role } from "@/services/api/roles/types"

export const roleColumns: ColumnDef<Role>[] = [
  {
    id: "Role",
    accessorKey: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Role" />,
    cell: ({ row }) => {
      const role = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1 max-w-xs">
          <div className="flex items-center gap-2">
            <Shield className="size-4 text-muted-foreground shrink-0" />
            <span className="font-medium">{role.name}</span>
            <SystemBadge isSystem={role.is_system} />
            {role.is_default && <Badge variant="outline" className="text-xs">Default</Badge>}
          </div>
          <span className="text-sm text-muted-foreground truncate">{role.description}</span>
        </div>
      )
    },
  },
  {
    id: "Status",
    accessorKey: "status",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => {
      const role = row.original
      return (
        <div className="px-3 py-1">
          <StatusBadge status={role.status} />
        </div>
      )
    },
  },
  {
    id: "Created",
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => {
      const role = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(new Date(role.created_at), { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {new Date(role.created_at).toLocaleDateString()}
          </span>
        </div>
      )
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const role = row.original
      return (
        <div className="px-3 py-1">
          <RoleActions role={role} />
        </div>
      )
    },
  },
]
