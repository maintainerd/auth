import type { ColumnDef } from "@tanstack/react-table"
import { FileText } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { PolicyActions } from "./PolicyActions"
import { DataTableColumnHeader } from "@/components/data-table"
import { SystemBadge, StatusBadge } from "@/components/badges"
import type { Policy } from "@/services/api/policies/types"

export const policyColumns: ColumnDef<Policy>[] = [
  {
    id: "Policy",
    accessorKey: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Policy" />,
    cell: ({ row }) => {
      const policy = row.original
      return (
        <div className="flex items-center gap-3 px-3 py-1 max-w-xs">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <FileText className="size-5" />
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            <div className="flex items-center gap-2">
              <span className="truncate font-medium">{policy.name}</span>
              <SystemBadge isSystem={policy.is_system} />
            </div>
            <span className="truncate text-sm text-muted-foreground">
              {policy.description || "No description"}
            </span>
          </div>
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
    id: "Version",
    accessorKey: "version",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Version" />,
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <span className="font-mono text-sm">{row.original.version}</span>
      </div>
    ),
  },
  {
    id: "Created",
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => {
      const policy = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(new Date(policy.created_at), { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {new Date(policy.created_at).toLocaleDateString()}
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
      const policy = row.original
      return (
        <div className="px-3 py-1">
          <PolicyActions policy={policy} />
        </div>
      )
    },
  },
]
