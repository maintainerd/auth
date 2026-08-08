import { describe, it, expect } from "vitest"
import { unwrap, assertSuccess } from "./unwrap"
import { ApiError } from "../client"
import type { ApiResponse } from "../types"

describe("unwrap", () => {
  it("returns data on success", () => {
    const res: ApiResponse<number> = { success: true, data: 42 }
    expect(unwrap(res, "fetch thing")).toBe(42)
  })

  it("returns falsy-but-valid payloads (empty array, 0)", () => {
    expect(unwrap({ success: true, data: [] } as ApiResponse<unknown[]>, "fetch")).toEqual([])
    expect(unwrap({ success: true, data: 0 } as ApiResponse<number>, "fetch")).toBe(0)
  })

  it("throws ApiError with the response message when not successful", () => {
    const res: ApiResponse<number> = { success: false, message: "boom" }
    expect(() => unwrap(res, "fetch thing")).toThrow(ApiError)
    expect(() => unwrap(res, "fetch thing")).toThrow("boom")
  })

  it("throws with a default action message when data is missing", () => {
    const res: ApiResponse<number> = { success: true }
    expect(() => unwrap(res, "fetch thing")).toThrow("Failed to fetch thing")
  })
})

describe("assertSuccess", () => {
  it("does not throw on success", () => {
    expect(() => assertSuccess({ success: true }, "delete thing")).not.toThrow()
  })

  it("throws ApiError with the message on failure", () => {
    expect(() => assertSuccess({ success: false, message: "nope" }, "delete thing")).toThrow("nope")
  })

  it("throws with a default action message when no message is present", () => {
    expect(() => assertSuccess({ success: false }, "delete thing")).toThrow("Failed to delete thing")
  })
})
