import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { SettingsCard } from '@/components/card'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { FormInputField } from '@/components/form'
import { FormUrlField } from '@/components/inputs'
import { useToast } from '@/hooks/useToast'
import { fetchProfiles, createProfile, updateProfile, type UserProfile } from '@/services/api/account'
import { buildProfilePayload, validateNotCleared, type ProfileFormValues } from './profilePayload'

// Mirrors the backend RuneLength rules on user.ProfileRequestDTO so the user is
// told in place instead of receiving a 400 that lists fields this form has no
// visible requirement for.
const NAME_MAX_LENGTH = 100
const PROFILE_URL_MAX_LENGTH = 1000

// is.URL server-side. Requiring an absolute http(s) URL is the narrower rule:
// anything that passes here passes there, and "https://…" is what the field's
// placeholder already asks for.
function validateProfileUrl(value: string): string | true {
  if (!value.trim()) return true
  let parsed: URL
  try {
    parsed = new URL(value.trim())
  } catch {
    return 'Enter a full URL, starting with https://'
  }
  return parsed.protocol === 'http:' || parsed.protocol === 'https:'
    ? true
    : 'Enter a full URL, starting with https://'
}

export default function ProfileFormPage() {
  const { profileId } = useParams<{ profileId: string }>()
  const isEdit = Boolean(profileId)
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { showError, showSuccess } = useToast()

  const { data: profiles = [], isLoading } = useQuery({
    queryKey: ['account', 'profiles'],
    queryFn: fetchProfiles,
    enabled: isEdit,
  })
  const editing = isEdit ? profiles.find((p: UserProfile) => p.profile_id === profileId) : undefined

  const form = useForm<ProfileFormValues>({
    defaultValues: { display_name: '', first_name: '', last_name: '', profile_url: '' },
  })

  useEffect(() => {
    if (editing) {
      form.reset({
        display_name: editing.display_name ?? '',
        first_name: editing.first_name ?? '',
        last_name: editing.last_name ?? '',
        profile_url: editing.profile_url ?? '',
      })
    }
  }, [editing, form])

  const invalidate = () => qc.invalidateQueries({ queryKey: ['account', 'profiles'] })

  const createMutation = useMutation({
    mutationFn: (data: ProfileFormValues) => createProfile(buildProfilePayload(data)),
    onSuccess: () => { showSuccess('Profile created'); invalidate(); navigate('/account/profile') },
    onError: (err) => showError(err, 'Could not create profile'),
  })

  const updateMutation = useMutation({
    // Only the fields the user touched are sent, so a concurrent edit from
    // another device is not reverted — see buildProfilePayload.
    mutationFn: (data: ProfileFormValues) =>
      updateProfile(profileId!, buildProfilePayload(data, form.formState.dirtyFields)),
    onSuccess: () => { showSuccess('Profile updated'); invalidate(); navigate('/account/profile') },
    onError: (err) => showError(err, 'Could not update profile'),
  })

  const onSubmit = form.handleSubmit((data) => {
    if (isEdit) updateMutation.mutate(data)
    else createMutation.mutate(data)
  })

  // dirtyFields is measured against the values form.reset() loaded, so an edit
  // submitted before the profile arrives would mark every field changed and
  // overwrite the stored values with the empty defaults.
  const pending = createMutation.isPending || updateMutation.isPending
  const waitingForProfile = isEdit && !editing

  return (
    <AccountLayout title="Profile">
      <Link
        to="/account/profile"
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> Back to profiles
      </Link>

      {isEdit && !isLoading && !editing ? (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            Profile not found.
          </CardContent>
        </Card>
      ) : (
        <SettingsCard
          title={isEdit ? 'Edit profile' : 'New profile'}
          description="Keep your display details consistent across the identity experience."
          contentClassName="space-y-6"
        >
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <FormInputField
                label="Display name"
                placeholder="Display name"
                containerClassName="sm:col-span-2"
                error={form.formState.errors.display_name?.message}
                {...form.register('display_name', {
                  validate: validateNotCleared('Display name', editing?.display_name),
                  maxLength: { value: NAME_MAX_LENGTH, message: `Display name must be at most ${NAME_MAX_LENGTH} characters.` },
                })}
              />
              <FormInputField
                label="First name"
                placeholder="First name"
                required
                error={form.formState.errors.first_name?.message}
                {...form.register('first_name', {
                  // Required by the server; the form never said so, so the rule
                  // only ever arrived as a 400.
                  validate: (value) => value.trim().length > 0 || 'First name is required.',
                  maxLength: { value: NAME_MAX_LENGTH, message: `First name must be at most ${NAME_MAX_LENGTH} characters.` },
                })}
              />
              <FormInputField
                label="Last name"
                placeholder="Last name"
                error={form.formState.errors.last_name?.message}
                {...form.register('last_name', {
                  validate: validateNotCleared('Last name', editing?.last_name),
                  maxLength: { value: NAME_MAX_LENGTH, message: `Last name must be at most ${NAME_MAX_LENGTH} characters.` },
                })}
              />
              <FormUrlField
                label="Avatar URL"
                placeholder="https://..."
                containerClassName="sm:col-span-2"
                error={form.formState.errors.profile_url?.message}
                {...form.register('profile_url', {
                  validate: {
                    notCleared: validateNotCleared('Avatar URL', editing?.profile_url),
                    httpUrl: validateProfileUrl,
                  },
                  maxLength: { value: PROFILE_URL_MAX_LENGTH, message: `Avatar URL must be at most ${PROFILE_URL_MAX_LENGTH} characters.` },
                })}
              />
            </div>
            {/* Column-reverse on mobile puts the primary action under the thumb,
                matching the security forms. */}
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <Button
                type="button"
                variant="outline"
                className="w-full sm:w-auto"
                onClick={() => navigate('/account/profile')}
              >
                Cancel
              </Button>
              <Button type="submit" className="w-full sm:w-auto" disabled={pending || waitingForProfile}>
                {pending ? 'Saving…' : isEdit ? 'Save changes' : 'Create profile'}
              </Button>
            </div>
          </form>
        </SettingsCard>
      )}
    </AccountLayout>
  )
}
