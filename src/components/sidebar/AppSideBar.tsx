import * as React from "react"
import { NavMain } from "@/components/sidebar/NavMain"
import {
  Sidebar,
  SidebarContent,
} from "@/components/ui/sidebar"
import { data } from "./constants"

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar
      data-console-sidebar
      collapsible="offcanvas"
      {...props}
      className="!top-14 !h-[calc(100svh-3.5rem)] [&_[data-sidebar=sidebar]]:overflow-y-auto [&_[data-sidebar=sidebar]]:border-sidebar-border [&_[data-sidebar=sidebar]]:bg-sidebar [&_[data-sidebar=sidebar]]:text-sidebar-foreground"
    >
      <SidebarContent className="flex-none gap-1 overflow-visible bg-sidebar px-3 pb-4 pt-4">
        <NavMain sections={data.navSections} />
      </SidebarContent>
    </Sidebar>
  )
}
