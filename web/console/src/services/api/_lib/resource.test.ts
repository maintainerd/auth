import { describe, it, expect, vi, beforeEach } from "vitest"

const { getMock, postMock, putMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  putMock: vi.fn(),
  deleteMock: vi.fn(),
}))

vi.mock("../client", () => ({
  get: getMock,
  post: postMock,
  put: putMock,
  deleteRequest: deleteMock,
  ApiError: class ApiError extends Error {
    status = 0
  },
}))

import { createResourceApi } from "./resource"

interface Row {
  id: string
}

describe("createResourceApi", () => {
  const api = createResourceApi<Row, { n: string }, { n: string }, Row[]>("/things", "thing")

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("list() calls GET on the base and unwraps", async () => {
    getMock.mockResolvedValue({ success: true, data: [{ id: "1" }] })
    await expect(api.list()).resolves.toEqual([{ id: "1" }])
    expect(getMock).toHaveBeenCalledWith("/things")
  })

  it("list(params) appends the query string", async () => {
    getMock.mockResolvedValue({ success: true, data: [] })
    await api.list({ page: 2 })
    expect(getMock).toHaveBeenCalledWith("/things?page=2")
  })

  it("getById() calls GET on /base/:id", async () => {
    getMock.mockResolvedValue({ success: true, data: { id: "1" } })
    await expect(api.getById("1")).resolves.toEqual({ id: "1" })
    expect(getMock).toHaveBeenCalledWith("/things/1")
  })

  it("create() POSTs the body", async () => {
    postMock.mockResolvedValue({ success: true, data: { id: "2" } })
    await expect(api.create({ n: "x" })).resolves.toEqual({ id: "2" })
    expect(postMock).toHaveBeenCalledWith("/things", { n: "x" })
  })

  it("update() PUTs to /base/:id", async () => {
    putMock.mockResolvedValue({ success: true, data: { id: "1" } })
    await expect(api.update("1", { n: "y" })).resolves.toEqual({ id: "1" })
    expect(putMock).toHaveBeenCalledWith("/things/1", { n: "y" })
  })

  it("remove() DELETEs /base/:id and resolves void", async () => {
    deleteMock.mockResolvedValue({ success: true })
    await expect(api.remove("1")).resolves.toBeUndefined()
    expect(deleteMock).toHaveBeenCalledWith("/things/1")
  })
})
