import { Controller, useForm } from 'react-hook-form'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Globe } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { SettingsCard } from '@/components/card'
import { Button } from '@/components/ui/button'
import { FormSelectField } from '@/components/form'
import { timezoneOptions, languageOptions } from '@/lib/constants'
import { useToast } from '@/hooks/useToast'
import { fetchUserSettings, updateUserSettings, type UserSettings } from '@/services/api/account'

interface PreferencesForm {
  language: string
  timezone: string
}

function toForm(settings: UserSettings): PreferencesForm {
  return {
    language: settings.language ?? '',
    timezone: settings.timezone ?? '',
  }
}

export default function AccountSettingsPage() {
  const { data: settings, isLoading } = useQuery({
    queryKey: ['account', 'settings'],
    queryFn: fetchUserSettings,
  })

  if (isLoading) {
    return (
      <AccountLayout title="Preferences">
        <p className="text-sm text-muted-foreground">Loading preferences…</p>
      </AccountLayout>
    )
  }

  return <PreferencesFormInner key={`${settings?.language ?? ''}-${settings?.timezone ?? ''}`} initial={settings ?? {}} />
}

function PreferencesFormInner({ initial }: { initial: UserSettings }) {
  const queryClient = useQueryClient()
  const { showError, showSuccess } = useToast()
  const { control, handleSubmit, reset, formState: { isDirty } } = useForm<PreferencesForm>({
    defaultValues: toForm(initial),
  })

  const saveMutation = useMutation({
    mutationFn: (data: PreferencesForm) => {
      const optional = (value: string) => (value.trim() ? value.trim() : undefined)
      return updateUserSettings({
        language: optional(data.language),
        timezone: optional(data.timezone),
      })
    },
    onSuccess: (_result, variables) => {
      reset(variables)
      showSuccess('Preferences saved')
      queryClient.invalidateQueries({ queryKey: ['account', 'settings'] })
    },
    onError: (err) => showError(err, 'Could not save preferences'),
  })

  return (
    <AccountLayout title="Preferences">
      <form onSubmit={handleSubmit((data) => saveMutation.mutate(data))} className="grid gap-6">
        <SettingsCard
          title="Localization"
          description="Your timezone and language preferences."
          icon={Globe}
        >
          <div className="grid gap-4 sm:grid-cols-2">
            <Controller
              control={control}
              name="timezone"
              render={({ field }) => (
                <FormSelectField
                  label="Timezone"
                  placeholder="Select timezone"
                  options={timezoneOptions}
                  value={field.value}
                  onValueChange={field.onChange}
                />
              )}
            />
            <Controller
              control={control}
              name="language"
              render={({ field }) => (
                <FormSelectField
                  label="Language"
                  placeholder="Select language"
                  options={languageOptions}
                  value={field.value}
                  onValueChange={field.onChange}
                />
              )}
            />
          </div>
        </SettingsCard>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" disabled={!isDirty || saveMutation.isPending} onClick={() => reset()}>
            Cancel
          </Button>
          <Button type="submit" disabled={!isDirty || saveMutation.isPending}>
            {saveMutation.isPending ? 'Saving…' : 'Save preferences'}
          </Button>
        </div>
      </form>
    </AccountLayout>
  )
}
