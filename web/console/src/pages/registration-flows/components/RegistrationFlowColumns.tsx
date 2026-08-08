import type { ColumnDef } from "@tanstack/react-table"
import { Workflow } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import type { RegistrationFlow } from "@/services/api/registration-flows/types"
import { RegistrationFlowActions } from "./RegistrationFlowActions"
import { CopyableCode } from "@/components/inputs"
import { DataTableColumnHeader } from "@/components/data-table"
import { SystemBadge } from "@/components/badges"
import { StatusBadge } from "@/components/details/StatusBadge"

export const registrationFlowColumns: ColumnDef<RegistrationFlow>[] = [
  {
    id: "Registration Flow",
    accessorKey: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Registration Flow" />,
    cell: ({ row }) => {
      const flow = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1 max-w-xs">
          <div className="flex min-w-0 items-center gap-2">
            <Workflow className="size-4 text-muted-foreground shrink-0" />
            {/* The name is the public registration-link selector, so it reads as
                code and is copyable straight from the listing. stopPropagation
                because the row itself navigates. */}
            <CopyableCode value={flow.name} label="Flow name" stopPropagation />
            <SystemBadge isSystem={flow.is_system} />
          </div>
          <span className="text-sm text-muted-foreground truncate">{flow.description}</span>
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
      const flow = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(new Date(flow.created_at), { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {new Date(flow.created_at).toLocaleDateString()}
          </span>
        </div>
      )
    },
  },
  {
    id: "actions",
    cell: ({ row }) => (
      <div className="px-3 py-1">
        <RegistrationFlowActions registrationFlow={row.original} />
      </div>
    ),
    enableSorting: false,
    enableHiding: false,
  },
]
