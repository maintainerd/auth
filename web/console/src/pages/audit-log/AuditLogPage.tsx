import { ClipboardList } from "lucide-react"
import { PageHeader } from "@/components/layout"
import { AuditLogListing } from "./components"

export default function AuditLogPage({ standalone = true }: { standalone?: boolean }) {
  if (!standalone) return <AuditLogListing tableInCard />

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Audit Log"
        description="Track all administrative actions across your tenant."
        icon={ClipboardList}
      />
      <AuditLogListing tableInCard />
    </div>
  )
}
