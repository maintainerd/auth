import type { ColumnDef } from "@tanstack/react-table"
import { Mail } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { DataTableColumnHeader } from "@/components/data-table"
import { StatusBadge } from "@/components/details/StatusBadge"
import { safeFormat } from "@/lib/formatDate"
import type { Invite } from "@/services/api/invites/types"
import { InviteActions } from "./InviteActions"

export const inviteColumns: ColumnDef<Invite>[] = [
  {
    id: "Email",
    accessorKey: "invited_email",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Email" />,
    cell: ({ row }) => (
      <div className="flex items-center gap-2 px-3 py-1 max-w-sm">
        <Mail className="size-4 text-muted-foreground shrink-0" />
        <span className="truncate font-medium">{row.original.invited_email}</span>
      </div>
    ),
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
    id: "Registration flow",
    accessorKey: "registration_flow_name",
    enableSorting: false,
    header: ({ column }) => <DataTableColumnHeader column={column} title="Registration flow" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <span className="text-sm">
          {row.original.registration_flow_name ?? (
            <span className="text-muted-foreground">Default registration</span>
          )}
        </span>
      </div>
    ),
  },
  {
    id: "Expires",
    accessorKey: "expires_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Expires" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <span className="text-sm">{safeFormat(row.original.expires_at, "PP") || "—"}</span>
      </div>
    ),
  },
  {
    id: "Invited",
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Invited" />,
    cell: ({ row }) => {
      const created = row.original.created_at
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(new Date(created), { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {new Date(created).toLocaleDateString()}
          </span>
        </div>
      )
    },
  },
  {
    id: "actions",
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <InviteActions invite={row.original} />
      </div>
    ),
    enableSorting: false,
    enableHiding: false,
  },
]
