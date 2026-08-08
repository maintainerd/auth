import { useParams, useNavigate, useSearchParams, useLocation } from "react-router-dom"
import {
  LayoutTemplate,
  Palette,
  Users,
} from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useBranding } from "@/hooks/useBranding"
import { useClients } from "@/hooks/useClients"
import { THEME_TOKENS, tokensFromMetadata, tokenId, isHex } from "../themeTokens"
import { BrandingHeader } from "./components/BrandingHeader"
import type { Branding } from "@/services/api/branding/types"
import {
  authUiTemplateIdFromMetadata,
  getAuthUiTemplate,
} from "@/lib/branding/authUiTemplates"
import { BRANDING_THEMES_LIST_URL } from "../brandingNavigation"

const TABS = [
  { value: "overview", label: "Overview", icon: LayoutTemplate },
  { value: "theme", label: "Theme", icon: Palette },
  { value: "clients", label: "Clients", icon: Users },
] as const

type BrandingDetailsTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set(TABS.map((tab) => tab.value))

function isBrandingDetailsTab(tab: string): tab is BrandingDetailsTab {
  return TAB_VALUES.has(tab as BrandingDetailsTab)
}

export default function BrandingDetailsPage() {
  const { brandingId } = useParams<{ brandingId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()
  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? BRANDING_THEMES_LIST_URL
  const backLabel = navState?.backLabel ?? "Back to Themes"

  const requestedTab = searchParams.get("tab") || ""
  const normalizedRequestedTab =
    requestedTab === "ui-templates" || requestedTab === "login-templates" || requestedTab === "details"
      ? "overview"
      : requestedTab
  const activeTab: BrandingDetailsTab = isBrandingDetailsTab(normalizedRequestedTab)
    ? normalizedRequestedTab
    : "overview"
  const handleTabChange = (tab: string) => setSearchParams({ tab })

  const { data: branding, isLoading, isError } = useBranding(brandingId)

  return (
    <DetailLayout
      backLabel={backLabel}
      onBack={() => navigate(backTo)}
      isLoading={isLoading}
      isError={isError || !branding}
      notFoundTitle="Branding not found"
      notFoundDescription="The branding template you're looking for doesn't exist or may have been removed."
    >
      {branding && (
        <>
          <BrandingHeader branding={branding} brandingId={brandingId!} />

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
              <OverviewTab branding={branding} />
            </TabsContent>
            <TabsContent value="theme">
              <ThemeTab branding={branding} />
            </TabsContent>
            <TabsContent value="clients">
              <ClientsTab brandingId={branding.branding_id} />
            </TabsContent>
          </DetailTabs>
        </>
      )}
    </DetailLayout>
  )
}

function OverviewTab({ branding }: { branding: Branding }) {
  return (
    <div className="space-y-6">
      <BrandAssetsLinksSection branding={branding} />
      <SelectedLoginTemplateSection branding={branding} />
    </div>
  )
}

function SelectedLoginTemplateSection({ branding }: { branding: Branding }) {
  const selectedTemplateId = authUiTemplateIdFromMetadata(branding.metadata, branding.layout)
  const selectedTemplate = getAuthUiTemplate(selectedTemplateId)

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Login Template</CardTitle>
        <p className="text-sm text-muted-foreground">
          The selected hosted-auth layout shell.
        </p>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0 space-y-1">
            <h3 className="text-base font-semibold">{selectedTemplate.label}</h3>
            <p className="text-sm text-muted-foreground">{selectedTemplate.summary}</p>
          </div>
          <span className="w-fit shrink-0 rounded-md border bg-muted px-2 py-1 text-xs font-medium text-muted-foreground">
            {selectedTemplate.layout.replace("_", " ")}
          </span>
        </div>
      </CardContent>
    </Card>
  )
}

