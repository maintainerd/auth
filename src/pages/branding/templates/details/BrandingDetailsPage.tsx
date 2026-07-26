import type { CSSProperties, KeyboardEvent, ReactNode } from "react"
import { useState } from "react"
import { useParams, useNavigate, useSearchParams, useLocation } from "react-router-dom"
import {
  CheckCircle2,
  Eye,
  LayoutTemplate,
  Link2,
  Loader2,
  Maximize2,
  Minimize2,
  Palette,
  ShieldCheck,
  Users,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DetailTabs } from "@/components/details/DetailTabs"
import { DetailLayout } from "@/components/details"
import { useBranding, useUpdateBranding } from "@/hooks/useBranding"
import { useClients } from "@/hooks/useClients"
import { useToast } from "@/hooks/useToast"
import { THEME_TOKENS, tokensFromMetadata, tokenId, isHex } from "../themeTokens"
import { BrandingHeader } from "./components/BrandingHeader"
import type { Branding, BrandingRequest } from "@/services/api/branding/types"
import {
  AUTH_UI_TEMPLATES,
  authUiTemplateIdFromMetadata,
  authUiTemplateSupportsImage,
  getAuthUiTemplate,
  type AuthUiTemplate,
} from "@/lib/branding/authUiTemplates"
import {
  loginPagePreviewsFromMetadata,
  loginTemplateImageUrlFromMetadata,
  type LoginPageElement,
  type LoginPagePreview,
  type LoginPagePreviewId,
} from "@/lib/branding/loginPageContent"
import { BRANDING_THEMES_LIST_URL } from "../brandingNavigation"

const TABS = [
  { value: "theme", label: "Theme", icon: Palette },
  { value: "details", label: "Defaults", icon: Link2 },
  { value: "clients", label: "Clients", icon: Users },
  { value: "login-templates", label: "Login Templates", icon: LayoutTemplate },
  { value: "preview", label: "Preview", icon: Eye },
] as const

type BrandingDetailsTab = typeof TABS[number]["value"]

const TAB_VALUES = new Set(TABS.map((tab) => tab.value))

function isBrandingDetailsTab(tab: string): tab is BrandingDetailsTab {
  return TAB_VALUES.has(tab as BrandingDetailsTab)
}

function brandingRequestFromTemplate(branding: Branding, template: AuthUiTemplate): BrandingRequest {
  return {
    name: branding.name,
    layout: template.layout,
    company_name: branding.company_name ?? "",
    logo_url: branding.logo_url ?? "",
    favicon_url: branding.favicon_url ?? "",
    support_url: branding.support_url ?? "",
    privacy_policy_url: branding.privacy_policy_url ?? "",
    terms_of_service_url: branding.terms_of_service_url ?? "",
    metadata: {
      ...(branding.metadata ?? {}),
      auth_ui_template: template.id,
    },
  }
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
  const normalizedRequestedTab = requestedTab === "ui-templates" ? "login-templates" : requestedTab
  const activeTab: BrandingDetailsTab = isBrandingDetailsTab(normalizedRequestedTab)
    ? normalizedRequestedTab
    : "theme"
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

            <TabsContent value="theme">
              <ThemeTab branding={branding} />
            </TabsContent>
            <TabsContent value="details">
              <DetailsTab branding={branding} />
            </TabsContent>
            <TabsContent value="clients">
              <ClientsTab brandingId={branding.branding_id} />
            </TabsContent>
            <TabsContent value="login-templates">
              <LoginTemplatesTab branding={branding} />
            </TabsContent>
            <TabsContent value="preview">
              <PreviewTab branding={branding} />
            </TabsContent>
          </DetailTabs>
        </>
      )}
    </DetailLayout>
  )
}

