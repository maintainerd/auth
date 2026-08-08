import { useState, useEffect, type FormEvent } from "react"
import { Plus, Search } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { usePolicies } from "@/hooks/usePolicies"
import { useDebouncedSearch } from "@/hooks/useDebouncedSearch"
import { useServicePolicyMutations } from "../hooks/useServicePolicyMutations"
import { useToast } from "@/hooks/useToast"

/**
 * One page is all this dialog fetches. Search is server-side, so this is a cap
 * on what one query returns, not a cap on what is reachable — see PAGE_LIMIT's
 * use alongside the "refine your search" hint below.
 */
const PAGE_LIMIT = 100

interface PolicyAssignDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  serviceId: string
  existingPolicyIds: string[]
}

export function PolicyAssignDialog({
  open,
  onOpenChange,
  serviceId,
  existingPolicyIds,
}: PolicyAssignDialogProps) {
  const [selectedPolicies, setSelectedPolicies] = useState<string[]>([])
  // Search is sent to the API (PolicyFilterDTO.Name is matched with ILIKE), not
  // applied to an already-truncated page — a client-side filter over the first
  // 100 rows made policy 101 unreachable in a required picker.
  const { searchInput, debouncedValue, handleSearchChange, handleKeyDown, setSearchValue } =
    useDebouncedSearch()

  const { showSuccess, showError } = useToast()
  const { assignPolicy } = useServicePolicyMutations(serviceId)

  const { data: policiesData, isLoading: isLoadingPolicies } = usePolicies(
    {
      page: 1,
      limit: PAGE_LIMIT,
      sort_by: 'name',
      sort_order: 'asc',
      name: debouncedValue || undefined,
    },
    { enabled: open }
  )

  useEffect(() => {
    if (!open) {
      setSelectedPolicies([])
      setSearchValue("")
    }
  }, [open, setSearchValue])

  const handlePolicyToggle = (policyId: string) => {
    setSelectedPolicies(prev =>
      prev.includes(policyId)
        ? prev.filter(id => id !== policyId)
        : [...prev, policyId]
    )
  }

  const handleSubmit = async (e?: FormEvent) => {
    e?.preventDefault()
    if (selectedPolicies.length === 0) {
      showError("Please select at least one policy")
      return
    }

    try {
      await Promise.all(
        selectedPolicies.map(policyId => assignPolicy.mutateAsync(policyId))
      )

      showSuccess(`${selectedPolicies.length} polic${selectedPolicies.length !== 1 ? 'ies' : 'y'} assigned successfully`)
      onOpenChange(false)
    } catch (error) {
      showError(error)
    }
  }

  const isLoading = assignPolicy.isPending

  // The API has no "exclude these ids" filter, so already-assigned policies are
  // the one thing still dropped client-side.
  const filteredPolicies = policiesData?.rows?.filter(
    policy => !existingPolicyIds.includes(policy.policy_id)
  ) ?? []

  const visiblePolicyIds = filteredPolicies.map(p => p.policy_id)
  // Compared against the rendered rows only. Under a search the fetched page is
  // already narrowed, so "all selected" must mean "all of what you can see".
  const selectedVisibleCount = visiblePolicyIds.filter(id => selectedPolicies.includes(id)).length
  const allVisibleSelected =
    visiblePolicyIds.length > 0 && selectedVisibleCount === visiblePolicyIds.length
  // Tri-state, so a partial selection reports aria-checked="mixed" instead of
  // an unchecked box that hides the fact that some rows are already ticked.
  const selectAllState: boolean | "indeterminate" = allVisibleSelected
    ? true
    : selectedVisibleCount > 0
      ? "indeterminate"
      : false

  const handleSelectAll = () => {
    setSelectedPolicies(prev =>
      allVisibleSelected
        ? prev.filter(id => !visiblePolicyIds.includes(id))
        : [...new Set([...prev, ...visiblePolicyIds])]
    )
  }

  // The server may hold more matches than one page returns; saying so is what
  // stops a user concluding their policy does not exist.
  const hasMoreMatches = (policiesData?.total ?? 0) > (policiesData?.rows?.length ?? 0)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Assign Policies to Service</DialogTitle>
          <DialogDescription>
            Select policies to assign to this service. Already assigned policies are not shown.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label>Select Policies</Label>
              {filteredPolicies.length > 0 && (
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="select-all-policies"
                    checked={selectAllState}
                    onCheckedChange={handleSelectAll}
                    disabled={isLoading}
                  />
                  <Label
                    htmlFor="select-all-policies"
                    className="cursor-pointer text-xs font-normal"
                  >
                    Select all
                  </Label>
                </div>
              )}
            </div>

            <div className="relative">
              <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search policies..."
                value={searchInput}
                onChange={(e) => handleSearchChange(e.target.value)}
                onKeyDown={handleKeyDown}
                className="pl-8"
              />
            </div>
          </div>

          <div className="border rounded-lg max-h-[400px] overflow-y-auto">
            {isLoadingPolicies && (
              <div className="text-center py-8 text-muted-foreground text-sm">
                Loading policies...
              </div>
            )}

            {!isLoadingPolicies && filteredPolicies.length === 0 && (
              <div className="text-center py-8 text-muted-foreground text-sm">
                {debouncedValue
                  ? "No policies found matching your search"
                  : "All available policies are already assigned"}
              </div>
            )}

            {!isLoadingPolicies && filteredPolicies.length > 0 && (
              <div className="divide-y">
                {filteredPolicies.map((policy) => (
                  <div
                    key={policy.policy_id}
                    className="flex items-start gap-3 p-3 hover:bg-accent/50 transition-colors"
                  >
                    <Checkbox
                      id={policy.policy_id}
                      checked={selectedPolicies.includes(policy.policy_id)}
                      onCheckedChange={() => handlePolicyToggle(policy.policy_id)}
                      disabled={isLoading}
                    />
                    <label
                      htmlFor={policy.policy_id}
                      className="flex-1 cursor-pointer"
                    >
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{policy.name}</span>
                        {policy.is_system && (
                          <span className="text-xs px-1.5 py-0.5 rounded bg-secondary text-secondary-foreground">
                            System
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-muted-foreground mt-0.5">
                        {policy.description}
                      </div>
                    </label>
                  </div>
                ))}
              </div>
            )}
          </div>

          {!isLoadingPolicies && hasMoreMatches && (
            <p className="text-sm text-muted-foreground">
              Showing the first {policiesData?.rows?.length ?? 0} of {policiesData?.total ?? 0} policies. Refine your search to narrow the list.
            </p>
          )}

          {selectedPolicies.length > 0 && (
            <p className="text-sm text-muted-foreground">
              {selectedPolicies.length} polic{selectedPolicies.length !== 1 ? 'ies' : 'y'} selected
            </p>
          )}
        </div>

        <DialogFooter>
          <form onSubmit={handleSubmit} className="contents">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={selectedPolicies.length === 0 || isLoading}
              className="gap-2"
            >
              {isLoading ? (
                <>Assigning...</>
              ) : (
                <>
                  <Plus className="h-4 w-4" />
                  Assign Policies
                </>
              )}
            </Button>
          </form>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
