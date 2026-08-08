import { describe, it, expect } from "vitest"
import {
  clientSchema,
  validateClientOAuthConfig,
  validateRedirectUri,
  isPublicClientType,
  authMethodRequiresSecret,
  CLIENT_AUTH_METHODS,
} from "./clientSchema"

const validBasics = {
  name: "my-client",
  displayName: "My Client App",
  clientType: "spa",
  domain: "app.example.com",
  status: "active",
}

describe("clientSchema basics", () => {
  it("accepts a valid client", async () => {
    await expect(clientSchema.validate(validBasics)).resolves.toBeTruthy()
  })

  it("requires a slug-shaped name", async () => {
    for (const name of ["My Client", "UPPER", "has space", "dot.name"]) {
      await expect(clientSchema.validate({ ...validBasics, name })).rejects.toThrow(/lowercase/i)
    }
  })

  // The domain becomes the token issuer and is compared in the private_key_jwt
  // audience check, so free text there is load-bearing. Backend allows 253 and
  // requires a host shape.
  it("validates the domain as a host or https URL, up to 253 chars", async () => {
    for (const domain of ["app.example.com", "https://app.example.com", "localhost:3000"]) {
      await expect(clientSchema.validate({ ...validBasics, domain })).resolves.toBeTruthy()
    }
    await expect(clientSchema.validate({ ...validBasics, domain: "has space" })).rejects.toThrow(
      /hostname or an https URL/i,
    )
    await expect(
      clientSchema.validate({ ...validBasics, domain: "a".repeat(254) }),
    ).rejects.toThrow(/not exceed 253/i)
  })

  it("rejects an unknown client type and status", async () => {
    await expect(clientSchema.validate({ ...validBasics, clientType: "public" })).rejects.toThrow()
    await expect(clientSchema.validate({ ...validBasics, status: "draft" })).rejects.toThrow()
  })
})

describe("client auth method helpers", () => {
  it("treats only spa and mobile as public", () => {
    expect(isPublicClientType("spa")).toBe(true)
    expect(isPublicClientType("mobile")).toBe(true)
    expect(isPublicClientType("traditional")).toBe(false)
    expect(isPublicClientType("m2m")).toBe(false)
    // Unknown must not be treated as public — that is the permissive direction.
    expect(isPublicClientType(undefined)).toBe(false)
    expect(isPublicClientType("")).toBe(false)
  })

  it("knows which methods need a secret", () => {
    expect(authMethodRequiresSecret("client_secret_basic")).toBe(true)
    expect(authMethodRequiresSecret("client_secret_post")).toBe(true)
    expect(authMethodRequiresSecret("client_secret_jwt")).toBe(true)
    expect(authMethodRequiresSecret("none")).toBe(false)
    expect(authMethodRequiresSecret("private_key_jwt")).toBe(false)
  })

  // The backend refuses these at write time because no certificate-binding path
  // exists, so offering them would only produce a client that cannot authenticate.
  it("does not offer the unimplemented mTLS methods", () => {
    expect(CLIENT_AUTH_METHODS).not.toContain("tls_client_auth")
    expect(CLIENT_AUTH_METHODS).not.toContain("self_signed_tls_client_auth")
  })
})

