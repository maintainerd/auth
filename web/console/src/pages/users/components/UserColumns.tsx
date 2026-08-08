import type { ColumnDef } from "@tanstack/react-table"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { formatDistanceToNow } from "date-fns"
import { UserActions } from "./UserActions"
import { DataTableColumnHeader } from "@/components/data-table"
import { StatusBadge } from "@/components/details/StatusBadge"
import type { User } from "@/services/api/users/types"

// Two-letter avatar fallback: initials of the first two whitespace-separated
// parts, or the first two characters of a single token (e.g. "jdoe" → "JD").
const initials = (value: string) => {
  const v = value.trim()
  if (!v) return "?"
  const parts = v.split(/\s+/)
  const letters = parts.length > 1 ? parts.slice(0, 2).map((p) => p[0]).join("") : v.slice(0, 2)
  return letters.toUpperCase() || "?"
}

export const userColumns: ColumnDef<User>[] = [
  {
    id: "Username",
    accessorKey: "username",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Username" />,
    cell: ({ row }) => {
      const user = row.original
      return (
        <div className="flex items-center gap-3 px-3 py-1">
          <Avatar className="size-9">
            <AvatarFallback className="bg-muted text-xs font-medium text-muted-foreground">
              {initials(user.username)}
            </AvatarFallback>
          </Avatar>
          <span className="font-medium">{user.username}</span>
        </div>
      )
    },
  },
  {
    id: "Status",
    accessorKey: "status",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => {
      const user = row.original
      return (
        <div className="px-3 py-1">
          <StatusBadge status={user.status} />
        </div>
      )
    },
  },
  {
    id: "Created",
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => {
      const user = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(new Date(user.created_at), { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {new Date(user.created_at).toLocaleDateString()}
          </span>
        </div>
      )
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const user = row.original
      return (
        <div className="px-3 py-1">
          <UserActions user={user} />
        </div>
      )
    },
  },
];

