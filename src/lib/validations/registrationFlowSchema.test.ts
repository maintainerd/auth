import { describe, it, expect } from "vitest"
import { registrationFlowSchema } from "./registrationFlowSchema"

const valid = {
  name: "seller-signup",
  description: "Sellers onboard here",
  status: "active",
  clientId: "c-uuid",
  verificationRequired: false,
  requiredFields: [],
  roleIds: [],
}

describe("registrationFlowSchema", () => {
  it("accepts a valid flow", async () => {
    await expect(registrationFlowSchema.validate(valid)).resolves.toMatchObject({ name: "seller-signup" })
  })

  // The name IS the public registration-link selector, so it must be URL-safe.
  it("accepts slug names with hyphens, underscores and digits", async () => {
    for (const name of ["seller", "seller-signup", "seller_signup", "flow2026", "a1-b2_c3"]) {
      await expect(registrationFlowSchema.validate({ ...valid, name })).resolves.toBeTruthy()
    }
  })

  it("rejects names that would not survive a URL", async () => {
    for (const name of ["Seller Signup", "seller signup", "SELLER", "seller:signup", "-seller", "seller-"]) {
      await expect(registrationFlowSchema.validate({ ...valid, name })).rejects.toThrow(/lowercase letters/i)
    }
  })

  it("rejects 'draft' as a status — the backend only accepts active/inactive", async () => {
    await expect(registrationFlowSchema.validate({ ...valid, status: "draft" })).rejects.toThrow(/invalid status/i)
  })

  it("accepts both statuses the backend allows", async () => {
    await expect(registrationFlowSchema.validate({ ...valid, status: "active" })).resolves.toBeTruthy()
    await expect(registrationFlowSchema.validate({ ...valid, status: "inactive" })).resolves.toBeTruthy()
  })

  it("treats description as optional (the backend has no minimum length)", async () => {
    await expect(registrationFlowSchema.validate({ ...valid, description: undefined })).resolves.toBeTruthy()
    await expect(registrationFlowSchema.validate({ ...valid, description: "" })).resolves.toBeTruthy()
    // Previously required a 10-character minimum that the backend never had.
    await expect(registrationFlowSchema.validate({ ...valid, description: "short" })).resolves.toBeTruthy()
  })

  it("caps description at 500 characters", async () => {
    await expect(
      registrationFlowSchema.validate({ ...valid, description: "x".repeat(501) }),
    ).rejects.toThrow(/not exceed 500/i)
  })

  it("requires a name and a client", async () => {
    await expect(registrationFlowSchema.validate({ ...valid, name: "" })).rejects.toThrow(/name is required/i)
    await expect(registrationFlowSchema.validate({ ...valid, clientId: "" })).rejects.toThrow(/client is required/i)
  })

  it("caps name at 100 characters", async () => {
    await expect(registrationFlowSchema.validate({ ...valid, name: "x".repeat(101) })).rejects.toThrow(
      /not exceed 100/i,
    )
  })

  it("only allows the required fields the backend supports", async () => {
    await expect(
      registrationFlowSchema.validate({ ...valid, requiredFields: ["email", "fullname", "phone"] }),
    ).resolves.toBeTruthy()
    await expect(
      registrationFlowSchema.validate({ ...valid, requiredFields: ["ssn"] }),
    ).rejects.toThrow()
  })

  it("has no identifier field — the name is the selector", () => {
    expect(Object.keys(registrationFlowSchema.fields)).not.toContain("identifier")
  })

  it("defaults the flow-behaviour fields so they are always part of the form", () => {
    const defaults = registrationFlowSchema.getDefault()
    expect(defaults.verificationRequired).toBe(false)
    expect(defaults.requiredFields).toEqual([])
    expect(defaults.roleIds).toEqual([])
  })
})
