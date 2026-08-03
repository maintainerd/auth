import { useLayoutEffect, type ReactNode } from "react"
import { useAppSelector } from "@/store/hooks"
import { applyConsoleTheme, clearConsoleTheme } from "@/lib/branding/consoleTheme"

interface Props {
  children: ReactNode
}

export function ConsoleBrandingProvider({ children }: Props) {
  const branding = useAppSelector((state) => state.tenant.currentTenant?.branding)

  // useLayoutEffect, not useEffect: the theme must reach the document BEFORE the
  // browser paints, or the splash renders unbranded and then visibly flips once
  // the tenant resolves. Matches the identity app's applyBranding.
  useLayoutEffect(() => {
    applyConsoleTheme(branding)

    return () => {
      clearConsoleTheme()
    }
  }, [branding])

  return <>{children}</>
}
