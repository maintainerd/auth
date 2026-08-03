import { type CSSProperties, type ReactNode, useEffect, useState } from "react"
import { useParams, useNavigate, useLocation } from "react-router-dom"
import { Controller, useForm } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import {
  ArrowLeft,
  AlertCircle,
  CalendarDays,
  Check,
  ChevronDown,
  ChevronRight,
  ChevronsUpDown,
  Eye,
  Globe2,
  Hash,
  HelpCircle,
  KeyRound,
  LayoutDashboard,
  LayoutTemplate,
  Lock,
  Mail,
  Maximize2,
  Menu,
  MoreHorizontal,
  Palette,
  Phone,
  Plus,
  Search,
  Settings,
  Shield,
  Smartphone,
  Trash2,
  Upload,
  UserRound,
  Users,
  X,
  type LucideIcon,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import { FormCheckboxField, FormInputField, FormSelectField, FormSubmitButton, FormTextareaField } from "@/components/form"
import { FormFileUploadField, FormUrlField } from "@/components/inputs"
import { ConfirmationDialog } from "@/components/dialog"
import { cn } from "@/lib/utils"
import { resolveBrandingLogoUrl } from "@/utils/branding"
import { brandingSchema, type BrandingFormData } from "@/lib/validations"
import { useBranding, useCreateBranding, useUpdateBranding } from "@/hooks/useBranding"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import {
  THEME_SECTIONS,
  STATUS_BADGE_TYPES,
  BADGE_GROUP_MEMBERS,
  DEFAULT_TOKENS,
  tokenId,
  hexToColorInputValue,
  hexToRgba,
  tokensFromMetadata,
  metadataFromTokens,
  type ThemeSection,
  type ThemeToken,
} from "../themeTokens"
import {
  LOGIN_FORM_LOGO_PLACEMENTS,
  SPLIT_SHOWCASE_VISUAL_STYLES,
  authUiTemplatePresentationFromMetadata,
  authUiTemplatePresentationMetadata,
  authUiTemplateIdFromMetadata,
  authUiTemplateOptions,
  getAuthUiTemplate,
  type AuthUiTemplatePresentation,
  type AuthUiTemplate,
  type LoginFormLogoPlacement,
  type SplitShowcaseVisualStyle,
} from "@/lib/branding/authUiTemplates"
import {
  LOGIN_PAGE_PREVIEW_GROUPS,
  loginPageContentCollectionMetadata,
  loginPagePreviewsFromMetadata,
  type LoginPageCopy,
  type LoginPageElement,
  type LoginPagePreview,
  type LoginPagePreviewId,
} from "@/lib/branding/loginPageContent"
import { BRANDING_THEMES_LIST_URL } from "../brandingNavigation"

type ThemeSectionGroup = {
  component: string
  label: string
  title: string
  description: string
  icon?: LucideIcon
  square?: boolean
}

// The button types grouped inside the merged Buttons theme section. Each group
// renders its own preview and token rows.
const BUTTON_GROUPS: readonly ThemeSectionGroup[] = [
  {
    component: "primaryButton",
    label: "Primary",
    title: "Primary button",
    description: "Primary actions, including the Create button in the top nav.",
  },
  {
    component: "secondaryButton",
    label: "Secondary",
    title: "Secondary button",
    description: "Secondary action buttons.",
  },
  {
    component: "outlineButton",
    label: "Outline",
    title: "Outline button",
    description: "Bordered surface buttons used for secondary commands — Edit, Cancel, listing actions, filters, and detail-page header buttons.",
  },
  {
    component: "destructiveButton",
    label: "Destructive",
    title: "Destructive button",
    description: "Danger and destructive action buttons.",
  },
  {
    component: "ghostButton",
    label: "Ghost",
    title: "Ghost button",
    description: "No background by default with a subtle hover tint — back links, icon row actions, and quiet text buttons.",
  },
] as const

// The badges grouped inside the Badges section, matching the semantic tone
// grouping in themeTokens (BADGE_GROUP_MEMBERS).
const BADGE_GROUPS: readonly ThemeSectionGroup[] = [
  {
    component: "positive",
    label: "Positive",
    title: "Positive badges",
    description: "Active, enabled, verified, and accepted states.",
  },
  {
    component: "in-progress",
    label: "In progress",
    title: "In-progress badges",
    description: "Pending, draft, configuring, and maintenance states.",
  },
  {
    component: "neutral",
    label: "Neutral",
    title: "Neutral badges",
    description: "Inactive, disabled, archived, and expired states.",
  },
  {
    component: "negative",
    label: "Negative",
    title: "Negative badges",
    description: "Suspended, blocked, revoked, quarantined, and deprecated states.",
  },
] as const

// The card surfaces grouped inside the Card section — the base card, the
// listing cards, and the sub-containers (metadata rows) inside them.
const CARD_GROUPS: readonly ThemeSectionGroup[] = [
  {
    component: "card",
    label: "Card",
    title: "Card",
    description: "Card surface defaults used by form panels, details, and repeated content.",
  },
  {
    component: "listing-card",
    label: "Listing card",
    title: "Listing card",
    description: "Card-like list items used inside detail tabs such as roles, identities, MFA, sessions, activity, and trusted devices.",
  },
  {
    component: "sub-container",
    label: "Sub-container",
    title: "Sub-container",
    description: "Boxed content inside a listing card — metadata rows, key/value pairs, and nested surfaces. Every sub-container shares this one config.",
  },
  {
    component: "option-card",
    label: "Option card",
    title: "Option card",
    description: "Clickable option rows used for shortcuts and navigation — quick actions, security links, and integration options.",
  },
  {
    component: "alert",
    label: "Alert",
    title: "Alert",
    description: "Inline notice banners on forms and pages — system warnings, status notices, and rotation messages.",
  },
] as const

// The input/select/switch surfaces grouped inside the Inputs and selects
// section.
const INPUT_GROUPS: readonly ThemeSectionGroup[] = [
  {
    component: "inputs",
    label: "Inputs & selects",
    title: "Inputs and selects",
    description: "Text inputs, ordinary dropdown/select triggers, and date pickers outside the top panel.",
  },
  {
    component: "textarea",
    label: "Textarea",
    title: "Textarea",
    description: "Multi-line text inputs with their own radius and surface.",
  },
  {
    component: "switch",
    label: "Switch",
    title: "Switch",
    description: "Toggle control styling.",
  },
  {
    component: "switch-sub-container",
    label: "Switch box",
    title: "Switch box",
    description: "The bordered box that wraps a switch field — allow registration, token federation, and JIT provisioning.",
  },
  {
    component: "checkbox-sub-container",
    label: "Checkbox list",
    title: "Checkbox list",
    description: "The bordered box that wraps a checkbox option list — roles and permissions pickers.",
  },
] as const

const SECTION_GROUPS: Record<string, readonly ThemeSectionGroup[]> = {
  buttons: BUTTON_GROUPS,
  badges: BADGE_GROUPS,
  card: CARD_GROUPS,
  inputs: INPUT_GROUPS,
}

// Backend snake_case field keys → form field names.
const BACKEND_FIELD_MAP: Record<string, keyof BrandingFormData> = {
  name: "name",
  layout: "layout",
  ui_template: "ui_template",
  company_name: "company_name",
  logo_label: "logo_label",
  show_logo_label: "show_logo_label",
  logo_url: "logo_url",
  favicon_url: "favicon_url",
  support_url: "support_url",
  privacy_policy_url: "privacy_policy_url",
  terms_of_service_url: "terms_of_service_url",
}

type LoginPageCopyDraft = Record<LoginPagePreviewId, LoginPageCopy>

function pageCopyDraftFromPages(pages: LoginPagePreview[]): LoginPageCopyDraft {
  return pages.reduce<LoginPageCopyDraft>((acc, page) => ({
    ...acc,
    [page.id]: {
      title: page.title,
      subtitle: page.subtitle,
    },
  }), {} as LoginPageCopyDraft)
}

export default function BrandingForm() {
  const { brandingId } = useParams<{ brandingId?: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError, parseError } = useToast()

  const isEditing = Boolean(brandingId)
  const isCreating = !isEditing

  // Honour where the user came from so the back button and post-submit
  // navigation return there. Falls back to the listing.
  const listUrl = BRANDING_THEMES_LIST_URL
  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? listUrl
  const backLabel = navState?.backLabel ?? "Back to Themes"

  const { data: branding, isLoading: isFetching } = useBranding(brandingId)
  const createMutation = useCreateBranding()
  const updateMutation = useUpdateBranding()

  const [tokens, setTokens] = useState<Record<string, string>>({ ...DEFAULT_TOKENS })
  const [logoMode, setLogoMode] = useState<'url' | 'file'>('url')
  const [logoData, setLogoData] = useState<string | null>(null)
  const [logoContentType, setLogoContentType] = useState<string | null>(null)
  const [logoPreview, setLogoPreview] = useState<string | null>(null)
  const [logoFileError, setLogoFileError] = useState<string | null>(null)
  const [selectedLoginPageId, setSelectedLoginPageId] = useState<LoginPagePreviewId>("login")
  const [loginPageDraft, setLoginPageDraft] = useState<LoginPageCopyDraft>(
    () => pageCopyDraftFromPages(loginPagePreviewsFromMetadata(undefined)),
  )
  const [loginTemplatePresentation, setLoginTemplatePresentation] = useState<AuthUiTemplatePresentation>(
    () => authUiTemplatePresentationFromMetadata(undefined),
  )

  const {
    register,
    control,
    handleSubmit,
    watch,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<BrandingFormData>({
    resolver: yupResolver(brandingSchema),
    defaultValues: {
      name: "",
      layout: "centered",
      ui_template: "centered-card",
      company_name: "",
      logo_label: "Maintainerd-IAM",
      show_logo_label: true,
      logo_url: "",
      favicon_url: "",
      support_url: "",
      privacy_policy_url: "",
      terms_of_service_url: "",
    },
    mode: "onTouched",
    reValidateMode: "onChange",
  })

  useEffect(() => {
    if (isEditing && branding) {
      reset({
        name: branding.name ?? "",
        layout: branding.layout ?? "centered",
        ui_template: authUiTemplateIdFromMetadata(branding.metadata, branding.layout),
        company_name: branding.company_name ?? "",
        logo_label: branding.logo_label ?? branding.company_name ?? "Maintainerd-IAM",
        show_logo_label: branding.show_logo_label ?? true,
        logo_url: branding.logo_url ?? "",
        favicon_url: branding.favicon_url ?? "",
        support_url: branding.support_url ?? "",
        privacy_policy_url: branding.privacy_policy_url ?? "",
        terms_of_service_url: branding.terms_of_service_url ?? "",
      })
      setTokens(tokensFromMetadata(branding.metadata))
      setLogoMode('url')
      setLogoData(null)
      setLogoPreview(null)
      setLogoContentType(null)
      setLogoFileError(null)
      setLoginPageDraft(pageCopyDraftFromPages(loginPagePreviewsFromMetadata(branding.metadata)))
      setLoginTemplatePresentation(authUiTemplatePresentationFromMetadata(branding.metadata))
    }
  }, [isEditing, branding, reset])

  const setToken = (id: string, value: string) => setTokens((t) => ({ ...t, [id]: value }))

  const handleLogoFile = async (file: File | null) => {
    if (!file) return
    setLogoFileError(null)
    const allowedTypes = ['image/png', 'image/jpeg', 'image/webp']
    if (!allowedTypes.includes(file.type)) {
      setLogoFileError('Only PNG, JPEG, or WebP images are allowed.')
      return
    }
    if (file.size > 262144) {
      setLogoFileError('File must be 256 KB or smaller.')
      return
    }
    const base64 = await toBase64(file)
    setLogoData(base64)
    setLogoContentType(file.type)
    const reader = new FileReader()
    reader.onload = () => setLogoPreview(reader.result as string)
    reader.readAsDataURL(file)
  }

  const isLoading = createMutation.isPending || updateMutation.isPending || isSubmitting
  const selectedUiTemplate = getAuthUiTemplate(watch("ui_template"))
  const selectedTemplateUsesSplitArtwork =
    selectedUiTemplate.previewVariant === "split-showcase" ||
    selectedUiTemplate.previewVariant === "stepper-flow" ||
    selectedUiTemplate.previewVariant === "editorial-cover"
  const loginPages = loginPagePreviewsFromMetadata(branding?.metadata)
  const draftLoginPages = loginPages.map((page) => ({
    ...page,
    ...loginPageDraft[page.id],
  }))
  const selectedLoginPage =
    draftLoginPages.find((page) => page.id === selectedLoginPageId) ?? draftLoginPages[0]
  const loginPageOptions = LOGIN_PAGE_PREVIEW_GROUPS.flatMap((group) =>
    draftLoginPages
      .filter((page) => page.group === group)
      .map((page) => ({
        value: page.id,
        label: `${group} - ${page.label}`,
      })),
  )
  const isLoginPageContentDirty = draftLoginPages.some((page) => {
    const original = loginPages.find((item) => item.id === page.id)
    return original?.title !== page.title || original?.subtitle !== page.subtitle
  })
  const savedLoginTemplatePresentation = authUiTemplatePresentationFromMetadata(branding?.metadata)
  const isLoginTemplatePresentationDirty =
    savedLoginTemplatePresentation.logoPlacement !== loginTemplatePresentation.logoPlacement ||
    savedLoginTemplatePresentation.logoDetail !== loginTemplatePresentation.logoDetail ||
    savedLoginTemplatePresentation.splitShowcaseVisualStyle !== loginTemplatePresentation.splitShowcaseVisualStyle ||
    savedLoginTemplatePresentation.splitShowcaseTitle !== loginTemplatePresentation.splitShowcaseTitle ||
    savedLoginTemplatePresentation.splitShowcaseSubtitle !== loginTemplatePresentation.splitShowcaseSubtitle ||
    savedLoginTemplatePresentation.splitShowcaseImageUrl !== loginTemplatePresentation.splitShowcaseImageUrl
  const logoLabel = watch("logo_label") || "Maintainerd-IAM"
  const showLogoLabel = watch("show_logo_label") ?? true
  const logoUrl = watch("logo_url") || ""
  const topPanelPreviewBranding = {
    logoLabel,
    showLogoLabel,
    logoUrl: logoMode === "file" ? logoPreview : logoUrl,
  }
  const updateLoginPageCopy = (field: keyof LoginPageCopy, value: string) => {
    setLoginPageDraft((current) => ({
      ...current,
      [selectedLoginPage.id]: {
        ...current[selectedLoginPage.id],
        [field]: value,
      },
    }))
  }
  const updateLoginTemplatePresentation = <K extends keyof AuthUiTemplatePresentation>(
    field: K,
    value: AuthUiTemplatePresentation[K],
  ) => {
    setLoginTemplatePresentation((current) => ({
      ...current,
      [field]: value,
    }))
  }
  const loginTemplateSelector = selectedLoginPage && (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-2">
        <Controller
          name="ui_template"
          control={control}
          render={({ field }) => (
            <FormSelectField
              label="Login template"
              options={authUiTemplateOptions()}
              value={field.value}
              onValueChange={field.onChange}
              disabled={isLoading}
              error={errors.ui_template?.message}
              description="Saved as configuration for the hosted login experience."
              required
            />
          )}
        />
        <FormSelectField
          label="Page preview"
          options={loginPageOptions}
          value={selectedLoginPage.id}
          onValueChange={(value) => setSelectedLoginPageId(value as LoginPagePreviewId)}
          disabled={isLoading}
          description="Choose which hosted-auth page state is shown below."
        />
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        {!selectedTemplateUsesSplitArtwork && (
          <FormSelectField
            label="Form logo"
            options={[...LOGIN_FORM_LOGO_PLACEMENTS]}
            value={loginTemplatePresentation.logoPlacement}
            onValueChange={(value) => updateLoginTemplatePresentation("logoPlacement", value as LoginFormLogoPlacement)}
            disabled={isLoading}
            description="Place the brand logo inside the form or above it."
          />
        )}
        <FormInputField
          label="Logo detail"
          value={loginTemplatePresentation.logoDetail}
          onChange={(event) => updateLoginTemplatePresentation("logoDetail", event.target.value)}
          disabled={isLoading}
          placeholder="Optional short text below the logo label"
          description="Shown under the logo label on hosted auth templates. Leave blank to enlarge the label."
        />
        {selectedTemplateUsesSplitArtwork && (
          <FormSelectField
            label="Visual design"
            options={[...SPLIT_SHOWCASE_VISUAL_STYLES]}
            value={loginTemplatePresentation.splitShowcaseVisualStyle}
            onValueChange={(value) => updateLoginTemplatePresentation("splitShowcaseVisualStyle", value as SplitShowcaseVisualStyle)}
            disabled={isLoading}
            description="Choose the treatment for the visual panel."
          />
        )}
      </div>
      {selectedTemplateUsesSplitArtwork && (
        <div className="grid gap-4 md:grid-cols-2">
          <FormInputField
            label="Visual panel title"
            value={loginTemplatePresentation.splitShowcaseTitle}
            onChange={(event) => updateLoginTemplatePresentation("splitShowcaseTitle", event.target.value)}
            disabled={isLoading}
          />
          <FormTextareaField
            label="Visual panel supporting text"
            value={loginTemplatePresentation.splitShowcaseSubtitle}
            onChange={(event) => updateLoginTemplatePresentation("splitShowcaseSubtitle", event.target.value)}
            disabled={isLoading}
            rows={3}
          />
        </div>
      )}
      {selectedTemplateUsesSplitArtwork && loginTemplatePresentation.splitShowcaseVisualStyle === "image" && (
        <FormUrlField
          label="Visual panel image URL"
          value={loginTemplatePresentation.splitShowcaseImageUrl}
          onChange={(event) => updateLoginTemplatePresentation("splitShowcaseImageUrl", event.target.value)}
          disabled={isLoading}
          placeholder="https://example.com/secure-access.jpg"
          description="Used as the visual panel background. Overlay and text colors still come from the theme."
        />
      )}
      <div className="grid gap-4 md:grid-cols-2">
        <FormInputField
          label="Page title"
          value={selectedLoginPage.title}
          onChange={(event) => updateLoginPageCopy("title", event.target.value)}
          disabled={isLoading}
        />
        <FormTextareaField
          label="Supporting text"
          value={selectedLoginPage.subtitle}
          onChange={(event) => updateLoginPageCopy("subtitle", event.target.value)}
          disabled={isLoading}
          rows={3}
        />
      </div>
    </div>
  )

  // Non-RHF state (tokens, logoMode) isn't captured by isDirty — same trade-off
  // as the legacy client form. The guard still protects against leaving after
  // editing RHF-tracked fields (name, URLs, layout, etc.).
  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(
    isDirty || isLoginPageContentDirty || isLoginTemplatePresentationDirty,
  )

  const onSubmit = async (data: BrandingFormData) => {
    const payload = {
      name: data.name.trim(),
      layout: selectedUiTemplate.layout,
      company_name: (data.company_name ?? "").trim(),
      logo_label: (data.logo_label ?? "").trim(),
      show_logo_label: data.show_logo_label ?? true,
      logo_url: logoMode === 'url' ? (data.logo_url ?? "").trim() : "",
      favicon_url: (data.favicon_url ?? "").trim(),
      support_url: (data.support_url ?? "").trim(),
      privacy_policy_url: (data.privacy_policy_url ?? "").trim(),
      terms_of_service_url: (data.terms_of_service_url ?? "").trim(),
      metadata: loginPageContentCollectionMetadata(
        authUiTemplatePresentationMetadata({
          ...(branding?.metadata ?? {}),
          ...metadataFromTokens(tokens),
          auth_ui_template: selectedUiTemplate.id,
        }, loginTemplatePresentation),
        draftLoginPages,
      ),
      logo_data: logoMode === 'file' && logoData ? logoData : undefined,
      logo_content_type: logoMode === 'file' && logoContentType ? logoContentType : undefined,
    }

    try {
      if (isCreating) {
        await createMutation.mutateAsync(payload)
        showSuccess("Branding created successfully")
      } else {
        await updateMutation.mutateAsync({ brandingId: brandingId!, data: payload })
        showSuccess("Branding updated successfully")
      }
      navigate(backTo)
    } catch (error) {
      const parsed = parseError(error)
      let mappedToField = false
      if (parsed.fieldErrors) {
        for (const [field, message] of Object.entries(parsed.fieldErrors)) {
          const formField = BACKEND_FIELD_MAP[field]
          if (formField) {
            setError(formField, { type: "server", message })
            mappedToField = true
          }
        }
      }
      if (!mappedToField) {
        const lower = parsed.message.toLowerCase()
        const keywordOrder: Array<[string, keyof BrandingFormData]> = [
          ["logo url", "logo_url"],
          ["logo_url", "logo_url"],
          ["favicon url", "favicon_url"],
          ["favicon_url", "favicon_url"],
          ["support url", "support_url"],
          ["support_url", "support_url"],
          ["privacy policy", "privacy_policy_url"],
          ["privacy_policy_url", "privacy_policy_url"],
          ["terms of service", "terms_of_service_url"],
          ["terms_of_service_url", "terms_of_service_url"],
          ["company name", "company_name"],
          ["company_name", "company_name"],
          ["logo label", "logo_label"],
          ["logo_label", "logo_label"],
          ["layout", "layout"],
          ["name", "name"],
        ]
        const hit = keywordOrder.find(([keyword]) => lower.includes(keyword))
        if (hit) {
          setError(hit[1], { type: "server", message: parsed.message })
        }
      }
      showError(error)
    }
  }

  const pageTitle = isCreating ? "Create Branding Template" : `Edit ${branding?.name || "Branding Template"}`
  const submitButtonText = isCreating ? "Create Template" : "Update Template"

  // Loading state while fetching the branding to edit
  if (isEditing && isFetching && !branding) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            onBack={() => guard(() => navigate(backTo))}
            title="Edit Branding Template"
            description="Update the branding theme and its values"
          />
          <Card>
            <CardContent className="space-y-4 pt-6">
              <Skeleton className="h-5 w-40" />
              <div className="grid gap-4 md:grid-cols-2">
                {Array.from({ length: 2 }).map((_, i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
              <Skeleton className="h-24 w-full" />
            </CardContent>
          </Card>
        </div>
      </DetailsContainer>
    )
  }

  // Not-found state
  if (isEditing && !isFetching && !branding) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            onBack={() => guard(() => navigate(backTo))}
            title="Edit Branding Template"
            description="Update the branding theme and its values"
          />
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <AlertCircle className="size-6" />
              </div>
              <div className="space-y-1">
                <h2 className="text-lg font-semibold">Branding not found</h2>
                <p className="text-sm text-muted-foreground">
                  The branding template you're looking for doesn't exist or may have been removed.
                </p>
              </div>
              <Button variant="outline" onClick={() => guard(() => navigate(backTo))}>
                <ArrowLeft className="mr-2 size-4" />
                {backLabel}
              </Button>
            </CardContent>
          </Card>
        </div>
      </DetailsContainer>
    )
  }

  return (
    <DetailsContainer>
      <div className="flex flex-col gap-6">
        <FormPageHeader
          backUrl={backTo}
          backLabel={backLabel}
          onBack={() => guard(() => navigate(backTo))}
          title={pageTitle}
          description={
            isCreating
              ? "Create a new branding theme. It won't be active until you set it as the active template."
              : "Update this branding theme. Setting it as active is a separate action."
          }
          showSystemBadge={!!branding?.is_system}
          showWarning={!!branding?.is_system}
          warningMessage="This is a system template. You can edit its values, but it can't be deleted."
        />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6" key={brandingId || "create"}>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Basic Information</CardTitle>
              <p className="text-sm text-muted-foreground">The template name and company.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <FormInputField
                  label="Name"
                  placeholder="e.g., acme-light"
                  disabled={isLoading || !!branding?.is_system}
                  description={branding?.is_system ? "System theme names are fixed so restore can always find the seeded default." : undefined}
                  required
                  error={errors.name?.message}
                  {...register("name")}
                />
                <FormInputField
                  label="Company name"
                  placeholder="Acme Inc."
                  disabled={isLoading}
                  error={errors.company_name?.message}
                  {...register("company_name")}
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Brand assets &amp; links</CardTitle>
              <p className="text-sm text-muted-foreground">
                Logo, favicon, and the URLs surfaced across the auth experience.
              </p>
            </CardHeader>
            <CardContent className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="flex flex-col gap-2">
                <div className="grid gap-4 sm:grid-cols-[1fr_auto]">
                  <FormInputField
                    label="Logo label"
                    placeholder="Maintainerd-IAM"
                    disabled={isLoading}
                    error={errors.logo_label?.message}
                    description="Shown beside the logo in the console top panel and hosted auth templates."
                    {...register("logo_label")}
                  />
                  <Controller
                    name="show_logo_label"
                    control={control}
                    render={({ field }) => (
                      <FormCheckboxField
                        label="Show label"
                        checked={field.value ?? true}
                        onCheckedChange={field.onChange}
                        disabled={isLoading}
                        containerClassName="self-end pb-2"
                      />
                    )}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium leading-none">Logo</span>
                  <div className="flex rounded-md border text-xs overflow-hidden">
                    <button
                      type="button"
                      className={`px-3 py-1.5 transition-colors ${logoMode === 'url' ? 'bg-primary text-primary-foreground' : 'bg-background text-muted-foreground hover:text-foreground'}`}
                      onClick={() => setLogoMode('url')}
                      disabled={isLoading}
                    >
                      URL
                    </button>
                    <button
                      type="button"
                      className={`px-3 py-1.5 transition-colors ${logoMode === 'file' ? 'bg-primary text-primary-foreground' : 'bg-background text-muted-foreground hover:text-foreground'}`}
                      onClick={() => setLogoMode('file')}
                      disabled={isLoading}
                    >
                      Upload
                    </button>
                  </div>
                </div>
                {logoMode === 'url' ? (
                  <FormUrlField
                    label="Logo URL"
                    placeholder="https://…/logo.svg"
                    disabled={isLoading}
                    error={errors.logo_url?.message}
                    {...register("logo_url")}
                  />
                ) : (
                  <div className="flex flex-col gap-2">
                    <FormFileUploadField
                      label="Logo file"
                      description="PNG, JPEG, or WebP — max 256 KB."
                      accept="image/png,image/jpeg,image/webp"
                      disabled={isLoading}
                      error={logoFileError ?? undefined}
                      onChange={handleLogoFile}
                    />
                    {logoPreview && (
                      <img
                        src={logoPreview}
                        alt="Logo preview"
                        className="mt-1 h-12 w-auto rounded-md border object-contain"
                      />
                    )}
                    <p className="text-xs text-muted-foreground">PNG, JPEG, or WebP — max 256 KB.</p>
                  </div>
                )}
              </div>
              <FormUrlField
                label="Favicon URL"
                placeholder="https://…/favicon.ico"
                disabled={isLoading}
                error={errors.favicon_url?.message}
                {...register("favicon_url")}
              />
              <FormUrlField
                label="Support URL"
                placeholder="https://…/support"
                disabled={isLoading}
                error={errors.support_url?.message}
                {...register("support_url")}
              />
              <FormUrlField
                label="Privacy policy URL"
                placeholder="https://…/privacy"
                disabled={isLoading}
                error={errors.privacy_policy_url?.message}
                {...register("privacy_policy_url")}
              />
              <FormUrlField
                label="Terms of service URL"
                placeholder="https://…/terms"
                disabled={isLoading}
                error={errors.terms_of_service_url?.message}
                containerClassName="sm:col-span-2"
                {...register("terms_of_service_url")}
              />
            </CardContent>
          </Card>

          <div className="space-y-2">
            {THEME_SECTIONS.map((section) => (
              <ThemeSectionEditor
                key={section.id}
                section={section}
                tokens={tokens}
                previewBranding={topPanelPreviewBranding}
                selectedTemplate={selectedUiTemplate}
                selectedLoginPage={selectedLoginPage}
                loginTemplatePresentation={loginTemplatePresentation}
                loginTemplateSelector={section.id === "login-template" ? loginTemplateSelector : undefined}
                disabled={isLoading}
                onTokenChange={setToken}
              />
            ))}
          </div>

          <div className="flex justify-end gap-3">
            <Button type="button" variant="outline" onClick={() => guard(() => navigate(backTo))} disabled={isLoading}>
              Cancel
            </Button>
            <FormSubmitButton isSubmitting={isLoading} submitText={submitButtonText} />
          </div>
        </form>

        <ConfirmationDialog
          open={isPromptOpen}
          onOpenChange={(open) => { if (!open) cancelLeave() }}
          onConfirm={confirmLeave}
          title="Discard changes?"
          description="You have unsaved changes. If you leave now, they will be lost."
          confirmText="Discard changes"
          cancelText="Keep editing"
          variant="destructive"
        />
      </div>
    </DetailsContainer>
  )
}

function ThemeSectionEditor({
  section,
  tokens,
  previewBranding,
  selectedTemplate,
  selectedLoginPage,
  loginTemplatePresentation,
  loginTemplateSelector,
  disabled,
  onTokenChange,
}: {
  section: ThemeSection
  tokens: Record<string, string>
  previewBranding: TopPanelPreviewBranding
  selectedTemplate: AuthUiTemplate
  selectedLoginPage: LoginPagePreview
  loginTemplatePresentation: AuthUiTemplatePresentation
  loginTemplateSelector?: ReactNode
  disabled?: boolean
  onTokenChange: (id: string, value: string) => void
}) {
  const sideBySide = section.id === "side-panel"
  const groups = SECTION_GROUPS[section.id]
  const [isOpen, setIsOpen] = useState(false)
  const sectionContentId = `theme-section-${section.id}`

  return (
    <Card className="gap-0 overflow-hidden !py-0">
      <button
        type="button"
        className="flex w-full items-center justify-between gap-4 px-6 py-3 text-left transition-colors hover:bg-muted/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        aria-expanded={isOpen}
        aria-controls={sectionContentId}
        onClick={() => setIsOpen((open) => !open)}
      >
        <span className="min-w-0 space-y-1">
          <CardTitle className="text-base">{section.title}</CardTitle>
          <span className="block text-sm font-normal text-muted-foreground">{section.description}</span>
        </span>
        <span className="flex shrink-0 items-center gap-2 text-xs font-medium text-muted-foreground">
          {section.tokens.length} settings
          <ChevronDown
            className={`size-4 transition-transform ${isOpen ? "rotate-180" : ""}`}
            aria-hidden
          />
        </span>
      </button>
      {isOpen &&
        (groups ? (
          <CardContent id={sectionContentId} className="space-y-10 border-t py-6">
            {groups.map((group) => {
              const groupTokens = section.tokens.filter((t) => t.group === group.component)
              return (
                <div key={group.component} className="space-y-4">
                  <div className="flex items-baseline justify-between gap-4">
                    <div>
                      <h4 className="text-sm font-semibold">{group.title}</h4>
                      <p className="text-xs text-muted-foreground">{group.description}</p>
                    </div>
                    <span className="shrink-0 text-xs font-medium text-muted-foreground">
                      {groupTokens.length} settings
                    </span>
                  </div>
                  {section.id === "badges" ? (
                    <BadgeGroupPreview group={group.component} tokens={tokens} />
                  ) : section.id === "card" ? (
                    group.component === "listing-card" ? (
                      <ListingPreview tokens={tokens} />
                    ) : group.component === "sub-container" ? (
                      <SubContainerPreview tokens={tokens} />
                    ) : group.component === "option-card" ? (
                      <OptionCardPreview tokens={tokens} />
                    ) : group.component === "alert" ? (
                      <AlertPreview tokens={tokens} />
                    ) : (
                      <CardComponentPreview tokens={tokens} />
                    )
                  ) : section.id === "inputs" ? (
                    group.component === "switch" ? (
                      <SwitchPreview tokens={tokens} />
                    ) : group.component === "switch-sub-container" ? (
                      <SwitchSubContainerPreview tokens={tokens} />
                    ) : group.component === "checkbox-sub-container" ? (
                      <CheckboxSubContainerPreview tokens={tokens} />
                    ) : group.component === "textarea" ? (
                      <TextareaPreview tokens={tokens} />
                    ) : (
                      <InputsPreview tokens={tokens} />
                    )
                  ) : (
                    <ButtonTypePreview component={group.component} label={group.label} tokens={tokens} />
                  )}
                  <div className="divide-y rounded-md border bg-background/40 px-4">
                    {groupTokens.map((t) => (
                      <ThemeTokenRow
                        key={tokenId(t)}
                        token={t}
                        value={tokens[tokenId(t)] ?? ""}
                        disabled={disabled}
                        onChange={(v) => onTokenChange(tokenId(t), v)}
                      />
                    ))}
                  </div>
                </div>
              )
            })}
          </CardContent>
        ) : (
          <CardContent
            id={sectionContentId}
            className={sideBySide ? "grid gap-6 border-t py-6 lg:grid-cols-[minmax(280px,0.9fr)_minmax(0,1.1fr)]" : "space-y-6 border-t py-6"}
          >
            <div className={sideBySide ? "self-start lg:sticky lg:top-20" : undefined}>
              {loginTemplateSelector && (
                <div className="mb-4 rounded-md border bg-background/40 p-4">
                  {loginTemplateSelector}
                </div>
              )}
              <ThemeSectionPreview
                sectionId={section.id}
                tokens={tokens}
                previewBranding={previewBranding}
                selectedTemplate={selectedTemplate}
                selectedLoginPage={selectedLoginPage}
                loginTemplatePresentation={loginTemplatePresentation}
              />
            </div>
            <div className="divide-y rounded-md border bg-background/40 px-4">
              {section.tokens.map((t) => (
                <ThemeTokenRow
                  key={tokenId(t)}
                  token={t}
                  value={tokens[tokenId(t)] ?? ""}
                  disabled={disabled}
                  onChange={(v) => onTokenChange(tokenId(t), v)}
                />
              ))}
            </div>
          </CardContent>
        ))}
    </Card>
  )
}

function toBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve((reader.result as string).split(',')[1])
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

function ThemeTokenRow({
  token,
  value,
  disabled,
  onChange,
}: {
  token: ThemeToken
  value: string
  disabled?: boolean
  onChange: (v: string) => void
}) {
  const isColor = token.kind === "color"

  return (
    <div className="grid grid-cols-1 items-center gap-2 py-3 sm:grid-cols-[1fr_300px]">
      <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
        <Lock className="size-3 shrink-0 text-muted-foreground/60" aria-hidden />
        <span className="text-sm font-medium">{token.label}</span>
        <code className="text-[11px] text-muted-foreground">
          {token.path.join(".")}
        </code>
      </div>
      {isColor ? (
        <div className="flex items-center gap-2">
          <span
            className="size-8 shrink-0 border bg-[linear-gradient(45deg,#e5e7eb_25%,transparent_25%),linear-gradient(-45deg,#e5e7eb_25%,transparent_25%),linear-gradient(45deg,transparent_75%,#e5e7eb_75%),linear-gradient(-45deg,transparent_75%,#e5e7eb_75%)] bg-[length:8px_8px] bg-[position:0_0,0_4px,4px_-4px,-4px_0px]"
            aria-hidden
          >
            <span
              className="block size-full rounded-[inherit]"
              style={{ backgroundColor: value || "transparent" }}
            />
          </span>
          <Input
            type="color"
            value={hexToColorInputValue(value)}
            onChange={(e) => onChange(e.target.value)}
            disabled={disabled}
            className="h-9 w-12 shrink-0 cursor-pointer rounded-none p-0"
            aria-label={`${token.label} color picker`}
          />
          <Input
            value={value}
            onChange={(e) => onChange(e.target.value)}
            disabled={disabled}
            placeholder="#000000"
            className="font-mono text-sm"
            aria-label={`${token.label} value`}
          />
        </div>
      ) : token.kind === "select" ? (
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          disabled={disabled}
          className="border-input h-10 w-full rounded-md border bg-transparent px-3.5 py-2 text-sm outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50"
          aria-label={`${token.label} value`}
        >
          {(token.options ?? []).map((option) => (
            <option key={option} value={option}>
              {option.toUpperCase()}
            </option>
          ))}
        </select>
      ) : (
        <Input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          disabled={disabled}
          placeholder={token.path.join(".") === "font.family" ? "Inter, system-ui, sans-serif" : "1px"}
          aria-label={`${token.label} value`}
        />
      )}
    </div>
  )
}

type TopPanelPreviewBranding = {
  logoLabel: string
  showLogoLabel: boolean
  logoUrl?: string | null
}

function ThemeSectionPreview({
  sectionId,
  tokens,
  previewBranding,
  selectedTemplate,
  selectedLoginPage,
  loginTemplatePresentation,
}: {
  sectionId: string
  tokens: Record<string, string>
  previewBranding: TopPanelPreviewBranding
  selectedTemplate: AuthUiTemplate
  selectedLoginPage: LoginPagePreview
  loginTemplatePresentation: AuthUiTemplatePresentation
}) {
  if (sectionId === "top-panel") return <TopPanelPreview tokens={tokens} branding={previewBranding} />
  if (sectionId === "login-template") return <LoginTemplateThemePreview tokens={tokens} branding={previewBranding} template={selectedTemplate} page={selectedLoginPage} presentation={loginTemplatePresentation} />
  if (sectionId === "side-panel") return <SidePanelPreview tokens={tokens} />
  if (sectionId === "icon-containers") return <IconContainersPreview tokens={tokens} />
  if (sectionId === "table") return <TablePreview tokens={tokens} />
  return <CorePalettePreview tokens={tokens} />
}

function CorePalettePreview({ tokens }: { tokens: Record<string, string> }) {
  const palette = [
    ["Primary", tokens["colors.primary"]],
    ["Secondary", tokens["colors.secondary"]],
    ["Accent", tokens["colors.accent"]],
    ["App", tokens["colors.appBackground"]],
    ["Top", tokens["colors.topPanelBackground"]],
    ["Side", tokens["colors.sidePanelBackground"]],
    ["Card", tokens["colors.cardBackground"]],
    ["Border", tokens["colors.border"]],
  ]

  return (
    <div
      className="space-y-4 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div
        className="rounded-md border p-4"
        style={{
          backgroundColor: tokens["colors.cardBackground"],
          borderColor: tokens["colors.border"],
        }}
      >
        <p className="text-sm font-semibold">Maintainerd-IAM</p>
        <p className="mt-1 text-xs" style={{ color: tokens["colors.textMuted"] }}>
          Tenant administration console
        </p>
      </div>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        {palette.map(([label, color]) => (
          <div key={label} className="overflow-hidden rounded-md border" style={{ borderColor: tokens["colors.border"] }}>
            <div className="h-12" style={{ backgroundColor: color }} />
            <div className="bg-card px-2 py-1.5 text-[11px] font-medium">{label}</div>
          </div>
        ))}
      </div>
    </div>
  )
}

function TopPanelPreview({
  tokens,
  branding,
}: {
  tokens: Record<string, string>
  branding: TopPanelPreviewBranding
}) {
  const topPanelStyle: CSSProperties = {
    backgroundColor: tokens["colors.topPanelBackground"],
    borderColor: tokens["colors.topPanelBorder"],
    color: tokens["colors.topPanelText"],
    fontFamily: tokens["font.family"],
  }
  const topControlStyle = styleForComponent(tokens, "topPanelControl")
  const topDropdownStyle = styleForComponent(tokens, "topPanelDropdownTrigger")
  const topProfileStyle = styleForComponent(tokens, "topPanelProfileTrigger")
  const topCreateStyle = styleForComponent(tokens, "topPanelCreateButton")
  const logoSrc = resolveBrandingLogoUrl(branding.logoUrl) ?? "/logo.png"
  const logoLabel = branding.logoLabel || "Maintainerd-IAM"

  return (
    <div className="overflow-x-auto rounded-md border" style={{ borderColor: tokens["colors.topPanelBorder"] }}>
      <div className="flex h-14 min-w-[760px] items-center gap-3 border-b px-4" style={topPanelStyle}>
        <span className="flex size-10 items-center justify-center" style={topControlStyle}>
          <Menu className="size-4" />
        </span>
        <img src={logoSrc} alt={logoLabel} className="h-7 w-auto shrink-0 object-contain" />
        {branding.showLogoLabel && (
          <span className="min-w-0">
            <span className="block truncate text-base font-semibold leading-none" style={{ color: tokens["colors.topPanelText"] }}>
              {logoLabel}
            </span>
          </span>
        )}
        <span className="ml-1 hidden h-10 min-w-32 items-center justify-between gap-2 px-2 text-sm sm:flex" style={topDropdownStyle}>
          <span className="min-w-0 flex-1 truncate text-left font-medium">
            System
          </span>
          <ChevronsUpDown className="size-4 shrink-0 opacity-75" />
        </span>
        <span className="ml-auto flex h-10 items-center gap-2 px-5 text-sm font-medium" style={topCreateStyle}>
          <Plus className="size-4" />
          Create
          <ChevronDown className="size-4 opacity-80" />
        </span>
        <span className="flex size-10 items-center justify-center" style={topControlStyle}>
          <HelpCircle className="size-5" />
        </span>
        <span className="flex h-10 items-center gap-2 px-2 text-sm" style={topProfileStyle}>
          <span className="flex size-8 items-center justify-center rounded-full text-xs text-white" style={{ backgroundColor: "#334155" }}>
            US
          </span>
          <span className="hidden max-w-28 truncate text-sm font-medium lg:inline">
            User
          </span>
          <ChevronDown className="size-4 text-slate-400" />
        </span>
      </div>
      <div className="flex h-14 min-w-[760px] items-center gap-3 px-4" style={topPanelStyle}>
        <span className="flex size-10 items-center justify-center" style={{ ...topControlStyle, backgroundColor: componentValue(tokens, "topPanelControl", "hoverColor") }}>
          <Menu className="size-4" />
        </span>
        <img src={logoSrc} alt="" className="h-7 w-auto shrink-0 object-contain" />
        {branding.showLogoLabel && (
          <span className="min-w-0">
            <span className="block truncate text-base font-semibold leading-none" style={{ color: tokens["colors.topPanelText"] }}>
              {logoLabel}
            </span>
          </span>
        )}
        <span className="ml-1 hidden h-10 min-w-32 items-center justify-between gap-2 px-2 text-sm sm:flex" style={{ ...topDropdownStyle, backgroundColor: componentValue(tokens, "topPanelDropdownTrigger", "hoverColor") }}>
          <span className="min-w-0 flex-1 truncate text-left font-medium">
            System
          </span>
          <ChevronsUpDown className="size-4 shrink-0 opacity-75" />
        </span>
        <span className="ml-auto flex h-10 items-center gap-2 px-5 text-sm font-medium" style={{ ...topCreateStyle, backgroundColor: componentValue(tokens, "topPanelCreateButton", "hoverColor") }}>
          <Plus className="size-4" />
          Create
          <ChevronDown className="size-4 opacity-80" />
        </span>
        <span className="flex size-10 items-center justify-center" style={{ ...topControlStyle, backgroundColor: componentValue(tokens, "topPanelControl", "hoverColor") }}>
          <HelpCircle className="size-5" />
        </span>
        <span className="flex h-10 items-center gap-2 px-2 text-sm" style={{ ...topProfileStyle, backgroundColor: componentValue(tokens, "topPanelProfileTrigger", "hoverColor") }}>
          <span className="flex size-8 items-center justify-center rounded-full text-xs text-white" style={{ backgroundColor: "#334155" }}>
            US
          </span>
          <span className="hidden max-w-28 truncate text-sm font-medium lg:inline">
            User
          </span>
          <ChevronDown className="size-4 text-slate-400" />
        </span>
      </div>
    </div>
  )
}

function LoginTemplateThemePreview({
  tokens,
  branding,
  template,
  page,
  presentation,
}: {
  tokens: Record<string, string>
  branding: TopPanelPreviewBranding
  template: AuthUiTemplate
  page: LoginPagePreview
  presentation: AuthUiTemplatePresentation
}) {
  const [isFullscreenOpen, setIsFullscreenOpen] = useState(false)
  const pageStyle: CSSProperties = {
    backgroundColor: tokens["colors.authPageBackground"],
    color: tokens["colors.authFormPanelText"],
    fontFamily: tokens["font.family"],
  }
  const panelStyle: CSSProperties = {
    backgroundColor: tokens["colors.authVisualPanelBackground"],
    color: tokens["colors.authVisualPanelText"],
  }
  const formPanelStyle: CSSProperties = {
    backgroundColor: tokens["colors.authFormPanelBackground"],
    borderColor: tokens["colors.authFormPanelBorder"],
    color: tokens["colors.authFormPanelText"],
    borderRadius: componentValue(tokens, "card", "borderRadius"),
  }
  const inputStyle = styleForComponent(tokens, "input")
  const primaryButtonStyle = styleForComponent(tokens, "primaryButton")
  const secondaryButtonStyle = styleForComponent(tokens, "secondaryButton")

  return (
    <div className="overflow-hidden rounded-md border" style={{ ...pageStyle, borderColor: tokens["colors.border"] }}>
      <div className="flex items-center justify-between border-b px-4 py-3" style={{ borderColor: tokens["colors.authFormPanelBorder"] }}>
        <div className="flex min-w-0 items-center gap-2">
          <LayoutTemplate className="size-4 shrink-0" />
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold">{template.label}</p>
            <p className="truncate text-xs" style={{ color: tokens["colors.textMuted"] }}>
              {template.layout.replace("_", " ")} hosted-auth shell
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-8 gap-2"
            onClick={() => setIsFullscreenOpen(true)}
          >
            <Maximize2 className="size-3.5" />
            Fullscreen
          </Button>
        </div>
      </div>
      <LoginTemplatePreviewCanvas
        template={template}
        branding={branding}
        pageStyle={pageStyle}
        panelStyle={panelStyle}
        formPanelStyle={formPanelStyle}
        inputStyle={inputStyle}
        primaryButtonStyle={primaryButtonStyle}
        secondaryButtonStyle={secondaryButtonStyle}
        tokens={tokens}
        page={page}
        presentation={presentation}
      />
      {isFullscreenOpen && (
        <div className="fixed inset-0 z-50 overflow-hidden" style={pageStyle}>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="absolute right-4 top-4 z-20 size-10 shadow-sm"
            onClick={() => setIsFullscreenOpen(false)}
            aria-label="Exit fullscreen preview"
          >
            <X className="size-4" />
          </Button>
          <LoginTemplatePreviewCanvas
            template={template}
            branding={branding}
            pageStyle={pageStyle}
            panelStyle={panelStyle}
            formPanelStyle={formPanelStyle}
            inputStyle={inputStyle}
            primaryButtonStyle={primaryButtonStyle}
            secondaryButtonStyle={secondaryButtonStyle}
            tokens={tokens}
            page={page}
            presentation={presentation}
            fullscreen
          />
        </div>
      )}
    </div>
  )
}

function LoginTemplatePreviewCanvas({
  template,
  branding,
  pageStyle,
  panelStyle,
  formPanelStyle,
  inputStyle,
  primaryButtonStyle,
  secondaryButtonStyle,
  tokens,
  page,
  presentation,
  fullscreen,
}: {
  template: AuthUiTemplate
  branding: TopPanelPreviewBranding
  pageStyle: CSSProperties
  panelStyle: CSSProperties
  formPanelStyle: CSSProperties
  inputStyle: CSSProperties
  primaryButtonStyle: CSSProperties
  secondaryButtonStyle: CSSProperties
  tokens: Record<string, string>
  page: LoginPagePreview
  presentation: AuthUiTemplatePresentation
  fullscreen?: boolean
}) {
  const canvasHeightClass = fullscreen ? "h-full min-h-0" : "min-h-[440px]"
  const visualPanel = (
    <LoginVisualPanel
      branding={branding}
      template={template}
      style={panelStyle}
      tokens={tokens}
      presentation={presentation}
      visualStyle={presentation.splitShowcaseVisualStyle}
    />
  )
  const form = (
    <LoginFormSample
      branding={branding}
      formPanelStyle={formPanelStyle}
      inputStyle={inputStyle}
      primaryButtonStyle={primaryButtonStyle}
      secondaryButtonStyle={secondaryButtonStyle}
      tokens={tokens}
      page={page}
      logoDetail={presentation.logoDetail}
      logoPlacement={
        template.previewVariant === "split-showcase"
          ? "none"
          : template.previewVariant === "editorial-cover"
            ? "none"
            : presentation.logoPlacement
      }
    />
  )
  const editorialForm = (
    <LoginFormSample
      branding={branding}
      formPanelStyle={formPanelStyle}
      inputStyle={inputStyle}
      primaryButtonStyle={primaryButtonStyle}
      secondaryButtonStyle={secondaryButtonStyle}
      tokens={tokens}
      page={page}
      logoDetail={presentation.logoDetail}
      logoPlacement="none"
    />
  )
  const embeddedForm = (
    <LoginFormSample
      branding={branding}
      formPanelStyle={formPanelStyle}
      inputStyle={inputStyle}
      primaryButtonStyle={primaryButtonStyle}
      secondaryButtonStyle={secondaryButtonStyle}
      tokens={tokens}
      page={page}
      logoDetail={presentation.logoDetail}
      logoPlacement="inside-form"
      embedded
    />
  )

  if (template.previewVariant === "split-showcase") {
    return (
      <div className={`grid ${canvasHeightClass} lg:grid-cols-[0.95fr_1.05fr]`} style={pageStyle}>
        {visualPanel}
        <div className="flex items-center justify-center p-6">{form}</div>
      </div>
    )
  }

  if (template.previewVariant === "editorial-cover") {
    return (
      <div className={`grid ${canvasHeightClass} lg:grid-cols-[1.05fr_0.95fr]`} style={pageStyle}>
        <div className="flex min-h-[440px] flex-col p-6">
          <div className="shrink-0">
            <BrandPreviewLockup branding={branding} logoDetail={presentation.logoDetail} />
          </div>
          <div className="flex flex-1 items-center justify-center py-6">
            {editorialForm}
          </div>
        </div>
        {visualPanel}
      </div>
    )
  }

  if (template.previewVariant === "stepper-flow") {
    return (
      <div className={`flex ${canvasHeightClass} flex-col items-center justify-center p-6 md:p-10`} style={pageStyle}>
        <div
          className="grid w-full max-w-sm items-stretch overflow-hidden rounded-md border md:min-h-[440px] md:max-w-4xl md:grid-cols-2"
          style={{ ...formPanelStyle, boxShadow: tokens["effects.authFormPanelShadow"] }}
        >
          <main className="flex min-h-[440px] items-center justify-center p-6 md:p-8">
            <div className="w-full max-w-sm">{embeddedForm}</div>
          </main>
          <div className="hidden min-h-[440px] md:block md:h-full">
            {visualPanel}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className={`flex ${canvasHeightClass} items-center justify-center p-6`} style={pageStyle}>
      {form}
    </div>
  )
}

function LoginVisualPanel({
  branding,
  template,
  style,
  tokens,
  presentation,
  visualStyle,
}: {
  branding: TopPanelPreviewBranding
  template: AuthUiTemplate
  style: CSSProperties
  tokens: Record<string, string>
  presentation: AuthUiTemplatePresentation
  visualStyle: SplitShowcaseVisualStyle
}) {
  const usesSplitArtwork =
    template.previewVariant === "split-showcase" ||
    template.previewVariant === "stepper-flow" ||
    template.previewVariant === "editorial-cover"
  const showVisualBrand = template.previewVariant === "split-showcase"
  const panelTitle = usesSplitArtwork ? presentation.splitShowcaseTitle : template.label
  const panelSubtitle = usesSplitArtwork ? presentation.splitShowcaseSubtitle : template.flowTreatment

  return (
    <div className="relative flex h-full min-h-[280px] flex-col justify-between overflow-hidden p-8" style={style}>
      <SplitShowcaseVisualDesign
        styleId={usesSplitArtwork ? visualStyle : "default"}
        imageUrl={presentation.splitShowcaseImageUrl}
        tokens={tokens}
      />
      {showVisualBrand ? (
        <div className="relative">
          <BrandPreviewLockup branding={branding} logoDetail={presentation.logoDetail} panel />
        </div>
      ) : (
        <div className="relative" />
      )}
      <div className="relative max-w-sm space-y-3">
        <p className="text-3xl font-semibold leading-tight">
          {panelTitle}
        </p>
        <p className="text-sm opacity-80">{panelSubtitle}</p>
      </div>
      <div className="relative flex flex-wrap gap-4 text-xs opacity-80">
        <span>Support</span>
        <span>Privacy</span>
        <span>Terms</span>
      </div>
    </div>
  )
}

function SplitShowcaseVisualDesign({
  styleId,
  imageUrl,
  tokens,
}: {
  styleId: SplitShowcaseVisualStyle
  imageUrl: string
  tokens: Record<string, string>
}) {
  const light = tokens["colors.authDecorativeLight"]
  const dark = tokens["colors.authDecorativeDark"]
  const overlay = tokens["colors.authVisualPanelOverlay"]
  const text = tokens["colors.authVisualPanelText"]

  if (styleId === "image") {
    return (
      <>
        {imageUrl ? (
          <>
            <img
              src={imageUrl}
              alt=""
              className="pointer-events-none absolute inset-0 size-full object-cover"
            />
            <div className="pointer-events-none absolute inset-0" style={{ backgroundColor: withAlpha(overlay, 0.42) }} />
          </>
        ) : (
          <>
            <div
              className="pointer-events-none absolute inset-0"
              style={{
                backgroundImage: `linear-gradient(135deg, ${withAlpha(light, 0.16)}, transparent 38%), linear-gradient(315deg, ${withAlpha(dark, 0.18)}, transparent 42%)`,
              }}
            />
            <div className="pointer-events-none absolute inset-0" style={{ backgroundColor: withAlpha(overlay, 0.42) }} />
          </>
        )}
      </>
    )
  }

  if (styleId === "identity-mesh") {
    const nodes = [
      ["12%", "18%", "26px"],
      ["36%", "12%", "18px"],
      ["64%", "20%", "34px"],
      ["84%", "42%", "22px"],
      ["66%", "72%", "26px"],
      ["34%", "80%", "20px"],
      ["14%", "56%", "32px"],
      ["48%", "48%", "46px"],
    ] as const
    return (
      <>
        <div
          className="pointer-events-none absolute inset-0"
          style={{
            backgroundImage: `radial-gradient(circle at 14% 20%, ${withAlpha(light, 0.22)}, transparent 28%), radial-gradient(circle at 82% 68%, ${withAlpha(dark, 0.22)}, transparent 34%), linear-gradient(135deg, ${withAlpha(overlay, 0.28)}, transparent 58%)`,
          }}
        />
        <div
          className="pointer-events-none absolute inset-0 opacity-30"
          style={{
            backgroundImage: `linear-gradient(90deg, ${withAlpha(text, 0.08)} 1px, transparent 1px), linear-gradient(${withAlpha(text, 0.08)} 1px, transparent 1px)`,
            backgroundSize: "56px 56px",
          }}
        />
        <span className="pointer-events-none absolute left-[12%] top-[18%] h-px w-[72%] rotate-[17deg]" style={{ backgroundColor: withAlpha(text, 0.16) }} />
        <span className="pointer-events-none absolute left-[13%] top-[56%] h-px w-[68%] -rotate-[13deg]" style={{ backgroundColor: withAlpha(text, 0.14) }} />
        <span className="pointer-events-none absolute left-[34%] top-[80%] h-px w-[48%] -rotate-[34deg]" style={{ backgroundColor: withAlpha(text, 0.12) }} />
        <span className="pointer-events-none absolute left-[48%] top-[16%] h-[70%] w-px rotate-[12deg]" style={{ backgroundColor: withAlpha(text, 0.12) }} />
        {nodes.map(([left, top, size], index) => (
          <span
            key={`${left}-${top}`}
            className="pointer-events-none absolute rounded-full border"
            style={{
              left,
              top,
              width: size,
              height: size,
              borderColor: withAlpha(text, 0.22),
              backgroundColor: withAlpha(index % 2 === 0 ? light : dark, 0.16),
              boxShadow: `0 0 0 6px ${withAlpha(text, 0.05)}`,
            }}
          />
        ))}
      </>
    )
  }

  if (styleId === "access-grid") {
    return (
      <>
        <div
          className="pointer-events-none absolute inset-0 opacity-25"
          style={{
            backgroundImage: `linear-gradient(${withAlpha(text, 0.12)} 1px, transparent 1px), linear-gradient(90deg, ${withAlpha(text, 0.12)} 1px, transparent 1px)`,
            backgroundSize: "40px 40px",
          }}
        />
        {[0, 1, 2, 3, 4, 5].map((item) => (
          <span
            key={item}
            className="pointer-events-none absolute rounded-md border"
            style={{
              left: `${14 + (item % 3) * 22}%`,
              top: `${18 + Math.floor(item / 3) * 28}%`,
              width: item === 1 ? "92px" : "68px",
              height: item === 1 ? "46px" : "34px",
              borderColor: withAlpha(text, item === 1 ? 0.3 : 0.16),
              backgroundColor: withAlpha(item === 1 ? light : text, item === 1 ? 0.16 : 0.07),
            }}
          />
        ))}
      </>
    )
  }

  if (styleId === "security-radar") {
    return (
      <>
        <div
          className="pointer-events-none absolute inset-0"
          style={{
            backgroundImage: `radial-gradient(circle at 58% 48%, transparent 0 21%, ${withAlpha(text, 0.08)} 21% 21.5%, transparent 21.5% 36%, ${withAlpha(text, 0.07)} 36% 36.5%, transparent 36.5% 52%, ${withAlpha(text, 0.06)} 52% 52.5%, transparent 52.5%), linear-gradient(160deg, ${withAlpha(overlay, 0.25)}, transparent)`,
          }}
        />
        <div
          className="pointer-events-none absolute left-1/2 top-1/2 size-[34rem] -translate-x-1/2 -translate-y-1/2 rounded-full"
          style={{
            background: `conic-gradient(from 24deg, ${withAlpha(light, 0.26)}, transparent 22%, ${withAlpha(text, 0.1)} 42%, transparent 64%, ${withAlpha(dark, 0.2)}, transparent 86%)`,
          }}
        />
        <span className="pointer-events-none absolute left-0 top-1/2 h-px w-full" style={{ backgroundColor: withAlpha(text, 0.08) }} />
        <span className="pointer-events-none absolute left-1/2 top-0 h-full w-px" style={{ backgroundColor: withAlpha(text, 0.08) }} />
        {[
          ["24%", "28%", light, "16px"],
          ["72%", "22%", dark, "10px"],
          ["80%", "60%", text, "12px"],
          ["44%", "70%", light, "9px"],
          ["18%", "58%", text, "7px"],
        ].map(([left, top, color, size]) => (
          <span
            key={`${left}-${top}`}
            className="pointer-events-none absolute rounded-full"
            style={{ left, top, width: size, height: size, backgroundColor: color, boxShadow: `0 0 0 8px ${withAlpha(color, 0.08)}` }}
          />
        ))}
        <div
          className="pointer-events-none absolute bottom-[14%] left-[12%] right-[12%] rounded-md border p-3"
          style={{ borderColor: withAlpha(text, 0.16), backgroundColor: withAlpha(text, 0.07) }}
        >
          <div className="grid grid-cols-4 gap-2">
            {[0, 1, 2, 3].map((item) => (
              <span key={item} className="h-9 rounded-sm border" style={{ borderColor: withAlpha(text, 0.12), backgroundColor: withAlpha(item === 2 ? light : text, item === 2 ? 0.18 : 0.08) }} />
            ))}
          </div>
        </div>
      </>
    )
  }

  if (styleId === "trust-circuit") {
    return (
      <>
        <div
          className="pointer-events-none absolute inset-0"
          style={{
            backgroundImage: `linear-gradient(90deg, transparent 0 12%, ${withAlpha(text, 0.08)} 12% 12.3%, transparent 12.3% 34%, ${withAlpha(text, 0.08)} 34% 34.3%, transparent 34.3% 66%, ${withAlpha(text, 0.08)} 66% 66.3%, transparent 66.3%), linear-gradient(145deg, ${withAlpha(overlay, 0.25)}, transparent 65%)`,
          }}
        />
        {[14, 26, 38, 50, 62, 74].map((top, index) => (
          <div
            key={top}
            className="pointer-events-none absolute"
            style={{
              left: index % 2 === 0 ? "10%" : "22%",
              right: index % 2 === 0 ? "18%" : "8%",
              top: `${top}%`,
            }}
          >
            <span
              className="absolute left-0 right-0 top-1/2 h-px"
              style={{ backgroundColor: withAlpha(text, 0.12 + index * 0.012) }}
            />
            <span
              className="absolute left-0 top-1/2 size-4 -translate-y-1/2 rounded-sm border"
              style={{ borderColor: withAlpha(text, 0.18), backgroundColor: withAlpha(index % 2 === 0 ? light : dark, 0.15) }}
            />
            <span
              className="absolute right-0 top-1/2 size-4 -translate-y-1/2 rounded-sm border"
              style={{ borderColor: withAlpha(text, 0.18), backgroundColor: withAlpha(index % 2 === 0 ? dark : light, 0.15) }}
            />
          </div>
        ))}
        {[0, 1, 2, 3].map((item) => (
          <span
            key={item}
            className="pointer-events-none absolute rounded-md border"
            style={{
              left: `${18 + item * 17}%`,
              top: `${20 + (item % 2) * 42}%`,
              width: "76px",
              height: "44px",
              borderColor: withAlpha(text, 0.16),
              backgroundColor: withAlpha(item % 2 === 0 ? light : dark, 0.11),
            }}
          />
        ))}
      </>
    )
  }

  if (styleId === "audit-trail") {
    return (
      <>
        <div
          className="pointer-events-none absolute inset-0"
          style={{
            backgroundImage: `linear-gradient(180deg, ${withAlpha(overlay, 0.24)}, transparent), linear-gradient(90deg, ${withAlpha(text, 0.05)} 1px, transparent 1px)`,
            backgroundSize: "100% 100%, 48px 48px",
          }}
        />
        <span className="pointer-events-none absolute bottom-[10%] left-[22%] top-[12%] w-px" style={{ backgroundColor: withAlpha(text, 0.18) }} />
        <span className="pointer-events-none absolute bottom-[16%] right-[18%] top-[18%] w-px" style={{ backgroundColor: withAlpha(text, 0.1) }} />
        {[0, 1, 2, 3, 4].map((item) => (
          <div
            key={item}
            className="pointer-events-none absolute rounded-md border p-3"
            style={{
              left: item % 2 === 0 ? "26%" : "44%",
              right: item % 2 === 0 ? "16%" : "8%",
              top: `${13 + item * 15}%`,
              borderColor: withAlpha(text, 0.15),
              backgroundColor: withAlpha(item % 2 === 0 ? light : dark, 0.09),
            }}
          >
            <div className="flex items-center gap-3">
              <span className="size-5 rounded-full border" style={{ borderColor: withAlpha(text, 0.2), backgroundColor: withAlpha(text, 0.08) }} />
              <span className="h-2 flex-1 rounded-full" style={{ backgroundColor: withAlpha(text, 0.18) }} />
              <span className="h-2 w-14 rounded-full" style={{ backgroundColor: withAlpha(text, 0.11) }} />
            </div>
          </div>
        ))}
      </>
    )
  }

  if (styleId === "session-orbit") {
    return (
      <>
        <div
          className="pointer-events-none absolute inset-0"
          style={{
            backgroundImage: `radial-gradient(circle at 52% 48%, ${withAlpha(light, 0.18)}, transparent 18%, ${withAlpha(text, 0.07)} 18.5%, transparent 19%, transparent 32%, ${withAlpha(text, 0.07)} 32.5%, transparent 33%, transparent 47%, ${withAlpha(text, 0.06)} 47.5%, transparent 48%), linear-gradient(145deg, ${withAlpha(overlay, 0.22)}, transparent 70%)`,
          }}
        />
        <div
          className="pointer-events-none absolute left-1/2 top-1/2 w-36 -translate-x-1/2 -translate-y-1/2 rounded-md border p-3"
          style={{ borderColor: withAlpha(text, 0.22), backgroundColor: withAlpha(light, 0.12) }}
        >
          <div className="mb-3 h-2 w-20 rounded-full" style={{ backgroundColor: withAlpha(text, 0.26) }} />
          <div className="space-y-1.5">
            <span className="block h-1.5 w-full rounded-full" style={{ backgroundColor: withAlpha(text, 0.16) }} />
            <span className="block h-1.5 w-2/3 rounded-full" style={{ backgroundColor: withAlpha(text, 0.12) }} />
          </div>
        </div>
        {[
          ["18%", "20%", "52px", "34px", dark],
          ["72%", "18%", "34px", "56px", light],
          ["78%", "66%", "54px", "34px", dark],
          ["18%", "68%", "58px", "34px", light],
          ["50%", "10%", "42px", "30px", text],
          ["46%", "80%", "42px", "30px", text],
        ].map(([left, top, width, height, color]) => (
          <span
            key={`${left}-${top}`}
            className="pointer-events-none absolute rounded-md border"
            style={{
              left,
              top,
              width,
              height,
              borderColor: withAlpha(text, 0.18),
              backgroundColor: withAlpha(color, 0.13),
              boxShadow: `0 0 0 8px ${withAlpha(color, 0.04)}`,
            }}
          />
        ))}
        <span className="pointer-events-none absolute left-[22%] top-[29%] h-px w-[56%] rotate-[14deg]" style={{ backgroundColor: withAlpha(text, 0.11) }} />
        <span className="pointer-events-none absolute left-[22%] top-[70%] h-px w-[56%] -rotate-[12deg]" style={{ backgroundColor: withAlpha(text, 0.1) }} />
      </>
    )
  }

  return (
    <>
      <span
        className="pointer-events-none absolute -right-24 -top-24 size-80 rounded-full opacity-15"
        style={{ backgroundColor: light }}
      />
      <span
        className="pointer-events-none absolute -bottom-40 -left-20 size-96 rounded-full opacity-15"
        style={{ backgroundColor: dark }}
      />
      <div
        className="pointer-events-none absolute inset-0 opacity-25"
        style={{ backgroundColor: overlay }}
      />
    </>
  )
}

function LoginFormSample({
  branding,
  formPanelStyle,
  inputStyle,
  primaryButtonStyle,
  secondaryButtonStyle,
  tokens,
  page,
  logoPlacement = "inside-form",
  embedded,
  logoDetail,
}: {
  branding: TopPanelPreviewBranding
  formPanelStyle: CSSProperties
  inputStyle: CSSProperties
  primaryButtonStyle: CSSProperties
  secondaryButtonStyle: CSSProperties
  tokens: Record<string, string>
  page: LoginPagePreview
  logoPlacement?: LoginFormLogoPlacement | "none"
  embedded?: boolean
  logoDetail?: string
}) {
  const content = (
    <>
      {logoPlacement === "inside-form" && <BrandPreviewLockup branding={branding} logoDetail={logoDetail} centered />}
      <div className="space-y-1 text-center">
        <p className="text-2xl font-semibold tracking-tight">{page.title}</p>
        <p className="text-sm" style={{ color: tokens["colors.textMuted"] }}>
          {page.subtitle}
        </p>
      </div>
      <div className="space-y-4">
        {page.elements.map((element, index) => (
          <LoginTemplatePreviewElement
            key={`${element.type}-${index}`}
            element={element}
            inputStyle={inputStyle}
            primaryButtonStyle={primaryButtonStyle}
            secondaryButtonStyle={secondaryButtonStyle}
            tokens={tokens}
          />
        ))}
      </div>
    </>
  )

  if (!embedded && logoPlacement === "above-form") {
    return (
      <div className="w-full max-w-md space-y-4">
        <BrandPreviewLockup branding={branding} logoDetail={logoDetail} centered />
        <div
          className="w-full space-y-6 rounded-md border p-7"
          style={{ ...formPanelStyle, boxShadow: tokens["effects.authFormPanelShadow"] }}
        >
          {content}
        </div>
      </div>
    )
  }

  return (
    <div
      className={`${embedded ? "" : "w-full max-w-md rounded-md border p-7"} space-y-6`}
      style={embedded ? undefined : { ...formPanelStyle, boxShadow: tokens["effects.authFormPanelShadow"] }}
    >
      {content}
    </div>
  )
}

function LoginTemplatePreviewElement({
  element,
  inputStyle,
  primaryButtonStyle,
  secondaryButtonStyle,
  tokens,
}: {
  element: LoginPageElement
  inputStyle: CSSProperties
  primaryButtonStyle: CSSProperties
  secondaryButtonStyle: CSSProperties
  tokens: Record<string, string>
}) {
  if (element.type === "field") {
    const icon = element.kind === "password"
      ? <Lock className="size-4" />
      : element.kind === "email"
        ? <Mail className="size-4" />
        : element.kind === "tel"
          ? <Phone className="size-4" />
          : element.kind === "code"
            ? <Hash className="size-4" />
            : undefined
    return (
      <InputPreviewField
        label={element.label}
        value={element.value ?? ""}
        icon={icon}
        rightIcon={element.kind === "password" ? <Eye className="size-4" /> : undefined}
        style={inputStyle}
      />
    )
  }

  if (element.type === "readonly") {
    return <InputPreviewField label={element.label} value={element.value} style={{ ...inputStyle, color: tokens["colors.textMuted"] }} />
  }

  if (element.type === "select") {
    return (
      <InputPreviewField
        label={element.label}
        value={element.value}
        rightIcon={<ChevronDown className="size-4" />}
        style={inputStyle}
      />
    )
  }

  if (element.type === "button") {
    const buttonStyle = !element.variant || element.variant === "primary"
      ? primaryButtonStyle
      : element.variant === "ghost"
        ? {
            backgroundColor: "transparent",
            borderColor: "transparent",
            color: primaryButtonStyle.backgroundColor,
          }
        : secondaryButtonStyle
    return (
      <button type="button" className="h-10 w-full px-5 text-sm font-medium" style={buttonStyle}>
        {element.label}
      </button>
    )
  }

  if (element.type === "link") {
    return <p className="text-center text-sm font-medium text-primary underline-offset-4">{element.label}</p>
  }

  if (element.type === "divider") {
    return (
      <div className="flex items-center gap-3">
        <span className="h-px flex-1" style={{ backgroundColor: tokens["colors.border"] }} />
        <span className="text-xs" style={{ color: tokens["colors.textMuted"] }}>{element.label}</span>
        <span className="h-px flex-1" style={{ backgroundColor: tokens["colors.border"] }} />
      </div>
    )
  }

  if (element.type === "checkbox") {
    return (
      <div className="flex items-start gap-2 rounded-md border p-3 text-sm" style={{ borderColor: inputStyle.borderColor }}>
        <span className="mt-0.5 size-4 rounded border" style={{ backgroundColor: inputStyle.backgroundColor, borderColor: inputStyle.borderColor }} />
        <span>{element.label}</span>
      </div>
    )
  }

  if (element.type === "alert") {
    const alertColor = element.tone === "warning" ? "#f59e0b" : element.tone === "info" ? "#2563eb" : "#dc2626"
    return (
      <div className="rounded-md border p-3 text-sm" style={{ borderColor: withAlpha(alertColor, 0.35), backgroundColor: withAlpha(alertColor, 0.1) }}>
        {element.text}
      </div>
    )
  }

  if (element.type === "section") {
    return (
      <div className="rounded-md border p-3" style={{ borderColor: inputStyle.borderColor }}>
        <p className="text-sm font-medium">{element.title}</p>
        {element.description && <p className="mt-1 text-xs" style={{ color: tokens["colors.textMuted"] }}>{element.description}</p>}
      </div>
    )
  }

  if (element.type === "scope-list") {
    return (
      <div className="grid gap-2">
        {element.items.map((item) => (
          <div key={item} className="rounded-md border px-3 py-2 font-mono text-xs" style={{ borderColor: inputStyle.borderColor }}>
            {item}
          </div>
        ))}
      </div>
    )
  }

  if (element.type === "tile-list") {
    return (
      <div className={element.columns === 1 ? "grid gap-2" : "grid gap-2 sm:grid-cols-2"}>
        {element.items.map((item) => (
          <div key={item.title} className="rounded-md border p-3" style={{ borderColor: inputStyle.borderColor }}>
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-xs font-medium">{item.title}</p>
                {item.description && <p className="text-xs" style={{ color: tokens["colors.textMuted"] }}>{item.description}</p>}
                {item.scopes && (
                  <div className="mt-2 flex flex-wrap gap-1.5">
                    {item.scopes.map((scope) => (
                      <span
                        key={scope}
                        className="rounded border px-2 py-0.5 font-mono text-xs"
                        style={{ borderColor: inputStyle.borderColor, color: tokens["colors.textMuted"] }}
                      >
                        {scope}
                      </span>
                    ))}
                  </div>
                )}
              </div>
              {item.actionLabel && (
                <button
                  type="button"
                  aria-label={item.actionLabel}
                  className="flex size-8 shrink-0 items-center justify-center rounded-md border"
                  style={{
                    borderColor: secondaryButtonStyle.borderColor,
                    backgroundColor: secondaryButtonStyle.backgroundColor,
                    color: secondaryButtonStyle.color,
                  }}
                >
                  <Trash2 className="size-3.5" aria-hidden="true" />
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    )
  }

  return null
}

function BrandPreviewLockup({
  branding,
  logoDetail,
  panel,
  centered,
}: {
  branding: TopPanelPreviewBranding
  logoDetail?: string
  panel?: boolean
  centered?: boolean
}) {
  const logoSrc = resolveBrandingLogoUrl(branding.logoUrl) ?? "/logo.png"
  const label = branding.logoLabel || "Maintainerd-IAM"
  const detail = logoDetail?.trim()

  return (
    <div className={`flex items-center gap-3 ${centered ? "justify-center" : ""}`}>
      <img
        src={logoSrc}
        alt={label}
        className={`size-9 rounded-md object-contain ${panel ? "bg-white/15 p-1" : ""}`}
      />
      {branding.showLogoLabel && (
        <div className="min-w-0">
          <p className={`truncate font-semibold ${detail ? "text-sm" : "text-lg"}`}>{label}</p>
          {detail && <p className="text-xs opacity-75">{detail}</p>}
        </div>
      )}
    </div>
  )
}

function SidePanelPreview({ tokens }: { tokens: Record<string, string> }) {
  const sidePanelStyle: CSSProperties = {
    backgroundColor: tokens["colors.sidePanelBackground"],
    borderColor: tokens["colors.sidePanelBorder"],
    color: tokens["colors.textPrimary"],
    fontFamily: tokens["font.family"],
  }
  const sideLabelStyle = styleForComponent(tokens, "sidePanelSectionLabel")
  const sideItemStyle = styleForComponent(tokens, "sidePanelItem")
  const sideActiveStyle = styleForComponent(tokens, "sidePanelItemActive")
  const sideSubItemStyle = styleForComponent(tokens, "sidePanelSubItem")
  const sideSubActiveStyle = styleForComponent(tokens, "sidePanelSubItemActive")

  return (
    <aside className="max-w-[17rem] rounded-md border px-3 pb-4 pt-4" style={sidePanelStyle}>
      <div className="space-y-3">
        <div className="space-y-1.5">
          <div className="flex h-7 items-center px-2 text-xs font-semibold" style={sideLabelStyle}>
            Overview
          </div>
          <SidePreviewItem
            icon={<LayoutDashboard className="size-[18px]" />}
            label="Dashboard"
            style={sideItemStyle}
            iconColor={tokens["colors.sidePanelItemIcon"]}
          />
        </div>

        <div className="space-y-1.5">
          <div className="flex h-7 items-center px-2 text-xs font-semibold" style={sideLabelStyle}>
            Identity & Access
          </div>
          <SidePreviewItem
            icon={<Users className="size-[18px]" />}
            label="User Management"
            style={sideActiveStyle}
            iconColor={tokens["colors.sidePanelItemActiveIcon"]}
            chevronColor={tokens["colors.sidePanelChevron"]}
            open
            active
          />
          <div className="space-y-0.5 py-1">
            <div className="flex h-8 items-center px-2 pl-[34px] text-sm font-semibold" style={sideSubActiveStyle}>
              Users
            </div>
            <div className="flex h-8 items-center px-2 pl-[34px] text-sm font-medium" style={sideSubItemStyle}>
              Roles
            </div>
          </div>
          <SidePreviewItem
            icon={<KeyRound className="size-[18px]" />}
            label="Authentication"
            style={sideItemStyle}
            iconColor={tokens["colors.sidePanelItemIcon"]}
            chevronColor={tokens["colors.sidePanelChevron"]}
          />
        </div>

        <div className="space-y-1.5">
          <div className="flex h-7 items-center px-2 text-xs font-semibold" style={sideLabelStyle}>
            Branding
          </div>
          <SidePreviewItem
            icon={<Palette className="size-[18px]" />}
            label="Branding"
            style={sideItemStyle}
            iconColor={tokens["colors.sidePanelItemIcon"]}
          />
          <SidePreviewItem
            icon={<Shield className="size-[18px]" />}
            label="Security"
            style={{
              ...sideItemStyle,
              backgroundColor: componentValue(tokens, "sidePanelItem", "hoverColor"),
              color: tokens["colors.sidePanelItemHoverText"],
            }}
            iconColor={tokens["colors.sidePanelItemIconHover"]}
          />
        </div>
      </div>
    </aside>
  )
}

function ButtonTypePreview({
  component,
  label,
  tokens,
}: {
  component: string
  label: string
  tokens: Record<string, string>
}) {
  const buttonStyle = styleForComponent(tokens, component)

  return (
    <div
      className="space-y-4 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div className="flex flex-wrap gap-3">
        <button type="button" className="flex h-10 items-center gap-2 px-5 text-sm font-medium" style={buttonStyle}>
          {label}
        </button>
        <button
          type="button"
          className="flex h-10 items-center gap-2 px-5 text-sm font-medium"
          style={{ ...buttonStyle, backgroundColor: componentValue(tokens, component, "hoverColor") }}
        >
          {label} hover
        </button>
      </div>
    </div>
  )
}

function TablePreview({ tokens }: { tokens: Record<string, string> }) {
  const containerStyle = styleForComponent(tokens, "tableContainer")
  const headerStyle = styleForComponent(tokens, "tableHeader")
  const rowStyle = styleForComponent(tokens, "tableRow")
  const rowHoverStyle = { ...rowStyle, backgroundColor: componentValue(tokens, "tableRow", "hoverColor") }
  const cellStyle = styleForComponent(tokens, "tableCell")

  // Mirrors the console exactly: the wrapper rounds all four corners, the
  // header only its top corners, and only the last body row rounds its bottom
  // corners. Row borders come from the theme tokens, not hardcoded classes.
  const headerRadius = componentValue(tokens, "tableHeader", "borderRadius")
  const rowRadius = componentValue(tokens, "tableRow", "borderRadius")
  const rowBorderWidth = componentValue(tokens, "tableRow", "borderThickness")
  const rowBorderColor = componentValue(tokens, "tableRow", "borderColor")
  const headerBorderColor = componentValue(tokens, "tableHeader", "borderColor")

  const headerCorners: CSSProperties = {
    ...headerStyle,
    minHeight: 0,
    borderRadius: `${headerRadius} ${headerRadius} 0 0`,
    borderBottom: `${rowBorderWidth} solid ${headerBorderColor}`,
  }
  const baseRow: CSSProperties = { ...rowStyle, minHeight: 0, borderRadius: 0, borderBottom: `${rowBorderWidth} solid ${rowBorderColor}` }
  const lastRowCorners: CSSProperties = {
    ...rowHoverStyle,
    minHeight: 0,
    borderBottom: `${rowBorderWidth} solid ${rowBorderColor}`,
    borderRadius: `0 0 ${rowRadius} ${rowRadius}`,
  }
  const headCellStyle: CSSProperties = { color: headerStyle.color, fontSize: headerStyle.fontSize }
  const bodyCellStyle: CSSProperties = { color: cellStyle.color, fontSize: cellStyle.fontSize }

  return (
    <div
      className="rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div className="overflow-hidden" style={containerStyle}>
        <div className="grid grid-cols-[1.2fr_0.8fr_0.8fr]" style={headerCorners}>
          {["Name", "Status", "Updated"].map((header) => (
            <div key={header} className="flex h-10 items-center px-2 text-left font-medium whitespace-nowrap" style={headCellStyle}>
              {header}
            </div>
          ))}
        </div>
        {[
          ["Console SPA", "Active", "Today"],
          ["Identity API", "Pending", "Yesterday"],
        ].map((row, index) => (
          <div key={row[0]} className="grid grid-cols-[1.2fr_0.8fr_0.8fr]" style={index === 0 ? baseRow : lastRowCorners}>
            {row.map((cell) => (
              <div key={cell} className="flex min-h-10 items-center px-2 text-left align-middle whitespace-nowrap" style={bodyCellStyle}>
                {cell}
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  )
}

function ListingPreview({ tokens }: { tokens: Record<string, string> }) {
  const itemStyle = styleForComponent(tokens, "listingItem")
  const iconStyle = styleForComponent(tokens, "iconContainer")
  const metaStyle = styleForComponent(tokens, "listingItemMeta")

  return (
    <div
      className="space-y-3 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div className="flex items-start justify-between gap-3 p-4" style={itemStyle}>
        <div className="flex min-w-0 items-start gap-3">
          <span className="flex size-9 items-center justify-center" style={iconStyle}>
            <Shield className="size-4" />
          </span>
          <div className="min-w-0 space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <p className="text-sm font-semibold">Admin role</p>
              <BadgePreviewBadge status="active" group="positive" tokens={tokens} />
            </div>
            <p className="text-sm" style={{ color: tokens["colors.textMuted"] }}>
              Full access for trusted administrators.
            </p>
            <div className="flex flex-wrap items-center gap-3 text-xs" style={metaStyle}>
              <span>Added today</span>
              <span>System tenant</span>
            </div>
          </div>
        </div>
        <MoreHorizontal className="size-4 opacity-60" />
      </div>
      <p className="text-xs" style={{ color: tokens["colors.textMuted"] }}>
        Meta lines like dates and attributes stay plain text — they are not sub-containers.
      </p>
    </div>
  )
}

function IconContainersPreview({ tokens }: { tokens: Record<string, string> }) {
  const iconStyle = styleForComponent(tokens, "iconContainer")
  const rows: Array<{ icon: LucideIcon; title: string; description: string }> = [
    { icon: Shield, title: "Role assignment", description: "Used in user details role rows" },
    { icon: UserRound, title: "User record", description: "Used in identities and members lists" },
    { icon: Smartphone, title: "MFA method", description: "Used in security setup options" },
  ]

  return (
    <div
      className="space-y-3 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      {rows.map(({ icon: Icon, title, description }) => (
        <div
          key={title}
          className="flex items-center gap-3 rounded-md border p-3"
          style={{
            backgroundColor: tokens["components.listingItem.background"],
            borderColor: tokens["components.listingItem.borderColor"],
          }}
        >
          <span className="flex size-9 shrink-0 items-center justify-center" style={iconStyle}>
            <Icon className="size-4" />
          </span>
          <span className="min-w-0">
            <span className="block text-sm font-medium">{title}</span>
            <span className="block text-xs" style={{ color: tokens["colors.textMuted"] }}>
              {description}
            </span>
          </span>
        </div>
      ))}
    </div>
  )
}

function SubContainerPreview({ tokens }: { tokens: Record<string, string> }) {
  const subStyle = styleForComponent(tokens, "listingSubContainer")
  const subRadius = componentValue(tokens, "listingSubContainer", "borderRadius")

  // Mirrors the console: the sub-container's own radius rounds all four corners
  // of the box, the first row rounds its top corners, and the last row its
  // bottom corners.
  const firstRowCorners: CSSProperties = { borderRadius: `${subRadius} ${subRadius} 0 0` }
  const lastRowCorners: CSSProperties = { borderRadius: `0 0 ${subRadius} ${subRadius}` }

  return (
    <div
      className="space-y-3 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div className="overflow-hidden p-1" style={subStyle}>
        <div className="divide-y">
          {[
            ["environment", "production"],
            ["region", "us-east-1"],
            ["owner", "platform-team"],
          ].map(([key, value], index) => (
            <div
              key={key}
              className="grid grid-cols-[minmax(0,1fr)_minmax(0,2fr)] gap-4 px-3 py-2.5"
              style={index === 0 ? firstRowCorners : index === 2 ? lastRowCorners : undefined}
            >
              <span className="text-sm font-medium">{key}</span>
              <span className="font-mono text-sm" style={{ color: tokens["colors.textMuted"] }}>
                {value}
              </span>
            </div>
          ))}
        </div>
      </div>
      <p className="text-xs" style={{ color: tokens["colors.textMuted"] }}>
        Metadata rows, key/value pairs, and nested surfaces inside a listing card.
      </p>
    </div>
  )
}

function OptionCardPreview({ tokens }: { tokens: Record<string, string> }) {
  const optionStyle = styleForComponent(tokens, "optionCard")
  const hoverStyle: CSSProperties = {
    ...optionStyle,
    backgroundColor: componentValue(tokens, "optionCard", "hoverColor"),
  }

  return (
    <div
      className="space-y-3 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div className="flex items-center justify-between gap-3 px-3 py-2.5" style={optionStyle}>
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
            <Settings className="size-4" />
          </span>
          <div className="min-w-0 space-y-0.5">
            <p className="truncate text-sm font-medium">IAM Shortcuts</p>
            <p className="truncate text-xs" style={{ color: tokens["colors.textMuted"] }}>
              Common control-plane workflows
            </p>
          </div>
        </div>
        <ChevronRight className="size-4 shrink-0 opacity-60" />
      </div>
      <div className="flex items-center justify-between gap-3 px-3 py-2.5" style={hoverStyle}>
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
            <Shield className="size-4" />
          </span>
          <div className="min-w-0 space-y-0.5">
            <p className="truncate text-sm font-medium">Security Posture</p>
            <p className="truncate text-xs" style={{ color: tokens["colors.textMuted"] }}>
              Configure authentication policy
            </p>
          </div>
        </div>
        <ChevronRight className="size-4 shrink-0 opacity-60" />
      </div>
      <p className="text-xs" style={{ color: tokens["colors.textMuted"] }}>
        Clickable option rows — quick actions, shortcuts, and navigation links.
      </p>
    </div>
  )
}

function CardComponentPreview({ tokens }: { tokens: Record<string, string> }) {
  const cardStyle = styleForComponent(tokens, "card")

  return (
    <div
      className="rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div className="space-y-3 p-4 shadow-xs" style={cardStyle}>
        <p className="text-sm font-semibold">Security policy</p>
        <p className="text-xs" style={{ color: tokens["colors.textMuted"] }}>
          Password and session requirements
        </p>
      </div>
    </div>
  )
}

function AlertPreview({ tokens }: { tokens: Record<string, string> }) {
  const alertStyle = styleForComponent(tokens, "alert")

  return (
    <div
      className="space-y-3 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div className="flex items-start gap-3 px-4 py-3 text-sm" style={alertStyle}>
        <Shield className="mt-0.5 size-4 shrink-0" aria-hidden />
        <div className="min-w-0 space-y-0.5">
          <p className="font-medium">This is a system client.</p>
          <p>Some settings may be restricted and cannot be modified.</p>
        </div>
      </div>
      <p className="text-xs" style={{ color: tokens["colors.textMuted"] }}>
        Inline notice banners on form pages and detail views.
      </p>
    </div>
  )
}

function InputsPreview({ tokens }: { tokens: Record<string, string> }) {
  const inputStyle = styleForComponent(tokens, "input")
  const datePickerStyle = styleForComponent(tokens, "datePicker")
  const selectStyle: CSSProperties = {
    ...inputStyle,
    paddingLeft: "0.75rem",
    paddingRight: "0.75rem",
  }
  const secondaryButtonStyle = styleForComponent(tokens, "secondaryButton")
  const errorStyle: CSSProperties = {
    ...inputStyle,
    borderColor: "#ef4444",
    boxShadow: "0 0 0 3px rgba(239,68,68,0.14)",
  }
  const hoverStyle: CSSProperties = {
    ...inputStyle,
    backgroundColor: componentValue(tokens, "input", "hoverColor"),
  }

  return (
    <div
      className="space-y-5 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div className="grid gap-4 md:grid-cols-2">
        <InputPreviewField label="Text" value="Maintainerd Console" style={inputStyle} />
        <InputPreviewField label="Search" value="Search users..." icon={<Search className="size-4" />} style={hoverStyle} />
        <InputPreviewField label="Email" value="admin@example.com" icon={<Mail className="size-4" />} style={inputStyle} />
        <InputPreviewField label="URL" value="https://auth.example.com" icon={<Globe2 className="size-4" />} style={inputStyle} />
        <InputPreviewField
          label="Password"
          value="************"
          icon={<Lock className="size-4" />}
          rightIcon={<Eye className="size-4" />}
          style={inputStyle}
        />
        <InputPreviewField label="Number" value="300" icon={<Hash className="size-4" />} style={inputStyle} />
        <InputPreviewField label="Phone" value="+1 555 123 4567" icon={<Phone className="size-4" />} style={inputStyle} />
        <InputPreviewSelect label="Select" value="System tenant" style={selectStyle} />
        <InputPreviewField
          label="Date picker"
          value="08/02/2026"
          rightIcon={<CalendarDays className="size-4" />}
          style={{
            ...datePickerStyle,
            borderColor: tokens["colors.primary"],
            boxShadow: `0 0 0 3px ${withAlpha(tokens["colors.primary"], 0.2)}`,
          }}
        />
        <InputPreviewField label="Disabled" value="Locked value" style={{ ...inputStyle, opacity: 0.5 }} />
      </div>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <InputPreviewPhoneWithCountry inputStyle={inputStyle} />
        <InputPreviewScope inputStyle={inputStyle} buttonStyle={secondaryButtonStyle} />
      </div>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <InputPreviewTextarea style={inputStyle} />
        <InputPreviewFileUpload tokens={tokens} inputStyle={inputStyle} buttonStyle={secondaryButtonStyle} />
      </div>

      <InputPreviewField
        label="Validation error"
        value="missing-domain"
        description="Use the same invalid border and field error treatment."
        error="Enter a valid domain."
        style={errorStyle}
      />
    </div>
  )
}

function InputPreviewField({
  label,
  value,
  icon,
  rightIcon,
  description,
  error,
  style,
}: {
  label: string
  value: string
  icon?: ReactNode
  rightIcon?: ReactNode
  description?: string
  error?: string
  style: CSSProperties
}) {
  return (
    <div className="space-y-1.5">
      <InputPreviewLabel>{label}</InputPreviewLabel>
      <div
        className="flex h-10 w-full min-w-0 items-center gap-2 px-3.5 py-2 text-base transition-[color,box-shadow] outline-none md:text-sm"
        style={inputControlStyle(style)}
      >
        {icon && <span className="shrink-0 opacity-60">{icon}</span>}
        <span className="min-w-0 flex-1 truncate">{value}</span>
        {rightIcon && (
          <span className="flex h-full shrink-0 items-center px-0.5 opacity-60">
            {rightIcon}
          </span>
        )}
      </div>
      {description && !error && (
        <p className="text-xs leading-normal text-muted-foreground">{description}</p>
      )}
      {error && <p className="text-xs leading-normal text-red-600">{error}</p>}
    </div>
  )
}

function InputPreviewSelect({
  label,
  value,
  style,
}: {
  label: string
  value: string
  style: CSSProperties
}) {
  return (
    <div className="space-y-1.5">
      <InputPreviewLabel>{label}</InputPreviewLabel>
      <div className="relative">
        <div
          className="flex h-10 w-full min-w-0 items-center justify-between gap-2 px-3 py-2 text-base transition-[color,box-shadow] outline-none md:text-sm"
          style={inputControlStyle(style)}
        >
          <span className="min-w-0 flex-1 truncate">{value}</span>
          <ChevronDown className="size-4 shrink-0 opacity-50" />
        </div>
        <div
          className="absolute left-0 top-11 z-10 hidden w-full rounded-md border p-1 shadow-md sm:block"
          style={{
            backgroundColor: tokensFallback(style.backgroundColor, "#ffffff"),
            borderColor: style.borderColor,
            color: style.color,
          }}
        >
          {["System tenant", "Acme tenant", "Disabled option"].map((item, index) => (
            <div
              key={item}
              className="flex items-center gap-2 rounded-sm px-2 py-1.5 text-sm"
              style={{ backgroundColor: index === 1 ? "rgba(148,163,184,0.16)" : "transparent", opacity: index === 2 ? 0.5 : 1 }}
            >
              <span className="min-w-0 flex-1 truncate">{item}</span>
              {index === 0 && <Check className="size-3.5" />}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function InputPreviewPhoneWithCountry({
  inputStyle,
}: {
  inputStyle: CSSProperties
}) {
  const countrySegmentStyle = joinedPhoneSegmentStyle(inputStyle, "left")
  const phoneSegmentStyle = joinedPhoneSegmentStyle(inputStyle, "right")

  return (
    <div className="space-y-1.5">
      <InputPreviewLabel>Phone with country</InputPreviewLabel>
      <div className="flex">
        <div
          className="flex h-10 w-[100px] shrink-0 items-center justify-between gap-1.5 px-3 text-base transition-[color,box-shadow] outline-none md:text-sm"
          style={countrySegmentStyle}
        >
          <span className="text-sm font-medium">US</span>
          <span>+1</span>
          <ChevronsUpDown className="size-3 shrink-0 opacity-50" />
        </div>
        <div
          className="flex h-10 min-w-0 flex-1 items-center px-3.5 py-2 text-base transition-[color,box-shadow] outline-none md:text-sm"
          style={phoneSegmentStyle}
        >
          <span className="truncate">(555) 123-4567</span>
        </div>
      </div>
    </div>
  )
}

function InputPreviewScope({
  inputStyle,
  buttonStyle,
}: {
  inputStyle: CSSProperties
  buttonStyle: CSSProperties
}) {
  return (
    <div className="space-y-1.5">
      <InputPreviewLabel>Scope picker</InputPreviewLabel>
      <div className="flex gap-2">
        <div className="flex h-10 min-w-0 flex-1 items-center px-3.5 py-2 text-sm" style={inputControlStyle(inputStyle)}>
          <span className="truncate">openid, profile, email</span>
        </div>
        <div className="flex h-10 shrink-0 items-center gap-1.5 px-3 text-sm font-medium" style={inputControlStyle(buttonStyle)}>
          Browse
          <ChevronsUpDown className="size-3.5 opacity-60" />
        </div>
      </div>
    </div>
  )
}

function InputPreviewTextarea({ style }: { style: CSSProperties }) {
  return (
    <div className="space-y-1.5">
      <InputPreviewLabel>Textarea</InputPreviewLabel>
      <div className="min-h-16 w-full px-3 py-2 text-base shadow-xs transition-[color,box-shadow] outline-none md:text-sm" style={inputControlStyle(style)}>
        <p>Users created through this registration flow inherit the selected roles.</p>
      </div>
    </div>
  )
}

function InputPreviewFileUpload({
  tokens,
  inputStyle,
  buttonStyle,
}: {
  tokens: Record<string, string>
  inputStyle: CSSProperties
  buttonStyle: CSSProperties
}) {
  return (
    <div className="space-y-1.5">
      <InputPreviewLabel>File upload</InputPreviewLabel>
      <div
        className="relative flex min-h-[8.5rem] flex-col items-center justify-center rounded-lg border border-dashed p-6 text-center transition-colors"
        style={{
          backgroundColor: inputStyle.backgroundColor,
          borderColor: inputStyle.borderColor ?? tokens["colors.textMuted"],
          borderRadius: inputStyle.borderRadius,
          color: inputStyle.color,
        }}
      >
        <Upload className="mb-3 size-9 opacity-60" />
        <p className="text-sm font-medium">Drop your image here, or click to browse</p>
        <p className="mt-1 text-xs" style={{ color: tokens["colors.textMuted"] }}>
          PNG, JPG, GIF up to 5MB
        </p>
        <span className="mt-3 inline-flex h-9 items-center px-3 text-sm font-medium" style={inputControlStyle(buttonStyle)}>
          Choose File
        </span>
      </div>
    </div>
  )
}

function InputPreviewLabel({ children }: { children: ReactNode }) {
  return <div className="flex w-fit gap-2 text-sm font-medium leading-snug">{children}</div>
}

function inputControlStyle(style: CSSProperties): CSSProperties {
  return {
    ...style,
    minHeight: "2.5rem",
  }
}

function joinedPhoneSegmentStyle(style: CSSProperties, side: "left" | "right"): CSSProperties {
  const radius = style.borderRadius
  const joinedStyle: CSSProperties = {
    ...inputControlStyle(style),
    borderRadius: 0,
  }

  if (side === "left") {
    return {
      ...joinedStyle,
      borderRightWidth: 0,
      borderTopLeftRadius: radius,
      borderBottomLeftRadius: radius,
      borderTopRightRadius: 0,
      borderBottomRightRadius: 0,
    }
  }

  return {
    ...joinedStyle,
    borderTopLeftRadius: 0,
    borderBottomLeftRadius: 0,
    borderTopRightRadius: radius,
    borderBottomRightRadius: radius,
  }
}

function tokensFallback(value: CSSProperties["backgroundColor"], fallback: string) {
  return typeof value === "string" && value && value !== "transparent" ? value : fallback
}

function withAlpha(hex: string | undefined, alpha: number) {
  return hexToRgba(hex, alpha)
}

function SwitchPreview({ tokens }: { tokens: Record<string, string> }) {
  const switchStyle = styleForComponent(tokens, "switch")
  const uncheckedStyle: CSSProperties = {
    ...switchStyle,
    backgroundColor: componentValue(tokens, "switch", "uncheckedBackground"),
  }
  const thumbStyle: CSSProperties = {
    backgroundColor: componentValue(tokens, "switch", "thumbColor"),
  }

  return (
    <div
      className="grid gap-4 rounded-md border p-4 sm:grid-cols-2"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <SwitchStatePreview label="Enabled" checked style={switchStyle} thumbStyle={thumbStyle} />
      <SwitchStatePreview label="Disabled" style={uncheckedStyle} thumbStyle={thumbStyle} />
    </div>
  )
}

function SwitchSubContainerPreview({ tokens }: { tokens: Record<string, string> }) {
  const boxStyle = styleForComponent(tokens, "switchSubContainer")
  const switchStyle = styleForComponent(tokens, "switch")
  const thumbStyle: CSSProperties = {
    backgroundColor: componentValue(tokens, "switch", "thumbColor"),
  }

  return (
    <div
      className="space-y-3 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div className="space-y-4 rounded-md border p-4" style={boxStyle}>
        <div className="flex items-center justify-between gap-3">
          <div className="min-w-0 space-y-0.5">
            <p className="text-sm font-medium">Allow registration</p>
            <p className="text-xs" style={{ color: tokens["colors.textMuted"] }}>
              Allow clients using this provider to create new accounts.
            </p>
          </div>
          <SwitchStatePreview label="" checked style={switchStyle} thumbStyle={thumbStyle} />
        </div>
      </div>
      <p className="text-xs" style={{ color: tokens["colors.textMuted"] }}>
        Bordered box around switch fields — allow registration, token federation, and JIT provisioning.
      </p>
    </div>
  )
}

function CheckboxSubContainerPreview({ tokens }: { tokens: Record<string, string> }) {
  const boxStyle = styleForComponent(tokens, "checkboxSubContainer")
  const rowHoverStyle: CSSProperties = {
    backgroundColor: componentValue(tokens, "checkboxSubContainer", "hoverColor"),
  }

  return (
    <div
      className="space-y-3 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div className="max-h-64 overflow-y-auto rounded-md border divide-y" style={boxStyle}>
        {[
          ["admin", "Full access for trusted administrators."],
          ["editor", "Manage content and publishing."],
          ["viewer", "Read-only access."],
        ].map(([title, description], index) => (
          <div
            key={title}
            className="flex items-start gap-3 p-3"
            style={index === 1 ? rowHoverStyle : undefined}
          >
            <span
              aria-hidden
              className={cn(
                "mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-[3px] border",
                index === 0
                  ? "border-primary bg-primary text-primary-foreground"
                  : "border-input",
              )}
            >
              {index === 0 && <Check className="size-3.5" />}
            </span>
            <div className="min-w-0 space-y-0.5">
              <p className="text-sm font-medium">{title}</p>
              <p className="break-words text-xs" style={{ color: tokens["colors.textMuted"] }}>
                {description}
              </p>
            </div>
          </div>
        ))}
      </div>
      <p className="text-xs" style={{ color: tokens["colors.textMuted"] }}>
        Bordered box around checkbox option lists — roles and permissions pickers.
      </p>
    </div>
  )
}

function TextareaPreview({ tokens }: { tokens: Record<string, string> }) {
  const textareaStyle = styleForComponent(tokens, "textarea")
  const hoverStyle: CSSProperties = {
    ...textareaStyle,
    backgroundColor: componentValue(tokens, "textarea", "hoverColor"),
  }

  return (
    <div
      className="space-y-3 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      <div className="grid gap-4 lg:grid-cols-2">
        <div className="space-y-1.5">
          <p className="text-sm font-medium">Textarea</p>
          <div className="min-h-16 w-full px-3 py-2 text-base outline-none md:text-sm" style={inputControlStyle(textareaStyle)}>
            <p>Users created through this registration flow inherit the selected roles.</p>
          </div>
        </div>
        <div className="space-y-1.5">
          <p className="text-sm font-medium">Textarea hover</p>
          <div className="min-h-16 w-full px-3 py-2 text-base outline-none md:text-sm" style={inputControlStyle(hoverStyle)}>
            <p>Policies applied when users accept an invitation.</p>
          </div>
        </div>
      </div>
      <p className="text-xs" style={{ color: tokens["colors.textMuted"] }}>
        Multi-line inputs keep their own radius and surface.
      </p>
    </div>
  )
}

function BadgeGroupPreview({
  group,
  tokens,
}: {
  group: string
  tokens: Record<string, string>
}) {
  const statuses = BADGE_GROUP_MEMBERS[group] ?? []

  return (
    <div
      className="flex flex-wrap gap-2 rounded-md border p-4"
      style={{
        backgroundColor: tokens["colors.appBackground"],
        borderColor: tokens["colors.border"],
        color: tokens["colors.textPrimary"],
        fontFamily: tokens["font.family"],
      }}
    >
      {statuses.map((status) => (
        <BadgePreviewBadge key={status} status={status} group={group} tokens={tokens} />
      ))}
    </div>
  )
}

function SidePreviewItem({
  icon,
  label,
  style,
  iconColor,
  chevronColor,
  open,
  active,
}: {
  icon: ReactNode
  label: string
  style: CSSProperties
  iconColor: string
  chevronColor?: string
  open?: boolean
  active?: boolean
}) {
  return (
    <div className={`flex h-9 items-center gap-2 px-2 text-sm ${active ? "font-semibold" : "font-medium"}`} style={style}>
      <span className="flex shrink-0" style={{ color: iconColor }}>
        {icon}
      </span>
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {chevronColor && (
        <span
          className="text-sm transition-transform"
          style={{
            color: chevronColor,
            transform: open ? "rotate(90deg)" : undefined,
          }}
        >
          ›
        </span>
      )}
    </div>
  )
}

function BadgePreviewBadge({
  status,
  group,
  tokens,
}: {
  status: (typeof STATUS_BADGE_TYPES)[number]
  group: string
  tokens: Record<string, string>
}) {
  const prefix = `components.badges.${group}.`
  const style: CSSProperties = {
    backgroundColor: tokens[`${prefix}background`],
    borderColor: tokens[`${prefix}borderColor`],
    borderStyle: "solid",
    borderWidth: tokens[`${prefix}borderThickness`],
    borderRadius: tokens[`${prefix}borderRadius`],
    color: tokens[`${prefix}textColor`],
    ...badgeSizeStyle(tokens[`${prefix}size`]),
  }

  return (
    <span className="inline-flex items-center gap-1.5 font-medium capitalize" style={style}>
      <span className="size-1.5 rounded-full" style={{ backgroundColor: tokens[`${prefix}dotColor`] }} />
      {status}
    </span>
  )
}

function styleForComponent(tokens: Record<string, string>, component: string): CSSProperties {
  return {
    backgroundColor: componentValue(tokens, component, "background"),
    borderColor: componentValue(tokens, component, "borderColor"),
    borderStyle: "solid",
    borderWidth: componentValue(tokens, component, "borderThickness"),
    borderRadius: componentValue(tokens, component, "borderRadius"),
    color: componentValue(tokens, component, "textColor"),
    ...sizeStyle(componentValue(tokens, component, "size")),
  }
}

function componentValue(tokens: Record<string, string>, component: string, key: string) {
  return tokens[`components.${component}.${key}`]
}

function sizeStyle(size: string | undefined): CSSProperties {
  if (size === "sm") return { fontSize: "0.75rem", minHeight: "1.75rem" }
  if (size === "lg") return { fontSize: "1rem", minHeight: "2.75rem" }
  return { fontSize: "0.875rem", minHeight: "2.25rem" }
}

function SwitchStatePreview({
  label,
  checked,
  style,
  thumbStyle,
}: {
  label: string
  checked?: boolean
  style: CSSProperties
  thumbStyle: CSSProperties
}) {
  return (
    <div className="flex items-center gap-3">
      <span
        className="relative inline-flex h-[1.15rem] w-8 shrink-0 items-center shadow-xs"
        style={{
          ...style,
          minHeight: "1.15rem",
          width: "2rem",
        }}
      >
        <span
          className="absolute block size-4 rounded-full"
          style={{
            ...thumbStyle,
            left: 0,
            transform: checked ? "translateX(calc(100% - 2px))" : "translateX(0)",
          }}
        />
      </span>
      <span className="text-sm">{label}</span>
    </div>
  )
}

function badgeSizeStyle(size: string | undefined): CSSProperties {
  if (size === "md") return { fontSize: "0.875rem", padding: "0.1875rem 0.625rem" }
  if (size === "lg") return { fontSize: "1rem", padding: "0.25rem 0.75rem" }
  return { fontSize: "0.75rem", padding: "0.125rem 0.5rem" }
}
