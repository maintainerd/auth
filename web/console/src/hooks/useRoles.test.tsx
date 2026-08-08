import { describe, it, expect, vi, beforeEach } from "vitest"
import type { ReactNode } from "react"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClientProvider } from "@tanstack/react-query"
import { createTestQueryClient } from "@/test/utils"
import { roleKeys, useAddRolePermissions, useRemoveRolePermission } from "./useRoles"

const { addRolePermissionsMock, removeRolePermissionMock } = vi.hoisted(() => ({
  addRolePermissionsMock: vi.fn(),
  removeRolePermissionMock: vi.fn(),
}))

vi.mock("@/services/api/roles", () => ({
  fetchRoles: vi.fn(),
  fetchRoleById: vi.fn(),
  createRole: vi.fn(),
  updateRole: vi.fn(),
  deleteRole: vi.fn(),
  updateRoleStatus: vi.fn(),
  addRolePermissions: (...args: unknown[]) => addRolePermissionsMock(...args),
  removeRolePermission: (...args: unknown[]) => removeRolePermissionMock(...args),
  fetchRolePermissions: vi.fn(),
}))

function setup() {
  const queryClient = createTestQueryClient()
  const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
  return { queryClient, invalidateSpy, wrapper }
}

/** The query keys passed to invalidateQueries, in call order. */
const invalidatedKeys = (spy: { mock: { calls: unknown[][] } }) =>
  spy.mock.calls.map(([arg]) => (arg as { queryKey: unknown }).queryKey)

describe("useRoles role-permission mutations", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("invalidates permissions, detail and lists after adding permissions", async () => {
    addRolePermissionsMock.mockResolvedValue(undefined)
    const { invalidateSpy, wrapper } = setup()

    const { result } = renderHook(() => useAddRolePermissions(), { wrapper })
    await result.current.mutateAsync({ roleId: "r1", data: { permissions: ["p1"] } })

    // RoleResponseDTO carries a `permissions` array (internal/iam/types.go), so
    // the detail and list caches go stale too — not just the permissions tab.
    await waitFor(() => {
      expect(invalidatedKeys(invalidateSpy)).toEqual(
        expect.arrayContaining([
          roleKeys.permissions("r1"),
          roleKeys.detail("r1"),
          roleKeys.lists(),
        ]),
      )
    })
  })

  it("invalidates permissions, detail and lists after removing a permission", async () => {
    removeRolePermissionMock.mockResolvedValue(undefined)
    const { invalidateSpy, wrapper } = setup()

    const { result } = renderHook(() => useRemoveRolePermission(), { wrapper })
    await result.current.mutateAsync({ roleId: "r1", permissionId: "p1" })

    await waitFor(() => {
      expect(invalidatedKeys(invalidateSpy)).toEqual(
        expect.arrayContaining([
          roleKeys.permissions("r1"),
          roleKeys.detail("r1"),
          roleKeys.lists(),
        ]),
      )
    })
  })

  it("scopes the invalidation to the mutated role", async () => {
    addRolePermissionsMock.mockResolvedValue(undefined)
    const { invalidateSpy, wrapper } = setup()

    const { result } = renderHook(() => useAddRolePermissions(), { wrapper })
    await result.current.mutateAsync({ roleId: "r1", data: { permissions: ["p1"] } })

    await waitFor(() => {
      expect(invalidatedKeys(invalidateSpy)).not.toContainEqual(roleKeys.detail("r2"))
    })
  })
})
