import { Gauge, FileText, Wrench } from "lucide-react"
import { useSearchParams } from "react-router-dom"
import { DetailsContainer } from "@/components/container"
import { PageHeader } from "@/components/layout"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  AuditSettingsPanel,
  MaintenanceSettingsPanel,
  RateLimitSettingsPanel,
} from "./components/SettingsPanels"

const SETTINGS_TABS = [
  {
    value: "rate-limit",
    label: "Rate Limits",
    icon: Gauge,
  },
  {
    value: "audit",
    label: "Audit",
    icon: FileText,
  },
  {
    value: "maintenance",
    label: "Maintenance",
    icon: Wrench,
  },
] as const

type SettingsTab = (typeof SETTINGS_TABS)[number]["value"]

const DEFAULT_TAB: SettingsTab = "rate-limit"

function isSettingsTab(value: string | null): value is SettingsTab {
  return SETTINGS_TABS.some((tab) => tab.value === value)
}

export function TenantSettingsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const requestedTab = searchParams.get("tab")
  const tab: SettingsTab = isSettingsTab(requestedTab) ? requestedTab : DEFAULT_TAB

  return (
    <DetailsContainer>
      <div className="flex flex-col gap-6">
        <PageHeader
          title="Tenant Settings"
          description="Manage operational controls for this tenant."
        />

        <Tabs
          value={tab}
          onValueChange={(value) => setSearchParams(value === DEFAULT_TAB ? {} : { tab: value })}
          className="space-y-6"
        >
          <TabsList className="h-auto flex-wrap justify-start">
            {SETTINGS_TABS.map((settingsTab) => (
              <TabsTrigger key={settingsTab.value} value={settingsTab.value} className="gap-2">
                <settingsTab.icon className="size-4" />
                {settingsTab.label}
              </TabsTrigger>
            ))}
          </TabsList>

          <TabsContent value="rate-limit">
            <RateLimitSettingsPanel />
          </TabsContent>
          <TabsContent value="audit">
            <AuditSettingsPanel />
          </TabsContent>
          <TabsContent value="maintenance">
            <MaintenanceSettingsPanel />
          </TabsContent>
        </Tabs>
      </div>
    </DetailsContainer>
  )
}
