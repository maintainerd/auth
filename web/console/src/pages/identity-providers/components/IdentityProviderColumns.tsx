import type { ColumnDef } from "@tanstack/react-table"
import { Badge } from "@/components/ui/badge"
import { Shield } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { IdentityProviderActions } from "./IdentityProviderActions"
import { DataTableColumnHeader } from "@/components/data-table"
import { SystemBadge } from "@/components/badges"
import { StatusBadge } from "@/components/details/StatusBadge"
import { PROVIDER_LABELS, ProviderLogo, getProviderBrand } from "@/components/provider-config"
import type { IdentityProvider, ProviderOption } from "@/services/api/identity-providers/types"

const getProviderBadge = (provider: ProviderOption, isSystem: boolean) => {
  if (isSystem) {
    return (
      <Badge variant="secondary" className="text-xs">
        <Shield className="h-3 w-3 mr-1" />
        {PROVIDER_LABELS[provider] ?? provider}
      </Badge>
    )
  }

  const { Icon, color } = getProviderBrand(provider)
  const label = PROVIDER_LABELS[provider] ?? provider

  return (
    <Badge variant="secondary" className="text-xs">
      <Icon className={`h-3 w-3 mr-1 ${color}`} />
      {label}
    </Badge>
  )
}

export const identityProviderColumns: ColumnDef<IdentityProvider>[] = [
  {
    id: "Identity Provider",
    accessorKey: "display_name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Identity Provider" />,
    cell: ({ row }) => {
      const provider = row.original
      return (
        <div className="flex items-center gap-3 px-3 py-1 max-w-xs">
          <ProviderLogo provider={provider.provider} className="size-10" />
          <div className="flex min-w-0 flex-col gap-1">
            <div className="flex items-center gap-2">
              <span className="font-medium truncate">{provider.display_name}</span>
              <SystemBadge isSystem={provider.is_system} />
            </div>
            <span className="text-sm text-muted-foreground truncate">{provider.name}</span>
          </div>
        </div>
      )
    },
  },
  {
    id: "Provider",
    accessorKey: "provider",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Provider" />,
    cell: ({ row }) => {
      return (
        <div className="px-3 py-1">
          {getProviderBadge(row.original.provider, row.original.is_system)}
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
      const provider = row.original
      return (
        <div className="flex flex-col gap-1 px-3 py-1">
          <span className="text-sm font-medium">
            {formatDistanceToNow(new Date(provider.created_at), { addSuffix: true })}
          </span>
          <span className="text-xs text-muted-foreground">
            {new Date(provider.created_at).toLocaleDateString()}
          </span>
        </div>
      )
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      return (
        <div className="px-3 py-1">
          <IdentityProviderActions provider={row.original} />
        </div>
      )
    },
    enableSorting: false,
    enableHiding: false,
  },
]
