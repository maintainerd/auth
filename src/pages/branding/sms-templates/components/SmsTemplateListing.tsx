import { useLocation, useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { smsTemplateColumns } from "./SmsTemplateColumns"
import { useSmsTemplatesList } from "@/hooks/useSmsTemplates"

const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: false }]
const SEARCH_FIELDS = ["name"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"] },
]

export function SmsTemplateListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()
  const location = useLocation()
  const from = `${location.pathname}${location.search}` || "/branding?tab=sms-templates"
  const navState = { from, backLabel: "Back to SMS Templates" }

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={smsTemplateColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search templates by name..."
      useData={useSmsTemplatesList}
      filterGroups={FILTER_GROUPS}
      onRowClick={(template) =>
        navigate(`/branding/sms-templates/${template.smsTemplateId}`, { state: navState })
      }
      emptyTitle="No SMS templates found"
      emptyDescription="SMS templates are managed by the system and can be edited after creation."
    />
  )
}
