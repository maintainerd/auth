import { useCallback, useEffect } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { CheckCircle, ArrowRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useAuth } from "@/hooks/useAuth"
import { useTenant } from "@/hooks/useTenant"
import { getRequestId } from "@/utils/oauthRedirect"
import { finishAuthStep } from "@/utils/oauthContinuation"

const RegisterProfileSuccess = () => {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { account } = useAuth()
  const { currentTenant } = useTenant()

  // Profile creation was the last registration step, so the account is now fully
  // registered. Apply the single shared continuation rule: resume the pending
  // OAuth authorize (request_id) via login-success, or land on the dashboard.
  const proceed = useCallback(() => {
    finishAuthStep({
      account,
      tenant: currentTenant,
      requestId: getRequestId(searchParams),
      navigate,
    })
  }, [account, currentTenant, searchParams, navigate])

  useEffect(() => {
    const timer = setTimeout(proceed, 5000)
    return () => clearTimeout(timer)
  }, [proceed])

  const handleContinue = () => {
    proceed()
  }

  return (
    <div className="flex flex-col gap-8 text-center">
      <div className="flex flex-col items-center gap-3">
        <div className="flex size-16 items-center justify-center rounded-full bg-emerald-500/10">
          <CheckCircle className="size-8 text-emerald-600" />
        </div>
        <h1 className="text-2xl font-semibold tracking-tight">All set!</h1>
        <p className="text-sm text-muted-foreground max-w-xs">
          Your profile has been created. You can now access the app.
        </p>
      </div>

        <Button onClick={handleContinue} className="w-full">
          Continue
        <ArrowRight className="ml-2 size-4" />
      </Button>

      <p className="text-xs text-muted-foreground">
        Redirecting automatically in a few seconds...
      </p>
    </div>
  )
}

export default RegisterProfileSuccess
