import { ScrollText, Server, Waypoints } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { DetailTabs } from "@/components/details/DetailTabs"
import { PageHeader } from "@/components/layout"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ServiceListing } from "@/pages/services/components/ServiceListing"
import { ApiListing } from "@/pages/apis/components/ApiListing"
import { PolicyListing } from "@/pages/policies/components/PolicyListing"

const TABS = [
  { value: "services", label: "Services", icon: Server },
  { value: "apis", label: "APIs", icon: Waypoints },
  { value: "policies", label: "Policies", icon: ScrollText },
] as const

export type ApisResourcesTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function ApisResourcesPage({
  defaultTab = "services",
}: {
  defaultTab?: ApisResourcesTab
}) {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: ApisResourcesTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as ApisResourcesTab
    : defaultTab

  const handleTabChange = (tab: string) => {
    navigate(`/apis-resources?tab=${tab}`)
  }

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="APIs & Resources"
        description="Manage services, APIs, and IAM policies that define protected resources and access rules."
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

        <TabsContent value="services">
          <ServiceListing tableInCard />
        </TabsContent>
        <TabsContent value="apis">
          <ApiListing tableInCard />
        </TabsContent>
        <TabsContent value="policies">
          <PolicyListing tableInCard />
        </TabsContent>
      </DetailTabs>
    </div>
  )
}
