import { useState } from "react"
import { useNavigate, Link, useSearchParams } from "react-router-dom"
import { KeyRound, ArrowLeft } from "lucide-react"
import { Button } from "@/components/ui/button"
import { FormCodeField } from "@/components/inputs"
import LoginLayout from "@/components/layout/LoginLayout"
import { useTenant } from "@/hooks/useTenant"
import { useToast } from "@/hooks/useToast"
import { post } from "@/services/api/client"
import { API_ENDPOINTS } from "@/services/api/config"

export default function BackupCodeRecoveryPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { currentTenant } = useTenant()
  const { showError } = useToast()

  const [code, setCode] = useState("")
  const [loading, setLoading] = useState(false)

  const clientId = searchParams.get("client_id")
  // Tenant comes from the domain bootstrap (its slug), never from a query param.
  const tenantId = currentTenant?.name

  const handleRecover = async () => {
    if (!code) return
    setLoading(true)
    try {
      const body: Record<string, string> = { backup_code: code }
      if (clientId) body.client_id = clientId
      if (tenantId) body.tenant_id = tenantId
      await post(API_ENDPOINTS.AUTH.RECOVERY_BACKUP_CODE, body)
      navigate("/", { replace: true })
    } catch (e: unknown) {
      showError(e, "Recovery failed")
    } finally {
      setLoading(false)
    }
  }

  return (
    <LoginLayout>
      <div className="space-y-6 w-full max-w-sm">
        <Link to="/login" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="h-3 w-3" /> Back to login
        </Link>

        <div className="space-y-2 text-center">
          <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-muted">
            <KeyRound className="size-6" />
          </div>
          <h1 className="text-2xl font-bold">Backup Code Recovery</h1>
          <p className="text-muted-foreground">
            Enter one of your saved backup codes to regain access to your account.
          </p>
        </div>

        <div className="space-y-3">
          <FormCodeField
            label="Backup code"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder="Enter your backup code"
            autoComplete="off"
          />

          <Button className="w-full" onClick={handleRecover} disabled={loading || !code}>
            {loading ? "Verifying..." : "Recover Account"}
          </Button>
        </div>
      </div>
    </LoginLayout>
  )
}
