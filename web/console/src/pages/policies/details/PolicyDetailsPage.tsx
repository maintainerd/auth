import { useParams, useNavigate, useSearchParams, useLocation } from "react-router-dom"
import { FileText, Server, History } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { usePolicy } from "@/hooks/usePolicies"
import { useServicesByPolicy } from "./hooks/useServicesByPolicy"
import { PolicyHeader, PolicyStatementsTab, PolicyServicesTab, PolicyHistoryTab } from "./components"

const TABS = [
  { value: "statements", label: "Statements", icon: FileText },
  { value: "services", label: "Services", icon: Server },
  { value: "history", label: "History", icon: History },
] as const

type PolicyDetailsTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function PolicyDetailsPage() {
  const { policyId } = useParams<{ policyId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()

  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/policies`
  const backLabel = navState?.backLabel ?? (backTo === `/policies` ? "Back to Policies" : "Back")

  const requestedTab = searchParams.get("tab")
  const activeTab: PolicyDetailsTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as PolicyDetailsTab
    : "statements"
  const handleTabChange = (tab: string) => setSearchParams({ tab })

  const { data: policy, isLoading, isError } = usePolicy(policyId || "")

  const { data: servicesData } = useServicesByPolicy({
    policyId: policyId || "",
    page: 1,
    limit: 1,
  })

  return (
    <DetailLayout
      backLabel={backLabel}
      onBack={() => navigate(backTo)}
      isLoading={isLoading}
      isError={isError || !policy}
      notFoundTitle="Policy not found"
      notFoundDescription="The policy you're looking for doesn't exist or may have been removed."
    >
      {policy && (
        <>
          <PolicyHeader policy={policy} policyId={policyId!} afterDeleteTo={backTo} />

          <DetailTabs value={activeTab} onValueChange={handleTabChange}>
            <TabsList>
              {TABS.map(({ value, label, icon: Icon }) => (
                <TabsTrigger key={value} value={value} className="gap-2">
                  <Icon className="size-4" />
                  {label}
                  {value === "services" && servicesData?.total ? ` (${servicesData.total})` : ""}
                </TabsTrigger>
              ))}
            </TabsList>

            <TabsContent value="statements">
              <PolicyStatementsTab
                documentVersion={policy.document.version}
                statements={policy.document.statement}
              />
            </TabsContent>
            <TabsContent value="services">
              <PolicyServicesTab policyId={policyId!} />
            </TabsContent>
            <TabsContent value="history">
              <PolicyHistoryTab policyId={policyId!} />
            </TabsContent>
          </DetailTabs>
        </>
      )}
    </DetailLayout>
  )
}
