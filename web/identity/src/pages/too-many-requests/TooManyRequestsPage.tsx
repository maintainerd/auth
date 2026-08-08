import { useEffect, useState } from "react"
import { Link, useLocation } from "react-router-dom"
import { AlertTriangle } from "lucide-react"
import { Button } from "@/components/ui/button"
import LoginLayout from "@/components/layout/LoginLayout"

export default function TooManyRequestsPage() {
  // The axios interceptor forwards the parsed `Retry-After` value (seconds) via
  // navigation state when the backend supplies one; we count it down so the user
  // knows exactly when they can try again.
  const location = useLocation()
  const initial = (location.state as { retryAfter?: number } | null)?.retryAfter
  const [remaining, setRemaining] = useState<number | undefined>(initial)

  useEffect(() => {
    if (remaining === undefined || remaining <= 0) return
    const timer = setInterval(() => {
      setRemaining((prev) => (prev === undefined ? prev : Math.max(0, prev - 1)))
    }, 1000)
    return () => clearInterval(timer)
  }, [remaining])

  return (
    <LoginLayout>
      <div className="flex flex-col items-center gap-6 text-center w-full max-w-sm">
        <div className="flex size-16 items-center justify-center rounded-full bg-red-50">
          <AlertTriangle className="size-8 text-red-600" />
        </div>
        <div className="space-y-2">
          <h1 className="text-2xl font-bold">Too Many Requests</h1>
          <p className="text-muted-foreground">
            You&apos;ve made too many attempts in a short period.
            {remaining !== undefined && remaining > 0
              ? ` Please wait ${remaining} second${remaining === 1 ? "" : "s"} before trying again.`
              : " Please wait a moment before trying again."}
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link to="/login">Back to Login</Link>
        </Button>
      </div>
    </LoginLayout>
  )
}
