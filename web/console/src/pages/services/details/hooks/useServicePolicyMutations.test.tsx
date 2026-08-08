import { describe, it, expect, vi, beforeEach } from "vitest"
import type { ReactNode } from "react"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClientProvider } from "@tanstack/react-query"
import { createTestQueryClient } from "@/test/utils"
import { policyServicesKeys } from "@/pages/policies/details/hooks/useServicesByPolicy"
import { useServicePolicyMutations } from "./useServicePolicyMutations"

const { assignMock, removeMock } = vi.hoisted(() => ({
  assignMock: vi.fn(),
  removeMock: vi.fn(),
}))

vi.mock("@/services", () => ({
  assignPolicyToService: (...args: unknown[]) => assignMock(...args),
  removePolicyFromService: (...args: unknown[]) => removeMock(...args),
}))

function setup() {
  const queryClient = createTestQueryClient()
  const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
  return { invalidateSpy, wrapper }
}

/** The query keys passed to invalidateQueries, in call order. */
const invalidatedKeys = (spy: { mock: { calls: unknown[][] } }) =>
  spy.mock.calls.map(([arg]) => (arg as { queryKey: unknown }).queryKey)

describe("useServicePolicyMutations", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("invalidates the policy's services listing after assigning", async () => {
    assignMock.mockResolvedValue(undefined)
    const { invalidateSpy, wrapper } = setup()

    const { result } = renderHook(() => useServicePolicyMutations("s1"), { wrapper })
    await result.current.assignPolicy.mutateAsync("pol1")

    // The link is bidirectional: the policy's own Services tab must refetch, and
    // its key starts with 'policy' so the 'policies' key does not cover it.
    await waitFor(() => {
      expect(invalidatedKeys(invalidateSpy)).toEqual(
        expect.arrayContaining([policyServicesKeys.all("pol1")]),
      )
    })
    expect(assignMock).toHaveBeenCalledWith("s1", "pol1")
  })

  it("invalidates the policy's services listing after removing", async () => {
    removeMock.mockResolvedValue(undefined)
    const { invalidateSpy, wrapper } = setup()

    const { result } = renderHook(() => useServicePolicyMutations("s1"), { wrapper })
    await result.current.removePolicy.mutateAsync("pol1")

    await waitFor(() => {
      expect(invalidatedKeys(invalidateSpy)).toEqual(
        expect.arrayContaining([policyServicesKeys.all("pol1")]),
      )
    })
    expect(removeMock).toHaveBeenCalledWith("s1", "pol1")
  })

  it("still invalidates the policy and service caches", async () => {
    assignMock.mockResolvedValue(undefined)
    const { invalidateSpy, wrapper } = setup()

    const { result } = renderHook(() => useServicePolicyMutations("s1"), { wrapper })
    await result.current.assignPolicy.mutateAsync("pol1")

    await waitFor(() => {
      expect(invalidatedKeys(invalidateSpy)).toEqual(
        expect.arrayContaining([["policies"], ["services", "detail", "s1"], ["services", "list"]]),
      )
    })
  })

  it("keys the services invalidation to the mutated policy only", async () => {
    assignMock.mockResolvedValue(undefined)
    const { invalidateSpy, wrapper } = setup()

    const { result } = renderHook(() => useServicePolicyMutations("s1"), { wrapper })
    await result.current.assignPolicy.mutateAsync("pol1")

    await waitFor(() => {
      expect(invalidatedKeys(invalidateSpy)).not.toContainEqual(policyServicesKeys.all("pol2"))
    })
  })
})
