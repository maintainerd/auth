import { useParams, useNavigate, useSearchParams } from "react-router-dom"
import { AppWindow, Braces, Building2, KeyRound, Link2, Server, Settings, Shield } from "lucide-react"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useClient } from "@/hooks/useClients"
import { ClientHeader, ClientInformation, ClientCredentials, ClientConfig, ClientMetadata, ClientApis, ClientUris, ClientIdentityProviders, ClientRoles } from "./components"

const TABS = [
  { value: "overview", label: "Overview", icon: AppWindow },
  { value: "identity-providers", label: "Identity Providers", icon: Building2 },
  { value: "credentials", label: "Credentials", icon: KeyRound },
  { value: "config", label: "OAuth & Tokens", icon: Settings },
  { value: "uris", label: "URIs & Origins", icon: Link2 },
  { value: "apis", label: "API Permissions", icon: Server },
  { value: "roles", label: "Roles", icon: Shield },
  { value: "metadata", label: "Metadata", icon: Braces },
] as const

type ClientDetailsTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set<string>(TABS.map((tab) => tab.value))

export default function ClientDetailsPage() {
  const { clientId } = useParams<{ clientId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const requestedTab = searchParams.get("tab")
  const activeTab: ClientDetailsTab = TAB_VALUES.has(requestedTab || "")
    ? requestedTab as ClientDetailsTab
    : "overview"

  const handleTabChange = (tab: string) => {
    setSearchParams({ tab })
  }

  const { data: clientData, isLoading, isError } = useClient(clientId || '')

  return (
    <DetailLayout
      backLabel="Back to Clients"
      onBack={() => navigate(`/clients`)}
      isLoading={isLoading}
      isError={isError || !clientData}
      notFoundTitle="Client not found"
      notFoundDescription="The client you're looking for doesn't exist or may have been removed."
    >
      <ClientHeader client={clientData!} clientId={clientId!} />

      <DetailTabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList>
          {TABS.map(({ value, label, icon: Icon }) => (
            <TabsTrigger key={value} value={value} className="gap-2">
              <Icon className="size-4" />
              {label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="overview">
          <ClientInformation client={clientData!} />
        </TabsContent>

        <TabsContent value="identity-providers">
          <ClientIdentityProviders client={clientData!} />
        </TabsContent>

        <TabsContent value="credentials">
          <ClientCredentials client={clientData!} />
        </TabsContent>

        <TabsContent value="config">
          <ClientConfig clientId={clientId!} />
        </TabsContent>

        <TabsContent value="uris">
          <ClientUris client={clientData!} />
        </TabsContent>

        <TabsContent value="apis">
          <ClientApis clientId={clientId!} />
        </TabsContent>

        <TabsContent value="roles">
          <ClientRoles clientId={clientId!} />
        </TabsContent>

        <TabsContent value="metadata">
          <ClientMetadata clientId={clientId!} />
        </TabsContent>
      </DetailTabs>
    </DetailLayout>
  )
}
