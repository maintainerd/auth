import type { FormEvent } from "react"
import { useEffect, useMemo, useState } from "react"
import { useLocation, useNavigate, useParams } from "react-router-dom"
import { AlertCircle, ArrowLeft, Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import { FormInputField, FormTextareaField } from "@/components/form"
import { FormUrlField } from "@/components/inputs"
import { ConfirmationDialog } from "@/components/dialog"
import { useBranding, useUpdateBranding } from "@/hooks/useBranding"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import {
  authUiTemplateIdFromMetadata,
  authUiTemplateSupportsImage,
  getAuthUiTemplate,
} from "@/lib/branding/authUiTemplates"
import {
  LOGIN_PAGE_PREVIEW_GROUPS,
  loginPageContentCollectionMetadata,
  loginPagePreviewsFromMetadata,
  loginTemplateImageUrlFromMetadata,
  type LoginPageCopy,
  type LoginPagePreview,
  type LoginPagePreviewId,
} from "@/lib/branding/loginPageContent"
import type { Branding, BrandingRequest } from "@/services/api/branding/types"

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

function brandingRequestWithMetadata(branding: Branding, metadata: Record<string, unknown>): BrandingRequest {
  return {
    name: branding.name,
    layout: branding.layout,
    company_name: branding.company_name ?? "",
    logo_url: branding.logo_url ?? "",
    favicon_url: branding.favicon_url ?? "",
    support_url: branding.support_url ?? "",
    privacy_policy_url: branding.privacy_policy_url ?? "",
    terms_of_service_url: branding.terms_of_service_url ?? "",
    metadata,
  }
}

export default function LoginTemplateFormsPage() {
  const { brandingId } = useParams<{ brandingId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { showSuccess, showError } = useToast()
  const updateMutation = useUpdateBranding()

  const navState = location.state as { from?: string; backLabel?: string } | null
  const backTo = navState?.from ?? `/branding/templates/${brandingId}?tab=login-templates`
  const backLabel = navState?.backLabel ?? "Back to Login Templates"

  const { data: branding, isLoading: isFetching, isError } = useBranding(brandingId)
  const pages = useMemo(() => loginPagePreviewsFromMetadata(branding?.metadata), [branding?.metadata])
  const selectedTemplate = getAuthUiTemplate(authUiTemplateIdFromMetadata(branding?.metadata, branding?.layout))
  const supportsImage = authUiTemplateSupportsImage(selectedTemplate)

  const [draft, setDraft] = useState<LoginPageCopyDraft>(() => pageCopyDraftFromPages(pages))
  const [imageUrl, setImageUrl] = useState("")

  useEffect(() => {
    setDraft(pageCopyDraftFromPages(pages))
    setImageUrl(loginTemplateImageUrlFromMetadata(branding?.metadata))
  }, [branding?.metadata, pages])

  const draftPages = pages.map((page) => ({
    ...page,
    ...draft[page.id],
  }))

  const hasCopyChanges = draftPages.some((page) => {
    const original = pages.find((item) => item.id === page.id)
    return original?.title !== page.title || original?.subtitle !== page.subtitle
  })
  const hasImageChanges = imageUrl !== loginTemplateImageUrlFromMetadata(branding?.metadata)
  const isDirty = hasCopyChanges || (supportsImage && hasImageChanges)
  const isSaving = updateMutation.isPending
  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const updateCopy = (pageId: LoginPagePreviewId, field: keyof LoginPageCopy, value: string) => {
    setDraft((current) => ({
      ...current,
      [pageId]: {
        ...current[pageId],
        [field]: value,
      },
    }))
  }

  const resetCopy = () => {
    setDraft(pageCopyDraftFromPages(pages))
    setImageUrl(loginTemplateImageUrlFromMetadata(branding?.metadata))
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!branding) return

    try {
      const metadata = loginPageContentCollectionMetadata(
        branding.metadata,
        draftPages,
        supportsImage ? imageUrl : undefined,
      )
      await updateMutation.mutateAsync({
        brandingId: branding.branding_id,
        data: brandingRequestWithMetadata(branding, metadata),
      })
      showSuccess("Login template forms updated successfully")
      navigate(backTo)
    } catch (error) {
      showError(error)
    }
  }

  if (isFetching && !branding) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            onBack={() => navigate(backTo)}
            title="Update Login Template Forms"
            description="Update the fixed hosted-auth page text."
          />
          <Card>
            <CardContent className="space-y-4 pt-6">
              <Skeleton className="h-5 w-48" />
              <div className="grid gap-4 lg:grid-cols-2">
                {Array.from({ length: 4 }).map((_, index) => (
                  <Skeleton key={index} className="h-40 w-full" />
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </DetailsContainer>
    )
  }

  if (isError || !branding) {
    return (
      <DetailsContainer>
        <div className="flex flex-col gap-6">
          <FormPageHeader
            backUrl={backTo}
            backLabel={backLabel}
            onBack={() => navigate(backTo)}
            title="Update Login Template Forms"
            description="Update the fixed hosted-auth page text."
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
              <Button variant="outline" onClick={() => navigate(backTo)}>
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
          title={`Update ${branding.name} Login Forms`}
          description="Override the default title and supporting text for each fixed hosted-auth page state."
          showSystemBadge={branding.is_system}
        />

        <form onSubmit={handleSubmit} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Login template assets</CardTitle>
              <p className="text-sm text-muted-foreground">
                {supportsImage
                  ? "Set the visual panel image used by split login templates."
                  : "This selected login template does not use a separate image panel."}
              </p>
            </CardHeader>
            <CardContent>
              {supportsImage ? (
                <FormUrlField
                  label="Image URL"
                  value={imageUrl}
                  onChange={(event) => setImageUrl(event.target.value)}
                  placeholder="https://example.com/login-panel.jpg"
                  disabled={isSaving}
                  description="Used by split-style login templates for the visual panel."
                />
              ) : (
                <div className="rounded-md border bg-muted/30 p-4">
                  <p className="text-sm font-medium">{selectedTemplate.label}</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    Switch to a split-style login template if you need a visual panel image.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>

          {LOGIN_PAGE_PREVIEW_GROUPS.map((group) => {
            const groupPages = draftPages.filter((page) => page.group === group)
            if (groupPages.length === 0) return null

            return (
              <Card key={group}>
                <CardHeader>
                  <CardTitle className="text-base">{group}</CardTitle>
                  <p className="text-sm text-muted-foreground">
                    {groupPages.length} fixed page state{groupPages.length === 1 ? "" : "s"}.
                  </p>
                </CardHeader>
                <CardContent className="grid gap-5 lg:grid-cols-2">
                  {groupPages.map((page) => (
                    <div key={page.id} className="space-y-4 rounded-md border p-4">
                      <div>
                        <p className="text-sm font-medium">{page.label}</p>
                        <p className="text-xs text-muted-foreground">Default values are shown until edited.</p>
                      </div>
                      <FormInputField
                        label="Title"
                        value={page.title}
                        onChange={(event) => updateCopy(page.id, "title", event.target.value)}
                        disabled={isSaving}
                      />
                      <FormTextareaField
                        label="Supporting text"
                        value={page.subtitle}
                        onChange={(event) => updateCopy(page.id, "subtitle", event.target.value)}
                        disabled={isSaving}
                        rows={3}
                      />
                    </div>
                  ))}
                </CardContent>
              </Card>
            )
          })}

          <div className="flex justify-end gap-3">
            <Button type="button" variant="outline" onClick={() => guard(() => navigate(backTo))} disabled={isSaving}>
              Cancel
            </Button>
            <Button type="button" variant="outline" disabled={!isDirty || isSaving} onClick={resetCopy}>
              Reset
            </Button>
            <Button type="submit" disabled={!isDirty || isSaving} className="gap-2">
              {isSaving && <Loader2 className="size-4 animate-spin" />}
              {isSaving ? "Saving..." : "Save Forms"}
            </Button>
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
