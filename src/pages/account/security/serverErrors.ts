/**
 * Server-error matching for the identifier-change forms.
 *
 * Both uniqueness checks in accountService return apperror.NewValidation(...),
 * and HandleServiceError maps a ValidationError to 400 — not the 409 these
 * pages used to branch on. The field-level message ("That email address is
 * already in use.") therefore never rendered and the user got only a generic
 * toast for a problem they could fix in the input right in front of them.
 */

/** The message the backend actually sent, preferring its response body. */
export function serverErrorMessage(err: unknown): string {
  const responseError = (err as { responseData?: { error?: string | object } }).responseData?.error
  if (typeof responseError === 'string' && responseError) return responseError
  return err instanceof Error ? err.message : ''
}

/**
 * True when the server rejected the value because it is already taken.
 *
 * Keyed off the server's own wording rather than the bare status, and accepting
 * 409 as well as 400, so this keeps working if the backend reclassifies these
 * as true conflicts.
 */
export function isAlreadyTakenError(err: unknown, wording: RegExp): boolean {
  const status = (err as { status?: number }).status
  if (status !== 400 && status !== 409) return false
  return wording.test(serverErrorMessage(err))
}
