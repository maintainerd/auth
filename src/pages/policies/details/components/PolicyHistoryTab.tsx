import { useState } from "react"
import { History, Eye } from "lucide-react"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { usePolicyHistory } from "@/hooks/usePolicies"
import { safeFormat } from "@/lib/formatDate"
import type { PolicyVersionHistory } from "@/services/api/policies/types"

interface PolicyHistoryTabProps {
  policyId: string
}

export function PolicyHistoryTab({ policyId }: PolicyHistoryTabProps) {
  const { data, isLoading, isError } = usePolicyHistory(policyId)
  const [selected, setSelected] = useState<PolicyVersionHistory | null>(null)
  const versions = data?.rows ?? []

  return (
    <>
      <InformationCard
        title="Version History"
        description="Previous versions of this policy's document, captured on every change"
        icon={History}
      >
        <div className="space-y-4">
          {isLoading && <ListSkeleton />}

          {isError && (
            <p className="py-8 text-center text-sm text-destructive">Failed to load history</p>
          )}

          {data && versions.length === 0 && (
            <EmptyState
              icon={History}
              title="No history"
              description="No previous versions of this policy exist."
            />
          )}

          {versions.length > 0 && (
            <div className="space-y-3">
              {versions.map((v) => (
                <div
                  key={v.version_number}
                  data-md-listing-item
                  className="flex items-start justify-between gap-3 rounded-lg border p-4"
                >
                  <div className="flex min-w-0 items-start gap-3">
                    <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                      <History className="size-4" />
                    </div>
                    <div className="min-w-0 space-y-1">
                      <span className="text-sm font-medium">Version {v.version_number}</span>
                      <p className="text-sm text-muted-foreground">
                        {safeFormat(v.snapshot_at, "PPpp")}
                        {v.changed_by_user_id && ` — by ${v.changed_by_user_id}`}
                      </p>
                    </div>
                  </div>
                  <Button variant="outline" size="sm" className="gap-2" onClick={() => setSelected(v)}>
                    <Eye className="size-4" />
                    View
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </InformationCard>

      <Dialog open={!!selected} onOpenChange={() => setSelected(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Version {selected?.version_number}</DialogTitle>
          </DialogHeader>
          {selected && (
            <pre className="max-h-96 overflow-auto rounded-md bg-muted p-3 text-xs">
              {JSON.stringify(selected.document, null, 2)}
            </pre>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}
