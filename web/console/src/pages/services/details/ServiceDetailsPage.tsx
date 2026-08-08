import { useParams, useNavigate, useSearchParams, useLocation } from "react-router-dom"
import { Server, FileText } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useService } from "@/hooks/useServices"
import { ServiceHeader, ServiceApisTab, ServicePoliciesTab } from "./components"

const TABS = [
  { value: "apis", label: "APIs", icon: Server },
  { value: "policies", label: "Policies", icon: FileText },
] as const

type ServiceDetailsTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function ServiceDetailsPage() {
  const { serviceId } = useParams<{ serviceId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()

  // Honour where the user came from (e.g. a policy's Services tab) so back
  // returns there. Falls back to the listing.
  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/services`
  const backLabel = navState?.backLabel ?? (backTo === `/services` ? "Back to Services" : "Back")

  const requestedTab = searchParams.get("tab")
  const activeTab: ServiceDetailsTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as ServiceDetailsTab
    : "apis"
  const handleTabChange = (tab: string) => setSearchParams({ tab })

  const { data: service, isLoading, isError } = useService(serviceId || "")

  return (
    <DetailLayout
      backLabel={backLabel}
      onBack={() => navigate(backTo)}
      isLoading={isLoading}
      isError={isError || !service}
      notFoundTitle="Service not found"
      notFoundDescription="The service you're looking for doesn't exist or may have been removed."
    >
      <ServiceHeader service={service!} serviceId={serviceId!} />

      <DetailTabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList>
          {TABS.map(({ value, label, icon: Icon }) => (
            <TabsTrigger key={value} value={value} className="gap-2">
              <Icon className="size-4" />
              {label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="apis">
          <ServiceApisTab serviceId={serviceId!} />
        </TabsContent>
        <TabsContent value="policies">
          <ServicePoliciesTab serviceId={serviceId!} />
        </TabsContent>
      </DetailTabs>
    </DetailLayout>
  )
}
