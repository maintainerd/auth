import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Download, Trash2 } from 'lucide-react'
import { Link } from 'react-router-dom'
import AccountLayout from '@/components/layout/AccountLayout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
      <div className="space-y-4">
        {/* Export */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <Download className="size-4" />
              Export your data
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Download a copy of your personal data including your profile, sessions, and activity.
            </p>
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
              <Button
                variant="outline"
                onClick={() => exportMutation.mutate()}
                disabled={exportMutation.isPending}
              >
                Request export
              </Button>
            )}
          </CardContent>
        </Card>

        {/* Delete */}
        <Card className="border-destructive/30">
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base text-destructive">
              <Trash2 className="size-4" />
              Delete account
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Permanently delete your account and all associated data. This action cannot be undone.
            </p>
            <Button variant="destructive" asChild>
              <Link to="/account/erasure">Delete my account</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </AccountLayout>
  )
}
