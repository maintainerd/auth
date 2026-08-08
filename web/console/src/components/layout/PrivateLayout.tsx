import type { CSSProperties } from "react"
import { Outlet } from "react-router-dom"
import { AppSidebar } from "@/components/sidebar/AppSideBar"
import { AppTopNav } from "@/components/navigation/AppTopNav"
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"

interface PrivateLayoutProps {
  fullWidth?: boolean
}

// Access gating (auth, registration completeness, tenant isolation) is handled
// centrally by AppBootstrap → RouteGuard; this layout only renders the chrome.
//
// Chrome: the brand bar spans the full app, while the sidebar and content sit
// beneath it on the same calm canvas color as the login side-panel template.
export function PrivateLayout({ fullWidth = false }: PrivateLayoutProps) {
  return (
    <div className="min-h-svh bg-background">
      <SidebarProvider style={{ "--sidebar-width": "17rem" } as CSSProperties}>
        <AppTopNav />
        <AppSidebar variant="sidebar" />
        <SidebarInset className="min-w-0 bg-background pt-14">
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
    </div>
  )
}