function LoginTemplatesTab({ branding }: { branding: Branding }) {
  const navigate = useNavigate()
  const updateMutation = useUpdateBranding()
  const { showSuccess, showError } = useToast()
  const [savingTemplateId, setSavingTemplateId] = useState<string | null>(null)
  const selectedTemplateId = authUiTemplateIdFromMetadata(branding.metadata, branding.layout)
  const selectedTemplate = getAuthUiTemplate(selectedTemplateId)
  const isSaving = updateMutation.isPending

  const selectTemplate = async (template: AuthUiTemplate) => {
    if (template.id === selectedTemplate.id || isSaving) return

    setSavingTemplateId(template.id)
    try {
      await updateMutation.mutateAsync({
        brandingId: branding.branding_id,
        data: brandingRequestFromTemplate(branding, template),
      })
      showSuccess(`"${template.label}" is now the login template`)
    } catch (error) {
      showError(error)
    } finally {
      setSavingTemplateId(null)
    }
  }

  const handleTemplateKeyDown = (event: KeyboardEvent<HTMLDivElement>, template: AuthUiTemplate) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault()
      void selectTemplate(template)
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <CardTitle className="text-base">Selected login template</CardTitle>
              <p className="text-sm text-muted-foreground">
                This is the saved hosted-auth layout shell for login, registration, MFA, and account-linking pages.
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              className="w-fit shrink-0"
              onClick={() =>
                navigate(`/branding/templates/${branding.branding_id}/forms`, {
                  state: {
                    from: `/branding/templates/${branding.branding_id}?tab=login-templates`,
                    backLabel: "Back to Login Templates",
                  },
                })
              }
            >
              Update forms
            </Button>
          </div>
        </CardHeader>
        <CardContent className="grid gap-4 lg:grid-cols-[1fr_280px]">
          <div className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="text-lg font-semibold">{selectedTemplate.label}</h3>
              <span className="rounded-md border bg-muted px-2 py-1 text-xs font-medium text-muted-foreground">
                {selectedTemplate.layout.replace("_", " ")}
              </span>
            </div>
            <p className="text-sm text-muted-foreground">{selectedTemplate.summary}</p>
            <p className="text-sm">{selectedTemplate.flowTreatment}</p>
          </div>
          <div className="rounded-md border bg-muted/30 p-4">
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Best for</p>
            <p className="mt-1 text-sm">{selectedTemplate.bestFor}</p>
            <p className="mt-4 text-xs font-medium uppercase tracking-wide text-muted-foreground">Configured pages</p>
            <div className="mt-2 flex flex-wrap gap-2">
              {selectedTemplate.features.map((feature) => (
                <span key={feature} className="rounded-md border bg-background px-2 py-1 text-xs">
                  {feature}
                </span>
              ))}
            </div>
            <div className="mt-4 border-t pt-4">
              <TemplateMeta
                label="Editable form text"
                value={`${loginPagePreviewsFromMetadata(branding.metadata).length} page states`}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        {AUTH_UI_TEMPLATES.map((template) => {
          const selected = template.id === selectedTemplate.id
          const saving = savingTemplateId === template.id
          return (
            <Card
              key={template.id}
              role="button"
              tabIndex={isSaving ? -1 : 0}
              aria-pressed={selected}
              onClick={() => void selectTemplate(template)}
              onKeyDown={(event) => handleTemplateKeyDown(event, template)}
              className={`cursor-pointer transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                selected ? "border-primary/50 bg-primary/5" : "hover:border-primary/40"
              } ${isSaving ? "pointer-events-none opacity-70" : ""}`}
            >
              <CardHeader>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <CardTitle className="text-base">{template.label}</CardTitle>
                    <p className="mt-1 text-sm text-muted-foreground">{template.summary}</p>
                  </div>
                  {saving ? (
                    <Loader2 className="size-5 shrink-0 animate-spin text-primary" />
                  ) : selected ? (
                    <CheckCircle2 className="size-5 shrink-0 text-primary" />
                  ) : null}
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                <TemplateWireframe template={template} />
                <div className="grid gap-3 sm:grid-cols-2">
                  <TemplateMeta label="Layout" value={template.layout.replace("_", " ")} />
                  <TemplateMeta label="Best for" value={template.bestFor} />
                </div>
                <div className="flex flex-wrap gap-2">
                  {template.features.map((feature) => (
                    <span key={feature} className="rounded-md border px-2 py-1 text-xs text-muted-foreground">
                      {feature}
                    </span>
                  ))}
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>
    </div>
  )
}

function PreviewTab({ branding }: { branding: Branding }) {
  const [selectedPageId, setSelectedPageId] = useState<LoginPagePreviewId>("login")
  const [isFullscreenPreview, setIsFullscreenPreview] = useState(false)
  const tokens = tokensFromMetadata(branding.metadata)
  const template = getAuthUiTemplate(authUiTemplateIdFromMetadata(branding.metadata, branding.layout))
  const pages = loginPagePreviewsFromMetadata(branding.metadata)
  const selectedPage = pages.find((page) => page.id === selectedPageId) ?? pages[0]

  return (
    <>
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Identity page preview</CardTitle>
            <p className="text-sm text-muted-foreground">
              Select one fixed hosted-auth page state to preview with the selected login template.
            </p>
          </CardHeader>
          <CardContent>
            <div className="w-full rounded-md border bg-muted/20 p-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0 space-y-1">
                  <p className="text-sm font-medium">Preview page state</p>
                  <p className="text-sm text-muted-foreground">
                    {selectedPage.group} - {selectedPage.label}
                  </p>
                </div>
                <Select
                  value={selectedPage.id}
                  onValueChange={(value) => setSelectedPageId(value as LoginPagePreviewId)}
                >
                  <SelectTrigger className="w-full sm:w-[340px]">
                    <SelectValue placeholder="Select page" />
                  </SelectTrigger>
                  <SelectContent className="w-[420px] max-w-[calc(100vw-3rem)]">
                    {pages.map((page) => (
                      <SelectItem key={page.id} value={page.id}>
                        <span className="flex flex-col gap-0.5 py-1">
                          <span className="font-medium">{page.label}</span>
                          <span className="text-xs text-muted-foreground">
                            {page.group} page state
                          </span>
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <p className="mt-3 text-xs text-muted-foreground">
                Layout changes come from the selected login template. Fields, links, and actions stay fixed to this identity page state.
              </p>
            </div>
          </CardContent>
        </Card>
        <AuthPreview
          branding={branding}
          template={template}
          tokens={tokens}
          page={selectedPage}
          onExpand={() => setIsFullscreenPreview(true)}
        />
      </div>

      {isFullscreenPreview && (
        <div className="fixed inset-0 z-50 bg-background">
          <AuthPreviewCanvas
            branding={branding}
            template={template}
            tokens={tokens}
            page={selectedPage}
            fullscreen
          />
          <Button
            type="button"
            variant="secondary"
            className="absolute right-4 top-4 z-10 gap-2 shadow-lg"
            onClick={() => setIsFullscreenPreview(false)}
          >
            <Minimize2 className="size-4" />
            Minimize
          </Button>
        </div>
      )}
    </>
  )
}

function AuthPreview({
  branding,
  template,
  tokens,
  page,
  onExpand,
}: {
  branding: Branding
  template: AuthUiTemplate
  tokens: Record<string, string>
  page: LoginPagePreview
  onExpand: () => void
}) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle className="text-base">{template.label} - {page.label} Preview</CardTitle>
            <p className="text-sm text-muted-foreground">{page.subtitle}</p>
          </div>
          <Button type="button" variant="outline" size="sm" className="shrink-0 gap-2" onClick={onExpand}>
            <Maximize2 className="size-4" />
            Expand
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <AuthPreviewCanvas branding={branding} template={template} tokens={tokens} page={page} />
      </CardContent>
    </Card>
  )
}

function AuthPreviewCanvas({
  branding,
  template,
  tokens,
  page,
  fullscreen = false,
}: {
  branding: Branding
  template: AuthUiTemplate
  tokens: Record<string, string>
  page: LoginPagePreview
  fullscreen?: boolean
}) {
  const previewStyle = {
    backgroundColor: tokens["colors.appBackground"],
    color: tokens["colors.textPrimary"],
    fontFamily: tokens["font.family"],
  }
  const panelStyle = {
    backgroundColor: tokens["colors.sidePanelBackground"],
    color: "#ffffff",
  }
  const imageUrl = loginTemplateImageUrlFromMetadata(branding.metadata)
  const imagePanelStyle: CSSProperties = imageUrl && authUiTemplateSupportsImage(template)
    ? {
        ...panelStyle,
        backgroundImage: `linear-gradient(rgba(15, 23, 42, 0.58), rgba(15, 23, 42, 0.58)), url("${imageUrl}")`,
        backgroundPosition: "center",
        backgroundSize: "cover",
      }
    : panelStyle
  const cardStyle = {
    backgroundColor: tokens["colors.cardBackground"],
    borderColor: tokens["colors.border"],
  }
  const primaryStyle = {
    backgroundColor: tokens["colors.primary"],
    color: "#ffffff",
  }
  const mutedStyle = {
    color: tokens["colors.textMuted"],
  }
  const frameHeight = fullscreen ? "min-h-screen" : "min-h-[440px]"

  return (
    <div
      className={`overflow-hidden ${fullscreen ? "h-screen" : "rounded-lg border"}`}
      style={previewStyle}
    >
      {template.previewVariant === "split-showcase" ? (
        <div className={`grid ${frameHeight} md:grid-cols-[0.95fr_1.05fr]`}>
          <div className="flex flex-col justify-between p-8" style={imagePanelStyle}>
            <BrandLockup branding={branding} />
            <div className="max-w-sm space-y-5">
              <div className="space-y-2">
                <p className="text-2xl font-semibold leading-tight">{page.title}</p>
                <p className="text-sm opacity-80">{template.flowTreatment}</p>
              </div>
            </div>
          </div>
          <div className="flex items-center justify-center p-8">
            <PreviewForm
              cardStyle={cardStyle}
              primaryStyle={primaryStyle}
              mutedStyle={mutedStyle}
              page={page}
            />
          </div>
        </div>
      ) : template.previewVariant === "side-panel" ? (
        <div className={`grid ${frameHeight} md:grid-cols-[260px_minmax(0,1fr)]`}>
          <aside
            className="flex flex-col justify-between gap-8 border-b p-6 md:border-b-0 md:border-r"
            style={panelStyle}
          >
            <div className="space-y-7">
              <BrandLockup branding={branding} />
              <div className="space-y-2">
                <p className="text-xl font-semibold leading-tight">Welcome back</p>
                <p className="text-sm opacity-80">
                  Continue with secure access for this application.
                </p>
              </div>
              <div className="rounded-lg border border-white/15 bg-white/10 p-4 shadow-sm backdrop-blur">
                <p className="text-xs font-medium uppercase opacity-70">Current app</p>
                <p className="mt-2 text-sm font-semibold">Application access</p>
                <p className="mt-1 text-xs leading-5 opacity-75">
                  Brand guidance, support, and access policy stay visible beside every auth step.
                </p>
              </div>
            </div>
            <div className="grid gap-2 text-xs opacity-80">
              {["Help center", "Privacy", "Terms"].map((item) => (
                <span key={item} className="flex items-center justify-between rounded-md bg-white/10 px-3 py-2">
                  {item}
                  <span className="size-1.5 rounded-full bg-white/50" />
                </span>
              ))}
            </div>
          </aside>
          <div className="grid gap-8 p-8 lg:grid-cols-[minmax(0,1fr)_380px] lg:items-center">
            <div className="space-y-5">
              <div className="space-y-2">
                <p className="text-2xl font-semibold leading-tight">{page.title}</p>
                <p className="max-w-md text-sm leading-6" style={mutedStyle}>{template.flowTreatment}</p>
              </div>
              <div className="grid max-w-xl gap-3 sm:grid-cols-3">
                {[
                  ["Any app", "Fits customer, workforce, and partner access."],
                  ["Clear help", "Support links remain close to the flow."],
                  ["Brand safe", "Works with quiet or expressive themes."],
                ].map(([title, description]) => (
                  <div key={title} className="rounded-lg border p-3" style={cardStyle}>
                    <CheckCircle2 className="mb-2 size-4 text-emerald-600" />
                    <p className="text-xs font-semibold">{title}</p>
                    <p className="mt-1 text-[11px] leading-4" style={mutedStyle}>{description}</p>
                  </div>
                ))}
              </div>
            </div>
            <PreviewForm cardStyle={cardStyle} primaryStyle={primaryStyle} mutedStyle={mutedStyle} page={page} />
          </div>
        </div>
      ) : template.previewVariant === "editorial-cover" ? (
        <div className={`grid ${frameHeight} md:grid-cols-[1.1fr_0.9fr]`}>
          <div className="flex items-center justify-center p-8">
            <PreviewForm cardStyle={cardStyle} primaryStyle={primaryStyle} mutedStyle={mutedStyle} page={page} />
          </div>
          <div className="flex flex-col justify-between p-8" style={imagePanelStyle}>
            <BrandLockup branding={branding} />
            <div className="max-w-sm space-y-4">
              <span className="inline-flex rounded-full border border-white/20 bg-white/10 px-3 py-1 text-xs font-medium">
                Hosted identity
              </span>
              <p className="text-3xl font-semibold leading-tight">A polished welcome moment before access.</p>
              <p className="text-sm opacity-80">{template.flowTreatment}</p>
            </div>
          </div>
        </div>
      ) : template.previewVariant === "full-page-minimal" ? (
        <div className={`${frameHeight} p-6`}>
          <div
            className="flex items-center justify-between rounded-lg border bg-white/40 px-4 py-3"
            style={{ borderColor: tokens["colors.border"] }}
          >
            <BrandLockup branding={branding} compact />
            <div className="hidden items-center gap-4 text-xs sm:flex" style={mutedStyle}>
              <span>Secure access</span>
              <span>Tenant policy</span>
              <span>Support</span>
            </div>
          </div>
          <div className="mx-auto flex max-w-3xl flex-col gap-6 py-9">
            <div className="space-y-2 text-center">
              <p className="text-2xl font-semibold">{page.title}</p>
              <p className="text-sm" style={mutedStyle}>{template.flowTreatment}</p>
            </div>
            <PreviewForm
              cardStyle={cardStyle}
              primaryStyle={primaryStyle}
              mutedStyle={mutedStyle}
              page={page}
              wide
            />
          </div>
        </div>
      ) : template.previewVariant === "stepper-flow" ? (
        <div className={`${frameHeight} p-6`}>
          <div className="mx-auto flex max-w-5xl flex-col gap-7">
            <div className="flex items-center justify-between">
              <BrandLockup branding={branding} compact />
              <span className="rounded-md border px-3 py-1 text-xs" style={mutedStyle}>
                Guided auth
              </span>
            </div>
            <div className="grid gap-6 lg:grid-cols-[280px_1fr]">
              <aside className="rounded-lg border p-5" style={cardStyle}>
                <p className="text-sm font-semibold">Progress</p>
                <div className="mt-5 space-y-4">
                  {["Account", "Verify", "Secure", "Continue"].map((step, index) => (
                    <div key={step} className="flex items-center gap-3">
                      <span
                        className="flex size-8 items-center justify-center rounded-full border text-xs font-medium"
                        style={{
                          backgroundColor: index === 0 ? tokens["colors.primary"] : "transparent",
                          borderColor: index === 0 ? tokens["colors.primary"] : tokens["colors.border"],
                          color: index === 0 ? "#ffffff" : tokens["colors.textPrimary"],
                        }}
                      >
                        {index + 1}
                      </span>
                      <div>
                        <p className="text-sm font-medium">{step}</p>
                        <p className="text-xs" style={mutedStyle}>{index === 0 ? "Current step" : "Next"}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </aside>
              <div className="flex items-center justify-center">
                <PreviewForm cardStyle={cardStyle} primaryStyle={primaryStyle} mutedStyle={mutedStyle} page={page} wide />
              </div>
            </div>
          </div>
        </div>
      ) : template.previewVariant === "security-console" ? (
        <div className={`grid ${frameHeight} gap-6 p-6 md:grid-cols-[320px_1fr]`}>
          <aside className="rounded-lg border p-5" style={cardStyle}>
            <div className="flex items-center justify-between">
              <p className="text-sm font-semibold">Security context</p>
              <span className="rounded-full border px-2 py-0.5 text-xs text-emerald-700">Low risk</span>
            </div>
            <div className="mt-5 rounded-md border bg-muted/30 p-4">
              <p className="text-xs font-medium" style={mutedStyle}>Current session</p>
              <p className="mt-1 text-sm font-semibold">Trusted browser</p>
              <p className="mt-1 text-xs" style={mutedStyle}>Policy and device checks passed.</p>
            </div>
            <div className="mt-4 space-y-3">
              {["Known device", "Policy matched", "MFA available", "Session protected"].map((item) => (
                <div key={item} className="flex items-center gap-2 rounded-md border px-3 py-2">
                  <CheckCircle2 className="size-4 text-emerald-600" />
                  <span className="text-xs">{item}</span>
                </div>
              ))}
            </div>
          </aside>
          <div className="flex items-center justify-center">
            <PreviewForm cardStyle={cardStyle} primaryStyle={primaryStyle} mutedStyle={mutedStyle} page={page} wide />
          </div>
        </div>
      ) : template.previewVariant === "compact-modal" ? (
        <div className={`relative flex ${frameHeight} items-center justify-center overflow-hidden p-6`}>
          <div className="absolute inset-0 grid grid-cols-[72px_minmax(0,1fr)] opacity-90">
            <div className="border-r p-4" style={panelStyle}>
              <div className="mx-auto size-8 rounded-md bg-white/20" />
              <div className="mt-8 space-y-3">
                <span className="block h-8 rounded-md bg-white/15" />
                <span className="block h-8 rounded-md bg-white/10" />
                <span className="block h-8 rounded-md bg-white/10" />
              </div>
            </div>
            <div className="p-5">
              <div
                className="mb-5 flex items-center justify-between rounded-lg border px-4 py-3"
                style={cardStyle}
              >
                <span className="h-2 w-28 rounded bg-muted-foreground/20" />
                <span className="h-8 w-24 rounded-md border bg-background" />
              </div>
              <div className="grid gap-4 sm:grid-cols-3">
                {[0, 1, 2].map((item) => (
                  <span key={item} className="h-20 rounded-lg border bg-background/60" />
                ))}
              </div>
            </div>
          </div>
          <div className="relative w-full max-w-md overflow-hidden rounded-lg border shadow-2xl" style={cardStyle}>
            <div className="flex items-center justify-between border-b px-5 py-4">
              <BrandLockup branding={branding} compact />
              <span className="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs" style={mutedStyle}>
                <ShieldCheck className="size-3.5 text-emerald-600" />
                Secure
              </span>
            </div>
            <PreviewForm
              cardStyle={cardStyle}
              primaryStyle={primaryStyle}
              mutedStyle={mutedStyle}
              page={page}
              embedded
            />
          </div>
        </div>
      ) : (
        <div className={`flex ${frameHeight} items-center justify-center p-8`}>
          <PreviewForm
            cardStyle={cardStyle}
            primaryStyle={primaryStyle}
            mutedStyle={mutedStyle}
            page={page}
            brand={<BrandLockup branding={branding} centered />}
          />
        </div>
      )}
    </div>
  )
}

function TemplateWireframe({ template }: { template: AuthUiTemplate }) {
  return (
    <div className="h-32 overflow-hidden rounded-md border bg-background p-3 shadow-sm" aria-hidden>
      {template.previewVariant === "split-showcase" && (
        <div className="grid h-full grid-cols-[0.9fr_1.1fr] gap-2">
          <div className="flex flex-col justify-between rounded bg-primary/15 p-2">
            <span className="block h-2 w-12 rounded bg-primary/50" />
            <div className="space-y-1.5">
              <span className="block h-3 w-20 rounded bg-primary/45" />
              <div className="grid grid-cols-3 gap-1">
                <span className="h-4 rounded bg-primary/25" />
                <span className="h-4 rounded bg-primary/20" />
                <span className="h-4 rounded bg-primary/25" />
              </div>
            </div>
          </div>
          <WireCard />
        </div>
      )}
      {template.previewVariant === "side-panel" && (
        <div className="grid h-full grid-cols-[0.82fr_1.18fr] gap-2">
          <div className="flex flex-col justify-between rounded bg-primary/15 p-2">
            <div className="space-y-2">
              <span className="block h-2 w-14 rounded bg-primary/45" />
              <span className="block h-2 w-20 rounded bg-primary/25" />
              <span className="block h-8 rounded bg-primary/20" />
            </div>
            <div className="space-y-1.5">
              <span className="block h-2 w-16 rounded bg-primary/25" />
              <span className="block h-2 w-12 rounded bg-primary/20" />
            </div>
          </div>
          <div className="grid grid-cols-[0.78fr_1fr] gap-2">
            <div className="space-y-2 pt-4">
              <span className="block h-2 w-16 rounded bg-muted-foreground/30" />
              <span className="block h-2 w-24 rounded bg-muted-foreground/20" />
              <span className="block h-2 w-20 rounded bg-muted-foreground/20" />
              <div className="grid grid-cols-3 gap-1 pt-1">
                <span className="h-5 rounded border bg-background" />
                <span className="h-5 rounded border bg-background" />
                <span className="h-5 rounded border bg-background" />
              </div>
            </div>
            <WireCard />
          </div>
        </div>
      )}
      {template.previewVariant === "full-page-minimal" && (
        <div className="h-full space-y-3">
          <div className="flex items-center justify-between rounded border px-2 py-1.5">
            <span className="block h-2 w-14 rounded bg-muted-foreground/30" />
            <span className="block h-2 w-24 rounded bg-muted-foreground/15" />
          </div>
          <div className="mx-auto max-w-[190px] pt-1">
            <WireCard />
          </div>
        </div>
      )}
      {template.previewVariant === "stepper-flow" && (
        <div className="grid h-full grid-cols-[0.7fr_1fr] gap-2">
          <div className="rounded border p-2">
            {Array.from({ length: 4 }).map((_, index) => (
              <div key={index} className="mb-2 flex items-center gap-1.5">
                <span className={`size-4 rounded-full ${index === 0 ? "bg-primary/60" : "bg-muted-foreground/20"}`} />
                <span className="h-1.5 w-10 rounded bg-muted-foreground/20" />
              </div>
            ))}
          </div>
          <WireCard />
        </div>
      )}
      {template.previewVariant === "compact-modal" && (
        <div className="relative flex h-full items-center justify-center overflow-hidden rounded bg-muted/30 p-2">
          <div className="absolute inset-0 grid grid-cols-[24px_1fr] opacity-80">
            <div className="bg-primary/15 p-1.5">
              <span className="block h-4 rounded bg-primary/35" />
              <span className="mt-3 block h-5 rounded bg-primary/20" />
              <span className="mt-1.5 block h-5 rounded bg-primary/15" />
            </div>
            <div className="p-2">
              <div className="mb-2 flex justify-between rounded border bg-background px-2 py-1">
                <span className="h-1.5 w-12 rounded bg-muted-foreground/20" />
                <span className="h-1.5 w-7 rounded bg-muted-foreground/15" />
              </div>
              <div className="grid grid-cols-3 gap-1">
                <span className="h-7 rounded border bg-background/70" />
                <span className="h-7 rounded border bg-background/70" />
                <span className="h-7 rounded border bg-background/70" />
              </div>
            </div>
          </div>
          <div className="relative w-40 overflow-hidden rounded-lg border bg-background shadow-md">
            <div className="flex items-center justify-between border-b px-2 py-1.5">
              <span className="h-2 w-14 rounded bg-muted-foreground/25" />
              <span className="size-3 rounded-full bg-emerald-500/50" />
            </div>
            <div className="p-2">
              <WireCard compact flat />
            </div>
          </div>
        </div>
      )}
      {template.previewVariant === "security-console" && (
        <div className="grid h-full grid-cols-[0.8fr_1fr] gap-2">
          <div className="rounded border bg-background p-2">
            <span className="mb-2 block h-2 w-14 rounded bg-muted-foreground/25" />
            {Array.from({ length: 4 }).map((_, index) => (
              <div key={index} className="mb-1.5 flex items-center gap-1.5">
                <span className="size-2 rounded-full bg-emerald-500/50" />
                <span className="h-1.5 flex-1 rounded bg-emerald-500/20" />
              </div>
            ))}
          </div>
          <WireCard />
        </div>
      )}
      {template.previewVariant === "editorial-cover" && (
        <div className="grid h-full grid-cols-[1fr_0.85fr] gap-2">
          <WireCard />
          <div className="flex flex-col justify-between rounded bg-primary/15 p-2">
            <span className="block h-4 w-16 rounded-full bg-primary/30" />
            <div>
              <span className="block h-3 w-20 rounded bg-primary/50" />
              <span className="mt-2 block h-2 w-24 rounded bg-primary/30" />
            </div>
          </div>
        </div>
      )}
      {template.previewVariant === "centered-card" && (
        <div className="flex h-full items-center justify-center">
          <div className="w-44">
            <WireCard />
          </div>
        </div>
      )}
    </div>
  )
}

function WireCard({ compact, flat }: { compact?: boolean; flat?: boolean }) {
  return (
    <div className={`${flat ? "" : "rounded border bg-background"} ${compact ? "p-1.5" : "p-2"}`}>
      <span className="block h-2 w-14 rounded bg-muted-foreground/30" />
      <span className="mt-2 block h-2 w-full rounded bg-muted-foreground/20" />
      <span className="mt-1.5 block h-2 w-full rounded bg-muted-foreground/20" />
      <span className="mt-2 block h-3 w-full rounded border bg-background" />
      <span className="mt-1.5 block h-3 w-full rounded border bg-background" />
      <span className="mt-2 block h-4 w-full rounded bg-primary/50" />
    </div>
  )
}

function BrandLockup({
  branding,
  compact,
  centered,
}: {
  branding: Branding
  compact?: boolean
  centered?: boolean
}) {
  return (
    <div className={`flex items-center gap-3 ${centered ? "justify-center" : ""}`}>
      {branding.logo_url ? (
        <img src={branding.logo_url} alt="" className="size-9 rounded-md object-contain" />
      ) : (
        <span className="flex size-9 items-center justify-center rounded-md bg-white/20 text-sm font-semibold">
          {(branding.company_name || branding.name || "M").slice(0, 1).toUpperCase()}
        </span>
      )}
      <div className={compact ? "hidden sm:block" : undefined}>
        <p className="text-sm font-semibold">{branding.company_name || branding.name}</p>
        {!compact && <p className="text-xs opacity-75">Identity access</p>}
      </div>
    </div>
  )
}

function PreviewForm({
  cardStyle,
  primaryStyle,
  mutedStyle,
  page,
  brand,
  wide,
  embedded,
}: {
  cardStyle: CSSProperties
  primaryStyle: CSSProperties
  mutedStyle: CSSProperties
  page: LoginPagePreview
  brand?: ReactNode
  wide?: boolean
  embedded?: boolean
}) {
  return (
    <div
      className={`w-full overflow-auto ${
        embedded
          ? "max-h-[calc(100vh-168px)] p-5"
          : `max-h-[calc(100vh-96px)] ${wide ? "max-w-2xl" : "max-w-sm"} rounded-lg border p-6 shadow-sm`
      }`}
      style={embedded ? undefined : cardStyle}
    >
      {brand && <div className="mb-6">{brand}</div>}
      <div className="space-y-1">
        <p className="text-lg font-semibold">{page.title}</p>
        <p className="text-sm" style={mutedStyle}>{page.subtitle}</p>
      </div>
      <div className="mt-5 space-y-3">
        {page.elements.map((element, index) => (
          <PreviewElement
            key={`${element.type}-${index}`}
            element={element}
            primaryStyle={primaryStyle}
            mutedStyle={mutedStyle}
          />
        ))}
      </div>
    </div>
  )
}

function PreviewElement({
  element,
  primaryStyle,
  mutedStyle,
}: {
  element: LoginPageElement
  primaryStyle: CSSProperties
  mutedStyle: CSSProperties
}) {
  if (element.type === "field") {
    return <PreviewField label={element.label} value={element.value ?? ""} kind={element.kind} />
  }

  if (element.type === "readonly") {
    return (
      <div className="space-y-1.5">
        <span className="text-xs font-medium">{element.label}</span>
        <span className="block rounded-md border bg-muted/50 px-3 py-2 text-sm text-muted-foreground">
          {element.value}
        </span>
      </div>
    )
  }

  if (element.type === "button") {
    const isPrimary = !element.variant || element.variant === "primary"
    return (
      <button
        type="button"
        className={`h-10 w-full rounded-md border px-3 text-sm font-medium ${
          element.variant === "ghost" ? "border-transparent bg-transparent" : ""
        }`}
        style={isPrimary ? primaryStyle : undefined}
      >
        {element.label}
      </button>
    )
  }

  if (element.type === "link") {
    return (
      <p className="text-center text-sm font-medium text-primary underline-offset-4">
        {element.label}
      </p>
    )
  }

  if (element.type === "checkbox") {
    return (
      <div className="flex items-start gap-2 rounded-md border p-3 text-sm">
        <span className="mt-0.5 size-4 rounded border bg-background" />
        <span>{element.label}</span>
      </div>
    )
  }

  if (element.type === "alert") {
    const toneClass = element.tone === "warning"
      ? "border-amber-500/30 bg-amber-500/10"
      : element.tone === "info"
        ? "border-blue-500/30 bg-blue-500/10"
        : "border-destructive/30 bg-destructive/5 text-destructive"
    return <div className={`rounded-md border p-3 text-sm ${toneClass}`}>{element.text}</div>
  }

  if (element.type === "divider") {
    return (
      <div className="flex items-center gap-3">
        <span className="h-px flex-1 bg-border" />
        <span className="text-xs" style={mutedStyle}>{element.label}</span>
        <span className="h-px flex-1 bg-border" />
      </div>
    )
  }

  if (element.type === "section") {
    return (
      <div className="rounded-md border p-3">
        <p className="text-sm font-medium">{element.title}</p>
        {element.description && <p className="mt-1 text-xs" style={mutedStyle}>{element.description}</p>}
      </div>
    )
  }

  if (element.type === "scope-list") {
    return (
      <div className="grid gap-2">
        {element.items.map((item) => (
          <div key={item} className="rounded-md border px-3 py-2 font-mono text-xs">
            {item}
          </div>
        ))}
      </div>
    )
  }

  if (element.type === "tile-list") {
    return (
      <div className="grid gap-2 sm:grid-cols-2">
        {element.items.map((item) => (
          <div key={item.title} className="rounded-md border p-3">
            <p className="text-xs font-medium">{item.title}</p>
            {item.description && <p className="text-xs" style={mutedStyle}>{item.description}</p>}
          </div>
        ))}
      </div>
    )
  }

  return null
}

function PreviewField({
  label,
  value,
  kind,
}: {
  label: string
  value: string
  kind?: "email" | "password" | "tel" | "code" | "text"
}) {
  return (
    <label className="block space-y-1.5">
      <span className="text-xs font-medium">{label}</span>
      <span
        className={`block h-10 rounded-md border bg-card px-3 py-2 text-sm text-foreground ${
          kind === "code" ? "text-center font-mono uppercase tracking-[0.25em]" : ""
        }`}
      >
        {value}
      </span>
    </label>
  )
}

function TemplateMeta({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className="mt-1 text-sm">{value}</p>
    </div>
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

function DetailsTab({ branding }: { branding: Branding }) {
  const links: { label: string; value: string }[] = [
    { label: "Company name", value: branding.company_name },
    { label: "Logo URL", value: branding.logo_url },
    { label: "Favicon URL", value: branding.favicon_url },
    { label: "Support URL", value: branding.support_url },
    { label: "Privacy policy URL", value: branding.privacy_policy_url },
    { label: "Terms of service URL", value: branding.terms_of_service_url },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Default brand assets &amp; links</CardTitle>
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
