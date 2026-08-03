import { ChevronRight } from "lucide-react"
import { type ComponentType, useEffect, useMemo, useState } from "react"
import { Link, useLocation } from "react-router-dom"

import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubItem,
  SidebarMenuSubButton,
} from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"

export type NavSubItem = {
  title: string
  route: string
}

export type NavItem = {
  title: string
  route: string
  activeRoutes?: string[]
  icon?: ComponentType<{ className?: string; active?: boolean }>
  items?: NavSubItem[]
}

export type NavSection = {
  label?: string
  items: NavItem[]
}

const parentMenuClass =
  "h-9 rounded-md border border-transparent px-2 text-sm font-medium text-sidebar-foreground/80 transition-colors [&>svg]:size-[18px] [&>svg]:text-sidebar-foreground/65 hover:border-transparent hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:[&>svg]:text-sidebar-accent-foreground active:!bg-sidebar-accent active:!text-sidebar-accent-foreground data-[state=open]:hover:!bg-sidebar-accent"

const parentMenuActiveClass =
  "bg-sidebar-accent font-semibold !text-sidebar-accent-foreground hover:bg-sidebar-accent hover:!text-sidebar-accent-foreground active:!bg-sidebar-accent active:!text-sidebar-accent-foreground data-[active=true]:!bg-sidebar-accent data-[active=true]:!text-sidebar-accent-foreground [&>svg]:!text-sidebar-primary"

const subMenuClass =
  "h-8 w-full translate-x-0 rounded-md pl-[34px] pr-2 text-sm font-medium text-sidebar-foreground/75 transition-colors hover:!bg-sidebar-accent hover:text-sidebar-accent-foreground active:!bg-sidebar-accent active:!text-sidebar-accent-foreground data-[active=true]:!bg-sidebar-accent data-[active=true]:!text-sidebar-accent-foreground"

export function NavMain({ sections }: { sections: NavSection[] }) {
  const location = useLocation()

  // Routes are flat (the tenant lives in the host subdomain), so this just
  // normalizes to a leading-slash absolute path while preserving query tabs.
  const buildRoute = (route: string) => (route.startsWith('/') ? route : `/${route}`)

  // Active on the exact route or any of its sub-paths (e.g. /users is active on
  // /users/:id). Query-string tab links require an exact pathname+search match.
  const isActive = (route: string) => {
    const r = buildRoute(route)
    const [pathname, search] = r.split('?')

    if (search !== undefined) {
      return location.pathname === pathname && location.search === `?${search}`
    }

    return location.pathname === pathname || location.pathname.startsWith(`${pathname}/`)
  }

  const isParentActive = (item: NavItem) => {
    // Check if current route matches the parent route
    if (isActive(item.route)) return true
    if (item.activeRoutes?.some((route) => isActive(route))) return true
    // Check if any submenu item is active
    if (item.items) {
      return item.items.some((subItem) => isActive(subItem.route))
    }
    return false
  }

  // Titles of collapsible groups containing the active route — kept open
  // automatically as the user navigates.
  const activeGroups = useMemo(() => {
    const set = new Set<string>()
    for (const section of sections) {
      for (const item of section.items) {
        if (item.items && isParentActive(item)) set.add(item.title)
      }
    }
    return set
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sections, location.pathname])

  const [openItems, setOpenItems] = useState<Set<string>>(() => new Set(activeGroups))

  useEffect(() => {
    setOpenItems((prev) => {
      const next = new Set(prev)
      activeGroups.forEach((title) => next.add(title))
      return next
    })
  }, [activeGroups])

  const toggleItem = (title: string) => {
    const newOpenItems = new Set(openItems)
    if (newOpenItems.has(title)) {
      newOpenItems.delete(title)
    } else {
      newOpenItems.add(title)
    }
    setOpenItems(newOpenItems)
  }

  return (
    <>
      {sections.map((section, index) => (
        <SidebarGroup key={section.label ?? index} className="px-0 py-1.5">
          {section.label && (
            <SidebarGroupLabel
              data-console-sidebar-section
              className="h-7 px-2 text-xs font-semibold text-[#667085]"
            >
              {section.label}
            </SidebarGroupLabel>
          )}
          <SidebarGroupContent className="flex flex-col gap-1">
            <SidebarMenu className="gap-0.5">
              {section.items.map((item) => {
                const active = isParentActive(item)

                return (
                  <SidebarMenuItem key={item.title}>
                    {item.items ? (
                      <>
                        <SidebarMenuButton
                          data-console-sidebar-item
                          isActive={active}
                          onClick={() => toggleItem(item.title)}
                          tooltip={item.title}
                          className={cn(parentMenuClass, active && parentMenuActiveClass)}
                        >
                          {item.icon && <item.icon active={active} />}
                          <span>{item.title}</span>
                          <ChevronRight
                            data-console-sidebar-chevron
                            className={cn(
                              "ml-auto h-4 w-4 text-[#7b8797] transition-transform",
                              active && "text-sidebar-foreground",
                              openItems.has(item.title) && "rotate-90",
                            )}
                          />
                        </SidebarMenuButton>
                        {openItems.has(item.title) && (
                          <SidebarMenuSub className="mx-0 gap-0.5 border-l-0 px-0 py-1">
                            {item.items.map((subItem) => {
                              const subActive = isActive(subItem.route)

                              return (
                                <SidebarMenuSubItem key={subItem.title}>
                                  <SidebarMenuSubButton
                                    data-console-sidebar-sub-item
                                    asChild
                                    isActive={subActive}
                                    className={cn(subMenuClass, subActive && "font-semibold !text-sidebar-accent-foreground")}
                                  >
                                    <Link to={buildRoute(subItem.route)}>
                                      <span>{subItem.title}</span>
                                    </Link>
                                  </SidebarMenuSubButton>
                                </SidebarMenuSubItem>
                              )
                            })}
                          </SidebarMenuSub>
                        )}
                      </>
                    ) : (
                      <SidebarMenuButton
                        data-console-sidebar-item
                        asChild
                        isActive={active}
                        tooltip={item.title}
                        className={cn(parentMenuClass, active && parentMenuActiveClass)}
                      >
                        <Link to={buildRoute(item.route)}>
                          {item.icon && <item.icon active={active} />}
                          <span>{item.title}</span>
                        </Link>
                      </SidebarMenuButton>
                    )}
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      ))}
    </>
  )
}
