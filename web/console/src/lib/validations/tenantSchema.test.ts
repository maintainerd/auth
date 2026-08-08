import { describe, it, expect } from "vitest"
import { createTenantSchema, RESERVED_TENANT_SLUGS } from "./tenantSchema"

const valid = {
  name: "acme-corp",
  display_name: "Acme Corporation",
  description: "The Acme tenant",
  status: "active",
}

describe("createTenantSchema", () => {
  it("accepts a valid tenant", async () => {
    await expect(createTenantSchema.validate(valid)).resolves.toMatchObject({ name: "acme-corp" })
  })

  // The name is the DNS subdomain, so the server anchors its pattern on an
  // alphanumeric at both ends (validation_tenant.go:14).
  it("accepts DNS-safe slugs", async () => {
    for (const name of ["abc", "acme-corp", "a1b2", "a-b-c-1", "x".repeat(63)]) {
      await expect(createTenantSchema.validate({ ...valid, name })).resolves.toBeTruthy()
    }
  })

  it("rejects leading and trailing hyphens the server rejects", async () => {
    for (const name of ["-acme", "acme-", "-acme-"]) {
      await expect(createTenantSchema.validate({ ...valid, name })).rejects.toThrow(/start and end/i)
    }
  })

  it("rejects uppercase and other characters outside the slug alphabet", async () => {
    for (const name of ["Acme", "acme corp", "acme_corp", "acme.corp"]) {
      await expect(createTenantSchema.validate({ ...valid, name })).rejects.toThrow()
    }
  })

  // Server enforces Length(3, 63); the client used to demand only 2 and cap at 100.
  it("matches the server's 3..63 character name bounds", async () => {
    await expect(createTenantSchema.validate({ ...valid, name: "ab" })).rejects.toThrow(/at least 3/i)
    await expect(createTenantSchema.validate({ ...valid, name: "abc" })).resolves.toBeTruthy()
    await expect(createTenantSchema.validate({ ...valid, name: "a".repeat(63) })).resolves.toBeTruthy()
    await expect(createTenantSchema.validate({ ...valid, name: "a".repeat(64) })).rejects.toThrow(/exceed 63/i)
  })

  it("rejects every reserved platform slug before the request is sent", async () => {
    for (const name of RESERVED_TENANT_SLUGS) {
      await expect(createTenantSchema.validate({ ...valid, name })).rejects.toThrow(/reserved/i)
    }
  })

  it("mirrors the server's reserved list exactly", () => {
    expect([...RESERVED_TENANT_SLUGS].sort()).toEqual(
      [
        "admin",
        "api",
        "auth",
        "console",
        "control",
        "control-api",
        "grafana",
        "prometheus",
        "rabbitmq",
        "root",
        "signoz",
        "system",
        "www",
      ].sort(),
    )
  })

  // Server enforces Length(8, 200); the client used to demand 10 and allow 500.
  it("matches the server's 8..200 character description bounds", async () => {
    await expect(createTenantSchema.validate({ ...valid, description: "1234567" })).rejects.toThrow(/at least 8/i)
    await expect(createTenantSchema.validate({ ...valid, description: "12345678" })).resolves.toBeTruthy()
    // 9 characters — accepted by the server but previously rejected client-side.
    await expect(createTenantSchema.validate({ ...valid, description: "123456789" })).resolves.toBeTruthy()
    await expect(createTenantSchema.validate({ ...valid, description: "d".repeat(200) })).resolves.toBeTruthy()
    await expect(createTenantSchema.validate({ ...valid, description: "d".repeat(201) })).rejects.toThrow(/exceed 200/i)
  })

  // Deliberately stricter than the server: the API stores an empty
  // display_name happily, but the console has no fallback label for a tenant
  // without one. Pinned so the rule is not "relaxed to match the server" by
  // accident during a later contract sweep.
  it("requires a display name even though the server does not", async () => {
    await expect(createTenantSchema.validate({ ...valid, display_name: "" })).rejects.toThrow(
      /display name is required/i,
    )
    await expect(
      createTenantSchema.validate({ ...valid, display_name: undefined }),
    ).rejects.toThrow(/display name is required/i)
  })

  // display_name is VARCHAR(255) with no server-side validation rule.
  it("allows display names up to the column width", async () => {
    await expect(createTenantSchema.validate({ ...valid, display_name: "A" })).resolves.toBeTruthy()
    await expect(createTenantSchema.validate({ ...valid, display_name: "n".repeat(255) })).resolves.toBeTruthy()
    await expect(createTenantSchema.validate({ ...valid, display_name: "n".repeat(256) })).rejects.toThrow(/exceed 255/i)
  })

  it("accepts every status the backend accepts, including pending", async () => {
    for (const status of ["active", "inactive", "pending", "suspended"]) {
      await expect(createTenantSchema.validate({ ...valid, status })).resolves.toBeTruthy()
    }
  })

  it("rejects a status outside the backend vocabulary", async () => {
    await expect(createTenantSchema.validate({ ...valid, status: "archived" })).rejects.toThrow(/invalid status/i)
  })
})