// Mirrors ValidateClientOAuthMatrix. These combinations are individually legal and
// jointly unsafe or unusable.
describe("validateClientOAuthConfig", () => {
  it("accepts a public client using none with authorization_code", () => {
    const errors = validateClientOAuthConfig({
      clientType: "spa",
      tokenEndpointAuthMethod: "none",
      grantTypes: ["authorization_code"],
    })
    expect(errors).toEqual({})
  })

  it("accepts an m2m client authenticating with a secret and declared scopes", () => {
    const errors = validateClientOAuthConfig({
      clientType: "m2m",
      tokenEndpointAuthMethod: "client_secret_basic",
      grantTypes: ["client_credentials"],
      allowedScopes: ["api:read"],
    })
    expect(errors).toEqual({})
  })

  // The exploit: client_id is public, so "no credential" here means anyone can
  // mint this client's tokens.
  it("rejects none on a confidential client", () => {
    for (const clientType of ["traditional", "m2m"]) {
      const errors = validateClientOAuthConfig({
        clientType,
        tokenEndpointAuthMethod: "none",
        grantTypes: ["client_credentials"],
        allowedScopes: ["api:read"],
      })
      expect(errors.tokenEndpointAuthMethod).toMatch(/only public clients/i)
    }
  })

  it("rejects a secret-based method on a public client", () => {
    const errors = validateClientOAuthConfig({
      clientType: "spa",
      tokenEndpointAuthMethod: "client_secret_basic",
      grantTypes: ["authorization_code"],
    })
    expect(errors.tokenEndpointAuthMethod).toMatch(/cannot keep a secret/i)
  })

  it("rejects client_credentials on a public client", () => {
    const errors = validateClientOAuthConfig({
      clientType: "spa",
      tokenEndpointAuthMethod: "none",
      grantTypes: ["client_credentials"],
      allowedScopes: ["api:read"],
    })
    expect(errors.grantTypes).toMatch(/not valid for a public client/i)
  })

  it("rejects authorization_code on m2m", () => {
    const errors = validateClientOAuthConfig({
      clientType: "m2m",
      tokenEndpointAuthMethod: "client_secret_basic",
      grantTypes: ["authorization_code"],
    })
    expect(errors.grantTypes).toMatch(/no user to authorize/i)
  })

  it("requires explicit scopes for client_credentials", () => {
    const errors = validateClientOAuthConfig({
      clientType: "m2m",
      tokenEndpointAuthMethod: "client_secret_basic",
      grantTypes: ["client_credentials"],
      allowedScopes: [],
    })
    expect(errors.allowedScopes).toMatch(/declare its allowed scopes/i)
  })

  it("requires at least one grant type", () => {
    const errors = validateClientOAuthConfig({
      clientType: "spa",
      tokenEndpointAuthMethod: "none",
      grantTypes: [],
    })
    expect(errors.grantTypes).toMatch(/at least one grant type/i)
  })

  // Mirrors chk_clients_token_ttl_order — a refresh token shorter than the access
  // token can never be used.
  it("requires the refresh token to outlive the access token", () => {
    const errors = validateClientOAuthConfig({
      clientType: "spa",
      tokenEndpointAuthMethod: "none",
      grantTypes: ["authorization_code"],
      accessTokenTtl: 3600,
      refreshTokenTtl: 60,
    })
    expect(errors.refreshTokenTtl).toMatch(/greater than or equal/i)
  })

  it("requires the absolute session timeout to outlive the idle timeout", () => {
    const errors = validateClientOAuthConfig({
      clientType: "spa",
      tokenEndpointAuthMethod: "none",
      grantTypes: ["authorization_code"],
      sessionIdleTimeout: 3600,
      sessionAbsoluteTimeout: 600,
    })
    expect(errors.sessionAbsoluteTimeout).toMatch(/greater than or equal/i)
  })

  it("requires an endpoint when back-channel logout session is required", () => {
    const errors = validateClientOAuthConfig({
      clientType: "spa",
      tokenEndpointAuthMethod: "none",
      grantTypes: ["authorization_code"],
      backchannelLogoutSessionRequired: true,
    })
    expect(errors.backchannelLogoutSessionRequired).toMatch(/needs a back-channel logout URI/i)
  })

  it("surfaces an invalid redirect URI", () => {
    const errors = validateClientOAuthConfig({
      clientType: "spa",
      tokenEndpointAuthMethod: "none",
      grantTypes: ["authorization_code"],
      redirectUris: ["http://app.example.com/cb"],
    })
    expect(errors.redirectUris).toMatch(/must use https/i)
  })
})

// Mirrors ValidateRegisteredRedirectURI. The rules differ per client type because
// a native app legitimately receives the response on a private-use scheme.
describe("validateRedirectUri", () => {
  it("accepts https for any client type", () => {
    for (const t of ["traditional", "spa", "mobile", "m2m"]) {
      expect(validateRedirectUri(t, "https://app.example.com/cb")).toBeNull()
    }
  })

  it("accepts http only on loopback", () => {
    expect(validateRedirectUri("spa", "http://127.0.0.1:3000/cb")).toBeNull()
    expect(validateRedirectUri("spa", "http://app.example.com/cb")).toMatch(/must use https/i)
  })

  it("rejects a fragment, credentials and relative URIs", () => {
    expect(validateRedirectUri("spa", "https://app.example.com/cb#x")).toMatch(/fragment/i)
    expect(validateRedirectUri("spa", "https://u:p@app.example.com/cb")).toMatch(/credentials/i)
    expect(validateRedirectUri("spa", "/callback")).toMatch(/absolute/i)
  })

  it("rejects code-executing schemes", () => {
    expect(validateRedirectUri("spa", "javascript:alert(1)")).toMatch(/javascript/i)
    expect(validateRedirectUri("spa", "data:text/html,x")).toMatch(/data:/i)
  })

  it("allows a reverse-domain custom scheme only for mobile", () => {
    expect(validateRedirectUri("mobile", "com.example.app:/oauth")).toBeNull()
    expect(validateRedirectUri("spa", "com.example.app:/oauth")).toMatch(/only allowed for mobile/i)
    // RFC 8252 §7.1 — a bare scheme can collide with another app's.
    expect(validateRedirectUri("mobile", "myapp:/oauth")).toMatch(/reverse-domain/i)
  })

  it("requires a value", () => {
    expect(validateRedirectUri("spa", "  ")).toMatch(/required/i)
  })
})

