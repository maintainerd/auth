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

interface ProfileForm {
  display_name: string
  first_name: string
  last_name: string
  profile_url: string
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

  const form = useForm<ProfileForm>({
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
    mutationFn: (data: ProfileForm) => createProfile(data),
    onSuccess: () => { showSuccess('Profile created'); invalidate(); navigate('/account/profile') },
    onError: (err) => showError(err, 'Could not create profile'),
  })

  const updateMutation = useMutation({
    mutationFn: (data: ProfileForm) => updateProfile(profileId!, data),
    onSuccess: () => { showSuccess('Profile updated'); invalidate(); navigate('/account/profile') },
    onError: (err) => showError(err, 'Could not update profile'),
  })

  const onSubmit = form.handleSubmit((data) => {
    if (isEdit) updateMutation.mutate(data)
    else createMutation.mutate(data)
  })

  const pending = createMutation.isPending || updateMutation.isPending

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
                {...form.register('display_name')}
              />
              <FormInputField
                label="First name"
                placeholder="First name"
                {...form.register('first_name')}
              />
              <FormInputField
                label="Last name"
                placeholder="Last name"
                {...form.register('last_name')}
              />
              <FormUrlField
                label="Avatar URL"
                placeholder="https://..."
                containerClassName="sm:col-span-2"
                {...form.register('profile_url')}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => navigate('/account/profile')}>
                Cancel
              </Button>
              <Button type="submit" disabled={pending}>
                {pending ? 'Saving…' : isEdit ? 'Save changes' : 'Create profile'}
              </Button>
            </div>
          </form>
        </SettingsCard>
      )}
    </AccountLayout>
  )
}
