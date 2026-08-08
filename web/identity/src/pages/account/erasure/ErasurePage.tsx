import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { Eraser, CheckCircle2, AlertCircle, ArrowLeft, ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import LoginLayout from '@/components/layout/LoginLayout'
import { requestDataErasure } from '@/services/api/auth'

export default function ErasurePage() {
  const [confirmed, setConfirmed] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)

  const mutation = useMutation({
    mutationFn: requestDataErasure,
    onSuccess: () => setConfirmed(true),
  })

  if (confirmed) {
    return (
      <LoginLayout>
        <div className="container mx-auto max-w-lg py-12 px-4 text-center">
          <CheckCircle2 className="h-12 w-12 mx-auto mb-4 text-emerald-600" />
          <h1 className="text-2xl font-bold mb-3">Request submitted</h1>
          <p className="text-muted-foreground mb-6">
            Your data will be anonymised within 30 days. You will continue to have access to your
            account until then.
          </p>
          <Link
            to="/login-success"
            className="text-sm text-muted-foreground hover:text-foreground inline-flex items-center gap-1"
          >
            <ArrowLeft className="h-3 w-3" /> Back to account
          </Link>
        </div>
      </LoginLayout>
    )
  }

  return (
    <LoginLayout>
      <div className="container mx-auto max-w-lg py-12 px-4">
        <Link
          to="/login-success"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-8"
        >
          <ArrowLeft className="h-3 w-3" /> Back
        </Link>

        <div className="flex justify-center mb-6">
          <div className="flex size-14 items-center justify-center rounded-xl bg-red-50 text-red-600">
            <Eraser className="h-7 w-7" />
          </div>
        </div>

        <h1 className="text-2xl font-bold text-center mb-2">Request account deletion</h1>
        <p className="text-center text-muted-foreground mb-8">
          Under GDPR Article 17 you have the right to have your personal data erased.
        </p>

        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 mb-6 flex gap-3">
          <ShieldAlert className="h-5 w-5 text-amber-600 shrink-0 mt-0.5" />
          <div className="text-sm text-amber-800 space-y-1">
            <p className="font-medium">Before you continue</p>
            <ul className="list-disc list-inside space-y-0.5 text-amber-700">
              <li>All your personal data will be permanently anonymised</li>
              <li>Your email, name, and profile info will be erased</li>
              <li>This process begins after a 30-day window</li>
              <li>This action cannot be undone</li>
            </ul>
          </div>
        </div>

        {mutation.isError && (
          <div className="flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive mb-4">
            <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
            <span>
              {mutation.error instanceof Error
                ? mutation.error.message
                : 'Request failed. Please try again.'}
            </span>
          </div>
        )}

        {!showConfirm ? (
          <Button
            variant="destructive"
            className="w-full"
            onClick={() => setShowConfirm(true)}
          >
            <Eraser className="h-4 w-4 mr-2" />
            Request data deletion
          </Button>
        ) : (
          <div className="space-y-3">
            <p className="text-center text-sm font-medium text-destructive">
              Are you sure? This cannot be undone.
            </p>
            <Button
              variant="destructive"
              className="w-full"
              onClick={() => mutation.mutate()}
              disabled={mutation.isPending}
            >
              {mutation.isPending ? 'Submitting…' : 'Yes, delete my data'}
            </Button>
            <Button
              variant="outline"
              className="w-full"
              onClick={() => setShowConfirm(false)}
              disabled={mutation.isPending}
            >
              Cancel
            </Button>
          </div>
        )}
      </div>
    </LoginLayout>
  )
}
