import { useState, useCallback, useMemo } from "react"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { IpRuleDialog } from "./IpRuleDialog"
import { ipRestrictionColumns } from "./IpRestrictionColumns"
import {
  useIpRestrictionRules,
  useCreateIpRestrictionRule,
  useUpdateIpRestrictionRule,
} from "@/hooks/useIpRestrictionRules"
import { useToast } from "@/hooks/useToast"
import type { IpRestrictionRule } from "@/services/api/ip-restriction-rules/types"

// Backend field name; the camelCase API key is not in the sort allow-list.
const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: false }]
const SEARCH_FIELDS = ["ip_address", "description"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "type", label: "Type", options: ["allow", "deny"] },
  { key: "status", label: "Status", options: ["active", "inactive"] },
]

export function IpRestrictionListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const { showSuccess, showError } = useToast()
  const createMutation = useCreateIpRestrictionRule()
  const updateMutation = useUpdateIpRestrictionRule()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [selectedRule, setSelectedRule] = useState<IpRestrictionRule | undefined>()

  const handleEdit = useCallback((rule: IpRestrictionRule) => {
    setSelectedRule(rule)
    setDialogOpen(true)
  }, [])

  const handleCreate = useCallback(() => {
    setSelectedRule(undefined)
    setDialogOpen(true)
  }, [])

  const handleDialogClose = useCallback(() => {
    setDialogOpen(false)
    setSelectedRule(undefined)
  }, [])

  const columns = useMemo(
    () => ipRestrictionColumns(handleEdit),
    [handleEdit],
  )

  const handleSubmit = async (formData: {
    description: string
    type: "allow" | "deny"
    ipAddress: string
    status: "active" | "inactive"
  }) => {
    const payload = {
      description: formData.description,
      type: formData.type,
      ip_address: formData.ipAddress,
      status: formData.status,
    }

    try {
      if (selectedRule) {
        await updateMutation.mutateAsync({
          id: selectedRule.ipRestrictionRuleId,
          data: payload,
        })
        showSuccess("IP rule updated successfully")
      } else {
        await createMutation.mutateAsync(payload)
        showSuccess("IP rule created successfully")
      }
      handleDialogClose()
    } catch (error) {
      showError(error)
    }
  }

  return (
    <>
      <ResourceListing
        tableInCard={tableInCard}
        columns={columns}
        defaultSort={DEFAULT_SORT}
        searchFields={SEARCH_FIELDS}
        searchPlaceholder="Search IP rules by address or description..."
        useData={useIpRestrictionRules}
        filterGroups={FILTER_GROUPS}
        onCreate={handleCreate}
        createLabel="Add IP Rule"
        emptyTitle="No IP rules yet"
        emptyDescription="Add your first IP restriction rule to control access based on IP address."
      />
      <IpRuleDialog
        open={dialogOpen}
        onOpenChange={handleDialogClose}
        onSubmit={handleSubmit}
        rule={selectedRule}
        isLoading={createMutation.isPending || updateMutation.isPending}
      />
    </>
  )
}
