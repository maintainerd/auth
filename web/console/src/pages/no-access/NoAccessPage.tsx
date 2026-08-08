import { useNavigate } from 'react-router-dom'
import { ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import LoginLayout from '@/components/layout/LoginLayout'
import { useAuth } from '@/hooks/useAuth'
import { useTenant } from '@/hooks/useTenant'
import { resolvePostAuthRoute } from '@/utils/postAuthRoute'

/**
 * Shown when an authenticated user opens a page they're not allowed to see
 * (e.g. a different tenant's URL). The bootstrap gate / route guard routes here.
 */
const NoAccessPage = () => {
  const navigate = useNavigate()
  const { isAuthenticated } = useAuth()
  const { currentTenant } = useTenant()

  const handleBack = () => {
    navigate(isAuthenticated ? resolvePostAuthRoute() : '/', {
      replace: true,
    })
  }

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="flex flex-col gap-8 text-center">
        <div className="flex flex-col items-center gap-3">
          <div className="flex size-14 items-center justify-center rounded-full bg-amber-100">
            <ShieldAlert className="size-7 text-amber-600" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">You don't have access</h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            You're not allowed to view this page. If you think this is a mistake, contact your
            administrator.
          </p>
        </div>
        <Button className="w-full" onClick={handleBack}>
          {isAuthenticated ? 'Back to your dashboard' : 'Sign in'}
        </Button>
      </div>
    </LoginLayout>
  )
}

export default NoAccessPage
