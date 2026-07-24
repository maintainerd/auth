import { useParams, useNavigate, useSearchParams } from "react-router-dom"
import { Users } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useTenantById } from "@/hooks/useTenants"
import { TenantHeader, TenantMembers } from "./components"

const TABS = [
  { value: "members", label: "Members", icon: Users },
] as const

type TenantDetailsTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function TenantDetailsPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: TenantDetailsTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as TenantDetailsTab
    : "members"

  const handleTabChange = (tab: string) => setSearchParams({ tab })

  const { data: tenant, isLoading, isError } = useTenantById(id)

  return (
    <DetailLayout
      backLabel="Back to Tenants"
      onBack={() => navigate(`/tenants`)}
      isLoading={isLoading}
      isError={isError || !tenant}
      notFoundTitle="Tenant not found"
      notFoundDescription="The tenant you're looking for doesn't exist or may have been removed."
    >
      {tenant && (
        <>
          <TenantHeader tenant={tenant} tenantId={id!} />

          <DetailTabs value={activeTab} onValueChange={handleTabChange}>
            <TabsList>
              {TABS.map(({ value, label, icon: Icon }) => (
                <TabsTrigger key={value} value={value} className="gap-2">
                  <Icon className="size-4" />
                  {label}
                </TabsTrigger>
              ))}
            </TabsList>

            <TabsContent value="members">
              <TenantMembers isSystemTenant={tenant.is_system} />
            </TabsContent>
          </DetailTabs>
        </>
      )}
    </DetailLayout>
  )
}