// Mirrors validateAdvancedClientConfig and the private_key_jwt arm of the matrix.
// The server refuses these writes, so catching them here saves a round trip — and a
// client that looked saved but could never authenticate.
describe("client keys", () => {
  const base = {
    clientType: "traditional",
    tokenEndpointAuthMethod: "private_key_jwt",
    grantTypes: ["authorization_code"],
  }
  const publicJwks = JSON.stringify({
    keys: [{ kty: "RSA", kid: "a", n: "0vx7", e: "AQAB" }],
  })

  it("accepts either a JWKS URL or an inline JWK Set", () => {
    expect(
      validateClientOAuthConfig({ ...base, jwksUri: "https://app.example.com/jwks.json" }),
    ).toEqual({})
    expect(validateClientOAuthConfig({ ...base, jwks: publicJwks })).toEqual({})
  })

  // RFC 7591 §2 — with both set, which source verifies an assertion depends on
  // lookup order rather than intent.
  it("rejects both key sources at once", () => {
    const errors = validateClientOAuthConfig({
      ...base,
      jwks: publicJwks,
      jwksUri: "https://app.example.com/jwks.json",
    })
    expect(errors.jwks).toMatch(/not both/i)
  })

  it("requires keys for private_key_jwt", () => {
    const errors = validateClientOAuthConfig(base)
    expect(errors.jwks).toMatch(/public keys/i)
  })

  it("does not require keys for other auth methods", () => {
    expect(
      validateClientOAuthConfig({
        clientType: "spa",
        tokenEndpointAuthMethod: "none",
        grantTypes: ["authorization_code"],
      }),
    ).toEqual({})
  })

  it("validates the JWK Set shape", () => {
    expect(validateClientOAuthConfig({ ...base, jwks: "{not json" }).jwks).toMatch(/valid JSON/i)
    expect(validateClientOAuthConfig({ ...base, jwks: "[]" }).jwks).toMatch(/"keys" array/i)
    expect(validateClientOAuthConfig({ ...base, jwks: '{"keys":[]}' }).jwks).toMatch(/non-empty/i)
    expect(validateClientOAuthConfig({ ...base, jwks: '{"keys":[{"kid":"a"}]}' }).jwks).toMatch(
      /"kty"/i,
    )
  })

  // A JWK Set is the client's PUBLIC keys. A private component means the operator is
  // handing us the signing key — a credential leak, and never needed to verify.
  it("rejects a private key component", () => {
    for (const component of ["d", "p", "q", "dp", "dq", "qi", "k"]) {
      const jwks = JSON.stringify({ keys: [{ kty: "RSA", n: "0vx7", e: "AQAB", [component]: "x" }] })
      expect(validateClientOAuthConfig({ ...base, jwks }).jwks).toMatch(/private key component/i)
    }
  })

  // The keys served from this URL decide whether an assertion is accepted, so the
  // fetch has to be authenticated and tamper-proof.
  it("requires the JWKS URL to be absolute https with no fragment", () => {
    expect(
      validateClientOAuthConfig({ ...base, jwksUri: "http://app.example.com/jwks.json" }).jwksUri,
    ).toMatch(/https/i)
    expect(validateClientOAuthConfig({ ...base, jwksUri: "/jwks.json" }).jwksUri).toMatch(
      /absolute/i,
    )
    expect(
      validateClientOAuthConfig({ ...base, jwksUri: "https://app.example.com/jwks.json#k" })
        .jwksUri,
    ).toMatch(/fragment/i)
  })
})
