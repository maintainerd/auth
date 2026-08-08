import { useParams, useNavigate, useSearchParams } from "react-router-dom"
import { ShieldCheck, KeyRound } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useWorkloadIdentity } from "@/hooks/useWorkloadIdentity"
import {
  WorkloadIdentityHeader,
  WorkloadIdentityTrust,
  WorkloadIdentityIssuedToken,
} from "./components"

// The two halves of a federation: what it ACCEPTS and what it ISSUES. This mirrors
// the two cards on the form. A federation owns no sub-resources — allowed_scopes and
// attribute_mapping are attributes of the row — so there is no third tab to add.
const TABS = [
  { value: "trust", label: "Trust", icon: ShieldCheck },
  { value: "issued-token", label: "Issued Token", icon: KeyRound },
] as const

type WorkloadIdentityDetailsTab = (typeof TABS)[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function WorkloadIdentityDetailsPage() {
  const { federationId } = useParams<{ federationId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: WorkloadIdentityDetailsTab = TAB_VALUES.has(requestedTab || "")
    ? (requestedTab as WorkloadIdentityDetailsTab)
    : "trust"

  const handleTabChange = (tab: string) => {
    setSearchParams({ tab })
  }

  const { data: federation, isLoading, isError } = useWorkloadIdentity(federationId || "")

  return (
    <DetailLayout
      backLabel="Back to Workload Identity"
      onBack={() => navigate(`/workload-identity`)}
      isLoading={isLoading}
      isError={isError || !federation}
      notFoundTitle="Federation not found"
      notFoundDescription="The workload identity federation you're looking for doesn't exist or may have been removed."
    >
      {federation && (
        <>
          <WorkloadIdentityHeader federation={federation} federationId={federationId!} />

          <DetailTabs value={activeTab} onValueChange={handleTabChange}>
            <TabsList>
              {TABS.map(({ value, label, icon: Icon }) => (
                <TabsTrigger key={value} value={value} className="gap-2">
                  <Icon className="size-4" />
                  {label}
                </TabsTrigger>
              ))}
            </TabsList>

            <TabsContent value="trust">
              <WorkloadIdentityTrust federation={federation} />
            </TabsContent>

            <TabsContent value="issued-token">
              <WorkloadIdentityIssuedToken federation={federation} />
            </TabsContent>
          </DetailTabs>
        </>
      )}
    </DetailLayout>
  )
}
