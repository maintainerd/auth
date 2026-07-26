import { describe, expect, it } from "vitest"
import { data } from "./constants"
import type { NavItem } from "./NavMain"

function subItems(item: NavItem): NonNullable<NavItem["items"]> {
  return Array.isArray(item.items) ? item.items : []
}

function allItemTitles() {
  return data.navSections.flatMap((section) =>
    section.items.flatMap((item) => {
      const navItem = item as NavItem
      return [navItem.title, ...subItems(navItem).map((subItem) => subItem.title)]
    }),
  )
}

describe("sidebar navigation data", () => {
  it("keeps monitoring as one sidenav item with page-level tabs", () => {
    const operations = data.navSections.find((section) => section.label === "Operations")
    const monitoring = operations?.items.find((item) => item.title === "Monitoring")

    expect(monitoring).toMatchObject({
      title: "Monitoring",
      route: "/monitoring",
    })
    expect(monitoring && "items" in monitoring ? monitoring.items : undefined).toBeUndefined()

    const itemTitles = allItemTitles()

    expect(itemTitles).not.toContain("Sign-in Logs")
    expect(itemTitles).not.toContain("Audit Log")
  })

  it("keeps user management as one sidenav item with page-level tabs", () => {
    const identity = data.navSections.find((section) => section.label === "Identity & Access")
    const userManagement = identity?.items.find((item) => item.title === "User Management")

    expect(userManagement).toMatchObject({
      title: "User Management",
      route: "/user-management",
      activeRoutes: ["/users", "/roles", "/invites"],
    })
    expect(userManagement && "items" in userManagement ? userManagement.items : undefined).toBeUndefined()

    const itemTitles = allItemTitles()

    expect(itemTitles).not.toContain("Users")
    expect(itemTitles).not.toContain("Roles")
    expect(itemTitles).not.toContain("Invitations")
  })

  it("keeps authentication as one sidenav item with page-level tabs", () => {
    const identity = data.navSections.find((section) => section.label === "Identity & Access")
    const authentication = identity?.items.find((item) => item.title === "Authentication")

    expect(authentication).toMatchObject({
      title: "Authentication",
      route: "/authentication",
      activeRoutes: ["/providers/identity", "/registration-flows"],
    })
    expect(authentication && "items" in authentication ? authentication.items : undefined).toBeUndefined()

    const itemTitles = allItemTitles()

    expect(itemTitles).not.toContain("Identity Providers")
    expect(itemTitles).not.toContain("Registration")
  })

  it("keeps applications as one sidenav item with page-level tabs", () => {
    const applicationsSection = data.navSections.find((section) => section.label === "Applications & APIs")
    const applications = applicationsSection?.items.find((item) => item.title === "Applications")

    expect(applications).toMatchObject({
      title: "Applications",
      route: "/applications",
      activeRoutes: ["/clients", "/workload-identity"],
    })
    expect(applications && "items" in applications ? applications.items : undefined).toBeUndefined()

    const itemTitles = allItemTitles()

    expect(itemTitles).not.toContain("Clients")
    expect(itemTitles).not.toContain("Workload Identity")
  })

  it("keeps APIs and resources as one sidenav item with page-level tabs", () => {
    const applicationsSection = data.navSections.find((section) => section.label === "Applications & APIs")
    const apisResources = applicationsSection?.items.find((item) => item.title === "APIs & Resources")

    expect(apisResources).toMatchObject({
      title: "APIs & Resources",
      route: "/apis-resources",
      activeRoutes: ["/services", "/apis", "/policies"],
    })
    expect(apisResources && "items" in apisResources ? apisResources.items : undefined).toBeUndefined()

    const itemTitles = allItemTitles()

    expect(itemTitles).not.toContain("Services")
    expect(itemTitles).not.toContain("APIs")
    expect(itemTitles).not.toContain("Policies")
  })
})
