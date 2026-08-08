import { useSearchParams } from "react-router-dom"
import { ClipboardList, TrendingUp } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { PageHeader } from "@/components/layout"

import LogMonitoringPage from "../log-monitoring/LogMonitoringPage"
import AuditLogPage from "../audit-log/AuditLogPage"

const TABS = [
  { value: "logs", label: "Sign-in Logs", icon: TrendingUp },
  { value: "audit", label: "Audit Log", icon: ClipboardList },
] as const

type MonitoringTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function MonitoringPage() {
  const [searchParams, setSearchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: MonitoringTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as MonitoringTab
    : "logs"

  const handleTabChange = (tab: string) => setSearchParams({ tab })

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Monitoring"
        description="View sign-in activity and track administrative changes across your tenant."
      />

      <DetailTabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList>
          {TABS.map(({ value, label, icon: Icon }) => (
            <TabsTrigger key={value} value={value} className="gap-2">
              <Icon className="size-4" />
              {label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="logs">
          <LogMonitoringPage standalone={false} />
        </TabsContent>
        <TabsContent value="audit">
          <AuditLogPage standalone={false} />
        </TabsContent>
      </DetailTabs>
    </div>
  )
}
