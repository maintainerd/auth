import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { useForm, type Resolver } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import * as yup from "yup"
import { AlertTriangle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { DetailsContainer } from "@/components/container"
import { FormPageHeader } from "@/components/header"
import { FormSelectField, FormSubmitButton, type SelectOption } from "@/components/form"
import { FormEmailField } from "@/components/inputs"
import { useSendInvite } from "@/hooks/useInvites"
import { useRegistrationFlows } from "@/hooks/useRegistrationFlows"
import { useClientUris } from "@/hooks/useClients"
import { useEmailConfigStatus } from "@/hooks/useNotifier"
import { useToast } from "@/hooks/useToast"
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard"
import { ConfirmationDialog } from "@/components/dialog"

const inviteSchema = yup.object({
  email: yup.string().trim().required("Email is required").email("Enter a valid email address"),
})

type InviteFormData = { email: string }

const NO_FLOW = "__none__"

export default function InviteForm() {
  const navigate = useNavigate()
  const { showSuccess, showError, parseError } = useToast()
  const listUrl = `/invites`

  const sendMutation = useSendInvite()
  const [registrationFlowId, setRegistrationFlowId] = useState<string>(NO_FLOW)
  const [callbackUrl, setCallbackUrl] = useState<string>("")

  const { data: emailStatus } = useEmailConfigStatus()
  const emailNotConfigured = emailStatus ? !emailStatus.configured : false

  const { data: flowsData } = useRegistrationFlows({ page: 1, limit: 100, sort_by: "name", sort_order: "asc", status: "active" })
  const flowOptions: SelectOption[] = [
    { value: NO_FLOW, label: "Default registration (no registration flow)" },
    ...(flowsData?.rows ?? []).map((f) => ({ value: f.registration_flow_id, label: f.name })),
  ]

  // A custom flow's callback must be one of its client's registered redirect
  // URIs (a dropdown, not free text — the backend enforces this too). No-flow
  // invites have no callback: the user lands on the normal post-registration
  // destination (profile if auto-authenticated, else the login/success page).
  const selectedFlow =
    registrationFlowId !== NO_FLOW
      ? (flowsData?.rows ?? []).find((f) => f.registration_flow_id === registrationFlowId)
      : undefined
  const { data: clientUris } = useClientUris(selectedFlow?.client_id ?? "")
  const redirectUriOptions: SelectOption[] = (clientUris?.uris ?? [])
    .filter((u) => u.type === "redirect_uri")
    .map((u) => ({ value: u.uri, label: u.uri }))

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<InviteFormData>({
    resolver: yupResolver(inviteSchema) as unknown as Resolver<InviteFormData>,
    defaultValues: { email: "" },
    mode: "onTouched",
    reValidateMode: "onChange",
  })

  const isLoading = sendMutation.isPending || isSubmitting

  const { guard, isPromptOpen, confirmLeave, cancelLeave } = useUnsavedChangesGuard(isDirty)

  const onSubmit = async (data: InviteFormData) => {
    try {
      await sendMutation.mutateAsync({
        email: data.email.trim(),
        registration_flow_uuid: registrationFlowId !== NO_FLOW ? registrationFlowId : undefined,
        callback_url: registrationFlowId !== NO_FLOW ? callbackUrl || undefined : undefined,
      })
      showSuccess(`Invitation sent to ${data.email.trim()}`)
      navigate(listUrl)
    } catch (error) {
      const parsed = parseError(error)
      const known = ["email"] as const
      let mappedToField = false
      if (parsed.fieldErrors) {
        for (const [field, message] of Object.entries(parsed.fieldErrors)) {
          if ((known as readonly string[]).includes(field)) {
            setError(field as (typeof known)[number], { type: "server", message })
            mappedToField = true
          }
        }
      }
      if (!mappedToField) {
        const lower = parsed.message.toLowerCase()
        const field = known.find((f) => lower.includes(f))
        if (field) {
          setError(field, { type: "server", message: parsed.message })
        }
      }
      showError(error)
    }
  }

  return (
    <DetailsContainer>
      <div className="flex flex-col gap-6">
        <FormPageHeader
          backUrl={listUrl}
          backLabel="Back to Invitations"
          onBack={() => guard(() => navigate(listUrl))}
          title="Invite User"
          description="Send a signed email invitation. The recipient registers under this email — they can't change it."
        />

        {emailNotConfigured && (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Email delivery is not configured</AlertTitle>
            <AlertDescription className="flex flex-col items-start gap-2">
              <span>
                Invitations are delivered by email. Configure an email provider so the invitation
                link can be sent.
              </span>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => navigate(`/messaging?tab=email`)}
              >
                Configure email delivery
              </Button>
            </AlertDescription>
          </Alert>
        )}

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Invitation</CardTitle>
              <p className="text-sm text-muted-foreground">Who to invite and how they onboard.</p>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormEmailField
                label="Email address"
                placeholder="person@example.com"
                required
                disabled={isLoading}
                error={errors.email?.message}
                {...register("email")}
              />

              <FormSelectField
                label="Registration flow"
                placeholder="Select registration flow"
                options={flowOptions}
                value={registrationFlowId}
                onValueChange={(v) => { setRegistrationFlowId(v); setCallbackUrl("") }}
                disabled={isLoading}
                description="Optional — default registration assigns only the registered role. Selecting a flow adds that flow's additional roles on completion."
              />

              {registrationFlowId !== NO_FLOW && (
                <FormSelectField
                  label="Post-registration callback URL"
                  placeholder={redirectUriOptions.length ? "Select a callback URL" : "No redirect URIs registered on this client"}
                  options={redirectUriOptions}
                  value={callbackUrl}
                  onValueChange={setCallbackUrl}
                  disabled={isLoading || redirectUriOptions.length === 0}
                  description="Where the user is redirected after registering — a redirect URI registered on the flow's client."
                />
              )}
            </CardContent>
          </Card>

          <div className="flex justify-end gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={() => guard(() => navigate(listUrl))}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <FormSubmitButton isSubmitting={isLoading} submitText="Send Invite" />
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
