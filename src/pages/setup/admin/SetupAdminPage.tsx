import { useEffect } from "react"
import { useNavigate } from "react-router-dom"
import LoginLayout from "@/components/layout/LoginLayout"
import SetupAdminForm from "./components/SetupAdminForm"
import { useSetupStatus } from "@/hooks/useSetup"

const SetupAdminPage = () => {
  const navigate = useNavigate()
  const { status, checkStatus, isLoading, isError } = useSetupStatus()

  useEffect(() => {
    checkStatus()
  }, [checkStatus])

  useEffect(() => {
    if (!status) return
    if (status.is_admin_setup || !status.is_tenant_setup) {
      navigate('/', { replace: true })
    }
  }, [status, navigate])

  // Returning null on failure left a permanently blank page during first-run
  // setup — no message, no retry, and reloading just repeated it.
  if (isError) {
    return (
      <LoginLayout>
        <div className="flex flex-col items-center gap-4 py-12 text-center">
          <p className="text-sm text-destructive">
            Could not reach the server to check setup status.
          </p>
          <button
            type="button"
            className="text-sm underline underline-offset-4"
            onClick={() => void checkStatus()}
          >
            Try again
          </button>
        </div>
      </LoginLayout>
    )
  }

  if (isLoading || !status) return null

  return (
    <LoginLayout>
      <SetupAdminForm />
    </LoginLayout>
  )
}

export default SetupAdminPage
