import { useCallback, useEffect, useState } from "react"

/**
 * Guards against losing unsaved form changes. Two layers:
 *
 *  1. A `beforeunload` prompt for browser refresh / tab close / navigating to an
 *     external URL — active only while `isDirty`.
 *  2. `guard(action)` wraps an in-app navigation (Cancel, Back, etc.): when the
 *     form is dirty it opens a confirmation and runs the action only once the
 *     user confirms; otherwise it runs immediately.
 *
 * Pair it with a `<ConfirmationDialog>` driven by `isPromptOpen` / `confirmLeave`
 * / `cancelLeave`.
 *
 * Note: the app uses a plain `<BrowserRouter>` (not a data router), so in-app
 * navigations the form does not route through `guard` (e.g. sidebar links)
 * cannot be intercepted; the `beforeunload` layer still covers hard reloads and
 * tab closes.
 */
export function useUnsavedChangesGuard(isDirty: boolean) {
  // Wrap the pending navigation in an object so setState never treats it as a
  // functional updater (which would invoke it instead of storing it).
  const [pendingAction, setPendingAction] = useState<{ run: () => void } | null>(null)

  useEffect(() => {
    if (!isDirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
      e.returnValue = ""
    }
    window.addEventListener("beforeunload", handler)
    return () => window.removeEventListener("beforeunload", handler)
  }, [isDirty])

  const guard = useCallback(
    (action: () => void) => {
      if (isDirty) setPendingAction({ run: action })
      else action()
    },
    [isDirty],
  )

  const confirmLeave = useCallback(() => {
    const action = pendingAction?.run
    setPendingAction(null)
    action?.()
  }, [pendingAction])

  const cancelLeave = useCallback(() => setPendingAction(null), [])

  return {
    /** Wrap a navigation callback; prompts when dirty, runs immediately otherwise. */
    guard,
    /** Whether the discard-changes confirmation should be shown. */
    isPromptOpen: pendingAction !== null,
    /** Proceed with the pending navigation (discard changes). */
    confirmLeave,
    /** Dismiss the prompt and stay on the form. */
    cancelLeave,
  }
}
