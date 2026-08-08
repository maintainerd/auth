import type { ReactNode } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { SettingsCard } from '@/components/card'
import { Button } from '@/components/ui/button'

const SECURITY_ROUTE = '/account/security'

/**
 * Shared scaffold for the dedicated security forms (email, username, password).
 *
 * These were modal dialogs. A credential change is a deliberate, focused task —
 * it deserves its own URL so it can be linked to, navigated back from, and
 * reloaded without losing state. Dialogs also cramp the password form, which
 * carries a live policy checklist, and on a phone a modal that outgrows the
 * viewport is genuinely hard to use.
 *
 * Extracted rather than repeated three times so the back link, card chrome,
 * button order and responsive behaviour cannot drift between them.
 */
export function SecurityFormPage({
  title,
  description,
  onSubmit,
  submitLabel,
  pendingLabel,
  pending = false,
  submitDisabled = false,
  hideSubmit = false,
  cancelLabel = 'Cancel',
  alert,
  children,
  footer,
}: {
  title: string
  description: string
  onSubmit: React.FormEventHandler<HTMLFormElement>
  submitLabel: string
  pendingLabel: string
  pending?: boolean
  submitDisabled?: boolean
  /** Hides the submit button once the form has reached a terminal state. */
  hideSubmit?: boolean
  cancelLabel?: string
  /**
   * Form-level failure that belongs to no single input — a throttle, say. Kept
   * out of the toast stack because it has to stay on screen while the user waits
   * it out, and off a field because retyping one is not what fixes it.
   */
  alert?: ReactNode
  children: ReactNode
  /** Secondary content between the fields and the actions (e.g. a reset link). */
  footer?: ReactNode
}) {
  const navigate = useNavigate()

  return (
    <AccountLayout title="Security">
      <Link
        to={SECURITY_ROUTE}
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> Back to security
      </Link>

      <SettingsCard title={title} description={description} contentClassName="space-y-6">
        {/* Fills the card, matching ProfileFormPage. A max-width here left the
            fields floating in a half-empty card on desktop and made these pages
            look unlike every other form in the account area. */}
        <form onSubmit={onSubmit} className="w-full space-y-4">
          {alert && (
            <p
              role="alert"
              className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
            >
              {alert}
            </p>
          )}
          {children}
          {footer}
          {/* Column-reverse on mobile puts the primary action at the bottom —
              under the thumb, and in the reading order the eye lands on last. */}
          <div className="flex flex-col-reverse gap-2 pt-2 sm:flex-row sm:justify-end">
            <Button
              type="button"
              variant="outline"
              className="w-full sm:w-auto"
              onClick={() => navigate(SECURITY_ROUTE)}
            >
              {cancelLabel}
            </Button>
            {!hideSubmit && (
              <Button type="submit" className="w-full sm:w-auto" disabled={pending || submitDisabled}>
                {pending ? pendingLabel : submitLabel}
              </Button>
            )}
          </div>
        </form>
      </SettingsCard>
    </AccountLayout>
  )
}

export { SECURITY_ROUTE }
