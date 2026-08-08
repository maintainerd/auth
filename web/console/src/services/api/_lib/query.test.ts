import { describe, it, expect } from "vitest"
import { buildQuery } from "./query"

describe("buildQuery", () => {
  it("returns an empty string when no params are given", () => {
    expect(buildQuery()).toBe("")
  })

  it("returns an empty string for an empty object", () => {
    expect(buildQuery({})).toBe("")
  })

  it("builds a query string from params", () => {
    expect(buildQuery({ page: 1, limit: 10 })).toBe("?page=1&limit=10")
  })

  it("skips undefined, null and empty-string values", () => {
    expect(buildQuery({ a: 1, b: undefined, c: null, d: "", e: "x" })).toBe("?a=1&e=x")
  })

  it("keeps falsy-but-meaningful values like 0 and false", () => {
    expect(buildQuery({ a: 0, b: false })).toBe("?a=0&b=false")
  })
})
