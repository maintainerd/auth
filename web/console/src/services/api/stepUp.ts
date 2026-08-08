/**
 * Step-up bridge.
 *
 * Decouples the (non-React) axios interceptor from the React step-up dialog.
 * The dialog provider registers a handler on mount; the interceptor calls
 * `requestStepUp()` when the backend answers a sensitive request with
 * 403 `step_up_required`. The handler runs the challenge/verify ceremony and
 * resolves with an elevated (acr=2) access token, which the interceptor then
 * attaches as a Bearer header to retry the original request.
 *
 * The in-flight ceremony is single-flighted: concurrent gated requests share
 * one prompt instead of stacking dialogs.
 */

/** Resolves with an elevated (acr=2) access token, or rejects if cancelled. */
export type StepUpHandler = () => Promise<string>

let handler: StepUpHandler | null = null
let pending: Promise<string> | null = null

/**
 * Registers the active step-up handler (called by StepUpProvider on mount).
 * Returns an unregister function for cleanup on unmount.
 */
export function registerStepUpHandler(h: StepUpHandler): () => void {
  handler = h
  return () => {
    if (handler === h) handler = null
  }
}

/**
 * Requests an elevated access token, prompting the user if needed. Concurrent
 * callers share a single ceremony. Rejects if no handler is mounted or the
 * user cancels.
 */
export function requestStepUp(): Promise<string> {
  if (!handler) {
    return Promise.reject(new Error("Step-up authentication is not available"))
  }
  if (!pending) {
    pending = handler().finally(() => {
      pending = null
    })
  }
  return pending
}
