import { describe, it, expect, vi, beforeEach } from "vitest"

const { getMock, postMock, putMock, patchMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  putMock: vi.fn(),
  patchMock: vi.fn(),
  deleteMock: vi.fn(),
}))

vi.mock("../client", () => ({
  get: getMock,
  post: postMock,
  put: putMock,
  patch: patchMock,
  deleteRequest: deleteMock,
  ApiError: class ApiError extends Error {
    status = 0
  },
}))

import {
  fetchRegistrationFlows,
  fetchRegistrationFlow,
  createRegistrationFlow,
  updateRegistrationFlow,
  updateRegistrationFlowStatus,
  fetchRegistrationFlowRoles,
  assignRegistrationFlowRoles,
  removeRegistrationFlowRole,
} from "./index"

const ROLE = {
  role_id: "r1",
  name: "seller",
  description: "Seller role",
  is_default: false,
  is_system: false,
  status: "active",
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
}

const LIST_ROW = {
  registration_flow_id: "f1",
  name: "Seller Signup",
  description: "Sellers",
  identifier: "seller-signup-k3f9qz7lm2xb8vrt",
  status: "active",
  client_id: "c-uuid",
  verification_required: true,
  is_system: false,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
}

const DETAIL = {
  ...LIST_ROW,
  required_fields: ["email"],
  client: {
    client_id: "c-uuid",
    name: "storefront",
    display_name: "Storefront",
    identifier: "storefront-abc123",
    status: "active",
  },
}

describe("registration flow API", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("fetchRegistrationFlows passes filters through and returns the rows untouched", async () => {
    getMock.mockResolvedValue({ success: true, data: { rows: [LIST_ROW], total: 1, page: 1, limit: 10, total_pages: 1 } })

    const result = await fetchRegistrationFlows({ search: "seller", status: "active,inactive", is_system: false })

    expect(getMock).toHaveBeenCalledWith(
      "/registration_flows?search=seller&status=active%2Cinactive&is_system=false",
    )
    // No hand-mapping: the row is returned exactly as the backend sent it.
    expect(result.rows).toEqual([LIST_ROW])
  })

  it("fetchRegistrationFlows defaults a missing rows array to empty", async () => {
    getMock.mockResolvedValue({ success: true, data: { total: 0, page: 1, limit: 10, total_pages: 0 } })
    await expect(fetchRegistrationFlows()).resolves.toMatchObject({ rows: [] })
  })

  it("fetchRegistrationFlow returns the detail projection including client + required_fields", async () => {
    getMock.mockResolvedValue({ success: true, data: DETAIL })

    const flow = await fetchRegistrationFlow("f1")

    expect(getMock).toHaveBeenCalledWith("/registration_flows/f1")
    expect(flow.required_fields).toEqual(["email"])
    expect(flow.client?.identifier).toBe("storefront-abc123")
  })

  it("fetchRegistrationFlow defaults a null required_fields to an empty array", async () => {
    getMock.mockResolvedValue({ success: true, data: { ...DETAIL, required_fields: null } })
    await expect(fetchRegistrationFlow("f1")).resolves.toMatchObject({ required_fields: [] })
  })

  it("createRegistrationFlow never sends an identifier", async () => {
    postMock.mockResolvedValue({ success: true, data: DETAIL })

    await createRegistrationFlow({
      name: "Seller Signup",
      description: "Sellers",
      status: "active",
      client_id: "c-uuid",
      verification_required: true,
      required_fields: ["email"],
      role_ids: ["r1"],
    })

    const [endpoint, body] = postMock.mock.calls[0]
    expect(endpoint).toBe("/registration_flows")
    expect(body).not.toHaveProperty("identifier")
  })

  it("updateRegistrationFlow PUTs only the mutable fields", async () => {
    putMock.mockResolvedValue({ success: true, data: DETAIL })

    await updateRegistrationFlow("f1", {
      name: "Seller Signup",
      verification_required: false,
      required_fields: ["email", "phone"],
    })

    const [endpoint, body] = putMock.mock.calls[0]
    expect(endpoint).toBe("/registration_flows/f1")
    expect(body).not.toHaveProperty("identifier")
    expect(body).not.toHaveProperty("client_id")
    expect(body).toMatchObject({ verification_required: false, required_fields: ["email", "phone"] })
  })

  it("updateRegistrationFlowStatus PATCHes the status sub-resource", async () => {
    patchMock.mockResolvedValue({ success: true, data: DETAIL })
    await updateRegistrationFlowStatus("f1", { status: "inactive" })
    expect(patchMock).toHaveBeenCalledWith("/registration_flows/f1/status", { status: "inactive" })
  })

  it("fetchRegistrationFlowRoles reads the paginated envelope", async () => {
    getMock.mockResolvedValue({
      success: true,
      data: { rows: [ROLE], total: 1, page: 1, limit: 10, total_pages: 1 },
    })

    const result = await fetchRegistrationFlowRoles("f1", { page: 1, limit: 10 })

    expect(getMock).toHaveBeenCalledWith("/registration_flows/f1/roles?page=1&limit=10")
    expect(result.rows).toEqual([ROLE])
    expect(result.total).toBe(1)
  })

  it("assignRegistrationFlowRoles resolves the BARE ARRAY the endpoint returns", async () => {
    // Regression: this endpoint returns `[role, ...]`, not `{rows: [...]}`. The
    // old code did `response.data.rows.map(...)`, which threw on a successful
    // assignment and surfaced an error toast for work that had actually happened.
    postMock.mockResolvedValue({ success: true, data: [ROLE] })

    const roles = await assignRegistrationFlowRoles("f1", ["r1"])

    expect(postMock).toHaveBeenCalledWith("/registration_flows/f1/roles", { role_uuids: ["r1"] })
    expect(roles).toEqual([ROLE])
  })

  it("assignRegistrationFlowRoles resolves an empty array without throwing", async () => {
    postMock.mockResolvedValue({ success: true, data: [] })
    await expect(assignRegistrationFlowRoles("f1", ["r1"])).resolves.toEqual([])
  })

  it("removeRegistrationFlowRole returns the updated flow", async () => {
    deleteMock.mockResolvedValue({ success: true, data: DETAIL })

    const flow = await removeRegistrationFlowRole("f1", "r1")

    expect(deleteMock).toHaveBeenCalledWith("/registration_flows/f1/roles/r1")
    expect(flow.registration_flow_id).toBe("f1")
  })

  it("throws the backend message when a call fails", async () => {
    getMock.mockResolvedValue({ success: false, message: "boom" })
    await expect(fetchRegistrationFlow("f1")).rejects.toThrow("boom")
  })
})
