import type { CSSProperties } from "react"
import { Outlet } from "react-router-dom"
import { AppSidebar } from "@/components/sidebar/AppSideBar"
import { TopNav } from "@/components/navigation/TopNav"
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar"
import { ConsoleBrandingProvider } from "@/components/theme/ConsoleBrandingProvider"
import { cn } from "@/lib/utils"

interface PrivateLayoutProps {
  fullWidth?: boolean
}

// Access gating (auth, registration completeness, tenant isolation) is handled
// centrally by AppBootstrap → RouteGuard; this layout only renders the chrome.
//
// Coolify-style chrome: a full-height left sidebar owns the brand + tenant
// switcher (see AppSidebar). The top bar is a slim header inside the content
// area holding only the user menu + sidebar collapse toggle.
export function PrivateLayout({ fullWidth = false }: PrivateLayoutProps) {
  return (
    <ConsoleBrandingProvider>
      <SidebarProvider style={{ "--sidebar-width": "17rem" } as CSSProperties}>
        <AppSidebar variant="sidebar" />
        <SidebarInset className="min-w-0 bg-white">
          <TopNav />
          <main
            className={cn(
              "flex-1 px-4 py-6 sm:px-6 sm:py-8",
              !fullWidth && "w-full max-w-6xl mx-auto",
            )}
          >
            <Outlet />
          </main>
        </SidebarInset>
      </SidebarProvider>
    </ConsoleBrandingProvider>
  )
}
