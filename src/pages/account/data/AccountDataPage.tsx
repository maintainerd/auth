import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Download, Trash2 } from 'lucide-react'
import { Link } from 'react-router-dom'
import AccountLayout from '@/components/layout/AccountLayout'
import { SettingsCard } from '@/components/card'
import { SettingsActionRow } from '@/components/settings'
import { Button } from '@/components/ui/button'
import { useToast } from '@/hooks/useToast'
import { requestDataExport } from '@/services/api/account'

export default function AccountDataPage() {
  const { showError } = useToast()
  const [exportResult, setExportResult] = useState<{ download_url?: string; message?: string } | null>(null)

  const exportMutation = useMutation({
    mutationFn: requestDataExport,
    onSuccess: (result) => setExportResult(result),
    onError: (err) => showError(err, 'Could not request data export'),
  })

  return (
    <AccountLayout title="Data & Privacy">
      <div className="grid gap-6">
        <SettingsCard
          title="Export your data"
          description="Download a copy of your account data."
          icon={Download}
          contentClassName="space-y-2"
        >
          {exportResult ? (
            <div className="rounded-md border border-emerald-500/30 bg-emerald-500/10 p-3">
              {exportResult.download_url ? (
                <a
                  href={exportResult.download_url}
                  className="text-sm font-medium text-emerald-700 underline dark:text-emerald-400"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Download your export
                </a>
              ) : (
                <p className="text-sm text-emerald-700 dark:text-emerald-400">
                  {exportResult.message ?? 'Your export is being prepared. You will receive an email when it is ready.'}
                </p>
              )}
            </div>
          ) : (
            <SettingsActionRow
              icon={Download}
              title="Personal data export"
              description="Download a copy of your profile, sessions, and activity."
              action={(
              <Button
                variant="outline"
                onClick={() => exportMutation.mutate()}
                disabled={exportMutation.isPending}
              >
                {exportMutation.isPending ? 'Requesting…' : 'Request export'}
              </Button>
              )}
            />
          )}
        </SettingsCard>

        <SettingsCard
          title="Delete account"
          description="Irreversible actions for your account."
          icon={Trash2}
          className="border-destructive/30"
          contentClassName="space-y-2"
        >
          <SettingsActionRow
            icon={Trash2}
            title="Delete account"
            description="Permanently delete your account and all associated data."
            action={(
              <Button variant="destructive" asChild>
                <Link to="/account/erasure">Delete my account</Link>
              </Button>
            )}
          />
        </SettingsCard>
      </div>
    </AccountLayout>
  )
}
