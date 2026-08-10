import { useState } from "react"
import { Key, Calendar, Globe, Unlink } from "lucide-react"
import { cn } from "@/lib/utils"
import { getProviderBrand } from "@/components/provider-config/providerBrand"
import { safeFormat } from "@/lib/formatDate"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { DataTablePagination, RowActions, usePaginationTable, type RowActionItem } from "@/components/data-table"
import { useUserIdentities, useUnlinkUserIdentity } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"
import type { UserIdentity } from "@/services/api/users/types"
import { type PaginationState } from "@tanstack/react-table"

interface UserIdentitiesProps {
  userId: string
}

export function UserIdentities({ userId }: UserIdentitiesProps) {
  const { showSuccess, showError } = useToast()
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data, isLoading, isError } = useUserIdentities(userId, {
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sort_by: 'created_at',
    sort_order: 'desc',
  })

  const unlinkMutation = useUnlinkUserIdentity()

  const handleUnlink = async (identityId: string) => {
    try {
      await unlinkMutation.mutateAsync({ userId, identityId })
      showSuccess("Identity unlinked")
    } catch (error) {
      showError(error)
    }
  }

  const table = usePaginationTable({
    pagination,
    onPaginationChange: setPagination,
    pageCount: data?.total_pages ?? 0,
  })

  // Never strand a user: only offer unlink when they have more than one identity.
  const canUnlink = (data?.total ?? 0) > 1

  const identityActions = (identity: UserIdentity): RowActionItem[] =>
    canUnlink
      ? [
          {
            key: "unlink",
            label: "Unlink identity",
            icon: Unlink,
            destructive: true,
            onSelect: () => handleUnlink(identity.user_identity_id),
            confirm: {
              title: "Unlink identity",
              description:
                "This removes this authentication method from the user. They will no longer be able to sign in with it. Continue?",
              confirmText: "Unlink",
            },
          },
        ]
      : []

  return (
    <InformationCard
      title="User Identities"
      description="How this user can authenticate — provider, subject, and the client each identity is linked to."
      icon={Key}
    >
      <div className="space-y-4">
        {isLoading && <ListSkeleton />}

        {isError && (
          <p className="py-8 text-center text-sm text-destructive">Failed to load identities</p>
        )}

        {data && data.rows.length === 0 && (
          <EmptyState
            icon={Key}
            title="No identities"
            description="This user has no linked authentication identities yet."
          />
        )}

        {data && data.rows.length > 0 && (
          <div className="space-y-3">
            {data.rows.map((identity: UserIdentity) => {
              const actions = identityActions(identity)
              // Show the actual provider's brand mark rather than a generic key.
              const { Icon: ProviderIcon, color: providerColor } = getProviderBrand(identity.provider)
              return (
                <div data-md-listing-item key={identity.user_identity_id} className="rounded-lg border p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex min-w-0 flex-1 items-start gap-3">
                      <div data-md-listing-icon className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                        <ProviderIcon className={cn("size-4", providerColor)} />
                      </div>
                      <div className="min-w-0 flex-1 space-y-3">
                        {/* Provider + subject */}
                        <div className="grid gap-3 sm:grid-cols-2">
                          <div className="space-y-0.5">
                            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                              Provider
                            </p>
                            <p className="text-sm font-medium capitalize">{identity.provider}</p>
                          </div>
                          <div className="min-w-0 space-y-0.5">
                            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                              Subject
                            </p>
                            <p className="break-all font-mono text-sm">{identity.sub}</p>
                          </div>
                        </div>

                        {/* Issuing identity provider — not a client. Which
                            applications can use this identity is configured on
                            the provider's connections, not stored here. */}
                        {identity.identity_provider_name && (
                          <div data-md-listing-nested className="space-y-2 rounded-md border bg-muted/30 p-3">
                            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                              <Globe className="size-3.5" />
                              Identity provider
                            </div>
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="text-sm font-medium">{identity.identity_provider_name}</span>
                            </div>
                          </div>
                        )}

                        {/* Footer */}
                        <div data-md-listing-meta className="flex items-center gap-1 text-xs text-muted-foreground">
                          <Calendar className="size-3" />
                          Connected {safeFormat(identity.created_at, "PPP")}
                        </div>
                      </div>
                    </div>
                    {actions.length > 0 && <RowActions items={actions} />}
                  </div>
                </div>
              )
            })}
          </div>
        )}

        {/* Pagination controls */}
        {data && data.total > 0 && (
          <div className="pt-4 border-t">
            <DataTablePagination table={table} rowCount={data.total} />
          </div>
        )}
      </div>
    </InformationCard>
  )
}
