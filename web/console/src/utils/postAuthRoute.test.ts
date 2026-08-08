import { describe, it, expect } from "vitest"
import { resolvePostAuthRoute, resolveGuardRedirect } from "./postAuthRoute"
import type { AccountEntity } from "@/services/api/auth/types"
import type { TenantEntity } from "@/services/api/tenants/types"

// Minimal fixtures — only the fields the routing logic reads. The tenant is
// identified by `name`; the host's tenant is the backend-resolved `currentTenant`
// passed into the guard, not a client-parsed slug.
const tenant = (overrides: Partial<TenantEntity> = {}): TenantEntity =>
  ({
    name: "acme",
    registration_config: { require_email_verification: true },
    ...overrides,
  } as TenantEntity)

const account = (overrides: Partial<AccountEntity> = {}): AccountEntity =>
  ({
    email_verified: true,
    profiles: [{ profile_id: "p1" }],
    tenant: { name: "acme" },
    ...overrides,
  } as AccountEntity)

describe("resolvePostAuthRoute", () => {
  it("always resolves to the flat dashboard route (the tenant is host-derived)", () => {
    expect(resolvePostAuthRoute()).toBe("/dashboard")
  })
})

describe("resolveGuardRedirect", () => {
  const guard = (
    pathname: string,
    isAuthenticated: boolean,
    acc: AccountEntity | null,
    hostTenant: TenantEntity | null = tenant(),
  ) => resolveGuardRedirect({ pathname, isAuthenticated, account: acc, tenant: hostTenant })

  it("never gates setup or no-access pages", () => {
    expect(guard("/setup/tenant", false, null)).toBeNull()
    expect(guard("/no-access", true, account())).toBeNull()
  })

  it("leaves unauthenticated root to the OAuth redirector, dashboard when complete", () => {
    expect(guard("/", false, null)).toBeNull()
    expect(guard("/", true, account())).toBe("/dashboard")
  })

  it("lets unauthenticated users see auth pages but bounces authenticated ones", () => {
    expect(guard("/login", false, null)).toBeNull()
    expect(guard("/login", true, account())).toBe("/dashboard")
  })

  it("leaves unauthenticated protected pages to the OAuth redirector", () => {
    expect(guard("/dashboard", false, null)).toBeNull()
  })

  it("allows unverified users through because verification belongs to identity", () => {
    expect(guard("/dashboard", true, account({ email_verified: false }))).toBeNull()
  })

  it("allows profile-less users through because profile completion belongs to identity", () => {
    expect(guard("/dashboard", true, account({ profiles: [] }))).toBeNull()
  })

  it("blocks access when the host's resolved tenant differs from the user's tenant", () => {
    expect(guard("/dashboard", true, account(), tenant({ name: "other" }))).toBe("/no-access")
  })

  it("allows access when the host's resolved tenant matches the user's tenant", () => {
    expect(guard("/users", true, account(), tenant({ name: "acme" }))).toBeNull()
  })

  it("does not block when the host tenant is unresolved", () => {
    expect(guard("/dashboard", true, account(), null)).toBeNull()
  })
})
