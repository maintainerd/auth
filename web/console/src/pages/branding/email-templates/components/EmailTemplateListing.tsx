import { useLocation, useNavigate } from "react-router-dom"
import type { SortingState } from "@tanstack/react-table"
import { ResourceListing, type FilterGroup } from "@/components/data-table"
import { emailTemplateColumns } from "./EmailTemplateColumns"
import { useEmailTemplatesList } from "@/hooks/useEmailTemplates"

const DEFAULT_SORT: SortingState = [{ id: "created_at", desc: false }]
const SEARCH_FIELDS = ["name", "subject"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"] },
]

export function EmailTemplateListing({ tableInCard }: { tableInCard?: boolean } = {}) {
  const navigate = useNavigate()
  const location = useLocation()
  const from = `${location.pathname}${location.search}` || "/branding?tab=email-templates"
  const navState = { from, backLabel: "Back to Email Templates" }

  return (
    <ResourceListing
      tableInCard={tableInCard}
      columns={emailTemplateColumns}
      defaultSort={DEFAULT_SORT}
      searchFields={SEARCH_FIELDS}
      searchPlaceholder="Search templates by name or subject..."
      useData={useEmailTemplatesList}
      filterGroups={FILTER_GROUPS}
      onRowClick={(template) =>
        navigate(`/branding/email-templates/${template.emailTemplateId}`, { state: navState })
      }
      emptyTitle="No email templates"
      emptyDescription="Email templates are seeded automatically and can be edited after creation."
    />
  )
}
