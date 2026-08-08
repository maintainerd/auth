import { describe, it, expect } from "vitest"
import { policySchema } from "./policySchema"

/**
 * Each case pins the client schema to a rule in the Go source, so a drift in
 * either direction (rejecting a payload the API accepts, or accepting one it
 * rejects) fails here rather than at submit time:
 *   internal/iam/validation_policy.go        — PolicyCreate/UpdateRequestDTO
 *   internal/setup/seeder/013_control_policy.go — writes version "v1"
 */
function makePolicy(overrides: Record<string, unknown> = {}) {
  return {
    name: "read-only",
    description: "",
    version: "1.0.0",
    status: "active",
    document: {
      statement: [{ effect: "allow", action: ["read"], resource: ["*"] }],
    },
    ...overrides,
  }
}

const isValid = (overrides: Record<string, unknown> = {}) =>
  policySchema.isValidSync(makePolicy(overrides))

describe("policySchema", () => {
  it("accepts a baseline policy", () => {
    expect(isValid()).toBe(true)
  })

  describe("name", () => {
    it("accepts up to the server's 150 characters and rejects 151", () => {
      expect(isValid({ name: "a".repeat(150) })).toBe(true)
      expect(isValid({ name: "a".repeat(151) })).toBe(false)
    })

    it("still enforces the 3-character minimum", () => {
      expect(isValid({ name: "ab" })).toBe(false)
      expect(isValid({ name: "abc" })).toBe(true)
    })

    it("accepts the slashes the server's pattern allows", () => {
      expect(isValid({ name: "svc/read-only" })).toBe(true)
      expect(isValid({ name: "svc\\read-only" })).toBe(true)
      expect(isValid({ name: "svc:read_only-1" })).toBe(true)
    })

    it("rejects characters outside the server's pattern", () => {
      expect(isValid({ name: "Read-Only" })).toBe(false)
      expect(isValid({ name: "read only" })).toBe(false)
      expect(isValid({ name: "read.only" })).toBe(false)
    })
  })

  describe("description", () => {
    it("is optional — the server DTO takes *string and the column defaults to ''", () => {
      expect(isValid({ description: "" })).toBe(true)
    })

    it("accepts short text the old 10-character minimum rejected", () => {
      expect(isValid({ description: "short" })).toBe(true)
    })

    it("caps at the server's 500 characters", () => {
      expect(isValid({ description: "a".repeat(500) })).toBe(true)
      expect(isValid({ description: "a".repeat(501) })).toBe(false)
    })
  })

  describe("version", () => {
    it("accepts the seeder's non-semver 'v1'", () => {
      expect(isValid({ version: "v1" })).toBe(true)
    })

    it("accepts any other free-form value within 20 characters", () => {
      expect(isValid({ version: "2024-01-01" })).toBe(true)
      expect(isValid({ version: "a".repeat(20) })).toBe(true)
    })

    it("rejects more than the server's 20 characters", () => {
      expect(isValid({ version: "a".repeat(21) })).toBe(false)
    })

    it("rejects blank and whitespace-only values (DB CHECK btrim(version) <> '')", () => {
      expect(isValid({ version: "" })).toBe(false)
      expect(isValid({ version: "   " })).toBe(false)
    })
  })

  describe("status", () => {
    it("accepts only active and inactive", () => {
      expect(isValid({ status: "active" })).toBe(true)
      expect(isValid({ status: "inactive" })).toBe(true)
      expect(isValid({ status: "archived" })).toBe(false)
    })
  })

  it("still requires at least one statement", () => {
    expect(isValid({ document: { statement: [] } })).toBe(false)
  })
})
