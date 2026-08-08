import { useParams, useNavigate, useSearchParams, useLocation } from "react-router-dom"
import { Key } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useApi } from "@/hooks/useApis"
import { ApiHeader, ApiPermissionsTab } from "./components"

const TABS = [
  { value: "permissions", label: "Permissions", icon: Key },
] as const

type ApiDetailsTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function ApiDetailsPage() {
  const { apiId } = useParams<{ apiId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()

  // Honour where the user came from (e.g. a service's APIs tab) so back
  // returns there. Falls back to the listing.
  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/apis`
  const backLabel = navState?.backLabel ?? (backTo === `/apis` ? "Back to APIs" : "Back")

  const requestedTab = searchParams.get("tab")
  const activeTab: ApiDetailsTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as ApiDetailsTab
    : "permissions"
  const handleTabChange = (tab: string) => setSearchParams({ tab })

  const { data: api, isLoading, isError } = useApi(apiId || "")

  return (
    <DetailLayout
      backLabel={backLabel}
      onBack={() => navigate(backTo)}
      isLoading={isLoading}
      isError={isError || !api}
      notFoundTitle="API not found"
      notFoundDescription="The API you're looking for doesn't exist or may have been removed."
    >
      <ApiHeader api={api!} apiId={apiId!} />

      <DetailTabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList>
          {TABS.map(({ value, label, icon: Icon }) => (
            <TabsTrigger key={value} value={value} className="gap-2">
              <Icon className="size-4" />
              {label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="permissions">
          <ApiPermissionsTab apiId={apiId!} />
        </TabsContent>
      </DetailTabs>
    </DetailLayout>
  )
}
