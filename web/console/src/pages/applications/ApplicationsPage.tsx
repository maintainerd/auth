import { AppWindow, Boxes } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { DetailTabs } from "@/components/details/DetailTabs"
import { PageHeader } from "@/components/layout"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ClientListing } from "@/pages/clients/components/ClientListing"
import { WorkloadIdentityListing } from "@/pages/workload-identity/components/WorkloadIdentityListing"

const TABS = [
  { value: "clients", label: "Clients", icon: AppWindow },
  { value: "workload-identity", label: "Workload Identity", icon: Boxes },
] as const

export type ApplicationsTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function ApplicationsPage({
  defaultTab = "clients",
}: {
  defaultTab?: ApplicationsTab
}) {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: ApplicationsTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as ApplicationsTab
    : defaultTab

  const handleTabChange = (tab: string) => {
    navigate(`/applications?tab=${tab}`)
  }

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Applications"
        description="Manage OAuth clients and workload identity federations for applications and services."
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

        <TabsContent value="clients">
          <ClientListing tableInCard />
        </TabsContent>
        <TabsContent value="workload-identity">
          <WorkloadIdentityListing tableInCard />
        </TabsContent>
      </DetailTabs>
    </div>
  )
}
