import { useEffect, useRef, useState } from "react"
import { registerStepUpHandler } from "@/services/api/stepUp"
import { StepUpDialog } from "./StepUpDialog"

/**
 * Mounts the global step-up dialog and wires it to the axios step-up bridge.
 *
 * When any request is answered with 403 `step_up_required`, the interceptor
 * calls the registered handler, which opens this dialog. On success the elevated
 * (acr=2) token resolves the handler promise; the interceptor then retries the
 * original request. Cancelling rejects the promise so the gated request fails
 * cleanly. Mount once, near the app root, inside the query + auth providers.
 */
export function StepUpProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false)
  // The promise callbacks for the in-flight ceremony. Held in a ref so the
  // dialog handlers can settle exactly one request without re-renders racing.
  const resolverRef = useRef<{ resolve: (token: string) => void; reject: (err: Error) => void } | null>(null)

  useEffect(() => {
    return registerStepUpHandler(
      () =>
        new Promise<string>((resolve, reject) => {
          resolverRef.current = { resolve, reject }
          setOpen(true)
        }),
    )
  }, [])

  const handleVerified = (token: string) => {
    resolverRef.current?.resolve(token)
    resolverRef.current = null
  }

  const handleCancel = () => {
    resolverRef.current?.reject(new Error("Step-up authentication was cancelled"))
    resolverRef.current = null
  }

  return (
    <>
      {children}
      <StepUpDialog
        open={open}
        onOpenChange={setOpen}
        onVerified={handleVerified}
        onCancel={handleCancel}
      />
    </>
  )
}
