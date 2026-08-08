import { describe, it, expect } from "vitest"
import { buildRegistrationUrl } from "./utils"

describe("buildRegistrationUrl", () => {
  it("composes the identity /register URL from the client identifier and flow name", () => {
    expect(
      buildRegistrationUrl("https://auth.example.com", "storefront-abc123", "seller-signup-k3f9qz7lm2xb8vrt"),
    ).toBe(
      "https://auth.example.com/register?client_id=storefront-abc123&registration_flow=seller-signup-k3f9qz7lm2xb8vrt",
    )
  })

  it("uses the client's OAuth identifier, never its UUID", () => {
    // The backend resolves ?client_id= with clientRepo.FindByIdentifier, so a UUID
    // here would resolve no client and the registration would be rejected.
    const url = buildRegistrationUrl(
      "https://auth.example.com",
      "storefront-abc123",
      "seller-signup-aaa",
    )
    const params = new URL(url).searchParams
    expect(params.get("client_id")).toBe("storefront-abc123")
    expect(params.get("client_id")).not.toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i,
    )
  })

  it("tolerates a trailing slash on the identity origin", () => {
    expect(buildRegistrationUrl("https://auth.example.com/", "c", "f")).toBe(
      "https://auth.example.com/register?client_id=c&registration_flow=f",
    )
  })

  it("percent-encodes values rather than emitting a broken URL", () => {
    const url = buildRegistrationUrl("https://auth.example.com", "a b&c", "flow=1")
    expect(url).toBe("https://auth.example.com/register?client_id=a+b%26c&registration_flow=flow%3D1")
    expect(new URL(url).searchParams.get("client_id")).toBe("a b&c")
  })
})
