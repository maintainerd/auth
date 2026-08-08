import type { ColumnDef } from "@tanstack/react-table"
import { MessageSquare } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { DataTableColumnHeader } from "@/components/data-table"
import { StatusBadge } from "@/components/details/StatusBadge"
import { SmsTemplateActions } from "./SmsTemplateActions"
import type { SmsTemplate } from "@/services/api/sms-templates/types"

export const smsTemplateColumns: ColumnDef<SmsTemplate>[] = [
  {
    id: "Name",
    accessorKey: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
    cell: ({ row }) => {
      const template = row.original
      return (
        <div className="flex items-center gap-3 px-3 py-1 max-w-xs">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <MessageSquare className="size-5" />
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            <span className="font-medium truncate">{template.name}</span>
            {template.description && (
              <span className="text-xs text-muted-foreground truncate">{template.description}</span>
            )}
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
    id: "Created",
    accessorKey: "createdAt",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => {
      const template = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(new Date(template.createdAt), { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {new Date(template.createdAt).toLocaleDateString()}
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
        <SmsTemplateActions template={row.original} />
      </div>
    ),
  },
]