function ThemeTab({ branding }: { branding: Branding }) {
  const tokens = tokensFromMetadata(branding.metadata)
  const colors = THEME_TOKENS.filter((t) => t.kind === "color")
  const fontFamily = tokens["font.family"]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Theme tokens</CardTitle>
        <p className="text-sm text-muted-foreground">
          The colors and typography applied by this template.
        </p>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {colors.map((t) => {
            const value = tokens[tokenId(t)]
            return (
              <div key={tokenId(t)} className="flex items-center gap-3">
                <span
                  className="size-10 shrink-0 rounded-lg border"
                  style={{ backgroundColor: isHex(value) ? value : "transparent" }}
                  aria-hidden
                />
                <div className="min-w-0">
                  <p className="text-sm font-medium">{t.label}</p>
                  <p className="font-mono text-xs text-muted-foreground">{value || "—"}</p>
                </div>
              </div>
            )
          })}
        </div>

        <div className="border-t pt-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Font family
          </p>
          <p className="mt-1 text-sm" style={{ fontFamily: fontFamily || undefined }}>
            {fontFamily || "—"}
          </p>
        </div>
      </CardContent>
    </Card>
  )
}

function BrandAssetsLinksSection({ branding }: { branding: Branding }) {
  const links: { label: string; value: string }[] = [
    { label: "Company name", value: branding.company_name },
    { label: "Logo label (console)", value: branding.logo_label },
    { label: "Logo detail (console)", value: branding.logo_detail },
    {
      label: "Show logo label (console)",
      value: (branding.show_logo_label ?? true) ? "Yes" : "No",
    },
    { label: "Logo label (identity)", value: branding.identity_logo_label },
    {
      label: "Show logo label (identity)",
      value: (branding.identity_show_logo_label ?? true) ? "Yes" : "No",
    },
    { label: "Logo URL", value: branding.logo_url },
    { label: "Favicon URL", value: branding.favicon_url },
    { label: "Support URL", value: branding.support_url },
    { label: "Privacy policy URL", value: branding.privacy_policy_url },
    { label: "Terms of service URL", value: branding.terms_of_service_url },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Brand Assets &amp; Links</CardTitle>
        <p className="text-sm text-muted-foreground">
          Company name and the URLs surfaced across the auth experience.
        </p>
      </CardHeader>
      <CardContent className="divide-y">
        {links.map((item) => {
          const isUrl = item.label.endsWith("URL") && !!item.value
          return (
            <div
              key={item.label}
              className="grid grid-cols-1 gap-1 py-3 sm:grid-cols-[220px_1fr] sm:items-center"
            >
              <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                {item.label}
              </span>
              {isUrl ? (
                <a
                  href={item.value}
                  target="_blank"
                  rel="noreferrer"
                  className="truncate text-sm text-primary hover:underline"
                >
                  {item.value}
                </a>
              ) : (
                <span className="truncate text-sm">{item.value || "—"}</span>
              )}
            </div>
          )
        })}
      </CardContent>
    </Card>
  )
}

function ClientsTab({ brandingId }: { brandingId: string }) {
  const { data: clientsData, isLoading } = useClients({ limit: 200 })

  const matchingClients = (clientsData?.rows ?? []).filter(
    (c) => c.branding_id === brandingId
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Clients using this branding</CardTitle>
        <p className="text-sm text-muted-foreground">
          {matchingClients.length > 0
            ? `${matchingClients.length} client${matchingClients.length !== 1 ? "s" : ""} explicitly using this branding.`
            : "No clients are explicitly using this branding template."}
        </p>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <p className="py-6 text-center text-sm text-muted-foreground">Loading clients…</p>
        ) : matchingClients.length === 0 ? (
          <div className="py-6 text-center space-y-2">
            <p className="text-sm text-muted-foreground">
              Clients without an explicit branding template fall back to the tenant's active branding.
            </p>
            <p className="text-xs text-muted-foreground">
              Deleting this branding template will return any using clients to the tenant's active branding fallback.
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            <ul className="divide-y">
              {matchingClients.map((client) => (
                <li key={client.client_id} className="py-2">
                  <span className="text-sm font-medium">{client.name}</span>
                  <span className="ml-2 font-mono text-xs text-muted-foreground">
                    {client.client_id}
                  </span>
                </li>
              ))}
            </ul>
            <p className="text-xs text-muted-foreground">
              Deleting this branding template will return these clients to the tenant's active branding fallback.
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
