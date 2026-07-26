import { useEffect, useRef, type ReactNode } from "react"
import { useAppSelector } from "@/store/hooks"
import { useBrandings } from "@/hooks/useBranding"

interface Props {
  children: ReactNode
}

function applyBranding(meta: Record<string, unknown> | undefined, companyName: string | undefined) {
  const root = document.documentElement

  if (!meta) {
    clearBranding()
    return
  }

  try {
    if (typeof meta["color.primary"] === "string") {
      root.style.setProperty("--branding-primary", meta["color.primary"] as string)
    }
    if (typeof meta["font.family"] === "string") {
      root.style.setProperty("--branding-font", meta["font.family"] as string)
      document.body.style.fontFamily = meta["font.family"] as string
    }
    if (typeof meta["color.background"] === "string") {
      root.style.setProperty("--branding-background", meta["color.background"] as string)
    }
  } catch {
    // parse error — skip
  }

  if (companyName) {
    document.title = companyName
  }
}

function clearBranding() {
  const root = document.documentElement
  root.style.removeProperty("--branding-primary")
  root.style.removeProperty("--branding-font")
  root.style.removeProperty("--branding-background")
  document.body.style.fontFamily = ""
  document.title = "Maintainerd IAM"
}

export function ConsoleBrandingProvider({ children }: Props) {
  const tenantId = useAppSelector((state) => state.tenant.currentTenant?.tenant_id)
  const prevTenantRef = useRef<string | undefined>(undefined)

  const { data: brandings } = useBrandings()
  const activeBranding = brandings?.find((b) => b.is_active)

  useEffect(() => {
    if (prevTenantRef.current && prevTenantRef.current !== tenantId) {
      clearBranding()
    }
    prevTenantRef.current = tenantId
  }, [tenantId])

  useEffect(() => {
    const meta = activeBranding?.metadata
      ? (typeof activeBranding.metadata === "string"
          ? JSON.parse(activeBranding.metadata)
          : activeBranding.metadata)
      : undefined
    applyBranding(meta, activeBranding?.company_name)

    return () => {
      clearBranding()
    }
  }, [activeBranding])

  return <>{children}</>
}
