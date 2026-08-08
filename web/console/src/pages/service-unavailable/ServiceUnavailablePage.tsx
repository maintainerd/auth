import { ServerCrash } from 'lucide-react'
import { Button } from '@/components/ui/button'
import LoginLayout from '@/components/layout/LoginLayout'
import { useTenant } from '@/hooks/useTenant'

const ServiceUnavailablePage = () => {
  const { currentTenant } = useTenant()

  const handleRetry = () => {
    window.location.reload()
  }

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="flex flex-col gap-8 text-center">
        <div className="flex flex-col items-center gap-3">
          <div className="flex size-14 items-center justify-center rounded-full bg-destructive/10">
            <ServerCrash className="size-7 text-destructive" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Service Unavailable</h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            We&apos;re unable to connect to our servers right now. This is usually temporary
            &mdash; please wait a moment and try again.
          </p>
        </div>
        <Button className="w-full" onClick={handleRetry}>
          Try again
        </Button>
      </div>
    </LoginLayout>
  )
}

export default ServiceUnavailablePage
