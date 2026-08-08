import { describe, it, expect } from "vitest"
import { getPassthroughClientConfig, getClientMetadata } from "./clientConfig"

// The add/update form rebuilds `config` from its own controls and the server replaces
// the blob wholesale, clearing any mirrored column whose key is absent. Anything the
// form does not own therefore has to be resent verbatim or saving revokes it.
describe("getPassthroughClientConfig", () => {
  it("carries through settings the form does not edit", () => {
    const passthrough = getPassthroughClientConfig({
      mtls_bound_cert_thumbprint: "A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8S9t0U1v",
      scope_claim_mappings: { profile: ["name"] },
      claim_mappers: { department: "metadata.dept" },
      id_token_lifetime: 300,
    })

    expect(passthrough).toEqual({
      mtls_bound_cert_thumbprint: "A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8S9t0U1v",
      scope_claim_mappings: { profile: ["name"] },
      claim_mappers: { department: "metadata.dept" },
      id_token_lifetime: 300,
    })
  })

  it("drops every key the form rebuilds", () => {
    const passthrough = getPassthroughClientConfig({
      grant_types: ["authorization_code"],
      response_types: ["code"],
      token_endpoint_auth_method: "none",
      allowed_scopes: ["openid"],
      require_consent: true,
      cors_enabled: true,
      access_token_lifetime: 3600,
      refresh_token_lifetime: 604800,
      refresh_token_rotation: true,
      multi_resource_refresh_token: false,
      required_acr: "2",
      session_idle_timeout: 900,
      session_absolute_timeout: 28800,
      jwks_uri: "https://app.example.com/jwks.json",
      custom: { team: "platform" },
    })

    expect(passthrough).toEqual({})
  })

  // The server resolves `require_pkce` before `pkce_required` and
  // `access_token_lifetime` before `access_token_ttl`. Carrying a stale alias through
  // would override the value the form just wrote.
  it("drops the legacy aliases of form-owned settings", () => {
    const passthrough = getPassthroughClientConfig({
      pkce_required: false,
      require_pkce: false,
      consent_required: false,
      access_token_ttl: 99999,
      refresh_token_ttl: 99999,
      session_idle_timeout_seconds: 60,
      session_absolute_timeout_seconds: 120,
    })

    expect(passthrough).toEqual({})
  })

  // An unrecognized key is already handled: getClientMetadata scoops it up and the
  // metadata editor re-emits it under `custom`. Carrying it through here too would
  // duplicate it AND make a metadata field the operator deleted immortal.
  it("leaves unknown keys to the metadata editor", () => {
    const config = { custom: { team: "platform" }, team_owner: "platform" }

    expect(getClientMetadata(config)).toMatchObject({ team: "platform", team_owner: "platform" })
    expect(getPassthroughClientConfig(config)).toEqual({})
  })
})
