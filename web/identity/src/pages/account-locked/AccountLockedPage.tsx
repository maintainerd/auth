import { Link } from "react-router-dom"
import { Lock } from "lucide-react"
import { Button } from "@/components/ui/button"
import LoginLayout from "@/components/layout/LoginLayout"

export default function AccountLockedPage() {
  return (
    <LoginLayout>
      <div className="flex flex-col items-center gap-6 text-center w-full max-w-sm">
        <div className="flex size-16 items-center justify-center rounded-full bg-amber-50">
          <Lock className="size-8 text-amber-600" />
        </div>
        <div className="space-y-2">
          <h1 className="text-2xl font-bold">Account Temporarily Locked</h1>
          <p className="text-muted-foreground">
            Your account has been locked due to too many failed login attempts.
            Please wait a few minutes and try again.
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link to="/login">Back to Login</Link>
        </Button>
        <p className="text-sm text-muted-foreground">
          If you&apos;re locked out, you can use the{" "}
          <Link to="/recovery" className="underline">backup code recovery</Link> option.
        </p>
      </div>
    </LoginLayout>
  )
}
