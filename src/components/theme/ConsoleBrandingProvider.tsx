import { useEffect, type ReactNode } from "react"
import { useAppSelector } from "@/store/hooks"
import { applyConsoleTheme, clearConsoleTheme } from "@/lib/branding/consoleTheme"

interface Props {
  children: ReactNode
}

export function ConsoleBrandingProvider({ children }: Props) {
  const branding = useAppSelector((state) => state.tenant.currentTenant?.branding)

  useEffect(() => {
    applyConsoleTheme(branding)

    return () => {
      clearConsoleTheme()
    }
  }, [branding])

  return <>{children}</>
}
