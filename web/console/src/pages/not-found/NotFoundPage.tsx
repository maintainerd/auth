import { useNavigate } from 'react-router-dom'
import { Compass } from 'lucide-react'
import { Button } from '@/components/ui/button'
import LoginLayout from '@/components/layout/LoginLayout'
import { useTenant } from '@/hooks/useTenant'

/**
 * Catch-all page for unknown routes. Keeps the user inside the branded shell
 * and offers a clear path back instead of a blank screen.
 */
const NotFoundPage = () => {
  const navigate = useNavigate()
  const { currentTenant } = useTenant()

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="flex flex-col gap-8 text-center">
        <div className="flex flex-col items-center gap-3">
          <div className="flex size-14 items-center justify-center rounded-full bg-muted">
            <Compass className="size-7 text-muted-foreground" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Page not found</h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            The page you're looking for doesn't exist or may have moved.
          </p>
        </div>
        <Button className="w-full" onClick={() => navigate('/', { replace: true })}>
          Back to sign in
        </Button>
      </div>
    </LoginLayout>
  )
}

export default NotFoundPage
