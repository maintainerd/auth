/**
 * Throttle / rate-limit detection for end-user facing forms.
 *
 * The backend meters these two ways and the SPA has to read both:
 *
 *  - The /account endpoints that verify current_password share one per-account
 *    counter and answer **429** ("Too many attempts. Please try again later." —
 *    checkAccountPasswordThrottle in internal/user/handler_account.go).
 *  - /recovery/backup-code raises apperror.NewForbidden("too many recovery
 *    attempts; try again later"), which HandleServiceError maps to **403**.
 *
 * Branching on the status alone renders the second one as "You don't have
 * permission to perform this action." — wrong, and it tells the user nothing
 * they can act on. Matching the server's wording as well as the status covers
 * both shapes, and keeps working unchanged once the 403 is reclassified as a
 * 429.
 */

const THROTTLE_WORDING = /\btoo many\b.*\b(attempts?|requests?)\b/i

function statusOf(err: unknown): number | undefined {
  return (err as { status?: number } | null)?.status
}

function messageOf(err: unknown): string {
  const responseError = (err as { responseData?: { error?: string | object } } | null)?.responseData?.error
  if (typeof responseError === 'string' && responseError) return responseError
  return err instanceof Error ? err.message : ''
}

/** True when the server refused because the caller has attempted too often. */
export function isRateLimitError(err: unknown): boolean {
  const status = statusOf(err)
  if (status === 429) return true
  // 403 only when the server said why — a real permission denial is also a 403
  // and must keep its own message.
  return status === 403 && THROTTLE_WORDING.test(messageOf(err))
}

/**
 * An actionable sentence for a throttled request.
 *
 * The server's own strings are terse and inconsistently cased ("too many
 * recovery attempts; try again later"), and the useful part — how long to wait —
 * only exists in the `Retry-After` header the client already parsed. This states
 * the wait when it is known and otherwise says the attempt was not lost.
 */
export function rateLimitMessage(err: unknown): string {
  const retryAfter = (err as { retryAfter?: number } | null)?.retryAfter
  if (typeof retryAfter === 'number' && retryAfter > 0) {
    const unit = retryAfter === 1 ? 'second' : 'seconds'
    return `Too many attempts. Wait ${retryAfter} ${unit} and try again.`
  }
  return 'Too many attempts. Wait a short while and try again — nothing was changed.'
}
